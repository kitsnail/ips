# IPS Web UI - Vue 3 + TypeScript + Vite Migration

## Phase 1 完成总结

### 已完成的工作

1. **初始化Vite项目** ✓
   - 使用 `npm create vite@latest . -- --template vue-ts` 创建项目
   - 安装依赖：axios, element-plus, @element-plus/icons-vue, pinia, vue-router
   - 总依赖包：141个（安全审计通过）

2. **配置TypeScript和Vite** ✓
   - `vite.config.ts`：
     - 配置路径别名 `@/` → `./src/`
     - 开发服务器代理：`/api` → `http://localhost:8080`
     - 构建输出：`../web/static/dist`（直接输出到Go服务器的静态目录）
     - Element Plus单独chunk打包优化
   - `tsconfig.app.json`：添加路径别名配置

3. **创建TypeScript接口定义** ✓
   - `src/types/api.ts`：定义所有API接口和类型
     - User相关：LoginRequest, User, CreateUserRequest
     - Task相关：Task, CreateTaskRequest, ListTasksRequest
     - ScheduledTask相关：ScheduledTask, ScheduledExecution
     - Library相关：LibraryImage, SaveImageRequest
     - Secret相关：Secret, CreateSecretRequest
     - 通用类型：PaginationState, FilterState

4. **创建API服务层** ✓
   - `src/services/api.ts`：封装所有API调用
     - 使用axios创建HTTP客户端
     - 自动注入JWT token到请求头
     - 401错误自动跳转登录页
     - 模块化API：authApi, taskApi, scheduledTaskApi, libraryApi, secretApi, userApi

5. **创建Vue Composables** ✓
   - `src/composables/useAuth.ts`：用户认证状态管理
   - `src/composables/useToast.ts`：Toast通知管理（未使用，可用）

6. **创建基础视图和路由** ✓
   - `src/main.ts`：应用入口，集成Element Plus和图标
   - `src/router/index.ts`：路由配置（dashboard, tasks）
   - `src/App.vue`：根组件，包含登录界面和路由守卫
   - `src/views/DashboardView.vue`：Dashboard页面（显示最近5个任务）
   - `src/views/TasksView.vue`：任务管理页面（显示任务列表）

### 构建结果

```
✓ built in 2.64s
index.html: 0.54 kB (gzip: 0.32 kB)
index.css: 349.95 kB (gzip: 47.42 kB) - 主要是Element Plus样式
index.js: 67.26 kB (gzip: 26.56 kB) - 应用代码
element-plus.js: 1,120.95 kB (gzip: 357.91 kB) - Element Plus库
```

**构建输出目录**：`web/static/dist/`

### 技术栈选择说明

- **Vue 3**：渐进式框架，可以逐步迁移，学习曲线平缓
- **TypeScript**：提供类型安全，减少运行时错误
- **Vite**：快速的开发服务器和优化的生产构建
- **Element Plus**：成熟的Vue组件库，适合管理后台
- **Axios**：HTTP客户端，支持拦截器和请求取消
- **Pinia**：状态管理（已安装，暂未使用）
- **Vue Router**：路由管理（基础配置完成）

### 后续渐进式迁移计划

#### Phase 2: 功能完善（当前任务）
- [ ] 创建通用UI组件（Toast, Modal等）
- [ ] 实现完整的Dashboard统计
- [ ] 实现任务CRUD完整功能
- [ ] 实现定时任务管理
- [ ] 实现镜像库管理
- [ ] 实现仓库认证管理
- [ ] 实现用户管理（管理员）

#### Phase 3: 渐进式替换现有页面
- [ ] 逐个替换 `web/static/index.html` 中的功能模块
- [ ] 逐步迁移样式（CSS变量保持一致）
- [ ] 测试每个模块的功能完整性

#### Phase 4: 清理和优化
- [ ] 删除旧的Vanilla JS代码
- [ ] 优化bundle大小（code splitting）
- [ ] 添加单元测试
- [ ] 性能优化

### 开发指南

#### 启动开发服务器
```bash
cd frontend
npm run dev
```
访问：http://localhost:5173

#### 构建生产版本
```bash
cd frontend
npm run build
```
输出到：`web/static/dist/`

#### 运行Go服务器（开发模式）
```bash
make run
```
Web UI访问：http://localhost:8080/web/

#### 运行Go服务器（生产模式）
构建后，Go服务器会自动使用 `web/static/dist/index.html`

### 已知问题

1. **Element Plus bundle较大**：1.12MB（gzip 358KB）
   - 方案1：按需导入（需要配置unplugin-vue-components）
   - 方案2：选择更轻量的组件库
   - 方案3：当前可接受，功能优先

2. **TypeScript类型定义文件较大**（270行）
   - 可选：拆分为多个文件（types/user.ts, types/task.ts等）

### 文件结构

```
frontend/
├── src/
│   ├── assets/          # 静态资源
│   ├── components/       # Vue组件
│   ├── composables/      # Vue组合式函数
│   │   ├── useAuth.ts
│   │   └── useToast.ts
│   ├── router/          # 路由配置
│   │   └── index.ts
│   ├── services/        # API服务
│   │   └── api.ts
│   ├── types/           # TypeScript类型定义
│   │   └── api.ts
│   ├── utils/           # 工具函数
│   ├── views/           # 页面组件
│   │   ├── DashboardView.vue
│   │   └── TasksView.vue
│   ├── App.vue          # 根组件
│   └── main.ts          # 入口文件
├── index.html
├── package.json
├── tsconfig.json
├── tsconfig.app.json
├── tsconfig.node.json
├── vite.config.ts
└── ...

web/static/
├── dist/               # Vite构建输出（Go服务器使用此目录）
│   ├── index.html
│   ├── vite.svg
│   └── assets/
├── index.html           # 旧的Vanilla JS版本（待删除）
└── app.js              # 旧的Vanilla JS版本（待删除）
```

### 兼容性说明

- 构建输出到 `web/static/dist/`，不影响现有的 `web/static/index.html`
- 可以并行运行新旧版本进行对比测试
- Go服务器需要更新路由以支持新版本（或直接替换）
