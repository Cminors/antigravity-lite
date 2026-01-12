# Changelog

All notable changes to this project will be documented in this file.

## [v1.1.0] - 2026-01-13

### ✨ New Features

#### Web 管理界面增强
- **服务配置面板**：端口、超时、API 密钥、OAuth 凭证全部可在 Web 端配置
- **调度模式选择**：缓存优先 / 平衡轮换 / 性能优先，可调节最大等待时长
- **模型路由 Web 管理**：一键应用预设映射，两列网格布局，实时保存
- **多标签页账号导入**：Refresh Token / OAuth 授权 / 数据库导入

#### 账号管理增强
- **批量 Token 导入**：支持一次粘贴多个 Token
- **智能格式识别**：自动识别单个 Token、JSON 数组、任意文本
- **账号类型筛选**：按 PRO/ULTRA/FREE 筛选
- **搜索功能**：按邮箱快速搜索

#### 部署简化
- **环境变量可选**：所有环境变量均可在 Web 界面设置
- **OAuth 凭证 Web 配置**：无需预先设置环境变量
- **3 步部署**：克隆 → 启动 → Web 配置

### 🔧 Improvements

- 更新 README 文档，添加详细使用说明
- 优化 Docker 部署配置
- 添加预设模型映射配置

### 📦 Technical Changes

- `config.go`：添加调度模式、OAuth 凭证等配置字段
- `web/index.html`：重构为多功能管理界面
- `web/app.js`：添加 Token 解析、筛选、预设映射等功能
- `web/style.css`：添加配置面板、标签页、调度模式等样式

---

## [v1.0.0] - 2026-01-12

### Initial Release

- 基础账号管理
- API 代理（OpenAI/Anthropic 兼容）
- 模型路由
- Web 管理界面
- Docker 支持
