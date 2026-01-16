既然我们要把它设计成一个部署在集群内的**“镜像预热服务（Image Pre-warming Service, IPS）”**，我们需要摒弃简单的“循环脚本”思维，转而采用 **Kubernetes Operator 模式** 或 **工作流引擎模式**。

这个服务不再是一个简单的 CronJob，而是一个具备**状态管理、容错能力、可观测性**的控制系统。

以下是基于 **Kubernetes Native** 理念设计的系统架构规划：

---

### 一、 核心架构理念：Controller-Worker 模型

我们将系统分为两层：**控制面（Control Plane）** 和 **数据面（Data Plane）**。

*   **控制面**：一个轻量级的 Controller（服务本体），负责接收请求、快照节点状态、分片计算、下发任务、监控进度。
*   **数据面**：一次性的、短周期的 K8s Job 或裸 Pod，负责实际的镜像拉取（干苦力）。

### 二、 模块划分与组件设计

我们将服务拆解为四个核心模块：

#### 1. 任务定义层 (API / CRD)
我们不需要开发复杂的 REST API Server，直接利用 Kubernetes 的 **CRD (Custom Resource Definition)** 作为交互接口。

*   **设计 `PrewarmTask` CRD**：
    *   `spec.images`: 需要预热的镜像列表。
    *   `spec.batchSize`: 并发批次大小（如 10）。
    *   `spec.nodeSelector`: 目标节点筛选器（支持 Label, Zone）。
    *   `spec.strategy`: 失败策略（Skip/Retry）。
    *   `spec.timeout`: 单节点超时时间。

#### 2. 调度核心 (The Brain: Controller)
这是原来的“脚本”逻辑的容器化升级版。它包含三个子组件：

*   **Node Snapshotter (节点快照器)**：
    *   当一个新 Task 进来时，它**不实时**去查 Node，而是在 Task 初始化的瞬间，拉取当前所有 Ready 节点，生成一个**静态列表**。
    *   **作用**：锁定预热范围。即使任务执行期间有新节点加入，也不会干扰当前批次；如果有节点挂了，根据静态列表去处理。
*   **Sharding Engine (分片引擎)**：
    *   输入：静态节点列表 + `batchSize`。
    *   输出：一个队列 `[[Node1, Node2...], [Node5, Node6...], ...]`。
    *   **作用**：实现了你想要的“上帝视角静态分片”。
*   **Task Dispatcher (任务派发器)**：
    *   负责从分片队列中取出一批，生成 Job，绑定 `nodeName`，提交给 APIServer。
    *   实现**栅栏（Barrier）机制**：监控当前批次的 Job 状态，只有满足条件（全部成功 或 达到超时）才解锁下一批。

#### 3. 执行单元 (The Worker: Agentless Jobs)
为了保持轻量，我们**不**在节点上部署 DaemonSet Agent。
*   **实现**：使用极其精简的 `Job`。
*   **内容**：只包含 `initContainers`（拉镜像）和极简的主容器（退出）。
*   **特性**：
    *   `nodeName`: 被 Controller 强行锁定。
    *   `activeDeadlineSeconds`: 被 Controller 强行注入，防止死锁。

#### 4. 状态观测器 (Observer & Reaper)
*   **Status Aggregator**：实时更新 CRD 的 `status` 字段。
    *   `status.phase`: Running / Completed / Failed
    *   `status.progress`: 45/100 Nodes
    *   `status.failedNodes`: [Node-X, Node-Y] (记录下来供运维排查)
*   **The Reaper (收割者)**：
    *   专门负责清理垃圾。任务完成后，删除生成的 100 个 Job，不留痕迹。
    *   处理“僵尸 Job”：如果 Controller 重启了，Reaper 启动时要能接管或清理之前遗留的 Job。

---

### 三、 详细工作流设计 (State Machine)

我们可以将整个过程看作一个**状态机**的流转：

1.  **Pending (挂起)**：
    *   用户提交 YAML (`PrewarmTask`)。
    *   Validating Webhook 校验参数合法性。

2.  **Snapshotting (快照中)**：
    *   Controller 锁定任务。
    *   拉取 Node List，过滤 NotReady/Unschedulable 节点。
    *   **关键点**：如果此时发现节点数过多（如 5000+），进行熔断保护或强制降级分批。
    *   生成内部的分片队列，存入内存或更新到 CRD Status 中（以便 Controller 重启后能恢复进度）。

3.  **Running (执行中)**：
    *   **Loop Start**:
        *   取出 Batch N。
        *   并发创建 Batch N 个 Jobs (指定 `nodeName`)。
    *   **Wait**:
        *   Controller 进入 Reconcile 循环，轮询这 N 个 Job 的 Pod 状态。
    *   **Analyze**:
        *   如果 Job 成功 -> 计数器 +1。
        *   如果 Job 失败/超时 -> 记录到 `failedNodes` 列表，**不重试**（Fail Fast 设计原则），继续推进。
    *   **Next**:
        *   如果本批次结束，Loop 回到 Start，取 Batch N+1。

4.  **Finalizing (收尾)**：
    *   所有批次跑完。
    *   触发 Reaper 清理所有 Job 资源。
    *   更新 CRD 状态为 `Completed`。
    *   发送 Kubernetes Event（通知 AlertManager 或钉钉/Slack）。

---

### 四、 针对核心问题的架构回应

#### 1. 如何规避坏节点？
*   **设计**：在 **Snapshotting** 阶段引入**健康度检查插件**。
*   **逻辑**：除了检查 `Ready` 状态，还可以检查 Prometheus 的指标（如磁盘使用率）。如果某节点 `disk_usage > 90%`，Snapshotter 直接将其剔除，并记录在 `ignoredNodes` 中。

#### 2. 如何保证控制面高可用与数据一致性？
*   **设计**：利用 CRD 的 `status` 存储分片游标（Cursor）。
*   **逻辑**：比如记录 `status.currentBatchIndex: 5`。如果 Controller 崩溃重启，它读取 CRD，发现跑到了第 5 批，于是从第 5 批继续，而不是从头开始。

#### 3. 如何处理超大规模（1000+节点）？
*   **设计**：引入**流式处理**而非全量生成。
*   **逻辑**：不要一次性创建 1000 个 Job 对象（会给 ETCD 造成压力）。Controller 应该严格遵守 `batchSize`，创建 -> 等待 -> 删除 -> 再创建。保持系统中同时存在的 Job 数不超过 `batchSize`。

---

### 五、 架构图示 (Mermaid Conceptual View)

```mermaid
graph TD
    User[DevOps Engineer] -->|kubectl apply| API[K8s API Server]
    API -->|Watch Event| Ctrl[**Prewarm Controller**]
    
    subgraph "Prewarm Service (In-Cluster)"
        Ctrl -->|1. List & Filter| NodeMgr[**Node Snapshotter**]
        NodeMgr -->|Healthy Nodes List| Sharding[**Sharding Engine**]
        Sharding -->|Batch Queue| Dispatcher[**Task Dispatcher**]
        
        Dispatcher -->|2. Create Job (pinned)| Job1[Job @ Node A]
        Dispatcher -->|2. Create Job (pinned)| Job2[Job @ Node B]
        
        Observer[**Status Observer**] -.->|3. Watch Jobs| API
        Observer -->|4. Update Status| Dispatcher
    end
    
    subgraph "K8s Cluster Nodes"
        NodeA[Node A]
        NodeB[Node B]
        Job1 -->|Pull Image| NodeA
        Job2 -->|Pull Image| NodeB
    end
```

### 六、 总结架构师建议

要实现这个服务，你不需要引入复杂的微服务架构，只需要写一个**标准的 Kubernetes Operator**。

**技术栈推荐**：
*   **语言**：Go (Kubebuilder / Operator SDK) —— 这是 K8s 一等公民，处理并发和 API 最顺手。
*   **存储**：直接依赖 K8s ETCD (CRD Status)，无需外部数据库。
*   **部署**：Deployment (1 Replica) + RBAC。

**核心价值**：
这套架构将“人肉运维脚本”升级为了“声明式基础设施能力”。用户只需要告诉集群“我要这 20 个镜像在所有计算节点就绪”，剩下的分片、容错、重试、清理，全部由控制器自动完成。这就是**平台工程（Platform Engineering）**的精髓。