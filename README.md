# Antigravity Lite

轻量级 API 网关，为无图形界面的 Linux 服务器设计。

## 功能特性

- 🔐 **账号管理** - 多账号管理，支持导入导出
- 🔌 **API 代理** - 兼容 OpenAI/Anthropic 协议
- 🔀 **模型路由** - 灵活的模型别名映射
- 📊 **配额监控** - 请求统计和使用分析
- 🌐 **Web 界面** - 现代暗色主题管理面板

## 资源占用

| 指标 | 数值 |
|------|------|
| 二进制大小 | ~10-15 MB |
| 内存占用 | ~20-50 MB |
| CPU | 极低 |

## 快速开始

### 方式一：Docker 一键部署 (推荐)

```bash
# 克隆项目
git clone https://github.com/Cminors/antigravity-lite.git
cd antigravity-lite

# 配置环境变量
cp .env.example .env
nano .env  # 填入 Google OAuth 凭证

# 编辑配置文件
nano config.yaml

# 一键启动
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 方式二：手动编译

```bash
# 安装 Go 1.21+
go mod tidy
go build -o antigravity-lite .

# 交叉编译 Linux 版本
GOOS=linux GOARCH=amd64 go build -o antigravity-lite-linux .
```

### 部署到服务器

```bash
# 上传文件
scp antigravity-lite-linux user@your-server:/opt/antigravity-lite/

# 设置环境变量并运行
export GOOGLE_CLIENT_ID="your-client-id"
export GOOGLE_CLIENT_SECRET="your-client-secret"
./antigravity-lite
```

### Systemd 服务

创建 `/etc/systemd/system/antigravity-lite.service`:

```ini
[Unit]
Description=Antigravity Lite API Gateway
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/antigravity-lite
Environment="GOOGLE_CLIENT_ID=your-client-id"
Environment="GOOGLE_CLIENT_SECRET=your-client-secret"
ExecStart=/opt/antigravity-lite/antigravity-lite
Restart=always

[Install]
WantedBy=multi-user.target
```

## 环境变量

| 变量名 | 必需 | 说明 |
|--------|------|------|
| `GOOGLE_CLIENT_ID` | Yes | Google OAuth Client ID |
| `GOOGLE_CLIENT_SECRET` | Yes | Google OAuth Client Secret |

## API 使用

**Python:**

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

**Claude CLI:**

```bash
export ANTHROPIC_API_KEY="your-api-key"
export ANTHROPIC_BASE_URL="http://127.0.0.1:8045"
claude
```

## 配置文件

编辑 `config.yaml` 自定义配置。

## 许可证

MIT License
