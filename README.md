# Eidolon Wiki

基于 [WeKnora](https://github.com/Tencent/WeKnora) 裁剪重建的团队 RAG 知识库，前后端单仓。

## 项目结构

| 目录 | 说明 |
| --- | --- |
| `web/` | Vue 3 + TypeScript + Vite 前端（SAG Wiki App） |
| `server/` | Go 纯 RAG 问答后端（WeKnora-Slim，含 docreader 文档解析服务） |

各自的启动方式、环境变量与详细说明见 `web/README.md` 和 `server/README.md`。
