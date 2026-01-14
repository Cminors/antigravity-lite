# Antigravity Lite

<p align="center">
  <strong>🚀 轻量级 API 网关，为无图形界面的 Linux 服务器设计</strong>
</p>

<p align="center">
  <em>极简部署：克隆项目 → 启动服务 → 打开 Web 界面配置一切！</em>
</p>

---

## ✨ 功能特性

### 🔐 账号管理
- 多账号管理和轮换
- 支持批量导入（单个 Token、JSON 数组、自动提取）
- 账号类型筛选（PRO/ULTRA/FREE）
- 一键检测所有账号状态
- 导入/导出 JSON 格式

### 🔌 API 代理
- 完全兼容 OpenAI API 格式
- 支持 Anthropic/Claude 协议
- 支持流式响应（SSE）
- 自动错误重试

### 🛤️ 模型路由
- 在 Web 界面直接管理模型映射
- 一键应用预设映射配置
- 支持通配符匹配（如 `gpt-4*` → `gemini-3-pro`）
- 自定义源模型到目标模型的映射

### 📊 智能调度
- **缓存优先模式**：绑定会话与账号，最大化 Prompt Cache 命中率
- **平衡轮换模式**：限流时自动切换账号（推荐）
- **性能优先模式**：纯随机轮换，适合高并发场景
- 可调节最大等待时长（0-300秒）

### 🌐 Web 管理界面
- 现代暗色主题设计
- 实时 Dashboard 统计
- 请求日志查看
- 全功能配置面板

---

## 📊 资源占用

| 指标 | 数值 |
|------|------|
| 二进制大小 | ~10-15 MB |
| 内存占用 | ~20-50 MB |
| CPU 占用 | 极低 |
| 依赖 | 无（单二进制运行） |

---

## 🚀 快速开始

### 方式一：Docker 一键部署（推荐）

只需 **3 步**，无需预先配置任何环境变量：

#### 步骤 1：克隆项目

```bash
git clone https://github.com/Cminors/antigravity-lite.git
cd antigravity-lite
```

#### 步骤 2：启动服务

```bash
# 新版 Docker（推荐）
docker compose up -d

# 或旧版 Docker
docker-compose up -d

# 查看日志（可选）
docker compose logs -f
```

#### 步骤 3：打开 Web 管理界面

访问 `http://your-server-ip:8045`

**所有配置都在 Web 界面完成：**

1. 进入 **Settings** 页面，配置 Google OAuth 凭证（如需使用 OAuth 授权添加账号）
2. 进入 **Accounts** 页面，添加账号（支持直接粘贴 Refresh Token）
3. 进入 **Model Router** 页面，配置模型映射（可一键应用预设）
4. 开始使用 API！

---

### 方式二：手动编译部署

适合需要自定义或无法使用 Docker 的环境：

```bash
# 1. 克隆项目
git clone https://github.com/Cminors/antigravity-lite.git
cd antigravity-lite

# 2. 安装依赖
go mod tidy

# 3. 编译
go build -o antigravity-lite .

# 4. 运行
chmod +x antigravity-lite
./antigravity-lite
```

访问 `http://localhost:8045` 进入 Web 管理界面。

---

### 方式三：Systemd 服务（生产环境）

创建服务文件 `/etc/systemd/system/antigravity-lite.service`：

```ini
[Unit]
Description=Antigravity Lite API Gateway
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/antigravity-lite
ExecStart=/opt/antigravity-lite/antigravity-lite
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable antigravity-lite
sudo systemctl start antigravity-lite
sudo systemctl status antigravity-lite
```

---

## ⚙️ 环境变量配置

**所有环境变量都是可选的**，均可在 Web 管理界面中配置。

| 变量名 | 必需 | 说明 | 默认值 |
|--------|------|------|--------|
| `GOOGLE_CLIENT_ID` | ❌ 可选 | Google OAuth 客户端 ID | Web 界面配置 |
| `GOOGLE_CLIENT_SECRET` | ❌ 可选 | Google OAuth 客户端密钥 | Web 界面配置 |
| `TZ` | ❌ 可选 | 时区设置 | `Asia/Shanghai` |

如果希望通过环境变量预设：

```bash
cp .env.example .env
nano .env
```

---

## 📝 Web 管理界面详细说明

### Dashboard（仪表盘）

显示实时统计信息：

- 📊 **账号总数** / **活跃账号数**
- 📈 **今日请求数**
- ⚡ **平均响应延迟**
- 🏆 **模型使用统计**

### Accounts（账号管理）

| 功能 | 说明 |
|------|------|
| 🔍 **搜索过滤** | 按邮箱搜索，按类型筛选（全部/PRO/ULTRA/FREE） |
| ➕ **添加账号** | 多方式添加：Refresh Token / OAuth 授权 / 数据库导入 |
| 📥 **批量导入** | 支持一次粘贴多个 Token，自动识别格式 |
| 🔍 **状态检测** | 一键检测所有账号状态 |
| 📤 **导出** | 导出为 JSON 文件备份 |

#### 支持的 Token 输入格式

1. **单个 Token**
   ```
   1//0gXXXXXXXXXXXXXXXXXXXXXX
   ```

2. **多个 Token（每行一个）**
   ```
   1//0gXXXXXXXXXXXX
   1//0gYYYYYYYYYYYY
   1//0gZZZZZZZZZZZZ
   ```

3. **JSON 数组格式**
   ```json
   [
     {"refresh_token": "1//0gXXXXXX", "email": "user1@gmail.com"},
     {"refresh_token": "1//0gYYYYYY", "email": "user2@gmail.com"}
   ]
   ```

4. **任意包含 Token 的文本**（自动提取 `1//` 开头的 Token）

### Model Router（模型路由）

在 Web 界面直接管理模型映射，无需编辑配置文件！

| 功能 | 说明 |
|------|------|
| ✨ **应用预设映射** | 一键加载常用模型映射配置 |
| 🔄 **重置映射** | 清空所有自定义映射 |
| ➕ **添加映射** | 自定义源模型→目标模型 |

#### 预设映射列表

| 源模型 | 目标模型 |
|--------|----------|
| `claude-haiku-*` | `gemini-2.5-flash-lite` |
| `claude-3-haiku-*` | `gemini-2.5-flash-lite` |
| `claude-3-5-sonnet-*` | `claude-sonnet-4-5` |
| `claude-3-opus-*` | `claude-opus-4-5-thinking` |
| `gpt-4o*` | `gemini-3-flash` |
| `gpt-4*` | `gemini-3-pro-high` |
| `gpt-3.5*` | `gemini-2.5-flash` |
| `o1-*` | `gemini-3-pro-high` |

### Settings（服务配置）

#### 基础配置

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| 监听端口 | 服务监听的端口号 | 8045 |
| 请求超时 | API 请求超时时间（秒） | 120 |
| 局域网访问 | 是否允许局域网其他设备访问 | 关闭 |
| 访问授权 | 是否启用 API 密钥验证 | 关闭 |

#### Google OAuth 凭证

在此配置 Google OAuth 客户端 ID 和密钥，用于 OAuth 授权方式添加账号。

#### API 密钥

- 显示当前 API 密钥
- 🔄 刷新：生成新的 API 密钥
- 📋 复制：复制到剪贴板

#### 调度模式

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| **缓存优先** | 绑定会话与账号，限流时继续等待 | 需要高 Prompt Cache 命中率 |
| **平衡轮换** | 绑定会话，限流时自动切换账号 | 日常使用（推荐） |
| **性能优先** | 无会话绑定，纯随机轮换 | 高并发、不考虑缓存 |

**最大等待时长**：限流时等待的最大秒数（0-300秒）

---

## 🔧 API 使用示例

### Python (OpenAI SDK)

```python
from openai import OpenAI

# 创建客户端，指向本地网关
client = OpenAI(
    base_url="http://127.0.0.1:8045/v1",
    api_key="your-api-key"  # 从 Web 界面 Settings 获取
)

# 发送请求
response = client.chat.completions.create(
    model="claude-sonnet-4-5",  # 会根据路由规则转发
    messages=[
        {"role": "user", "content": "Hello, how are you?"}
    ]
)

print(response.choices[0].message.content)
```

### Claude CLI

```bash
# 设置环境变量
export ANTHROPIC_API_KEY="your-api-key"
export ANTHROPIC_BASE_URL="http://127.0.0.1:8045"

# 启动 Claude CLI
claude
```

### cURL

```bash
curl http://127.0.0.1:8045/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "claude-sonnet-4-5",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "stream": false
  }'
```

### 流式请求

```bash
curl http://127.0.0.1:8045/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'
```

---

## 🔑 获取 Google OAuth 凭证

如需使用 OAuth 授权方式添加账号，需要配置 Google OAuth 凭证：

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. 创建新项目或选择已有项目
3. 进入 **API 和服务** → **凭据**
4. 点击 **创建凭据** → **OAuth 客户端 ID**
5. 应用类型选择 **Web 应用程序**
6. 在 **已获授权的重定向 URI** 中添加：
   ```
   http://localhost:8045/auth/callback
   http://your-server-ip:8045/auth/callback
   ```
7. 创建后，复制 **客户端 ID** 和 **客户端密钥**
8. 在 Web 界面的 **Settings** 页面填入

> 💡 **提示**：如果只使用 Refresh Token 方式添加账号，可以跳过此步骤。

---

## ❓ 常见问题

### Q: 无法访问 Web 界面？

```bash
# 1. 检查服务是否运行
docker-compose ps

# 2. 查看日志
docker-compose logs -f

# 3. 检查防火墙
sudo ufw allow 8045
```

### Q: 如何获取 Refresh Token？

Refresh Token 通常需要通过 Google OAuth 流程获取。您可以：
1. 使用本应用的 OAuth 授权功能（需配置 OAuth 凭证）
2. 使用其他工具获取后粘贴到本应用

### Q: 模型映射不生效？

1. 确认映射规则已保存
2. 检查源模型名称是否匹配（支持通配符 `*`）
3. 刷新页面后重试

### Q: 如何更新到最新版本？

```bash
cd antigravity-lite
git pull
docker-compose down
docker-compose up -d --build
```

---

## 📄 更新日志

### v1.2.0 (2026-01-13)

#### 新功能
- 📊 **使用图表**
  - 24小时请求趋势图 (Chart.js)
  - 模型使用分布饼图
- 🎯 **智能调度增强**
  - 配额优先调度 (ULTRA > PRO > FREE)
  - 429 限流自动跟踪和跳过
  - 会话粘性支持 (Prompt Cache 优化)
- 🔄 **配额监控**
  - 实时配额查询 API
  - Web 端一键刷新配额
  - 订阅类型自动识别
- 🐳 **Docker 优化**
  - 非 root 用户运行 (安全性)
  - 镜像体积优化
  - 资源限制配置
  - 一键部署脚本 (deploy.sh)

### v1.1.0 (2026-01-13)

#### 新功能
- 🎨 **全新 Web 管理界面**
  - 服务配置面板（端口、超时、API 密钥）
  - 调度模式选择（缓存优先/平衡轮换/性能优先）
  - 模型路由 Web 端管理
  - 多标签页账号导入
- 📥 **增强账号导入**
  - 支持批量 Token 粘贴
  - 智能格式识别（单个/JSON/自动提取）
  - 账号类型筛选（PRO/ULTRA/FREE）
- 🛤️ **模型路由增强**
  - 一键应用预设映射
  - 两列网格布局
- 🚀 **简化部署**
  - 环境变量全部可选
  - OAuth 凭证支持 Web 端配置

### v1.0.0

- 初始版本
- 基础账号管理
- API 代理功能
- 模型路由

---

## 📜 许可证

MIT License

---

<p align="center">
  <sub>Made with ❤️ for headless Linux servers</sub>
</p>
