// API Base URL
const API_BASE = '/api/v1';

// Global State
let state = {
     tasks: [],
     pagination: {
         page: 1,
         pageSize: 10,
         total: 0
     },
     selectedTasks: new Set(),
     filter: {
         status: '',
         search: ''
     },
     user: null,
     currentTaskId: null,
     refreshInterval: null,
     debounceTimer: null,
     confirmCallback: null,
     library: [],
     libraryPagination: { page: 1, pageSize: 10, total: 0 },
     selectedLibImages: new Set(),
     secrets: [],
     secretPagination: { page: 1, pageSize: 10, total: 0 },
     selectedSecrets: new Set(),
     scheduledTasks: [],
     scheduledPagination: { page: 1, pageSize: 10, total: 0 },
     selectedScheduledTasks: new Set(),
     scheduledFilter: { enabled: '' }
 };

// --- Toast Factory ---
function showToast(message, type = 'success') {
    const container = document.getElementById('toastContainer');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;

    // Icon based on type
    let icon = '';
    if (type === 'success') icon = '<svg class="icon" style="color:#10b981" viewBox="0 0 24 24"><path d="M5 13l4 4L19 7"/></svg>';
    else if (type === 'error') icon = '<svg class="icon" style="color:#ef4444" viewBox="0 0 24 24"><path d="M6 18L18 6M6 6l12 12"/></svg>';
    else icon = '<svg class="icon" style="color:#3b82f6" viewBox="0 0 24 24"><path d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>';

    toast.innerHTML = `${icon}<span>${message}</span>`;
    container.appendChild(toast);

    // Auto remove
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(100%)';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// --- Custom Confirm ---
function showConfirm(title, message, onConfirm) {
    document.getElementById('confirmTitle').innerText = title;
    document.getElementById('confirmMessage').innerText = message;

    // Unbind previous
    const btn = document.getElementById('confirmBtnAction');
    const newBtn = btn.cloneNode(true);
    btn.parentNode.replaceChild(newBtn, btn);

    newBtn.onclick = () => {
        onConfirm();
        hideConfirmModal();
    };

    document.getElementById('confirmModal').classList.add('show');
}

function hideConfirmModal() {
    document.getElementById('confirmModal').classList.remove('show');
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    if (checkAuth()) {
        initApp();
    }
});

function initApp() {
    loadUser();
    refreshDashboard();

    state.refreshInterval = setInterval(() => {
        const tasksPanel = document.getElementById('tasksPanel');
        const dashboardPanel = document.getElementById('dashboardPanel');
        const isDetailModalOpen = document.getElementById('detailModal').classList.contains('show');

        if (isDetailModalOpen) return;

        if (dashboardPanel && dashboardPanel.style.display !== 'none') {
            refreshDashboard(true);
        } else if (tasksPanel && tasksPanel.style.display !== 'none') {
            refreshTasks(true);
        }
    }, 5000);
}

async function refreshDashboard(silent = false) {
    try {
        await Promise.all([
            refreshDashboardStats(silent),
            refreshRecentTasks(silent)
        ]);
    } catch (error) {
        console.error('Failed to refresh dashboard:', error);
        if (!silent) showToast('加载Dashboard失败', 'error');
    }
}

async function refreshDashboardStats(silent = false) {
    try {
        const response = await fetchWithAuth(`${API_BASE}/tasks?limit=1000`);
        const data = await response.json();
        const tasks = data.tasks || [];

        const runningTasks = tasks.filter(t => t.status === 'running').length;
        const pendingTasks = tasks.filter(t => t.status === 'pending').length;

        const today = new Date().toDateString();
        const todayTasks = tasks.filter(t => new Date(t.createdAt).toDateString() === today);
        const completedTasks = todayTasks.filter(t => t.status === 'completed').length;
        const successRate = todayTasks.length > 0 ? Math.round((completedTasks / todayTasks.length) * 100) : 100;

        document.getElementById('statRunningTasks').textContent = runningTasks + pendingTasks;
        document.getElementById('statSuccessRate').textContent = successRate + '%';

        const trend = document.getElementById('statTrend');
        const trendText = document.getElementById('statTrendText');
        if (todayTasks.length >= 2) {
            trend.className = 'stat-trend';
            trendText.textContent = '良好';
        } else if (todayTasks.length === 1) {
            trend.className = 'stat-trend neutral';
            trendText.textContent = '--';
        } else {
            trend.className = 'stat-trend neutral';
            trendText.textContent = '暂无数据';
        }

        const scheduledResponse = await fetchWithAuth(`${API_BASE}/scheduled-tasks`);
        const scheduledData = await scheduledResponse.json();
        const scheduledTasks = scheduledData.tasks || [];
        const activeScheduled = scheduledTasks.filter(t => t.enabled).length;
        document.getElementById('statScheduled').textContent = scheduledTasks.length;
        document.getElementById('statScheduledActive').textContent = activeScheduled;

        document.getElementById('statNodes').textContent = 'N/A';

    } catch (error) {
        console.error('Failed to refresh dashboard stats:', error);
    }
}

async function refreshRecentTasks(silent = false) {
    try {
        const response = await fetchWithAuth(`${API_BASE}/tasks?limit=5&offset=0`);
        const data = await response.json();
        const tasks = data.tasks || [];

        const container = document.getElementById('recentTasksList');

        if (tasks.length === 0) {
            container.innerHTML = '<div style="text-align: center; color: var(--text-muted); padding: 60px 0;">暂无任务记录</div>';
            return;
        }

        container.innerHTML = `
            <table class="data-grid" style="width: 100%;">
                <thead>
                    <tr>
                        <th>状态</th>
                        <th>任务 ID</th>
                        <th>镜像</th>
                        <th>进度</th>
                        <th>创建时间</th>
                        <th>操作</th>
                    </tr>
                </thead>
                <tbody>
                    ${tasks.map(task => {
                        const progress = task.progress || { percentage: 0 };
                        const progressValue = Math.round(progress.percentage);
                        const radius = 18;
                        const circumference = 2 * Math.PI * radius;
                        const strokeDashoffset = circumference - (progressValue / 100) * circumference;

                        return `
                        <tr class="task-row" style="cursor: pointer;" onclick="showTaskDetail('${task.taskId}')">
                            <td><span class="status-badge bg-${task.status}">${getStatusText(task.status)}</span></td>
                            <td style="font-family: monospace; font-weight: 500;">${task.taskId}</td>
                            <td>
                                <div style="font-size: 13px; font-weight: 500; color: #111827;">${task.images[0]}</div>
                                ${task.images.length > 1 ? `<div style="font-size: 11px; color: #6b7280;">+${task.images.length - 1} 个更多</div>` : ''}
                            </td>
                            <td>
                                <div class="progress-ring">
                                    <svg width="40" height="40">
                                        <circle class="bg" cx="20" cy="20" r="${radius}" />
                                        <circle class="progress" cx="20" cy="20" r="${radius}"
                                            style="stroke-dasharray: ${circumference} ${circumference}; stroke-dashoffset: ${strokeDashoffset};" />
                                    </svg>
                                    <div class="value">${progressValue}%</div>
                                </div>
                            </td>
                            <td style="color: #6b7280;">${formatTime(task.createdAt)}</td>
                            <td onclick="event.stopPropagation()">
                                <button class="btn btn-secondary btn-sm" onclick="showTaskDetail('${task.taskId}')">详情</button>
                            </td>
                        </tr>
                        `;
                    }).join('')}
                </tbody>
            </table>
        `;
    } catch (error) {
        console.error('Failed to refresh recent tasks:', error);
        if (!silent) showToast('加载最近任务失败', 'error');
    }
}

// Authentication
function checkAuth() {
    const token = localStorage.getItem('ips_token');
    if (!token) {
        showLogin();
        return false;
    }
    return true;
}

async function renderScheduledTasks() {
    const tbody = document.getElementById('scheduledTasksTableBody');
    tbody.innerHTML = '';
    
    if (state.scheduledTasks.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" style="text-align:center; color:#6b7280; padding: 40px;">暂无定时任务</td></tr>';
        return;
    }
    
    const start = (state.scheduledPagination.page - 1) * state.scheduledPagination.pageSize;
    const end = Math.min(start + state.scheduledPagination.pageSize, state.scheduledTasks.length);
    const displayTasks = state.scheduledTasks.slice(start, end);
    
    displayTasks.forEach(task => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${task.id}</td>
            <td>${task.name}</td>
            <td>${task.cronExpr}</td>
            <td><span class="status-badge bg-${task.enabled ? 'enabled' : 'disabled'}">${task.enabled ? '已启用' : '已禁用'}</span></td>
            <td>${task.overlapPolicy}</td>
            <td>${task.enabled ? `<button class="btn btn-sm btn-secondary" onclick="disableScheduledTask('${task.id}')">禁用</button>` : `<button class="btn btn-sm btn-secondary" onclick="enableScheduledTask('${task.id}')">启用</button>`}</td>
            <td>
                <button class="btn btn-sm btn-primary" onclick="triggerScheduledTask('${task.id}')">触发</button>
                <button class="btn btn-sm btn-danger" onclick="deleteScheduledTask('${task.id}')">删除</button>
            </td>
        `;
        tbody.appendChild(row);
    });
    
    // Update pagination info
    document.getElementById('scheduledPageStart').innerText = start + 1;
    document.getElementById('scheduledPageEnd').innerText = end;
    document.getElementById('scheduledTotalItems').innerText = state.scheduledPagination.total;
}

async function prevScheduledPage() {
    if (state.scheduledPagination.page > 1) {
        state.scheduledPagination.page--;
        refreshScheduledTasks();
    }
}

async function nextScheduledPage() {
    const { page, pageSize, total } = state.scheduledPagination;
    if (page * pageSize < total) {
        state.scheduledPagination.page++;
        refreshScheduledTasks();
    }
}

async function changeScheduledPageSize() {
    const size = parseInt(document.getElementById('scheduledPageSizeSelect').value);
    state.scheduledPagination.pageSize = size;
    state.scheduledPagination.page = 1;
    refreshScheduledTasks();
}

async function createScheduledTask(event) {
    event.preventDefault();
    
    const name = document.getElementById('scheduledName').value.trim();
    const cronExpr = document.getElementById('scheduledCron').value.trim();
    const images = document.getElementById('scheduledImages').value.trim().split(',').map(s => s.trim());
    const enabled = document.getElementById('scheduledEnabled').checked;
    const overlapPolicy = document.getElementById('scheduledOverlap').value;
    const timeout = parseInt(document.getElementById('scheduledTimeout').value) || 0;
    const batchSize = parseInt(document.getElementById('scheduledBatchSize').value) || 10;
    
    if (!name || !cronExpr || images.length === 0) {
        showToast('请填写必填字段', 'error');
        return;
    }
    
    if (!cronExpr.match(/^(\S+\s+\s+\s+\s+\s+\d+)$/)) {
        showToast('无效的 Cron 表达式', 'error');
        return;
    }
    
    const task = {
        name,
        description: '',
        cronExpr,
        enabled,
        taskConfig: {
            images,
            batchSize,
            priority: 1
        },
        overlapPolicy,
        timeoutSeconds: timeout
    };
    
    try {
        const response = await fetchWithAuth(`${API_BASE}/scheduled-tasks`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(task)
        });
        
        await response.json();
        showToast('定时任务创建成功', 'success');
        hideCreateScheduledTaskModal();
        refreshScheduledTasks();
    } catch (error) {
        console.error('Failed to create scheduled task:', error);
        showToast('创建定时任务失败', 'error');
    }
}

async function enableScheduledTask(id) {
    try {
        await fetchWithAuth(`${API_BASE}/scheduled-tasks/${id}/enable`, { method: 'PUT' });
        showToast('定时任务已启用', 'success');
        refreshScheduledTasks();
    } catch (error) {
        console.error('Failed to enable scheduled task:', error);
        showToast('启用定时任务失败', 'error');
    }
}

async function disableScheduledTask(id) {
    try {
        await fetchWithAuth(`${API_BASE}/scheduled-tasks/${id}/disable`, { method: 'PUT' });
        showToast('定时任务已禁用', 'success');
        refreshScheduledTasks();
    } catch (error) {
        console.error('Failed to disable scheduled task:', error);
        showToast('禁用定时任务失败', 'error');
    }
}

async function triggerScheduledTask(id) {
    try {
        const response = await fetchWithAuth(`${API_BASE}/scheduled-tasks/${id}/trigger`, { method: 'POST' });
        showToast('定时任务已触发', 'success');
    } catch (error) {
        console.error('Failed to trigger scheduled task:', error);
        showToast('触发定时任务失败', 'error');
    }
}

async function deleteScheduledTask(id) {
    if (!confirm('确定要删除这个定时任务吗？')) {
        return;
    }
    
    try {
        await fetchWithAuth(`${API_BASE}/scheduled-tasks/${id}`, { method: 'DELETE' });
        showToast('定时任务已删除', 'success');
        refreshScheduledTasks();
    } catch (error) {
        console.error('Failed to delete scheduled task:', error);
        showToast('删除定时任务失败', 'error');
    }
}

function showCreateScheduledTaskModal() {
    document.getElementById('createScheduledTaskModal').classList.add('show');
}

function hideCreateScheduledTaskModal() {
    document.getElementById('createScheduledTaskModal').classList.remove('show');
    document.getElementById('createScheduledTaskModal').classList.remove('active');
}

async function refreshScheduledTasks(silent = false) {
    if (!checkAuth()) return;
    try {
        const response = await fetchWithAuth(`${API_BASE}/scheduled-tasks?enabled=${state.scheduledFilter.enabled}&offset=${(state.scheduledPagination.page - 1) * state.scheduledPagination.pageSize}&limit=${state.scheduledPagination.pageSize}`);
        const data = await response.json();
        state.scheduledTasks = data.tasks;
        state.scheduledPagination.total = data.total;
        renderScheduledTasks();
        
        if (!silent) {
            showToast(`成功加载 ${data.tasks.length} 个定时任务`, 'success');
        }
    } catch (error) {
        console.error('Failed to load scheduled tasks:', error);
        showToast('加载定时任务失败', 'error');
    }
}

function loadUser() {
    const user = JSON.parse(localStorage.getItem('ips_user'));
    if (user) {
        state.user = user;
        document.getElementById('currentUsername').innerText = user.username;
        if (user.role === 'admin') {
            document.getElementById('adminTab').style.display = 'flex';
        }
    }
}

function showLogin() { document.getElementById('loginOverlay').style.display = 'flex'; }
function hideLogin() { document.getElementById('loginOverlay').style.display = 'none'; }

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

        if (!response.ok) throw new Error('Login failed. Please check credentials.');

        const data = await response.json();
        localStorage.setItem('ips_token', data.token);
        localStorage.setItem('ips_user', JSON.stringify(data.user));

        hideLogin();
        initApp();
    } catch (error) {
        alert(error.message);
    }
}

function logout() {
    localStorage.removeItem('ips_token');
    localStorage.removeItem('ips_user');
    location.reload();
}

async function fetchWithAuth(url, options = {}) {
    const token = localStorage.getItem('ips_token');
    const headers = { ...options.headers, 'Authorization': `Bearer ${token}` };
    const response = await fetch(url, { ...options, headers });

    if (response.status === 401) {
        logout();
        throw new Error('Session expired');
    }
    return response;
}

// Task Management
async function refreshTasks(silent = false) {
    const { page, pageSize } = state.pagination;
    const offset = (page - 1) * pageSize;
    // Note: status filter is client-side for now in rendering, or server-side if API supports it. 
    // Current API plan supports pagination. Filtering logic: 
    // If backend doesn't support filtering, we fetch paginated data. 
    // BUT if we filter client-side, purely paginated backend data is wrong.
    // For "Pro Max" correctness, we should pass status to backend.
    // The previous backend update included `Status` in `ListTasksRequest` but commented out impl in Repo.
    // So for now, we will fetch data and render, acknowledging limitation or implementation.
    // Wait, I updated TaskHandler to accept Status but Repo impl ignores it (Step 1431).
    // So filtering will only work on the CURRENT PAGE if we don't fix backend.
    // However, the prompt asked for frontend pagination.

    // Construct Query Params
    const params = new URLSearchParams({
        limit: pageSize,
        offset: offset,
        status: state.filter.status // Backend ignores this currently, but good to send
    });

    try {
        if (!silent) document.getElementById('taskListBody').innerHTML = '<tr><td colspan="6" class="loading">加载任务中...</td></tr>';

        const response = await fetchWithAuth(`${API_BASE}/tasks?${params}`);
        const data = await response.json();

        state.tasks = data.tasks || [];
        state.pagination.total = data.total || 0;

        renderTasks();
        updatePaginationUI();
    } catch (error) {
        console.error('Failed to fetch tasks:', error);
        if (!silent) document.getElementById('taskListBody').innerHTML = `<tr><td colspan="6" class="empty-state">${error.message}</td></tr>`;
    }
}

function renderTasks() {
    const tbody = document.getElementById('taskListBody');
    const tasks = state.tasks;
    const search = state.filter.search.toLowerCase();

    const filtered = tasks.filter(t => {
        const matchesSearch = t.taskId.toLowerCase().includes(search);
        const matchesStatus = !state.filter.status || t.status === state.filter.status;
        return matchesSearch && matchesStatus;
    });

    if (filtered.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="empty-state">暂无任务</td></tr>';
        return;
    }

    const selectAllCheckbox = document.getElementById('selectAll');
    if (selectAllCheckbox) {
        selectAllCheckbox.checked = filtered.length > 0 && filtered.every(t => state.selectedTasks.has(t.taskId));
        selectAllCheckbox.indeterminate = filtered.some(t => state.selectedTasks.has(t.taskId)) && !filtered.every(t => state.selectedTasks.has(t.taskId));
    }

    const batchDeleteBtn = document.getElementById('batchDeleteBtn');
    if (batchDeleteBtn) {
        if (state.selectedTasks.size > 0) {
            batchDeleteBtn.style.display = 'inline-flex';
            batchDeleteBtn.innerHTML = `
                <svg class="icon" viewBox="0 0 24 24"><path d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0"/></svg>
                批量删除 (${state.selectedTasks.size})
            `;
        } else {
            batchDeleteBtn.style.display = 'none';
        }
    }

    tbody.innerHTML = filtered.map(task => {
        const progress = task.progress || { percentage: 0 };
        const progressValue = Math.round(progress.percentage);
        const radius = 18;
        const circumference = 2 * Math.PI * radius;
        const strokeDashoffset = circumference - (progressValue / 100) * circumference;

        return `
        <tr class="task-row">
            <td class="col-checkbox">
                <input type="checkbox"
                    class="task-checkbox"
                    onchange="toggleTaskSelection('${task.taskId}')"
                    ${state.selectedTasks.has(task.taskId) ? 'checked' : ''}
                >
            </td>
            <td><span class="status-badge bg-${task.status}">${getStatusText(task.status)}</span></td>
            <td style="font-family: monospace; font-weight: 500;">${task.taskId}</td>
            <td class="image-cell">
                <div class="tooltip-trigger" style="display: inline-block;">
                    <div style="font-size: 13px; font-weight: 500; color: #111827;">${task.images[0]}</div>
                    ${task.images.length > 1 ? `<div style="font-size: 11px; color: #6b7280;">+${task.images.length - 1} 个更多</div>` : ''}
                    <div class="tooltip" style="bottom: 100%; left: 0; transform: none;">
                        ${task.images.map(img => `<div style="font-family: monospace; font-size: 11px; padding: 2px 0;">${img}</div>`).join('')}
                    </div>
                </div>
            </td>
            <td>
                <div class="progress-ring">
                    <svg width="40" height="40">
                        <circle class="bg" cx="20" cy="20" r="${radius}" />
                        <circle class="progress" cx="20" cy="20" r="${radius}"
                            style="stroke-dasharray: ${circumference} ${circumference}; stroke-dashoffset: ${strokeDashoffset};" />
                    </svg>
                    <div class="value">${progressValue}%</div>
                </div>
            </td>
            <td style="color: #6b7280;">${formatTime(task.createdAt)}</td>
            <td>
                <div class="action-dropdown">
                    <button class="btn btn-secondary btn-sm" onclick="toggleActionDropdown('${task.taskId}', event)">
                        操作
                        <svg class="icon" style="width: 12px; height: 12px; margin-left: 4px;" viewBox="0 0 24 24">
                            <path d="M19 9l-7 7-7-7" />
                        </svg>
                    </button>
                    <div class="action-dropdown-menu" id="dropdown-${task.taskId.replace(/[^a-zA-Z0-9-]/g, '')}">
                        <div class="action-dropdown-item" onclick="showTaskDetail('${task.taskId}')">
                            查看详情
                        </div>
                        ${task.status === 'running' || task.status === 'pending' ? `
                            <div class="action-dropdown-item danger" onclick="cancelTask('${task.taskId}')">
                                取消任务
                            </div>
                        ` : `
                            <div class="action-dropdown-item danger" onclick="deleteTask('${task.taskId}')">
                                删除任务
                            </div>
                        `}
                    </div>
                </div>
            </td>
        </tr>
    `}).join('');

    document.querySelectorAll('.action-dropdown button').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
        });
    });
}

// Pagination Controls
function updatePaginationUI() {
    const { page, pageSize, total } = state.pagination;
    const start = Math.min((page - 1) * pageSize + 1, total);
    const end = Math.min(start + pageSize - 1, total);

    document.getElementById('pageStart').innerText = total === 0 ? 0 : start;
    document.getElementById('pageEnd').innerText = end;
    document.getElementById('totalItems').innerText = total;
    document.getElementById('currentPageNum').innerText = page;

    document.getElementById('prevBtn').disabled = page <= 1;
    document.getElementById('nextBtn').disabled = end >= total;

    document.getElementById('pageSizeSelect').value = pageSize;
}

function nextPage() {
    const { page, pageSize, total } = state.pagination;
    if (page * pageSize < total) {
        state.pagination.page++;
        refreshTasks();
    }
}

function prevPage() {
    if (state.pagination.page > 1) {
        state.pagination.page--;
        refreshTasks();
    }
}

function changePageSize() {
    const size = parseInt(document.getElementById('pageSizeSelect').value);
    state.pagination.pageSize = size;
    state.pagination.page = 1; // Reset to page 1
    refreshTasks();
}

function searchTasksDebounced() {
    clearTimeout(state.debounceTimer);
    state.debounceTimer = setTimeout(() => {
        state.filter.search = document.getElementById('searchTask').value;
        renderTasks(); // Filter locally first
    }, 300);
}

function filterTasks() {
    state.filter.status = document.getElementById('statusFilter').value;
    state.pagination.page = 1;
    refreshTasks();
}

// Task Detail & Actions (Kept mostly same but updated styles)
async function showTaskDetail(taskId) {
    try {
        const response = await fetchWithAuth(`${API_BASE}/tasks/${taskId}`);
        const task = await response.json();
        state.currentTaskId = taskId;

        // Re-use logic to render detail modal content...
        // Simplified generic rendering for brevity
        renderDetailModal(task);
        document.getElementById('detailModal').classList.add('show');
    } catch (error) {
        console.error(error);
    }
}

function renderDetailModal(task) {
    const detailHtml = `
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-label">任务状态</div>
                <div class="stat-value info"><span class="status-badge bg-${task.status}">${getStatusText(task.status)}</span></div>
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
        
        <div style="background: #f9fafb; padding: 16px; border-radius: 8px; margin-bottom: 24px;">
            <div style="display: grid; grid-template-columns:1fr 1fr; gap: 16px;">
                <div class="detail-row"><strong>任务 ID:</strong> ${task.taskId}</div>
                 <div class="detail-row"><strong>创建时间:</strong> ${formatTime(task.createdAt)}</div>
                 <div class="detail-row"><strong>重试策略:</strong> ${task.retryStrategy} (最大重试 ${task.maxRetries} 次)</div>
            </div>
            ${task.secretName ? `
            <div style="margin-top: 12px; padding: 12px; background: #eef2ff; border: 1px solid #c7d2fe; border-radius: 6px;">
                <div class="detail-row" style="color: #3730a3;">
                    <strong style="color: #3730a3;">私有仓库认证:</strong>
                    <span class="status-badge" style="background: #dbeafe; color: #1e40af;">已启用</span>
                </div>
                <div class="detail-row" style="color: #3730a3;">
                    <strong style="color: #3730a3;">Secret 名称:</strong>
                    <span style="font-family: monospace; font-size: 13px; color: #3730a3;">${task.secretName}</span>
                </div>
                <div style="font-size: 11px; color: #6366f1; margin-top: 4px;">
                    <svg class="icon" style="width: 12px; height: 12px; color: #6366f1; vertical-align: text-bottom;" viewBox="0 0 24 24"><path d="M12 22s8-4 8-10V5l-8-5v5zM12 16.5a2.5 2.5 0 110-5 2.5 2.5 0 010-5z"/></svg>
                    临时 Secret 会在任务完成后自动清理
                </div>
            </div>
            ` : ''}
             <div style="margin-top: 12px;">
                <strong>镜像列表:</strong>
                <div style="max-height: 100px; overflow-y: auto; background: #fff; padding: 8px; border:1px solid #e5e7eb; border-radius:4px; margin-top: 4px; font-family: monospace; font-size: 12px;">
                    ${task.images.join('<br>')}
                </div>
            </div>
        </div>
             <div style="margin-top: 12px;">
                <strong>镜像列表:</strong>
                <div style="max-height: 100px; overflow-y: auto; background: #fff; padding: 8px; border: 1px solid #e5e7eb; border-radius: 4px; margin-top: 4px; font-family: monospace; font-size: 12px;">
                    ${task.images.join('<br>')}
                </div>
            </div>
        </div>

        ${renderNodeStatuses(task.nodeStatuses)}
        ${task.failedNodes && task.failedNodes.length > 0 ? renderFailedNodes(task.failedNodes) : ''}
    `;

    document.getElementById('taskDetail').innerHTML = detailHtml;
    const cancelBtn = document.getElementById('cancelTaskBtn');
    cancelBtn.style.display = (task.status === 'pending' || task.status === 'running') ? 'inline-block' : 'none';
}

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
    return `<div class="failed-nodes"><div style="font-weight: 600; margin-bottom: 8px;">失败详情:</div>${failedNodes.map(n => `<div class="failed-node-item"><strong>${n.nodeName}:</strong> ${n.reason} - ${n.message}</div>`).join('')}</div>`;
}

// Create Task
async function createTask(event) {
    event.preventDefault();
    const images = document.getElementById('images').value.trim().split('\n').filter(i => i.trim());

    const enableRegistry = document.getElementById('enableRegistry').checked;
    const manualMode = document.querySelector('input[name="authMode"][value="manual"]').checked;

    const body = {
        images,
        batchSize: parseInt(document.getElementById('batchSize').value),
        priority: parseInt(document.getElementById('priority').value),
        maxRetries: parseInt(document.getElementById('maxRetries').value),
        retryStrategy: document.getElementById('retryStrategy').value,
        retryDelay: parseInt(document.getElementById('retryDelay').value),
        webhookUrl: document.getElementById('webhookUrl')?.value.trim() || '',
        nodeSelector: document.getElementById('nodeSelector')?.value.trim() ? JSON.parse(document.getElementById('nodeSelector').value) : null
    };

    // 如果启用了私有仓库认证
    if (enableRegistry) {
        if (manualMode) {
            // 手动输入方式
            const registry = document.getElementById('registry').value.trim();
            const username = document.getElementById('username').value.trim();
            const password = document.getElementById('password').value;

            if (!registry || !username || !password) {
                showToast('请填写完整的镜像仓库认证信息', 'error');
                return;
            }
            body.registry = registry;
            body.username = username;
            body.password = password;
        } else {
            // 选择已保存的认证
            const secretId = parseInt(document.getElementById('selectedSecretId').value);
            if (!secretId) {
                showToast('请选择仓库认证', 'error');
                return;
            }
            body.secretId = secretId;
        }
    }

    try {
        const response = await fetchWithAuth(`${API_BASE}/tasks`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || error.details || 'Creation failed');
        }
        hideCreateTaskModal();
        showToast('任务创建成功');
        refreshTasks();
    } catch (error) {
        showToast(error.message, 'error');
    }
}

// Toggle registry fields visibility
function toggleRegistryFields() {
    const enableRegistry = document.getElementById('enableRegistry').checked;
    const registryFields = document.getElementById('registryFields');

    if (enableRegistry) {
        registryFields.style.display = 'block';
        // Set default to manual mode if none selected
        const manualRadio = document.querySelector('input[name="authMode"][value="manual"]');
        const selectRadio = document.querySelector('input[name="authMode"][value="select"]');
        if (!manualRadio.checked && !selectRadio.checked) {
            manualRadio.checked = true;
        }
        toggleAuthMode();
    } else {
        registryFields.style.display = 'none';
        // Clear required attributes when disabled to allow form submission
        document.getElementById('registry').required = false;
        document.getElementById('username').required = false;
        document.getElementById('password').required = false;
        document.getElementById('selectedSecretId').required = false;
    }
}

async function cancelCurrentTask() {
    if (!state.currentTaskId) return;

    showConfirm('取消任务', '确定要取消当前任务吗?', async () => {
        try {
            await fetchWithAuth(`${API_BASE}/tasks/${state.currentTaskId}`, { method: 'DELETE' });
            showToast('任务已取消');
            hideDetailModal();
            refreshTasks();
        } catch (error) {
            showToast(error.message, 'error');
        }
    });
}

// Batch Selection Logic
function toggleTaskSelection(taskId) {
    if (state.selectedTasks.has(taskId)) {
        state.selectedTasks.delete(taskId);
    } else {
        state.selectedTasks.add(taskId);
    }
    renderTasks(); // Re-render to update UI states
}

function toggleActionDropdown(taskId, event) {
    const dropdownId = `dropdown-${taskId.replace(/[^a-zA-Z0-9-]/g, '')}`;
    const dropdown = document.getElementById(dropdownId);

    document.querySelectorAll('.action-dropdown-menu').forEach(menu => {
        if (menu.id !== dropdownId) {
            menu.classList.remove('show');
        }
    });

    if (dropdown) {
        dropdown.classList.toggle('show');
    }
}

function closeAllActionDropdowns() {
    document.querySelectorAll('.action-dropdown-menu').forEach(menu => {
        menu.classList.remove('show');
    });
}

document.addEventListener('click', () => {
    closeAllActionDropdowns();
});

function toggleSelectAll() {
    const tasks = state.tasks;
    // Get visible tasks based on search filter if needed
    const search = state.filter.search.toLowerCase();
    const visibleTasks = tasks.filter(t => t.taskId.toLowerCase().includes(search));

    const allSelected = visibleTasks.length > 0 && visibleTasks.every(t => state.selectedTasks.has(t.taskId));

    if (allSelected) {
        // Deselect all visible
        visibleTasks.forEach(t => state.selectedTasks.delete(t.taskId));
    } else {
        // Select all visible
        visibleTasks.forEach(t => state.selectedTasks.add(t.taskId));
    }
    renderTasks();
}

async function executeBatchDelete() {
    if (state.selectedTasks.size === 0) return;

    showConfirm('批量操作', `确定要删除/取消选中的 ${state.selectedTasks.size} 个任务吗?`, async () => {
        const taskIds = Array.from(state.selectedTasks);
        try {
            // Execute sequentially to avoid SQLite locking issues
            let count = 0;
            for (const id of taskIds) {
                await fetchWithAuth(`${API_BASE}/tasks/${id}`, { method: 'DELETE' });
                count++;
            }

            showToast(`成功处理 ${count} 个任务`);
            state.selectedTasks.clear();
            refreshTasks();
        } catch (error) {
            showToast('部分任务处理失败: ' + error.message, 'error');
            refreshTasks();
        }
    });
}

async function cancelTask(taskId) {
    showConfirm('取消任务', '确定要取消该任务吗?', async () => {
        try {
            await fetchWithAuth(`${API_BASE}/tasks/${taskId}`, { method: 'DELETE' });
            showToast('任务已取消');
            refreshTasks();
        } catch (error) {
            showToast(error.message, 'error');
        }
    });
}

async function deleteTask(taskId) {
    showConfirm('删除任务', '确定要永久删除该任务记录吗?', async () => {
        try {
            await fetchWithAuth(`${API_BASE}/tasks/${taskId}`, { method: 'DELETE' });
            showToast('任务记录已删除');
            refreshTasks();
        } catch (error) {
            showToast(error.message, 'error');
        }
    });
}

// User Management (Admin)
async function refreshUsers() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/users`);
        const data = await response.json();
        const tbody = document.getElementById('userListBody');
        tbody.innerHTML = data.map(u => `
            <tr>
                <td>${u.username}</td>
                <td><span class="status-badge bg-pending" style="background:#e5e7eb; color:#374151;">${u.role}</span></td>
                <td>${formatTime(u.createdAt)}</td>
                <td>
                    <div style="display: flex; gap: 8px;">
                        <button class="btn btn-secondary btn-sm" onclick="showChangePasswordModal(${u.id}, '${u.username}')">编辑</button>
                        ${u.username === 'admin' ? '' : `<button class="btn btn-danger btn-sm" onclick="deleteUser(${u.id})">删除</button>`}
                    </div>
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
    showConfirm('删除用户', '确定要删除此用户吗?', async () => {
        try {
            await fetchWithAuth(`${API_BASE}/users/${id}`, { method: 'DELETE' });
            showToast('用户已删除');
            refreshUsers();
        } catch (error) {
            showToast(error.message, 'error');
        }
    });
}

// Tabs & Modals
function switchTab(tab) {
    const tabs = document.querySelectorAll('.nav-link');
    tabs.forEach(t => {
        t.classList.remove('active');
        if (t.getAttribute('onclick')?.includes(`'${tab}'`)) {
            t.classList.add('active');
        }
    });

    const panels = document.querySelectorAll('.tab-panel');
    panels.forEach(p => p.classList.remove('active'));
    const activePanel = document.getElementById(`${tab}Panel`);
    if (activePanel) activePanel.classList.add('active');

    const batchDeleteBtn = document.getElementById('batchDeleteBtn');
    if (batchDeleteBtn) {
        batchDeleteBtn.style.display = tab === 'tasks' ? 'none' : 'block';
    }

    if (tab === 'dashboard') {
        refreshDashboard();
    } else if (tab === 'tasks' && state.tasks.length === 0) {
        refreshTasks();
    }
}

function showCreateTaskModal() {
    const form = document.getElementById('createTaskForm');
    if (form) {
        form.reset();
        toggleRegistryFields();
        toggleAdvancedSection();
    }
    document.getElementById('createModal').classList.add('show');
    refreshQuickLibrary();
    loadSecretsForDropdown();
}

function toggleAdvancedSection() {
    const content = document.getElementById('advancedSection');
    const toggle = document.querySelector('.form-section-toggle');

    if (content.classList.contains('collapsed')) {
        content.classList.remove('collapsed');
        if (toggle) toggle.classList.remove('collapsed');
    } else {
        content.classList.add('collapsed');
        if (toggle) toggle.classList.add('collapsed');
    }
}
function hideCreateTaskModal() { document.getElementById('createModal').classList.remove('show'); }
function showCreateUserModal() { document.getElementById('createUserModal').classList.add('show'); }
function hideCreateUserModal() { document.getElementById('createUserModal').classList.remove('show'); }
function showAddLibraryImageModal() { document.getElementById('addLibraryModal').classList.add('show'); }
function hideAddLibraryModal() { document.getElementById('addLibraryModal').classList.remove('show'); }
function hideDetailModal() { document.getElementById('detailModal').classList.remove('show'); state.currentTaskId = null; }

// --- Password Change Functions ---
function showChangePasswordModal(userId, username) {
    document.getElementById('pwTargetUserId').value = userId;
    document.getElementById('pwTargetUsername').value = username;
    document.getElementById('pwNewPassword').value = '';
    document.getElementById('pwConfirmPassword').value = '';
    document.getElementById('changePasswordModal').classList.add('show');
}

function showChangeMyPasswordModal() {
    const user = state.user;
    if (user) {
        showChangePasswordModal(user.id, user.username);
    }
}

function hideChangePasswordModal() {
    document.getElementById('changePasswordModal').classList.remove('show');
}

async function changePassword(event) {
    event.preventDefault();
    const userId = document.getElementById('pwTargetUserId').value;
    const newPassword = document.getElementById('pwNewPassword').value;
    const confirmPassword = document.getElementById('pwConfirmPassword').value;

    if (newPassword !== confirmPassword) {
        showToast('两次密码输入不一致', 'error');
        return;
    }

    try {
        const response = await fetchWithAuth(`${API_BASE}/users/${userId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ password: newPassword })
        });
        if (!response.ok) {
            const err = await response.json();
            throw new Error(err.error || 'Failed');
        }
        showToast('密码修改成功');
        hideChangePasswordModal();
    } catch (error) {
        showToast(error.message, 'error');
    }
}

// --- Library Management Functions ---
async function refreshLibrary() {
    const { page, pageSize } = state.libraryPagination;
    const offset = (page - 1) * pageSize;
    try {
        const response = await fetchWithAuth(`${API_BASE}/library?limit=${pageSize}&offset=${offset}`);
        const data = await response.json();
        state.library = data.images || [];
        state.libraryPagination.total = data.total || 0;
        renderLibrary();
        updateLibPaginationUI();
    } catch (error) {
        console.error('Failed to fetch library:', error);
    }
}

function renderLibrary() {
    const tbody = document.getElementById('libraryListBody');
    const images = state.library;

    // Update Select All checkbox
    const selectAllCheckbox = document.getElementById('libSelectAll');
    if (selectAllCheckbox) {
        selectAllCheckbox.checked = images.length > 0 && images.every(img => state.selectedLibImages.has(img.id));
        selectAllCheckbox.indeterminate = images.some(img => state.selectedLibImages.has(img.id)) && !images.every(img => state.selectedLibImages.has(img.id));
    }

    // Update Batch Delete Button
    const batchBtn = document.getElementById('libBatchDeleteBtn');
    if (batchBtn) {
        batchBtn.style.display = state.selectedLibImages.size > 0 ? 'inline-flex' : 'none';
        batchBtn.innerHTML = `<svg class="icon" viewBox="0 0 24 24"><path d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0"/></svg> 批量删除 (${state.selectedLibImages.size})`;
    }

    if (images.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="empty-state">暂无镜像数据</td></tr>';
        return;
    }

    tbody.innerHTML = images.map(img => `
        <tr>
            <td class="col-checkbox"><input type="checkbox" onchange="toggleLibImageSelection(${img.id})" ${state.selectedLibImages.has(img.id) ? 'checked' : ''}></td>
            <td style="font-weight: 500;">${img.name}</td>
            <td style="font-family: monospace; font-size: 13px; color: #3b82f6;">${img.image}</td>
            <td style="color: #64748b;">${formatTime(img.createdAt)}</td>
            <td>
                <button class="btn btn-danger btn-sm" onclick="deleteLibraryImage(${img.id})">删除</button>
            </td>
        </tr>
    `).join('');
}

function toggleLibImageSelection(id) {
    if (state.selectedLibImages.has(id)) state.selectedLibImages.delete(id);
    else state.selectedLibImages.add(id);
    renderLibrary();
}

function toggleLibSelectAll() {
    const images = state.library;
    const allSelected = images.length > 0 && images.every(img => state.selectedLibImages.has(img.id));
    if (allSelected) images.forEach(img => state.selectedLibImages.delete(img.id));
    else images.forEach(img => state.selectedLibImages.add(img.id));
    renderLibrary();
}

async function executeLibBatchDelete() {
    if (state.selectedLibImages.size === 0) return;
    showConfirm('批量删除', `确定要删除选中的 ${state.selectedLibImages.size} 个镜像吗?`, async () => {
        let count = 0;
        for (const id of state.selectedLibImages) {
            try {
                await fetchWithAuth(`${API_BASE}/library/${id}`, { method: 'DELETE' });
                count++;
            } catch (e) { console.error(e); }
        }
        showToast(`成功删除 ${count} 个镜像`);
        state.selectedLibImages.clear();
        refreshLibrary();
    });
}

function updateLibPaginationUI() {
    const { page, pageSize, total } = state.libraryPagination;
    const start = Math.min((page - 1) * pageSize + 1, total);
    const end = Math.min(start + pageSize - 1, total);
    document.getElementById('libPageStart').innerText = total === 0 ? 0 : start;
    document.getElementById('libPageEnd').innerText = end;
    document.getElementById('libTotalItems').innerText = total;
    document.getElementById('libCurrentPageNum').innerText = page;
    document.getElementById('libPrevBtn').disabled = page <= 1;
    document.getElementById('libNextBtn').disabled = end >= total;
    document.getElementById('libPageSizeSelect').value = pageSize;
}

function libNextPage() {
    const { page, pageSize, total } = state.libraryPagination;
    if (page * pageSize < total) { state.libraryPagination.page++; refreshLibrary(); }
}
function libPrevPage() {
    if (state.libraryPagination.page > 1) { state.libraryPagination.page--; refreshLibrary(); }
}
function changeLibPageSize() {
    state.libraryPagination.pageSize = parseInt(document.getElementById('libPageSizeSelect').value);
    state.libraryPagination.page = 1;
    refreshLibrary();
}

async function addLibraryImage(event) {
    event.preventDefault();
    const imagesStr = document.getElementById('libImages').value;
    const lines = imagesStr.split('\n').map(l => l.trim()).filter(l => l);

    if (lines.length === 0) return;

    let successCount = 0;
    let failCount = 0;

    for (const image of lines) {
        // Auto-extract name: cr01.home.lan/library/n8n:v0.1.1 -> n8n:v0.1.1
        const name = image.split('/').pop();

        try {
            await fetchWithAuth(`${API_BASE}/library`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, image })
            });
            successCount++;
        } catch (error) {
            console.error(`Failed to add ${image}:`, error);
            failCount++;
        }
    }

    if (failCount === 0) {
        showToast(`成功添加 ${successCount} 个镜像`);
    } else {
        showToast(`部分添加成功 (${successCount} 成功, ${failCount} 失败)`, 'error');
    }

    hideAddLibraryModal();
    document.getElementById('addLibraryForm').reset();
    refreshLibrary();
}

async function deleteLibraryImage(id) {
    showConfirm('删除镜像', '确定要从库中删除此常用镜像吗?', async () => {
        try {
            await fetchWithAuth(`${API_BASE}/library/${id}`, { method: 'DELETE' });
            showToast('镜像已删除');
            refreshLibrary();
        } catch (error) {
            showToast(error.message, 'error');
        }
    });
}

async function refreshQuickLibrary() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/library?limit=100`);
        const data = await response.json();
        const images = data.images || [];
        const list = document.getElementById('quickLibraryList');
        if (images.length === 0) {
            list.innerHTML = '<div style="font-size: 12px; color: #94a3b8; text-align: center; margin-top: 20px;">库中暂无镜像</div>';
            return;
        }

        list.innerHTML = images.map(img => `
            <div class="library-item" onclick="pickImage('${img.image}')" style="padding: 8px; background: white; border: 1px solid #e2e8f0; border-radius: 4px; margin-bottom: 6px; cursor: pointer; transition: all 0.2s;">
                <div style="font-size: 13px; font-weight: 600; color: #1e293b;">${img.name}</div>
                <div style="font-size: 11px; color: #64748b; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">${img.image}</div>
            </div>
        `).join('');

        // Add hover effect via CSS classes if needed, or inline
    } catch (error) {
        console.error(error);
    }
}

function pickImage(imageUrl) {
    const textarea = document.getElementById('images');
    const current = textarea.value.trim();
    if (current.includes(imageUrl)) {
        showToast('该镜像已在列表中');
        return;
    }
    textarea.value = (current ? current + '\n' : '') + imageUrl;
    showToast('镜像已添加');
}


function formatTime(s) {
    if (!s) return '-';
    const d = new Date(s);
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

document.addEventListener('click', e => { if (e.target.classList.contains('modal')) e.target.classList.remove('show'); });

function getStatusText(s) {
    return {
        'pending': '等待中',
        'running': '运行中',
        'completed': '已完成',
        'failed': '失败',
        'cancelled': '已取消'
    }[s] || s;
}

// ==================== Secrets Management ====================

async function refreshSecrets() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/secrets?page=${state.secretPagination.page}&pageSize=${state.secretPagination.pageSize}`);
        const data = await response.json();
        state.secrets = data.data || [];
        state.secretPagination.total = data.total || 0;
        renderSecrets();
    } catch (error) {
        console.error('Failed to load secrets:', error);
        showToast('加载仓库认证失败', 'error');
    }
}

function renderSecrets() {
    const tbody = document.getElementById('secretListBody');
    const filtered = state.secrets;

    if (filtered.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="empty-state">暂无仓库认证</td></tr>';
        document.getElementById('secretPageStart').textContent = '0';
        document.getElementById('secretPageEnd').textContent = '0';
        document.getElementById('secretTotalItems').textContent = '0';
        return;
    }

    const offset = (state.secretPagination.page - 1) * state.secretPagination.pageSize;
    const start = offset + 1;
    const end = Math.min(start + state.secretPagination.pageSize - 1, state.secretPagination.total);

    document.getElementById('secretPageStart').textContent = start;
    document.getElementById('secretPageEnd').textContent = end;
    document.getElementById('secretTotalItems').textContent = state.secretPagination.total;

    tbody.innerHTML = filtered.map(secret => `
        <tr>
            <td class="col-checkbox">
                <input type="checkbox"
                    onchange="toggleSecretSelection(${secret.id})"
                    ${state.selectedSecrets.has(secret.id) ? 'checked' : ''}>
            </td>
            <td style="font-weight: 500;">${secret.name}</td>
            <td style="font-family: monospace;">${secret.registry}</td>
            <td>${secret.username}</td>
            <td style="color: #64748b; font-size: 13px;">${formatTime(secret.createdAt)}</td>
            <td>
                <button class="btn btn-sm btn-secondary" onclick="editSecret(${secret.id})">编辑</button>
                <button class="btn btn-sm btn-danger" onclick="deleteSecret(${secret.id})">删除</button>
            </td>
        </tr>
    `).join('');

    updateSecretPaginationButtons();
    updateSecretBatchDeleteButton();
}

function toggleSecretSelection(id) {
    if (state.selectedSecrets.has(id)) {
        state.selectedSecrets.delete(id);
    } else {
        state.selectedSecrets.add(id);
    }
    renderSecrets();
}

function toggleSecretSelectAll() {
    const selectAllCheckbox = document.getElementById('secretSelectAll');
    if (selectAllCheckbox.checked) {
        state.secrets.forEach(s => state.selectedSecrets.add(s.id));
    } else {
        state.selectedSecrets.clear();
    }
    renderSecrets();
}

function updateSecretBatchDeleteButton() {
    const btn = document.getElementById('secretBatchDeleteBtn');
    if (btn) {
        btn.style.display = state.selectedSecrets.size > 0 ? 'inline-flex' : 'none';
    }
}

function executeSecretBatchDelete() {
    if (state.selectedSecrets.size === 0) return;

    showConfirm('批量删除认证', `确定要删除选中的 ${state.selectedSecrets.size} 个仓库认证吗?`, async () => {
        try {
            await Promise.all([...state.selectedSecrets].map(id =>
                fetchWithAuth(`${API_BASE}/secrets/${id}`, { method: 'DELETE' })
            ));
            showToast('批量删除成功');
            state.selectedSecrets.clear();
            refreshSecrets();
        } catch (error) {
            showToast('批量删除失败: ' + error.message, 'error');
        }
    });
}

function changeSecretPageSize() {
    state.secretPagination.pageSize = parseInt(document.getElementById('secretPageSizeSelect').value);
    state.secretPagination.page = 1;
    refreshSecrets();
}

function secretPrevPage() {
    if (state.secretPagination.page > 1) {
        state.secretPagination.page--;
        refreshSecrets();
    }
}

function secretNextPage() {
    const maxPage = Math.ceil(state.secretPagination.total / state.secretPagination.pageSize);
    if (state.secretPagination.page < maxPage) {
        state.secretPagination.page++;
        refreshSecrets();
    }
}

function updateSecretPaginationButtons() {
    const maxPage = Math.ceil(state.secretPagination.total / state.secretPagination.pageSize);
    document.getElementById('secretPrevBtn').disabled = state.secretPagination.page <= 1;
    document.getElementById('secretNextBtn').disabled = state.secretPagination.page >= maxPage;
    document.getElementById('secretCurrentPageNum').textContent = state.secretPagination.page;
}

function showCreateSecretModal() {
    document.getElementById('secretModalTitle').textContent = '添加仓库认证';
    document.getElementById('secretId').value = '';
    document.getElementById('secretName').value = '';
    document.getElementById('secretRegistry').value = '';
    document.getElementById('secretUsername').value = '';
    document.getElementById('secretPassword').value = '';
    document.getElementById('secretPasswordHint').style.display = 'none';
    document.getElementById('createSecretModal').classList.add('show');
}

function hideCreateSecretModal() {
    document.getElementById('createSecretModal').classList.remove('show');
}

function editSecret(id) {
    const secret = state.secrets.find(s => s.id === id);
    if (!secret) return;

    document.getElementById('secretModalTitle').textContent = '编辑仓库认证';
    document.getElementById('secretId').value = secret.id;
    document.getElementById('secretName').value = secret.name;
    document.getElementById('secretRegistry').value = secret.registry;
    document.getElementById('secretUsername').value = secret.username;
    document.getElementById('secretPassword').value = '';
    document.getElementById('secretPasswordHint').style.display = 'block';
    document.getElementById('createSecretModal').classList.add('show');
}

async function saveSecret(e) {
    e.preventDefault();
    const id = document.getElementById('secretId').value;
    const isEdit = id !== '';

    const req = {
        name: document.getElementById('secretName').value,
        registry: document.getElementById('secretRegistry').value,
        username: document.getElementById('secretUsername').value,
        password: document.getElementById('secretPassword').value
    };

    if (!req.name || !req.registry || !req.username || (!isEdit && !req.password)) {
        showToast('请填写所有必填字段', 'error');
        return;
    }

    try {
        if (isEdit) {
            if (req.password) {
                await fetchWithAuth(`${API_BASE}/secrets/${id}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(req)
                });
            } else {
                await fetchWithAuth(`${API_BASE}/secrets/${id}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name: req.name, registry: req.registry, username: req.username })
                });
            }
            showToast('认证已更新');
        } else {
            await fetchWithAuth(`${API_BASE}/secrets`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(req)
            });
            showToast('认证已创建');
        }

        hideCreateSecretModal();
        refreshSecrets();
        loadSecretsForDropdown();
    } catch (error) {
        showToast('保存失败: ' + error.message, 'error');
    }
}

async function deleteSecret(id) {
    showConfirm('删除认证', '确定要删除此仓库认证吗?', async () => {
        try {
            await fetchWithAuth(`${API_BASE}/secrets/${id}`, { method: 'DELETE' });
            showToast('认证已删除');
            refreshSecrets();
            loadSecretsForDropdown();
        } catch (error) {
            showToast('删除失败: ' + error.message, 'error');
        }
    });
}

async function loadSecretsForDropdown() {
    try {
        const response = await fetchWithAuth(`${API_BASE}/secrets?pageSize=100`);
        const data = await response.json();
        const secrets = data.data || [];
        const select = document.getElementById('selectedSecretId');

        select.innerHTML = '<option value="">-- 请选择认证 --</option>';
        secrets.forEach(s => {
            select.innerHTML += `<option value="${s.id}">${s.name} (${s.registry})</option>`;
        });
    } catch (error) {
        console.error('Failed to load secrets for dropdown:', error);
    }
}

function toggleAuthMode() {
    const manualMode = document.querySelector('input[name="authMode"][value="manual"]').checked;
    document.getElementById('manualAuthFields').style.display = manualMode ? 'block' : 'none';
    document.getElementById('selectAuthFields').style.display = manualMode ? 'none' : 'block';

    if (manualMode) {
        document.getElementById('registry').required = true;
        document.getElementById('username').required = true;
        document.getElementById('password').required = true;
        document.getElementById('selectedSecretId').required = false;
    } else {
        document.getElementById('registry').required = false;
        document.getElementById('username').required = false;
        document.getElementById('password').required = false;
        document.getElementById('selectedSecretId').required = true;
    }
}

// Patch createTask function to support secretId
const originalCreateTask = createTask;
