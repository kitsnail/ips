// API 基础地址
const API_BASE = '/api/v1';

// 全局状态
let tasks = [];
let currentTaskId = null;
let autoRefreshInterval = null;

// 初始化
document.addEventListener('DOMContentLoaded', function () {
    refreshTasks();
    // 每5秒自动刷新
    autoRefreshInterval = setInterval(refreshTasks, 5000);
});

// 刷新任务列表
async function refreshTasks() {
    try {
        const response = await fetch(`${API_BASE}/tasks`);
        const data = await response.json();
        tasks = data.tasks || [];
        renderTasks();
    } catch (error) {
        console.error('Failed to fetch tasks:', error);
        document.getElementById('taskList').innerHTML =
            '<div class="empty-state">加载失败，请检查网络连接</div>';
    }
}

// 渲染任务列表
function renderTasks() {
    const taskList = document.getElementById('taskList');
    const statusFilter = document.getElementById('statusFilter').value;

    // 过滤任务
    let filteredTasks = tasks;
    if (statusFilter) {
        filteredTasks = tasks.filter(t => t.status === statusFilter);
    }

    if (!filteredTasks || filteredTasks.length === 0) {
        taskList.innerHTML = '<div class="empty-state">暂无任务</div>';
        return;
    }

    // 按创建时间倒序排列
    filteredTasks.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));

    taskList.innerHTML = filteredTasks.map(task => `
        <div class="task-item" onclick="showTaskDetail('${task.taskId}')">
            <div class="task-header">
                <div class="task-id">${task.taskId}</div>
                <div class="task-status status-${task.status}">${getStatusText(task.status)}</div>
            </div>
            <div class="task-info">
                <span><strong>镜像数:</strong> ${task.images.length}</span>
                <span><strong>批次大小:</strong> ${task.batchSize}</span>
                <span><strong>优先级:</strong> ${task.priority}</span>
                <span><strong>创建时间:</strong> ${formatTime(task.createdAt)}</span>
                ${task.startedAt ? `<span><strong>开始时间:</strong> ${formatTime(task.startedAt)}</span>` : ''}
            </div>
            ${renderProgress(task)}
        </div>
    `).join('');
}

// 渲染进度条
function renderProgress(task) {
    if (!task.progress) return '';

    const percentage = task.progress.percentage || 0;
    return `
        <div class="progress-bar">
            <div class="progress-fill" style="width: ${percentage}%"></div>
        </div>
        <div class="task-info" style="margin-top: 4px; font-size: 12px;">
            <span>进度: ${percentage.toFixed(1)}%</span>
            <span>完成: ${task.progress.completedNodes}/${task.progress.totalNodes}</span>
            ${task.progress.failedNodes > 0 ? `<span style="color: #cf1322;">失败: ${task.progress.failedNodes}</span>` : ''}
            <span>批次: ${task.progress.currentBatch}/${task.progress.totalBatches}</span>
        </div>
    `;
}

// 显示任务详情
async function showTaskDetail(taskId) {
    try {
        const response = await fetch(`${API_BASE}/tasks/${taskId}`);
        const task = await response.json();
        currentTaskId = taskId;

        const detailHtml = `
            <!-- 状态统计板 -->
            <div class="stats-grid">
                <div class="stat-card">
                    <div class="stat-label">任务状态</div>
                    <div class="stat-value info" style="font-size: 18px;">
                        <span class="task-status status-${task.status}">${getStatusText(task.status)}</span>
                    </div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">总体进度</div>
                    <div class="stat-value info">${task.progress ? task.progress.percentage.toFixed(1) : 0}%</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">节点 (完成/总数)</div>
                    <div class="stat-value success">${task.progress ? task.progress.completedNodes : 0} / ${task.progress ? task.progress.totalNodes : 0}</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">异常节点</div>
                    <div class="stat-value failed">${task.progress ? task.progress.failedNodes : 0}</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">当前批次</div>
                    <div class="stat-value">${task.progress ? task.progress.currentBatch : 0} / ${task.progress ? task.progress.totalBatches : 0}</div>
                </div>
            </div>

            <!-- 元数据详情 -->
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 24px; margin-bottom: 24px;">
                <div class="config-pane" style="border: none; padding: 0;">
                    <div class="detail-row">
                        <span class="detail-label">任务ID:</span>
                        <span class="detail-value">${task.taskId}</span>
                    </div>
                    <div class="detail-row">
                        <span class="detail-label">创建时间:</span>
                        <span class="detail-value">${formatTime(task.createdAt)}</span>
                    </div>
                    <div class="detail-row">
                        <span class="detail-label">重试情况:</span>
                        <span class="detail-value">${task.retryCount} / ${task.maxRetries} (策略: ${task.retryStrategy === 'exponential' ? '指数' : '线性'})</span>
                    </div>
                </div>
                <div class="config-pane" style="border: none; padding: 0;">
                    <div class="detail-row">
                        <span class="detail-label">镜像列表:</span>
                        <div class="detail-value" style="word-break: break-all; font-size: 12px; max-height: 60px; overflow-y: auto; background: #fafafa; padding: 8px; border-radius: 4px;">
                            ${task.images.join('<br>')}
                        </div>
                    </div>
                </div>
            </div>

            <div class="progress-bar" style="height: 10px; margin-bottom: 24px;">
                <div class="progress-fill" style="width: ${task.progress ? task.progress.percentage : 0}%"></div>
            </div>

            ${renderNodeStatuses(task.nodeStatuses)}
            ${task.failedNodeDetails && task.failedNodeDetails.length > 0 ? renderFailedNodes(task.failedNodeDetails) : ''}
        `;

        document.getElementById('taskDetail').innerHTML = detailHtml;

        // 显示/隐藏取消按钮
        const cancelBtn = document.getElementById('cancelTaskBtn');
        const refreshDetailBtn = document.getElementById('refreshDetailBtn') || createRefreshDetailBtn();

        if (task.status === 'pending' || task.status === 'running') {
            cancelBtn.style.display = 'inline-block';
        } else {
            cancelBtn.style.display = 'none';
        }

        if (refreshDetailBtn) {
            refreshDetailBtn.onclick = () => showTaskDetail(taskId);
        }

        document.getElementById('detailModal').classList.add('show');
    } catch (error) {
        console.error('Failed to fetch task detail:', error);
        // 不弹窗，静默失败或在详情区显示错误
    }
}

// 创建详情刷新按钮（如果不存在）
function createRefreshDetailBtn() {
    // index.html 中使用的是 .form-actions 而不是 .modal-footer
    const footer = document.querySelector('#detailModal .form-actions');
    if (!footer) return null;

    // 检查是否已经有这个按钮
    let btn = document.getElementById('refreshDetailBtn');
    if (btn) return btn;

    btn = document.createElement('button');
    btn.id = 'refreshDetailBtn';
    btn.className = 'btn btn-primary';
    btn.innerText = '刷新仪表盘';
    btn.style.marginRight = '8px';
    footer.insertBefore(btn, footer.firstChild);
    return btn;
}

// 渲染节点镜像状态
function renderNodeStatuses(nodeStatuses) {
    if (!nodeStatuses || Object.keys(nodeStatuses).length === 0) {
        return `
            <div class="empty-state" style="margin-top: 16px; padding: 30px; border: 1px dashed #d9d9d9; background: #fafafa;">
                <div style="font-size: 24px; margin-bottom: 8px;">🕒</div>
                暂无节点详细镜像状态（可能正在收集或 Pod 已过期）
            </div>
        `;
    }

    const rows = Object.entries(nodeStatuses).map(([nodeName, images]) => {
        const imageTags = Object.entries(images).map(([image, status]) => {
            const className = status === 1 ? 'image-tag-success' : 'image-tag-failed';
            const label = status === 1 ? '成功' : '失败';
            return `<span class="${className}" style="display: inline-block; margin-bottom: 4px;">${image} (${label})</span>`;
        }).join(' ');

        return `
            <tr>
                <td class="node-name-cell" style="vertical-align: top;">
                    <div style="font-weight: 600;">${nodeName}</div>
                    <div style="font-size: 11px; color: #999;">Node Status</div>
                </td>
                <td>${imageTags}</td>
            </tr>
        `;
    }).join('');

    return `
        <div style="margin-top: 24px;">
            <div style="font-size: 16px; font-weight: 600; margin-bottom: 16px; display: flex; align-items: center; gap: 8px;">
                <span style="width: 4px; height: 16px; background: #1890ff; border-radius: 2px;"></span>
                节点镜像拉取详情
            </div>
            <table class="node-status-table">
                <thead>
                    <tr>
                        <th style="width: 250px;">节点名称</th>
                        <th>镜像执行结果 (每个镜像的拉取结果)</th>
                    </tr>
                </thead>
                <tbody>
                    ${rows}
                </tbody>
            </table>
        </div>
    `;
}

// 渲染失败节点
function renderFailedNodes(failedNodes) {
    return `
        <div class="failed-nodes">
            <div style="font-weight: 500; margin-bottom: 8px;">失败节点详情:</div>
            ${failedNodes.map(node => `
                <div class="failed-node-item">
                    <div><strong>节点:</strong> ${node.nodeName}</div>
                    <div><strong>原因:</strong> ${node.reason}</div>
                    ${node.message ? `<div><strong>消息:</strong> ${node.message}</div>` : ''}
                    <div><strong>时间:</strong> ${formatTime(node.timestamp)}</div>
                </div>
            `).join('')}
        </div>
    `;
}

// 创建任务
async function createTask(event) {
    event.preventDefault();

    const images = document.getElementById('images').value.trim().split('\n').filter(i => i.trim());
    const batchSize = parseInt(document.getElementById('batchSize').value);
    const priority = parseInt(document.getElementById('priority').value);
    const maxRetries = parseInt(document.getElementById('maxRetries').value);
    const retryStrategy = document.getElementById('retryStrategy').value;
    const retryDelay = parseInt(document.getElementById('retryDelay').value);
    const webhookUrl = document.getElementById('webhookUrl').value.trim();
    const nodeSelectorStr = document.getElementById('nodeSelector').value.trim();

    let nodeSelector = null;
    if (nodeSelectorStr) {
        try {
            nodeSelector = JSON.parse(nodeSelectorStr);
        } catch (e) {
            alert('节点选择器格式错误，请输入有效的JSON');
            return;
        }
    }

    const requestBody = {
        images,
        batchSize,
        priority,
        maxRetries,
        retryStrategy,
        retryDelay
    };

    if (webhookUrl) {
        requestBody.webhookUrl = webhookUrl;
    }

    if (nodeSelector) {
        requestBody.nodeSelector = nodeSelector;
    }

    try {
        const response = await fetch(`${API_BASE}/tasks`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestBody)
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || '创建失败');
        }

        const task = await response.json();
        // 移除弹窗，直接刷新列表并隐藏模态框
        console.log(`任务创建成功！任务ID: ${task.taskId}`);
        hideCreateTaskModal();
        document.getElementById('createTaskForm').reset();
        refreshTasks();
    } catch (error) {
        console.error('Failed to create task:', error);
        alert('创建任务失败: ' + error.message);
    }
}

// 取消任务
async function cancelCurrentTask() {
    if (!currentTaskId) return;

    if (!confirm('确定要取消这个任务吗？')) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/tasks/${currentTaskId}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || '取消失败');
        }

        console.log('任务已取消');
        hideDetailModal();
        refreshTasks();
    } catch (error) {
        console.error('Failed to cancel task:', error);
        alert('取消任务失败: ' + error.message);
    }
}

// 过滤任务
function filterTasks() {
    renderTasks();
}

// 显示创建任务模态框
function showCreateTaskModal() {
    document.getElementById('createModal').classList.add('show');
}

// 隐藏创建任务模态框
function hideCreateTaskModal() {
    document.getElementById('createModal').classList.remove('show');
}

// 隐藏详情模态框
function hideDetailModal() {
    document.getElementById('detailModal').classList.remove('show');
    currentTaskId = null;
}

// 获取状态文本
function getStatusText(status) {
    const statusMap = {
        'pending': '等待中',
        'running': '运行中',
        'completed': '已完成',
        'failed': '失败',
        'cancelled': '已取消'
    };
    return statusMap[status] || status;
}

// 格式化时间
function formatTime(timeStr) {
    if (!timeStr) return '-';
    const date = new Date(timeStr);
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

// 点击模态框外部关闭
document.addEventListener('click', function (e) {
    if (e.target.classList.contains('modal')) {
        e.target.classList.remove('show');
    }
});
