// API 基础地址
const API_BASE = '/api/v1';

// 全局状态
let tasks = [];
let users = [];
let currentTaskId = null;
let autoRefreshInterval = null;
let currentUser = null;

// 初始化
document.addEventListener('DOMContentLoaded', function () {
    if (checkAuth()) {
        initApp();
    }
});

function initApp() {
    refreshTasks();
    // 每5秒自动刷新任务
    autoRefreshInterval = setInterval(() => {
        if (document.getElementById('tasksPanel').style.display !== 'none') {
            refreshTasks();
        }
    }, 5000);

    // 设置用户名显示
    const user = JSON.parse(localStorage.getItem('ips_user'));
    if (user) {
        currentUser = user;
        document.getElementById('currentUsername').innerText = user.username;
        if (user.role === 'admin') {
            document.getElementById('adminTab').style.display = 'inline-block';
        }
    }
}

// 身份验证检查
function checkAuth() {
    const token = localStorage.getItem('ips_token');
    if (!token) {
        showLogin();
        return false;
    }
    hideLogin();
    return true;
}

function showLogin() {
    document.getElementById('loginOverlay').style.display = 'flex';
}

function hideLogin() {
    document.getElementById('loginOverlay').style.display = 'none';
}

// 登录
async function login(event) {
    event.preventDefault();
    const username = document.getElementById('loginUser').value;
    const password = document.getElementById('loginPass').value;

    try {
        const response = await fetch(`${API_BASE}/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        if (!response.ok) {
            throw new Error('登录失败，请检查用户名和密码');
        }

        const data = await response.json();
        localStorage.setItem('ips_token', data.token);
        localStorage.setItem('ips_user', JSON.stringify(data.user));

        hideLogin();
        initApp();
    } catch (error) {
        alert(error.message);
    }
}

// 退出
function logout() {
    localStorage.removeItem('ips_token');
    localStorage.removeItem('ips_user');
    location.reload();
}

// 带认证的 Fetch 包装器
async function fetchWithAuth(url, options = {}) {
    const token = localStorage.getItem('ips_token');
    const headers = {
        ...options.headers,
        'Authorization': `Bearer ${token}`
    };

    const response = await fetch(url, { ...options, headers });

    if (response.status === 401) {
        logout();
        throw new Error('登录已过期');
    }

    return response;
}

// 切换标签页
function switchTab(tab) {
    document.querySelectorAll('.tab-panel').forEach(p => p.style.display = 'none');
    document.querySelectorAll('.nav-link').forEach(l => l.classList.remove('active'));

    if (tab === 'tasks') {
        document.getElementById('tasksPanel').style.display = 'block';
        document.querySelector('a[onclick="switchTab(\'tasks\')"]').classList.add('active');
        refreshTasks();
    } else if (tab === 'admin') {
        document.getElementById('adminPanel').style.display = 'block';
        document.querySelector('a[onclick="switchTab(\'admin\')"]').classList.add('active');
        refreshUsers();
    }
}

// 刷新任务列表
async function refreshTasks() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/tasks`);
        const data = await response.json();
        tasks = data.tasks || [];
        renderTasks();
    } catch (error) {
        console.error('Failed to fetch tasks:', error);
        document.getElementById('taskList').innerHTML =
            `<div class="empty-state">${error.message === '登录已过期' ? '请重新登录' : '加载失败，请检查网络连接'}</div>`;
    }
}

// 渲染任务列表
function renderTasks() {
    const taskList = document.getElementById('taskList');
    const statusFilter = document.getElementById('statusFilter').value;

    let filteredTasks = tasks;
    if (statusFilter) {
        filteredTasks = tasks.filter(t => t.status === statusFilter);
    }

    if (!filteredTasks || filteredTasks.length === 0) {
        taskList.innerHTML = '<div class="empty-state">暂无任务</div>';
        return;
    }

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

// 渲染进度条 (同前)
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
        const response = await fetchWithAuth(`${API_BASE}/tasks/${taskId}`);
        const task = await response.json();
        currentTaskId = taskId;

        const detailHtml = `
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
            </div>
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 24px; margin-bottom: 24px;">
                <div class="config-pane" style="border: none; padding: 0;">
                    <div class="detail-row"><span class="detail-label">任务ID:</span><span class="detail-value">${task.taskId}</span></div>
                    <div class="detail-row"><span class="detail-label">创建时间:</span><span class="detail-value">${formatTime(task.createdAt)}</span></div>
                </div>
                <div class="config-pane" style="border: none; padding: 0;">
                    <div class="detail-row">
                        <span class="detail-label">镜像列表:</span>
                        <div class="detail-value" style="word-break: break-all; font-size: 11px; max-height: 80px; overflow-y: auto; background: #fafafa; padding: 8px;">
                            ${task.images.join('<br>')}
                        </div>
                    </div>
                </div>
            </div>
            ${renderNodeStatuses(task.nodeStatuses)}
            ${task.failedNodes && task.failedNodes.length > 0 ? renderFailedNodes(task.failedNodes) : ''}
        `;

        document.getElementById('taskDetail').innerHTML = detailHtml;
        const cancelBtn = document.getElementById('cancelTaskBtn');
        cancelBtn.style.display = (task.status === 'pending' || task.status === 'running') ? 'inline-block' : 'none';
        document.getElementById('detailModal').classList.add('show');
    } catch (error) {
        console.error('Failed to fetch task detail:', error);
    }
}

// 渲染节点镜像状态
function renderNodeStatuses(nodeStatuses) {
    if (!nodeStatuses || Object.keys(nodeStatuses).length === 0) return '<div class="empty-state">暂无节点拉取详情</div>';
    const rows = Object.entries(nodeStatuses).map(([node, images]) => `
        <tr>
            <td class="node-name-cell">${node}</td>
            <td>${Object.entries(images).map(([img, status]) => `<span class="${status === 1 ? 'image-tag-success' : 'image-tag-failed'}">${img}</span>`).join(' ')}</td>
        </tr>
    `).join('');
    return `<table class="node-status-table"><thead><tr><th>节点</th><th>镜像执行结果</th></tr></thead><tbody>${rows}</tbody></table>`;
}

function renderFailedNodes(failedNodes) {
    return `<div class="failed-nodes"><div style="font-weight: 600;">失败节点详情:</div>${failedNodes.map(n => `<div class="failed-node-item"><strong>${n.nodeName}:</strong> ${n.reason} - ${n.message}</div>`).join('')}</div>`;
}

// 创建任务
async function createTask(event) {
    event.preventDefault();
    const images = document.getElementById('images').value.trim().split('\n').filter(i => i.trim());
    const body = {
        images,
        batchSize: parseInt(document.getElementById('batchSize').value),
        priority: parseInt(document.getElementById('priority').value),
        maxRetries: parseInt(document.getElementById('maxRetries').value),
        retryStrategy: document.getElementById('retryStrategy').value,
        retryDelay: parseInt(document.getElementById('retryDelay').value),
        webhookUrl: document.getElementById('webhookUrl').value.trim(),
        nodeSelector: document.getElementById('nodeSelector').value.trim() ? JSON.parse(document.getElementById('nodeSelector').value) : null
    };

    try {
        const response = await fetchWithAuth(`${API_BASE}/tasks`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });
        if (!response.ok) throw new Error('创建失败');
        hideCreateTaskModal();
        refreshTasks();
    } catch (error) {
        alert(error.message);
    }
}

// 取消任务
async function cancelCurrentTask() {
    if (!currentTaskId || !confirm('确定取消？')) return;
    try {
        await fetchWithAuth(`${API_BASE}/tasks/${currentTaskId}`, { method: 'DELETE' });
        hideDetailModal();
        refreshTasks();
    } catch (error) {
        alert(error.message);
    }
}

// 用户列表管理
async function refreshUsers() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/users`);
        const data = await response.json();
        const tbody = document.getElementById('userListBody');
        tbody.innerHTML = data.map(u => `
            <tr>
                <td>${u.username}</td>
                <td>${u.role}</td>
                <td>${formatTime(u.createdAt)}</td>
                <td>
                    ${u.username === 'admin' ? '-' : `<button class="btn btn-secondary" onclick="deleteUser(${u.id})">删除</button>`}
                </td>
            </tr>
        `).join('');
    } catch (error) {
        console.error(error);
    }
}

async function createUser(event) {
    event.preventDefault();
    const username = document.getElementById('newUsername').value;
    const password = document.getElementById('newPassword').value;
    const role = document.getElementById('newRole').value;

    try {
        await fetchWithAuth(`${API_BASE}/users`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password, role })
        });
        hideCreateUserModal();
        refreshUsers();
    } catch (error) {
        alert(error.message);
    }
}

async function deleteUser(id) {
    if (!confirm('确定删除该用户？')) return;
    try {
        await fetchWithAuth(`${API_BASE}/users/${id}`, { method: 'DELETE' });
        refreshUsers();
    } catch (error) {
        alert(error.message);
    }
}

// 辅助函数
function showCreateTaskModal() { document.getElementById('createModal').classList.add('show'); }
function hideCreateTaskModal() { document.getElementById('createModal').classList.remove('show'); }
function showCreateUserModal() { document.getElementById('createUserModal').classList.add('show'); }
function hideCreateUserModal() { document.getElementById('createUserModal').classList.remove('show'); }
function hideDetailModal() { document.getElementById('detailModal').classList.remove('show'); currentTaskId = null; }
function filterTasks() { renderTasks(); }
function getStatusText(s) { return { 'pending': '等待中', 'running': '运行中', 'completed': '已完成', 'failed': '失败', 'cancelled': '已取消' }[s] || s; }
function formatTime(s) { if (!s) return '-'; return new Date(s).toLocaleString(); }

document.addEventListener('click', e => { if (e.target.classList.contains('modal')) e.target.classList.remove('show'); });
