# WeKnora-Slim

基于 [Tencent/WeKnora](https://github.com/Tencent/WeKnora) 裁剪的**纯 RAG 问答后端 API**：保留检索增强问答的完整链路（文档高精度解析 → 混合检索 → 重排 → 引用溯源），去掉了产品化外围（前端、IM 渠道、数据源同步、沙箱、联网搜索、OIDC 等）。本项目为本地存档仓库，不关联上游 remote。

## 保留的能力

- **Chat Pipeline 全套插件**：意图判断、查询改写、查询扩展、混合检索（关键词 + 向量）、rerank、结果合并、引用溯源。
- **知识库**：FAQ / 文档 / Wiki 三类知识库；父子分块；分块编辑与修订。
- **高精度解析**：docreader sidecar（gRPC）+ anydoc 进程内 Office 解析；PDF / Word / PPT / Excel / 图片等多模态。
- **Wiki**：agent 引擎骨架生成互链 Wiki 页 + 知识图谱（工具面已裁剪）。
- **模型接入**：OpenAI 兼容 API、Ollama 等；声明式内置模型（`config/builtin_models.yaml`）。
- **认证与权限**：登录 + API Key（空间级 full_access / 限定知识库）；单租户，空间角色强制鉴权焊死；SystemAdmin 后台路由。
- **对外 MCP Server**（`--profile full`）：把知识库发布给 Cursor / Claude 等 MCP 客户端。
- **Lite 模式**：单二进制 + SQLite + 内存队列，见 `make build-lite` / `scripts/package-lite.sh`。

## 裁剪掉的部分

前端（含 Vite 构建、frontend 镜像）、CLI、小程序、Chrome 扩展、IM 渠道（企微/飞书/Slack 等）、数据源同步（飞书/Confluence/Notion 等）、沙箱与 Agent Skills、联网搜索、长期记忆、embed 评价、审计面板、Langfuse 追踪、多空间 RBAC/组织/分享/邀请注册、OIDC、云对象存储（COS/S3/OSS 等）、多向量库矩阵（保留 postgres 与 qdrant）。

## 技术栈

| 组件 | 选型 |
|------|------|
| 业务库 + BM25 全文 | PostgreSQL / [ParadeDB](https://paradedb.com)（`pg_search`） |
| 向量库 | PostgreSQL(pgvector) 或 Qdrant（`RETRIEVE_DRIVER`，可逗号分隔并行） |
| 知识图谱 | Neo4j（`--profile neo4j`） |
| 对象存储 | MinIO / local（`STORAGE_TYPE`） |
| 高精度解析 | docreader sidecar（gRPC，含 ODL hybrid / Docling 选项） |
| 队列 | Redis（Asynq）；Lite 模式为内存队列 |

## 快速开始

需要 Docker 与 Docker Compose：

```bash
cp .env.example .env    # 按注释修改必填项（数据库、密钥、模型）
docker compose pull
docker compose up -d                          # 核心服务
docker compose --profile neo4j --profile minio --profile full up -d   # 可选组件
```

| 服务 | 地址 |
|------|------|
| 后端 API | `http://localhost:8080` |
| MCP Server | `http://localhost:8082`（`--profile full`） |

所有配置项及注释见 [`.env.example`](./.env.example)；升级、 Lite 单机部署等见 `scripts/` 与 `Makefile`。

## 自建前端

后端保留 SPA 静态托管：把构建产物（含 `index.html`）放进 `web/` 目录（或设置 `WEKNORA_WEB_DIR`）即可由后端直接服务，无需 nginx；`web/index.html` 不存在时该逻辑自动关闭。实现见 `internal/router/static.go`。

## 开发

```bash
make dev-start    # 启动基础设施（PG/Redis/docreader，按需 --qdrant/--neo4j/--minio）
make dev-app      # 本地运行后端（另一终端）
```

后端改动支持 Air 热重载；无需重建镜像。常用命令：`make fmt` / `make lint` / `make test` / `make dev-*`，详见 `make help`。

## 与上游的关系

- 上游仓库（WeKnora）保持只读，本仓库所有改动独立提交。
- 历史变更见 [`CHANGELOG.md`](./CHANGELOG.md)（上游遗留，仅作参考）。

## License

[MIT](./LICENSE)（继承自上游 Tencent/WeKnora）。
