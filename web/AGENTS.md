# AGENTS.md

自研 WeKnora 前端：Vue 3 + shadcn-vue + TS，直连 WeKnora 后端。观感基准是当前代码的管理系统式壳，不照官方前端的 SaaS 观感。

## 参考项目在哪（只读，禁止修改其中任何文件）

**WeKnora 完整源码**：`/Users/jujiuyey/ReferenceCode/WeKnora`（v0.7.2）

- `frontend/` — 官方前端（Vue 3 + TDesign）。理解产品行为、交互、接口语义的主参考；逻辑可近 1:1 借鉴，注明「参考实现」来源，观感不跟
- `docreader/` — Python 文档解析服务（parser/、splitter/），多格式解析的核心
- `internal/` — Go 主后端（handler、datasource、agent、sandbox、event、mcp 等）
- `client/` `cli/` `mcp-server/` `miniprogram/` `helm/` — 边缘能力，本项目不涉及

仓库内参考：

- `react-reference/` — 自家 sag-wiki React 前端的部分拷贝，组件模式（sag-ui、TreeSelect 等）的移植来源
- `~/Projects/sag-wiki` — 用户的自研全栈企业知识库（Go + React），RAG 概念可对照参考

## 边界

- 只改本仓库前端；不动参考项目、后端、接口协议、部署
- 已明确不追：agent 聊天、沙箱、附件、im、mcp、小程序
- 部署模型 = 单空间（官方两层已坍缩成一层）

## 工作方式

- 做功能前先读参考项目对应实现；交付时说明「参考实现 / 当前实现 / 差异」
- 信息架构、视觉方向、新依赖等方向性决定：先给方案，等用户拍板
- 不用占位数据或空按钮冒充完成；类型检查、构建过了再报完成；未验证的明说
- 中文沟通，直接、具体、以实际代码为依据
