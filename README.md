# Akasha (阿卡夏)

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![React Version](https://img.shields.io/badge/React-18+-61DAFB?logo=react)](https://react.dev/)

**Akasha** 是一个旨在汇聚全宇宙智慧（LLM）的 AI 网关系统。它致力于成为连接用户与各类 AI 服务的坚实桥梁，不仅提供基础的模型转发功能，更深度融合了现代化的用户管理、风控体系以及特色社区（如 LinuxDO）生态。

## ✨ 核心特性

*   **多模型支持**: 原生支持 OpenAI (GPT-3.5/4), Anthropic (Claude 3), Google (Gemini Pro), Midjourney 等主流模型。
*   **渠道管理**: 支持多渠道负载均衡、自动重试、优先级分组，确保服务高可用。
*   **Midjourney 深度集成**: 支持 Imagine, Upscale, Variation, Blend 等操作，包含任务追踪、自动扣费与失败退款机制。
*   **用户体系**:
    *   支持 邮箱、GitHub OAuth、LinuxDO OAuth 多种登录方式。
    *   **LinuxDO 深度集成**: 自动同步 LinuxDO 信任等级，支持按等级自定义初始额度。
*   **运营风控**:
    *   支持 充值兑换码 生成与使用。
    *   集成 **Cloudflare Turnstile** 人机验证，防止恶意注册。
    *   支持 SMTP 邮件服务，用于注册验证与通知。
*   **额度管理**: 精细化的额度控制（令牌级），支持自定义模型倍率。
*   **现代化前端**: 基于 React + HeroUI 构建的响应式管理界面，支持深色模式。

## 🛠️ 技术栈

*   **后端**: Go (Gin, GORM, SQLite/MySQL/PostgreSQL)
*   **前端**: React, Vite, HeroUI (NextUI), TailwindCSS
*   **部署**: Docker, Docker Compose

## 🚀 快速开始

### 前置要求

*   Go 1.21+
*   Node.js 18+
*   MySQL 5.7+ (可选，默认使用 SQLite)

### 本地开发

1.  **后端启动**

    ```bash
    cd backend
    go mod tidy
    go run main.go --port 8080 --driver sqlite --dsn "akasha.db"
    ```

2.  **前端启动**

    ```bash
    cd frontend
    npm install
    npm run dev
    ```

    访问 `http://localhost:5173` 即可看到管理界面。

### Docker 部署

*(即将推出 Docker 镜像)*

## ⚙️ 配置说明

系统启动后，请使用默认生成的管理员账号（首个注册用户自动成为 Root 管理员）登录，并在 **系统设置** 页面进行配置：

*   **GitHub OAuth**: 填入 Client ID 和 Secret。
*   **LinuxDO OAuth**: 填入 Client ID 和 Secret，并设置各等级初始额度。
*   **邮件服务**: 配置 SMTP 服务器信息以启用邮件验证。
*   **Midjourney**: 添加类型为 `midjourney` 的渠道，BaseURL 填写 `mj-proxy` 地址。

## 📜 开源协议

本项目采用 **GNU Affero General Public License v3.0 (AGPL-3.0)** 协议开源。

这意味着：
1.  如果您修改了本项目代码并对外提供网络服务，您**必须**公开您的修改源码。
2.  您必须在界面上保留原作者的版权声明。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---
Powered by [Akasha Team](https://github.com/yourusername/Akasha)
