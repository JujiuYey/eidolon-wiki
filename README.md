# SAG Wiki

SAG Wiki 是一个面向团队知识沉淀、检索与问答的可自托管 RAG 工作台。它将目录与权限、文档处理、向量索引、模型配置和检索增强问答放在同一个系统中。

[项目介绍站](https://jujiuyey.github.io/sag-wiki/) · [快速开始](https://jujiuyey.github.io/sag-wiki/guide/getting-started) · [系统架构](https://jujiuyey.github.io/sag-wiki/guide/architecture)

> 项目仍在开发阶段。知识库、文档处理、系统管理和后端 RAG 链路已有实现；部分前端会话数据及 Agent 页面仍包含演示或占位逻辑。

## 核心能力

- **知识库管理**：文件夹树、文档上传、内容编辑、下载、重新处理与切片查看。
- **权限边界**：按用户、角色和部门配置文件夹访问范围，检索前扩展可访问目录。
- **异步文档处理**：通过 Redis / Asynq 执行解析、切片和向量化任务，并提供队列状态查询。
- **RAG 问答**：基于 Qdrant 召回相关切片，组合模型配置生成回答。
- **模型与向量库管理**：管理 OpenAI-compatible 模型配置、Qdrant Collection 和 Point。
- **系统管理**：用户、角色、部门、头像、密码和任务队列管理。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| Web | React 19、TypeScript、Vite 7、TanStack Router / Query、Tailwind CSS 4、Tiptap |
| API | Go 1.25、Fiber、Bun ORM、Eino |
| 数据 | PostgreSQL 15、Qdrant、Redis 7、MinIO |
| 介绍站 | Rspress 2、React、GitHub Pages |

## 项目结构

```text
sag-wiki/
├── server/                 # Go API
│   ├── app/                # 按 ai、wiki、system、file 划分的业务模块
│   ├── infrastructure/     # PostgreSQL、Qdrant、Redis、MinIO 适配
│   ├── migrations/         # system、wiki、ai 数据库迁移
│   ├── middleware/         # JWT 与角色校验
│   └── router/             # HTTP 路由聚合
├── web/                    # React 管理端
│   └── src/
│       ├── api/            # API 客户端与类型
│       ├── components/     # 通用组件与编辑器组件
│       ├── pages/          # 知识库、AI、系统管理页面
│       ├── routes/         # TanStack Router 文件路由
│       └── stores/         # Zustand 状态
├── site/                   # Rspress 项目介绍与文档站
├── docs/                   # 设计记录与实现计划
└── docker-compose.yml      # 本地基础设施
```

## 本地启动

### 环境要求

- Go 1.25.3+
- Node.js 20.19+ 或 22.12+
- pnpm 10+
- Docker 与 Docker Compose

### 1. 启动基础设施

```bash
docker compose up -d
```

Docker Compose 会启动 PostgreSQL、Redis、MinIO 和 Qdrant。首次运行时创建数据库并按顺序执行迁移：

```bash
docker compose exec postgres createdb -U postgres sag_wiki
docker compose exec -T postgres psql -U postgres -d sag_wiki < server/migrations/system/001_schema.sql
docker compose exec -T postgres psql -U postgres -d sag_wiki < server/migrations/wiki/001_schema.sql
docker compose exec -T postgres psql -U postgres -d sag_wiki < server/migrations/ai/001_schema.sql
docker compose exec -T postgres psql -U postgres -d sag_wiki < server/migrations/ai/002_schema.sql
```

### 2. 启动后端

```bash
cd server
cp .env.example .env
# 修改 .env，确保数据库和对象存储凭据与 docker-compose.yml 一致
go mod download
go run main.go
```

API 默认运行在 `http://localhost:8080`。

### 3. 启动前端

```bash
cd web
cp .env.example .env
pnpm install
pnpm dev
```

Web 应用默认运行在 `http://localhost:5200`。

### 4. 启动介绍站

```bash
cd site
pnpm install
pnpm dev
```

## 常用命令

```bash
# Web
cd web
pnpm lint:type
pnpm test:run
pnpm build

# Go API
cd server
go test ./...
go build ./...

# Rspress 站点
cd site
pnpm build
pnpm preview
```

## GitHub Pages

介绍站由 [`.github/workflows/deploy-pages.yml`](.github/workflows/deploy-pages.yml) 自动部署。首次使用时，在仓库 `Settings > Pages > Build and deployment` 中将 Source 设为 **GitHub Actions**。之后推送到 `main` 的 `site/` 或部署工作流变更会触发发布。

当前仓库是项目 Pages，Rspress 的 `base` 已配置为 `/sag-wiki/`。如果改用自定义域名，需要同步调整 `site/rspress.config.ts`。

## License

项目尚未添加 License 文件。在明确许可证之前，请不要将仓库内容视为已授权的开源软件。
