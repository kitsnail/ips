// API 基础地址
const API_BASE = '/api/v1';

// 全局状态
let tasks = [];
let currentTaskId = null;
let autoRefreshInterval = null;

// 初始化
document.addEventListener('DOMContentLoaded', function() {
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
            <div class="detail-row">
                <span class="detail-label">任务ID:</span>
                <span class="detail-value">${task.taskId}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">状态:</span>
                <span class="task-status status-${task.status}">${getStatusText(task.status)}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">优先级:</span>
                <span class="detail-value">${task.priority}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">镜像列表:</span>
                <span class="detail-value">${task.images.join(', ')}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">批次大小:</span>
                <span class="detail-value">${task.batchSize}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">最大重试次数:</span>
                <span class="detail-value">${task.maxRetries}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">当前重试次数:</span>
                <span class="detail-value">${task.retryCount}</span>
            </div>
            <div class="detail-row">
                <span class="detail-label">重试策略:</span>
                <span class="detail-value">${task.retryStrategy === 'exponential' ? '指数退避' : '线性'}</span>
            </div>
            ${task.webhookUrl ? `
            <div class="detail-row">
                <span class="detail-label">Webhook URL:</span>
                <span class="detail-value">${task.webhookUrl}</span>
            </div>
            ` : ''}
            <div class="detail-row">
                <span class="detail-label">创建时间:</span>
                <span class="detail-value">${formatTime(task.createdAt)}</span>
            </div>
            ${task.startedAt ? `
            <div class="detail-row">
                <span class="detail-label">开始时间:</span>
                <span class="detail-value">${formatTime(task.startedAt)}</span>
            </div>
            ` : ''}
            ${task.finishedAt ? `
            <div class="detail-row">
                <span class="detail-label">完成时间:</span>
                <span class="detail-value">${formatTime(task.finishedAt)}</span>
            </div>
            ` : ''}
            ${task.progress ? `
            <div class="detail-row">
                <span class="detail-label">进度:</span>
                <span class="detail-value">
                    ${task.progress.percentage.toFixed(1)}%
                    (${task.progress.completedNodes}/${task.progress.totalNodes} 节点)
                </span>
            </div>
            <div class="detail-row">
                <span class="detail-label">批次进度:</span>
                <span class="detail-value">${task.progress.currentBatch}/${task.progress.totalBatches}</span>
            </div>
            ` : ''}
            ${task.failedNodeDetails && task.failedNodeDetails.length > 0 ? renderFailedNodes(task.failedNodeDetails) : ''}
        `;

        document.getElementById('taskDetail').innerHTML = detailHtml;

        // 显示/隐藏取消按钮
        const cancelBtn = document.getElementById('cancelTaskBtn');
        if (task.status === 'pending' || task.status === 'running') {
            cancelBtn.style.display = 'inline-block';
        } else {
            cancelBtn.style.display = 'none';
        }

        document.getElementById('detailModal').classList.add('show');
    } catch (error) {
        console.error('Failed to fetch task detail:', error);
        alert('获取任务详情失败');
    }
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
        alert(`任务创建成功！任务ID: ${task.taskId}`);
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

        alert('任务已取消');
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
document.addEventListener('click', function(e) {
    if (e.target.classList.contains('modal')) {
        e.target.classList.remove('show');
    }
});
