// API Base URL
const API_BASE = '';

// Toast notification
function showToast(message, type = 'success') {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.className = `toast show ${type}`;
    setTimeout(() => {
        toast.className = 'toast';
    }, 3000);
}

// Navigation
document.querySelectorAll('.nav-item').forEach(item => {
    item.addEventListener('click', () => {
        const page = item.dataset.page;
        switchPage(page);
    });
});

function switchPage(page) {
    document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
    document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
    
    document.querySelector(`[data-page="${page}"]`).classList.add('active');
    document.getElementById(`page-${page}`).classList.add('active');
    
    // Load page data
    switch(page) {
        case 'dashboard':
            loadDashboard();
            break;
        case 'accounts':
            loadAccounts();
            break;
        case 'routes':
            loadRoutes();
            break;
        case 'logs':
            loadLogs();
            break;
        case 'settings':
            loadConfig();
            break;
    }
}

// Modal functions
function showModal(id) {
    document.getElementById(id).classList.add('active');
}

function closeModal(id) {
    document.getElementById(id).classList.remove('active');
}

function showAddAccountModal() {
    document.getElementById('account-name').value = '';
    document.getElementById('account-email').value = '';
    document.getElementById('account-token').value = '';
    document.getElementById('account-type').value = 'free';
    showModal('modal-add-account');
}

function showImportModal() {
    document.getElementById('import-data').value = '';
    showModal('modal-import');
}

// Dashboard
async function loadDashboard() {
    try {
        const res = await fetch(`${API_BASE}/api/dashboard`);
        const data = await res.json();
        
        document.getElementById('stat-accounts').textContent = data.accounts?.total || 0;
        document.getElementById('stat-active').textContent = data.accounts?.active || 0;
        document.getElementById('stat-requests').textContent = data.stats?.requests_today || 0;
        document.getElementById('stat-latency').textContent = 
            data.stats?.avg_latency_ms ? `${Math.round(data.stats.avg_latency_ms)}ms` : '-';
        
        // Model stats
        const modelStatsEl = document.getElementById('model-stats');
        if (data.model_stats && data.model_stats.length > 0) {
            modelStatsEl.innerHTML = data.model_stats.slice(0, 5).map(s => `
                <div class="stats-list-item">
                    <span class="model-name">${s.model}</span>
                    <span class="request-count">${s.requests} 请求</span>
                </div>
            `).join('');
        } else {
            modelStatsEl.innerHTML = '<div class="stats-list-item"><span style="color:var(--text-muted)">暂无数据</span></div>';
        }
    } catch (err) {
        console.error('Failed to load dashboard:', err);
    }
}

function refreshDashboard() {
    loadDashboard();
    showToast('已刷新');
}

// Accounts
async function loadAccounts() {
    try {
        const res = await fetch(`${API_BASE}/api/accounts`);
        const accounts = await res.json() || [];
        
        const tbody = document.getElementById('accounts-table');
        if (accounts.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:var(--text-muted);padding:2rem;">暂无账号，点击"添加账号"开始</td></tr>';
            return;
        }
        
        tbody.innerHTML = accounts.map(acc => `
            <tr>
                <td><strong>${escapeHtml(acc.name)}</strong></td>
                <td>${escapeHtml(acc.email || '-')}</td>
                <td>${acc.account_type || 'free'}</td>
                <td><span class="status-badge status-${acc.status}">${getStatusText(acc.status)}</span></td>
                <td>${formatDate(acc.last_used_at)}</td>
                <td>
                    <button class="btn btn-sm btn-secondary" onclick="checkAccount(${acc.id})">检测</button>
                    <button class="btn btn-sm btn-danger" onclick="deleteAccount(${acc.id})">删除</button>
                </td>
            </tr>
        `).join('');
    } catch (err) {
        console.error('Failed to load accounts:', err);
    }
}

async function addAccount() {
    const name = document.getElementById('account-name').value.trim();
    const email = document.getElementById('account-email').value.trim();
    const refreshToken = document.getElementById('account-token').value.trim();
    const accountType = document.getElementById('account-type').value;
    
    if (!name || !refreshToken) {
        showToast('请填写名称和 Refresh Token', 'error');
        return;
    }
    
    try {
        const res = await fetch(`${API_BASE}/api/accounts`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, email, refresh_token: refreshToken, account_type: accountType })
        });
        
        if (res.ok) {
            closeModal('modal-add-account');
            loadAccounts();
            showToast('账号添加成功');
        } else {
            const err = await res.json();
            showToast(err.error || '添加失败', 'error');
        }
    } catch (err) {
        showToast('添加失败: ' + err.message, 'error');
    }
}

async function deleteAccount(id) {
    if (!confirm('确定要删除这个账号吗？')) return;
    
    try {
        await fetch(`${API_BASE}/api/accounts/${id}`, { method: 'DELETE' });
        loadAccounts();
        showToast('账号已删除');
    } catch (err) {
        showToast('删除失败', 'error');
    }
}

async function checkAccount(id) {
    try {
        showToast('正在检测...');
        await fetch(`${API_BASE}/api/accounts/${id}/check`, { method: 'POST' });
        loadAccounts();
        showToast('检测完成');
    } catch (err) {
        showToast('检测失败', 'error');
    }
}

async function checkAllAccounts() {
    try {
        showToast('正在检测所有账号...');
        await fetch(`${API_BASE}/api/accounts/check-all`, { method: 'POST' });
        loadAccounts();
        showToast('检测完成');
    } catch (err) {
        showToast('检测失败', 'error');
    }
}

async function importAccounts() {
    const data = document.getElementById('import-data').value.trim();
    
    try {
        JSON.parse(data); // Validate JSON
    } catch {
        showToast('JSON 格式无效', 'error');
        return;
    }
    
    try {
        const res = await fetch(`${API_BASE}/api/accounts/import`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: data
        });
        
        const result = await res.json();
        closeModal('modal-import');
        loadAccounts();
        showToast(`成功导入 ${result.imported} 个账号`);
    } catch (err) {
        showToast('导入失败', 'error');
    }
}

async function exportAccounts() {
    try {
        const res = await fetch(`${API_BASE}/api/accounts/export`);
        const blob = await res.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'accounts.json';
        a.click();
        URL.revokeObjectURL(url);
        showToast('导出成功');
    } catch (err) {
        showToast('导出失败', 'error');
    }
}

// Routes
async function loadRoutes() {
    try {
        const res = await fetch(`${API_BASE}/api/routes`);
        const routes = await res.json() || {};
        
        const container = document.getElementById('routes-container');
        container.innerHTML = '';
        
        Object.entries(routes).forEach(([pattern, target]) => {
            addRouteRow(pattern, target);
        });
        
        if (Object.keys(routes).length === 0) {
            addRouteRow('gpt-4', 'gemini-2.0-flash');
            addRouteRow('claude-3-opus', 'gemini-2.0-pro');
        }
    } catch (err) {
        console.error('Failed to load routes:', err);
    }
}

function addRouteRow(pattern = '', target = '') {
    const container = document.getElementById('routes-container');
    const row = document.createElement('div');
    row.className = 'route-row';
    row.innerHTML = `
        <input type="text" placeholder="源模型 (如 gpt-4)" value="${escapeHtml(pattern)}">
        <span class="arrow">→</span>
        <input type="text" placeholder="目标模型 (如 gemini-2.0-flash)" value="${escapeHtml(target)}">
        <button class="btn-remove" onclick="this.parentElement.remove()">×</button>
    `;
    container.appendChild(row);
}

async function saveRoutes() {
    const routes = {};
    document.querySelectorAll('.route-row').forEach(row => {
        const inputs = row.querySelectorAll('input');
        const pattern = inputs[0].value.trim();
        const target = inputs[1].value.trim();
        if (pattern && target) {
            routes[pattern] = target;
        }
    });
    
    try {
        await fetch(`${API_BASE}/api/routes`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(routes)
        });
        showToast('路由已保存');
    } catch (err) {
        showToast('保存失败', 'error');
    }
}

// Logs
async function loadLogs() {
    try {
        const res = await fetch(`${API_BASE}/api/logs?limit=50`);
        const logs = await res.json() || [];
        
        const tbody = document.getElementById('logs-table');
        if (logs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:var(--text-muted);padding:2rem;">暂无日志</td></tr>';
            return;
        }
        
        tbody.innerHTML = logs.map(log => `
            <tr>
                <td>${formatDateTime(log.created_at)}</td>
                <td>${escapeHtml(log.account_name)}</td>
                <td><code>${escapeHtml(log.model)}</code></td>
                <td>${log.tokens_in} / ${log.tokens_out}</td>
                <td>${log.latency_ms}ms</td>
                <td><span class="status-badge ${log.status_code === 200 ? 'status-active' : 'status-expired'}">${log.status_code}</span></td>
            </tr>
        `).join('');
    } catch (err) {
        console.error('Failed to load logs:', err);
    }
}

function refreshLogs() {
    loadLogs();
    showToast('已刷新');
}

// Config
async function loadConfig() {
    try {
        const res = await fetch(`${API_BASE}/api/config`);
        const config = await res.json();
        
        document.getElementById('config-timeout').value = config.proxy?.timeout || 120;
        document.getElementById('config-retries').value = config.proxy?.max_retries || 3;
        document.getElementById('config-autorotate').checked = config.proxy?.auto_rotate !== false;
    } catch (err) {
        console.error('Failed to load config:', err);
    }
}

async function saveConfig() {
    const config = {
        proxy: {
            timeout: parseInt(document.getElementById('config-timeout').value) || 120,
            max_retries: parseInt(document.getElementById('config-retries').value) || 3,
            auto_rotate: document.getElementById('config-autorotate').checked
        }
    };
    
    try {
        await fetch(`${API_BASE}/api/config`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });
        showToast('设置已保存');
    } catch (err) {
        showToast('保存失败', 'error');
    }
}

// Helpers
function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/[&<>"']/g, m => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[m]));
}

function getStatusText(status) {
    const map = {
        'active': '✓ 正常',
        'expired': '⚠ 过期',
        'banned': '✗ 封禁',
        'checking': '⟳ 检测中',
        'unknown': '? 未知'
    };
    return map[status] || status;
}

function formatDate(dateStr) {
    if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return '-';
    try {
        return new Date(dateStr).toLocaleDateString('zh-CN');
    } catch {
        return '-';
    }
}

function formatDateTime(dateStr) {
    if (!dateStr) return '-';
    try {
        return new Date(dateStr).toLocaleString('zh-CN', {
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
        });
    } catch {
        return '-';
    }
}

// Initial load
document.addEventListener('DOMContentLoaded', () => {
    loadDashboard();
});
