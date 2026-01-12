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

#### 第四步：编辑配置文件（可选）

```bash
# 编辑配置文件
nano config.yaml
```

可以根据需要修改端口、数据库路径等配置。

#### 第五步：启动服务

```bash
# 使用 Docker Compose 一键启动
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看实时日志
docker-compose logs -f
```

#### 第六步：访问 Web 管理界面

打开浏览器，访问：

```
http://your-server-ip:8045
```

---

### 方式二：手动编译部署

> 适合需要自定义编译或无法使用 Docker 的用户。

#### 第一步：安装 Go 环境

**Ubuntu/Debian:**

```bash
# 下载 Go 1.21+
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz

# 解压到 /usr/local
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# 添加到 PATH（写入 ~/.bashrc）
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 验证安装
go version
```

**CentOS/RHEL:**

```bash
sudo yum install golang
```

#### 第二步：克隆并编译

```bash
# 克隆项目
git clone https://github.com/Cminors/antigravity-lite.git
cd antigravity-lite

# 下载依赖
go mod tidy

# 编译（本机运行）
go build -o antigravity-lite .

# 或：交叉编译 Linux amd64 版本（在 Windows/Mac 上编译给 Linux 用）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o antigravity-lite-linux-amd64 .
```

#### 第三步：上传到服务器

```bash
# 在服务器上创建目录
ssh user@your-server "mkdir -p /opt/antigravity-lite"

# 上传二进制文件
scp antigravity-lite-linux-amd64 user@your-server:/opt/antigravity-lite/antigravity-lite

# 上传配置文件
scp config.yaml user@your-server:/opt/antigravity-lite/
```

#### 第四步：在服务器上配置并运行

```bash
# SSH 登录到服务器
ssh user@your-server

# 进入应用目录
cd /opt/antigravity-lite

# 赋予执行权限
chmod +x antigravity-lite

# 设置环境变量并运行
export GOOGLE_CLIENT_ID="你的客户端ID"
export GOOGLE_CLIENT_SECRET="你的客户端密钥"
./antigravity-lite
```

---

### 方式三：使用 Systemd 服务（生产环境推荐）

> 适合需要开机自启、后台运行的生产环境。

#### 第一步：创建服务文件

```bash
sudo nano /etc/systemd/system/antigravity-lite.service
```

粘贴以下内容：

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

#### 第二步：启用并启动服务

```bash
# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 启用开机自启
sudo systemctl enable antigravity-lite

# 启动服务
sudo systemctl start antigravity-lite

# 查看服务状态
sudo systemctl status antigravity-lite

# 查看实时日志
sudo journalctl -u antigravity-lite -f
```

#### 第三步：常用管理命令

```bash
# 停止服务
sudo systemctl stop antigravity-lite

# 重启服务
sudo systemctl restart antigravity-lite

# 禁用开机自启
sudo systemctl disable antigravity-lite
```

---

## 环境变量

| 变量名 | 必需 | 说明 | 示例 |
|--------|------|------|------|
| `GOOGLE_CLIENT_ID` | ✅ 是 | Google OAuth 客户端 ID | `123456789.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET` | ✅ 是 | Google OAuth 客户端密钥 | `GOCSPX-xxxxxx` |

---

## API 使用示例

### Python (OpenAI SDK)

```bash
# 安装 OpenAI SDK
pip install openai
```

```python
from openai import OpenAI

# 创建客户端，指向本地网关
client = OpenAI(
    base_url="http://127.0.0.1:8045/v1",
    api_key="your-api-key"  # 从 Web 界面获取
)

# 发送请求
response = client.chat.completions.create(
    model="claude-sonnet-4-5",
    messages=[
        {"role": "user", "content": "Hello, how are you?"}
    ]
)

# 打印回复
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
    ]
  }'
```

---

## 配置文件说明

编辑 `config.yaml` 自定义配置：

```yaml
server:
  port: 8045           # 服务端口
  host: "0.0.0.0"      # 监听地址

database:
  path: "./data/antigravity.db"  # SQLite 数据库路径

logging:
  level: "info"        # 日志级别: debug, info, warn, error
```

---

## 常见问题

### Q: 无法访问 Web 界面？

1. 检查防火墙是否开放了 8045 端口：
   ```bash
   sudo ufw allow 8045
   ```
2. 检查服务是否正常运行：
   ```bash
   sudo systemctl status antigravity-lite
   ```

### Q: Google OAuth 登录失败？

1. 确保 **重定向 URI** 配置正确
2. 确保环境变量 `GOOGLE_CLIENT_ID` 和 `GOOGLE_CLIENT_SECRET` 设置正确
3. 检查服务器时间是否准确

### Q: 如何更新到最新版本？

**Docker 方式：**
```bash
cd antigravity-lite
git pull
docker-compose down
docker-compose up -d --build
```

**二进制方式：**
```bash
# 下载新版本二进制文件并替换
sudo systemctl stop antigravity-lite
# 替换二进制文件...
sudo systemctl start antigravity-lite
```

---

## 许可证

MIT License
