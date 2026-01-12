# Antigravity Lite

轻量级 API 网关，为无图形界面的 Linux 服务器设计。

**✨ 极简部署：克隆项目 → 启动服务 → 打开 Web 界面配置一切！**

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

## 🚀 快速开始（Docker 一键部署）

只需 **3 步**：

### 步骤 1：克隆项目

```bash
git clone https://github.com/Cminors/antigravity-lite.git
cd antigravity-lite
```

### 步骤 2：启动服务

```bash
docker-compose up -d
```

### 步骤 3：打开 Web 管理界面

访问 `http://your-server-ip:8045`，**所有配置都在这里完成**：

1. 在 **Settings** 页面配置 Google OAuth 凭证
2. 在 **Accounts** 页面添加账号（支持 Refresh Token 批量导入）
3. 在 **Model Router** 页面配置模型映射
4. 开始使用！

---

## 环境变量（可选）

所有配置都可以在 Web 界面完成，环境变量是**可选的**。

| 变量名 | 必需 | 说明 | 默认值 |
|--------|------|------|--------|
| `GOOGLE_CLIENT_ID` | ❌ 可选 | Google OAuth 客户端 ID | 可在 Web 界面设置 |
| `GOOGLE_CLIENT_SECRET` | ❌ 可选 | Google OAuth 客户端密钥 | 可在 Web 界面设置 |
| `TZ` | ❌ 可选 | 时区 | `Asia/Shanghai` |

如果需要预设环境变量：

```bash
# 复制模板
cp .env.example .env

# 编辑（可选）
nano .env
```

---

## Web 管理界面功能

### ⚙️ Settings（服务配置）

| 配置项 | 说明 |
|--------|------|
| **监听端口** | 默认 8045 |
| **请求超时** | 范围 30-3600 秒 |
| **Google OAuth 凭证** | 客户端 ID 和密钥 |
| **API 密钥** | 显示、刷新、复制 |
| **局域网访问** | 允许其他设备访问 |
| **访问授权** | 启用 API 密钥验证 |

#### 调度模式

| 模式 | 说明 |
|------|------|
| **缓存优先** | 绑定会话与账号，最大化缓存命中 |
| **平衡轮换** | 限流时自动切换账号（推荐） |
| **性能优先** | 纯随机轮换，适合高并发 |

### 🔐 Accounts（账号管理）

- **搜索过滤**：按邮箱搜索，按类型筛选
- **批量导入**：一次粘贴多个 Token，自动识别格式
- **多方式添加**：Refresh Token / OAuth 授权 / 数据库导入
- **状态检测**：一键检测所有账号

#### 支持的 Token 格式

1. 单个 Token：`1//xxxxx...`
2. JSON 数组：`[{"refresh_token": "1//..."}]`
3. 任意文本：自动提取 Token

### 🛤️ Model Router（模型路由）

- **自定义映射**：源模型 → 目标模型
- **预设映射**：✨ 一键应用常用配置
- **重置映射**：🔄 清空所有

**预设映射：**

```
claude-haiku-* → gemini-2.5-flash-lite
claude-3-opus-* → claude-opus-4-5-thinking
gpt-4o* → gemini-3-flash
gpt-4* → gemini-3-pro-high
```

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

## 其他部署方式

### 手动编译

```bash
git clone https://github.com/Cminors/antigravity-lite.git
cd antigravity-lite
go mod tidy
go build -o antigravity-lite .
./antigravity-lite
```

### Systemd 服务（生产环境）

创建 `/etc/systemd/system/antigravity-lite.service`：

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

启动：

```bash
sudo systemctl daemon-reload
sudo systemctl enable antigravity-lite
sudo systemctl start antigravity-lite
```

---

## 获取 Google OAuth 凭证

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. 创建项目 → **API 和服务** → **凭据**
3. **创建凭据** → **OAuth 客户端 ID** → **Web 应用程序**
4. 添加重定向 URI：
   ```
   http://localhost:8045/auth/callback
   http://your-server-ip:8045/auth/callback
   ```
5. 复制 **客户端 ID** 和 **密钥**，在 Web 界面的 Settings 页面填入

---

## 常见问题

### 无法访问 Web 界面？

```bash
# 检查防火墙
sudo ufw allow 8045

# 检查服务状态
docker-compose ps
docker-compose logs -f
```

### 如何更新？

```bash
cd antigravity-lite
git pull
docker-compose down
docker-compose up -d --build
```

---

## 许可证

MIT License
