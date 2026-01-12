# Antigravity Lite

轻量级 API 网关，为无图形界面的 Linux 服务器设计。**克隆项目后，全部在 Web 管理界面配置！**

## 功能特性

- 🔐 **账号管理** - 多账号管理，支持批量导入、类型筛选（PRO/ULTRA/FREE）
- 🔌 **API 代理** - 兼容 OpenAI/Anthropic 协议
- 🛤️ **模型路由** - Web 端管理，一键应用预设映射
- 📊 **调度模式** - 缓存优先/平衡轮换/性能优先
- 🌐 **Web 界面** - 现代暗色主题管理面板，全功能配置

## 资源占用

| 指标 | 数值 |
|------|------|
| 二进制大小 | ~10-15 MB |
| 内存占用 | ~20-50 MB |
| CPU | 极低 |

---

## 快速开始

### 方式一：Docker 一键部署 (推荐)

> 适合不想手动编译、希望快速上手的用户。

#### 第一步：获取 Google OAuth 凭证

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
7. 点击创建后，记下 **客户端 ID** 和 **客户端密钥**

#### 第二步：克隆项目

```bash
# 克隆仓库到本地
git clone https://github.com/Cminors/antigravity-lite.git

# 进入项目目录
cd antigravity-lite
```

#### 第三步：配置环境变量

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑环境变量文件
nano .env
```

在 `.env` 文件中填入你的 Google OAuth 凭证：

```env
GOOGLE_CLIENT_ID=你的客户端ID.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=你的客户端密钥
```

#### 第四步：启动服务

```bash
# 使用 Docker Compose 一键启动
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看实时日志
docker-compose logs -f
```

#### 第五步：访问 Web 管理界面配置一切

打开浏览器，访问：

```
http://your-server-ip:8045
```

**所有配置都可以在 Web 界面完成！** 无需再手动编辑任何配置文件。

---

### 方式二：手动编译部署

> 适合需要自定义编译或无法使用 Docker 的用户。

```bash
# 克隆项目
git clone https://github.com/Cminors/antigravity-lite.git
cd antigravity-lite

# 下载依赖
go mod tidy

# 编译
go build -o antigravity-lite .

# 设置环境变量并运行
export GOOGLE_CLIENT_ID="你的客户端ID"
export GOOGLE_CLIENT_SECRET="你的客户端密钥"
./antigravity-lite
```

然后访问 `http://localhost:8045` 进行 Web 端配置。

---

## Web 管理界面功能

### 📊 Dashboard

实时显示账号数量、活跃账号、今日请求数、平均延迟等统计信息。

### 🔐 Accounts（账号管理）

| 功能 | 说明 |
|------|------|
| **搜索过滤** | 按邮箱搜索，按类型筛选（PRO/ULTRA/FREE） |
| **批量导入** | 一次性粘贴多个 Token，自动识别格式 |
| **多方式添加** | Refresh Token / OAuth 授权 / 数据库导入 |
| **状态检测** | 一键检测所有账号状态 |
| **导入导出** | JSON 格式导入导出账号 |

#### 添加账号支持的格式

1. **单个 Token**：`1//xxxxx...`
2. **JSON 数组**：`[{"refresh_token": "1//..."}]`
3. **任意文本**：自动提取包含的 Token

### 🛤️ Model Router（模型路由）

在 Web 端直接管理模型映射，无需编辑配置文件！

| 功能 | 说明 |
|------|------|
| **自定义映射** | 添加源模型→目标模型的映射规则 |
| **预设映射** | ✨ 一键应用常用映射配置 |
| **重置映射** | 🔄 清空所有映射 |

**预设映射包括：**

```
claude-haiku-*     → gemini-2.5-flash-lite
claude-3-haiku-*   → gemini-2.5-flash-lite
claude-3-5-sonnet-* → claude-sonnet-4-5
claude-3-opus-*    → claude-opus-4-5-thinking
gpt-4o*            → gemini-3-flash
gpt-4*             → gemini-3-pro-high
gpt-3.5*           → gemini-2.5-flash
o1-*               → gemini-3-pro-high
```

### ⚙️ Settings（服务配置）

#### 基础配置

| 配置项 | 说明 |
|--------|------|
| **监听端口** | 默认 8045 |
| **请求超时** | 范围 30-3600 秒，默认 120 秒 |
| **局域网访问** | 开启后允许局域网其他设备访问 |
| **访问授权** | 开启后需要 API 密钥验证 |

#### API 密钥

- 显示当前 API 密钥
- 🔄 刷新生成新密钥
- 📋 一键复制密钥

#### 调度模式

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| **缓存优先** | 绑定会话与账号，限流时继续等待 | 最大化 Prompt Cache 命中率 |
| **平衡轮换** | 绑定会话，限流时自动切换账号 | 兼顾缓存与可用性（推荐） |
| **性能优先** | 无会话绑定，纯随机轮换 | 高并发场景 |

还可以设置 **最大等待时长**（0-300 秒）。

---

## API 使用示例

### Python (OpenAI SDK)

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://127.0.0.1:8045/v1",
    api_key="your-api-key"  # 从 Web 界面获取
)

response = client.chat.completions.create(
    model="claude-sonnet-4-5",
    messages=[{"role": "user", "content": "Hello!"}]
)

print(response.choices[0].message.content)
```

### Claude CLI

```bash
export ANTHROPIC_API_KEY="your-api-key"
export ANTHROPIC_BASE_URL="http://127.0.0.1:8045"
claude
```

### cURL

```bash
curl http://127.0.0.1:8045/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "claude-sonnet-4-5",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

---

## 环境变量

| 变量名 | 必需 | 说明 | 示例 |
|--------|------|------|------|
| `GOOGLE_CLIENT_ID` | ✅ 是 | Google OAuth 客户端 ID | `123456789.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET` | ✅ 是 | Google OAuth 客户端密钥 | `GOCSPX-xxxxxx` |

---

## 使用 Systemd 服务（生产环境推荐）

创建服务文件 `/etc/systemd/system/antigravity-lite.service`：

```ini
[Unit]
Description=Antigravity Lite API Gateway
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/antigravity-lite
Environment="GOOGLE_CLIENT_ID=你的客户端ID"
Environment="GOOGLE_CLIENT_SECRET=你的客户端密钥"
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
```

---

## 常见问题

### Q: 无法访问 Web 界面？

```bash
# 检查防火墙
sudo ufw allow 8045

# 检查服务状态
sudo systemctl status antigravity-lite
```

### Q: Google OAuth 登录失败？

1. 确保 **重定向 URI** 配置正确
2. 确保环境变量设置正确
3. 检查服务器时间是否准确

### Q: 如何更新到最新版本？

**Docker 方式：**
```bash
cd antigravity-lite
git pull
docker-compose down
docker-compose up -d --build
```

---

## 许可证

MIT License
