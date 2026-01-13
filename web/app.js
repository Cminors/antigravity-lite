// API Base URL
const API_BASE = '';

// Preset model mappings
const PRESET_MAPPINGS = {
    'claude-haiku-*': 'gemini-2.5-flash-lite',
    'claude-3-haiku-*': 'gemini-2.5-flash-lite',
    'claude-3-5-sonnet-*': 'claude-sonnet-4-5',
    'claude-3-opus-*': 'claude-opus-4-5-thinking',
    'gpt-4o*': 'gemini-3-flash',
    'gpt-4*': 'gemini-3-pro-high',
    'gpt-3.5*': 'gemini-2.5-flash',
    'o1-*': 'gemini-3-pro-high'
};

// State
let allAccounts = [];
let currentFilter = 'all';

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
    switch (page) {
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
    document.getElementById('account-token').value = '';
    document.getElementById('account-type').value = 'free';
    switchAccountTab('token');
    showModal('modal-add-account');
}

// Tab switching for add account modal
function switchAccountTab(tabName) {
    // Update tab buttons
    document.querySelectorAll('.modal-tab').forEach(tab => {
        tab.classList.remove('active');
        if (tab.dataset.tab === tabName) {
            tab.classList.add('active');
        }
    });

    // Update tab content
    document.querySelectorAll('#modal-add-account .tab-content').forEach(content => {
        content.classList.remove('active');
    });
    document.getElementById(`tab-${tabName}`).classList.add('active');
}

// Filter tabs for accounts
document.querySelectorAll('.filter-tab').forEach(tab => {
    tab.addEventListener('click', () => {
        document.querySelectorAll('.filter-tab').forEach(t => t.classList.remove('active'));
        tab.classList.add('active');
        currentFilter = tab.dataset.filter;
        renderAccounts();
    });
});

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

        // Update service status
        updateServiceStatus(data.accounts?.total || 0);

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

function updateServiceStatus(accountCount) {
    const statusEl = document.getElementById('service-status');
    if (statusEl) {
        statusEl.textContent = `● 服务运行中 (${accountCount} 个账号)`;
    }
}

// Accounts
async function loadAccounts() {
    try {
        const res = await fetch(`${API_BASE}/api/accounts`);
        allAccounts = await res.json() || [];
        updateAccountCounts();
        renderAccounts();
    } catch (err) {
        console.error('Failed to load accounts:', err);
    }
}

function updateAccountCounts() {
    const counts = { all: allAccounts.length, pro: 0, ultra: 0, free: 0 };
    allAccounts.forEach(acc => {
        const type = (acc.account_type || 'free').toLowerCase();
        if (counts[type] !== undefined) counts[type]++;
    });

    document.getElementById('count-all').textContent = counts.all;
    document.getElementById('count-pro').textContent = counts.pro;
    document.getElementById('count-ultra').textContent = counts.ultra;
    document.getElementById('count-free').textContent = counts.free;
}

function filterAccounts() {
    renderAccounts();
}

function renderAccounts() {
    const searchTerm = (document.getElementById('account-search')?.value || '').toLowerCase();

    let filtered = allAccounts;

    // Filter by type
    if (currentFilter !== 'all') {
        filtered = filtered.filter(acc =>
            (acc.account_type || 'free').toLowerCase() === currentFilter
        );
    }

    // Filter by search
    if (searchTerm) {
        filtered = filtered.filter(acc =>
            (acc.email || '').toLowerCase().includes(searchTerm) ||
            (acc.name || '').toLowerCase().includes(searchTerm)
        );
    }

    const tbody = document.getElementById('accounts-table');
    if (filtered.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:var(--text-muted);padding:2rem;">暂无账号，点击"添加账号"开始</td></tr>';
        return;
    }

    tbody.innerHTML = filtered.map((acc, index) => `
        <tr draggable="true" data-id="${acc.id}">
            <td class="drag-handle">⋮⋮</td>
            <td>
                <strong>${escapeHtml(acc.email || acc.name)}</strong>
                ${getAccountBadges(acc, index)}
            </td>
            <td class="quota-cell">
                ${getQuotaTags(acc)}
            </td>
            <td>${formatDateTime(acc.last_used_at)}</td>
            <td>
                <button class="btn btn-sm btn-secondary" onclick="checkAccount(${acc.id})" title="检测">🔍</button>
                <button class="btn btn-sm btn-secondary" onclick="toggleAccountStatus(${acc.id})" title="启用/禁用">⊘</button>
                <button class="btn btn-sm btn-danger" onclick="deleteAccount(${acc.id})" title="删除">🗑️</button>
            </td>
        </tr>
    `).join('');
}

function getAccountBadges(acc, index) {
    let badges = '';
    const type = (acc.account_type || 'free').toLowerCase();

    if (type === 'pro') {
        badges += '<span class="badge badge-pro">PRO</span>';
    } else if (type === 'ultra') {
        badges += '<span class="badge badge-ultra">ULTRA</span>';
    } else {
        badges += '<span class="badge badge-free">FREE</span>';
    }

    if (index === 0) {
        badges += '<span class="badge badge-current">当前</span>';
    }

    if (acc.status === 'banned' || acc.status === 'expired') {
        badges += '<span class="badge badge-disabled">已禁用</span>';
    }

    return badges;
}

function getQuotaTags(acc) {
    // Display quota if available from account data
    if (acc.quota_limit && acc.quota_limit > 0) {
        const used = acc.quota_used || 0;
        const limit = acc.quota_limit;
        const remaining = Math.max(0, limit - used);
        const percentage = Math.round((remaining / limit) * 100);

        let colorClass = 'quota-high';
        if (percentage < 30) colorClass = 'quota-low';
        else if (percentage < 60) colorClass = 'quota-medium';

        return `<span class="quota-tag ${colorClass}">${percentage}% 剩余</span>`;
    }

    // No quota data yet - show refresh button
    return `<button class="btn btn-sm btn-secondary" onclick="refreshQuota(${acc.id})" title="刷新配额">🔄 获取配额</button>`;
}

// Refresh quota for single account
async function refreshQuota(id) {
    try {
        showToast('正在查询配额...');
        const res = await fetch(`${API_BASE}/api/accounts/${id}/quota`, { method: 'POST' });
        const data = await res.json();

        if (data.error) {
            showToast('配额查询失败: ' + data.error, 'error');
            return;
        }

        // Show quota info
        if (data.models && data.models.length > 0) {
            const quotaInfo = data.models.map(m => `${m.name}: ${m.percentage}%`).join(', ');
            showToast(`配额信息: ${quotaInfo}`);
        }

        if (data.subscription_tier) {
            showToast(`订阅类型: ${data.subscription_tier}`);
        }

        loadAccounts();
    } catch (err) {
        showToast('配额查询失败: ' + err.message, 'error');
    }
}

// Refresh all quotas
async function refreshAllQuotas() {
    try {
        showToast('正在查询所有账号配额...');
        const res = await fetch(`${API_BASE}/api/accounts/refresh-quotas`, { method: 'POST' });
        const data = await res.json();

        if (data.error) {
            showToast('配额查询失败: ' + data.error, 'error');
            return;
        }

        showToast(`已刷新 ${data.refreshed} 个账号的配额`);
        loadAccounts();
        loadDashboard();
    } catch (err) {
        showToast('配额查询失败: ' + err.message, 'error');
    }
}

async function addAccountFromModal() {
    const activeTab = document.querySelector('#modal-add-account .tab-content.active');

    if (activeTab.id === 'tab-token') {
        await addAccountsFromTokens();
    } else if (activeTab.id === 'tab-oauth') {
        showToast('OAuth 授权功能开发中', 'error');
    } else if (activeTab.id === 'tab-database') {
        await importFromDatabase();
    }
}

async function addAccountsFromTokens() {
    const input = document.getElementById('account-token').value.trim();
    const accountType = document.getElementById('account-type').value;

    if (!input) {
        showToast('请输入 Refresh Token', 'error');
        return;
    }

    // Parse tokens from input
    const tokens = parseTokenInput(input);

    if (tokens.length === 0) {
        showToast('未能识别有效的 Token', 'error');
        return;
    }

    let successCount = 0;

    for (let i = 0; i < tokens.length; i++) {
        try {
            const res = await fetch(`${API_BASE}/api/accounts`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: `Account ${Date.now()}-${i}`,
                    refresh_token: tokens[i],
                    account_type: accountType
                })
            });

            if (res.ok) successCount++;
        } catch (err) {
            console.error('Failed to add account:', err);
        }
    }

    closeModal('modal-add-account');
    loadAccounts();
    showToast(`成功添加 ${successCount} 个账号`);
}

function parseTokenInput(text) {
    const tokens = [];

    // Try to parse as JSON array
    try {
        const parsed = JSON.parse(text);
        if (Array.isArray(parsed)) {
            parsed.forEach(item => {
                if (typeof item === 'string') {
                    tokens.push(item);
                } else if (item.refresh_token) {
                    tokens.push(item.refresh_token);
                }
            });
            return tokens;
        }
    } catch (e) {
        // Not JSON, continue with regex extraction
    }

    // Extract tokens using regex (format: 1//xxxxx)
    const tokenRegex = /1\/\/[\w\-_]+/g;
    const matches = text.match(tokenRegex);
    if (matches) {
        tokens.push(...matches);
    }

    // If no regex matches, treat each line as a token
    if (tokens.length === 0) {
        text.split('\n').forEach(line => {
            const trimmed = line.trim();
            if (trimmed && trimmed.length > 10) {
                tokens.push(trimmed);
            }
        });
    }

    return [...new Set(tokens)]; // Remove duplicates
}

async function importFromDatabase() {
    const dbType = document.getElementById('db-type').value;
    const connection = document.getElementById('db-connection').value;
    const table = document.getElementById('db-table').value;

    showToast('数据库导入功能开发中', 'error');
}

function testDbConnection() {
    showToast('数据库连接测试功能开发中', 'error');
}

async function startOAuthFlow() {
    try {
        showToast('正在启动 OAuth 授权...');
        const res = await fetch(`${API_BASE}/api/oauth/start`);
        const data = await res.json();

        if (data.error) {
            showToast(data.error, 'error');
            return;
        }

        // Show auth URL and copy to clipboard
        showToast('授权链接已生成，正在打开...');

        // Copy to clipboard
        navigator.clipboard.writeText(data.auth_url).then(() => {
            showToast('授权链接已复制到剪贴板');
        }).catch(() => { });

        // Open in new window/tab
        window.open(data.auth_url, '_blank');

        closeModal('modal-add-account');

        // Show instructions
        setTimeout(() => {
            showToast('请在浏览器中完成授权，授权后账号将自动添加');
        }, 1000);

        // Poll for new accounts
        setTimeout(() => {
            loadAccounts();
            loadDashboard();
        }, 5000);

    } catch (err) {
        console.error('OAuth flow error:', err);
        showToast('启动 OAuth 授权失败: ' + err.message, 'error');
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

async function toggleAccountStatus(id) {
    showToast('切换账号状态');
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
            // Add some default mappings
            addRouteRow('gpt-4', 'gemini-2.0-flash');
            addRouteRow('claude-3-opus', 'gemini-2.0-pro');
        }
    } catch (err) {
        console.error('Failed to load routes:', err);
    }
}

function addRouteRow(pattern = '', target = '') {
    const container = document.getElementById('routes-container');
    const item = document.createElement('div');
    item.className = 'mapping-item';
    item.innerHTML = `
        <input type="text" placeholder="源模型 (如 gpt-4*)" value="${escapeHtml(pattern)}">
        <span class="arrow">→</span>
        <input type="text" placeholder="目标模型" value="${escapeHtml(target)}">
        <button class="btn-remove" onclick="this.parentElement.remove()">×</button>
    `;
    container.appendChild(item);
}

function applyPresetMappings() {
    const container = document.getElementById('routes-container');
    container.innerHTML = '';

    Object.entries(PRESET_MAPPINGS).forEach(([pattern, target]) => {
        addRouteRow(pattern, target);
    });

    showToast('已应用预设映射');
}

function resetMappings() {
    if (!confirm('确定要重置所有映射吗？')) return;

    const container = document.getElementById('routes-container');
    container.innerHTML = '';
    showToast('映射已重置');
}

async function saveRoutes() {
    const routes = {};
    document.querySelectorAll('.mapping-item').forEach(item => {
        const inputs = item.querySelectorAll('input');
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

        document.getElementById('config-port').value = config.server?.port || 8045;
        document.getElementById('config-timeout').value = config.proxy?.timeout || 120;
        document.getElementById('config-autostart').checked = config.server?.autostart !== false;
        document.getElementById('config-lan-access').checked = config.server?.lan_access === true;
        document.getElementById('config-auth-enabled').checked = config.server?.auth_enabled === true;

        // Scheduling mode
        const mode = config.proxy?.schedule_mode || 'balance';
        document.querySelector(`input[name="schedule-mode"][value="${mode}"]`).checked = true;
        updateModeCards();

        // Wait time
        const waitTime = config.proxy?.max_wait_time || 60;
        document.getElementById('config-wait-time').value = waitTime;
        updateWaitTimeDisplay(waitTime);

        // API Key
        document.getElementById('api-key-value').textContent = config.server?.api_key || 'sk-xxxxxxxxxxxxx';

    } catch (err) {
        console.error('Failed to load config:', err);
    }
}

function updateModeCards() {
    document.querySelectorAll('.mode-card').forEach(card => card.classList.remove('active'));
    const checked = document.querySelector('input[name="schedule-mode"]:checked');
    if (checked) {
        checked.parentElement.querySelector('.mode-card').classList.add('active');
    }
}

// Listen for schedule mode changes
document.querySelectorAll('input[name="schedule-mode"]').forEach(radio => {
    radio.addEventListener('change', updateModeCards);
});

function updateWaitTimeDisplay(value) {
    document.getElementById('wait-time-value').textContent = `${value}s`;
}

async function saveConfig() {
    const config = {
        server: {
            port: parseInt(document.getElementById('config-port').value) || 8045,
            autostart: document.getElementById('config-autostart').checked,
            lan_access: document.getElementById('config-lan-access').checked,
            auth_enabled: document.getElementById('config-auth-enabled').checked,
            api_key: document.getElementById('api-key-value').textContent
        },
        proxy: {
            timeout: parseInt(document.getElementById('config-timeout').value) || 120,
            schedule_mode: document.querySelector('input[name="schedule-mode"]:checked')?.value || 'balance',
            max_wait_time: parseInt(document.getElementById('config-wait-time').value) || 60,
            auto_rotate: true
        }
    };

    try {
        const res = await fetch(`${API_BASE}/api/config`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(config)
        });
        const result = await res.json();
        if (result.success) {
            showToast('设置已保存！部分更改需要重启服务生效。');
        } else {
            showToast(result.error || '保存失败', 'error');
        }
    } catch (err) {
        showToast('保存失败: ' + err.message, 'error');
    }
}

function refreshApiKey() {
    if (!confirm('确定要刷新 API 密钥吗？现有的密钥将失效。')) return;

    const newKey = 'sk-' + generateRandomString(32);
    document.getElementById('api-key-value').textContent = newKey;
    showToast('API 密钥已刷新，请保存设置');
}

function copyApiKey() {
    const key = document.getElementById('api-key-value').textContent;
    navigator.clipboard.writeText(key);
    showToast('API 密钥已复制');
}

function generateRandomString(length) {
    const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < length; i++) {
        result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
}

function clearSessionBinding() {
    showToast('会话绑定已清除');
}

function openMonitor() {
    showToast('监控面板功能开发中');
}

function toggleService() {
    showToast('服务控制功能开发中');
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
            year: 'numeric',
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
    initCharts();
});

// ========== Charts ==========
let hourlyChart = null;
let modelChart = null;

function initCharts() {
    // Initialize hourly requests chart
    const hourlyCtx = document.getElementById('hourlyChart');
    if (hourlyCtx) {
        hourlyChart = new Chart(hourlyCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: '请求数',
                    data: [],
                    borderColor: '#58a6ff',
                    backgroundColor: 'rgba(88, 166, 255, 0.1)',
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: { display: false }
                },
                scales: {
                    x: {
                        grid: { color: 'rgba(255,255,255,0.1)' },
                        ticks: { color: '#8b949e' }
                    },
                    y: {
                        beginAtZero: true,
                        grid: { color: 'rgba(255,255,255,0.1)' },
                        ticks: { color: '#8b949e' }
                    }
                }
            }
        });
    }

    // Initialize model distribution chart
    const modelCtx = document.getElementById('modelChart');
    if (modelCtx) {
        modelChart = new Chart(modelCtx, {
            type: 'doughnut',
            data: {
                labels: [],
                datasets: [{
                    data: [],
                    backgroundColor: [
                        '#58a6ff', '#a371f7', '#3fb950',
                        '#d29922', '#f85149', '#8b949e'
                    ]
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'right',
                        labels: { color: '#e6edf3' }
                    }
                }
            }
        });
    }

    // Load chart data
    loadChartData();
}

async function loadChartData() {
    try {
        // Load hourly stats
        const hourlyRes = await fetch(`${API_BASE}/api/stats/hourly`);
        const hourlyData = await hourlyRes.json() || [];

        if (hourlyChart && hourlyData.length > 0) {
            hourlyChart.data.labels = hourlyData.map(d => {
                const date = new Date(d.hour);
                return date.getHours() + ':00';
            });
            hourlyChart.data.datasets[0].data = hourlyData.map(d => d.requests);
            hourlyChart.update();
        }

        // Load model stats
        const modelRes = await fetch(`${API_BASE}/api/stats/models`);
        const modelData = await modelRes.json() || [];

        if (modelChart && modelData.length > 0) {
            modelChart.data.labels = modelData.slice(0, 6).map(d => d.model || 'unknown');
            modelChart.data.datasets[0].data = modelData.slice(0, 6).map(d => d.requests);
            modelChart.update();
        }
    } catch (err) {
        console.error('Failed to load chart data:', err);
    }
}
