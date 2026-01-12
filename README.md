# Antigravity Lite

轻量�?API 网关，为无图形界面的 Linux 服务器设计�?

## 功能特�?

- 🔐 **账号管理** - 多账号管理，支持导入导出
- 🔌 **API 代理** - 兼容 OpenAI/Anthropic 协议
- 🔀 **模型路由** - 灵活的模型别名映�?
- 📊 **配额监控** - 请求统计和使用分�?
- 🌐 **Web 界面** - 现代暗色主题管理面板

## 资源占用

| 指标 | 数�?|
|------|------|
| 二进制大�?| ~10-15 MB |
| 内存占用 | ~20-50 MB |
| CPU | 极低 |

## 快速开�?

### 方式一：Docker 一键部�?(推荐)

```bash
# 克隆或下载项目后
cd antigravity-lite

# 编辑配置文件
nano config.yaml

# 一键启�?
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 方式二：手动编译

```bash
# 安装 Go 1.21+
# 在项目目录下执行�?
go mod tidy
go build -o antigravity-lite .

# 交叉编译 Linux 版本
GOOS=linux GOARCH=amd64 go build -o antigravity-lite-linux .
```

### 部署到服务器 (手动编译方式)

```bash
# 上传文件
scp antigravity-lite-linux user@your-server:/opt/antigravity-lite/antigravity-lite
scp config.yaml user@your-server:/opt/antigravity-lite/

# SSH 到服务器
ssh user@your-server

# 设置权限
chmod +x /opt/antigravity-lite/antigravity-lite

# 运行
cd /opt/antigravity-lite
./antigravity-lite
```

### 3. 设置开机自�?(systemd)

创建 `/etc/systemd/system/antigravity-lite.service`:

```ini
[Unit]
Description=Antigravity Lite API Gateway
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/antigravity-lite
ExecStart=/opt/antigravity-lite/antigravity-lite
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启用服务:

```bash
sudo systemctl daemon-reload
sudo systemctl enable antigravity-lite
sudo systemctl start antigravity-lite
sudo systemctl status antigravity-lite
```

## 使用方法

### 访问 Web 管理界面

打开浏览器访�? `http://你的服务器IP:8045`

### 添加账号

1. 在其他设备（有浏览器的电脑）获取 Google OAuth Refresh Token
2. �?Web 界面 �?账号管理 �?添加账号
3. 粘贴 Refresh Token

### API 使用

**Python (OpenAI SDK):**

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://127.0.0.1:8045/v1",
    api_key="your-api-key"
)

response = client.chat.completions.create(
    model="claude-sonnet-4-5",
    messages=[{"role": "user", "content": "Hello"}]
)

print(response.choices[0].message.content)
```

> **注意**: API Key 需要在 `config.yaml` 中配置�?

**Claude CLI:**

```bash
export ANTHROPIC_API_KEY="your-api-key"
export ANTHROPIC_BASE_URL="http://127.0.0.1:8045"
claude
```

**cURL:**

```bash
curl http://127.0.0.1:8045/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "gemini-3-flash",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## HTTPS 配置 (推荐)

使用 Nginx 反向代理:

```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:8045;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_buffering off;
    }
}
```

## 配置文件

编辑 `config.yaml` 自定义配置，详见文件内注释�?

## 许可�?

MIT License
