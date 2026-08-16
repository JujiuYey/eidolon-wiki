import {
  ArrowRight,
  BookOpenText,
  Boxes,
  Check,
  Database,
  FileText,
  Folder,
  Github,
  GitPullRequestArrow,
  Menu,
  MessageSquareText,
  Network,
  Search,
  ShieldCheck,
  Sparkles,
  Upload,
  Workflow,
} from "lucide-react";

import logoUrl from "./assets/logo.svg";

export const frontmatter = {
  pageType: "blank",
};

const repoUrl = "https://github.com/JujiuYey/sag-wiki";

function Brand() {
  return (
    <a className="site-brand" href="./" aria-label="返回 SAG Wiki 首页">
      <img src={logoUrl} alt="" />
      <span>SAG Wiki</span>
    </a>
  );
}

function SiteHeader() {
  return (
    <header className="site-header">
      <div className="site-shell header-inner">
        <Brand />
        <nav className="desktop-nav" aria-label="主导航">
          <a href="#workflow">工作流</a>
          <a href="#capabilities">核心能力</a>
          <a href="#architecture">架构</a>
          <a href="./guide/getting-started">文档</a>
        </nav>
        <div className="header-actions">
          <a className="icon-link" href={repoUrl} aria-label="在 GitHub 查看 SAG Wiki">
            <Github aria-hidden="true" />
          </a>
          <a className="header-cta" href="./guide/getting-started">
            开始部署
            <ArrowRight aria-hidden="true" />
          </a>
          <details className="mobile-menu">
            <summary aria-label="打开导航菜单">
              <Menu aria-hidden="true" />
            </summary>
            <nav aria-label="移动端导航">
              <a href="#workflow">工作流</a>
              <a href="#capabilities">核心能力</a>
              <a href="#architecture">架构</a>
              <a href="./guide/getting-started">文档</a>
              <a href={repoUrl}>GitHub</a>
            </nav>
          </details>
        </div>
      </div>
    </header>
  );
}

function ProductPreview() {
  return (
    <div className="product-preview" aria-label="SAG Wiki 产品界面预览">
      <div className="preview-bar">
        <div className="window-controls" aria-hidden="true">
          <span />
          <span />
          <span />
        </div>
        <div className="preview-location">
          <ShieldCheck aria-hidden="true" />
          团队知识空间 / 产品研发
        </div>
        <div className="sync-state"><span />已同步</div>
      </div>

      <div className="preview-body">
        <aside className="preview-sidebar">
          <div className="preview-brand">
            <img src={logoUrl} alt="" />
          </div>
          <div className="preview-nav-item is-active"><BookOpenText /><span>知识库</span></div>
          <div className="preview-nav-item"><MessageSquareText /><span>知识问答</span></div>
          <div className="preview-nav-item"><Boxes /><span>模型配置</span></div>
          <div className="preview-nav-item"><Workflow /><span>任务队列</span></div>
        </aside>

        <section className="preview-files">
          <div className="preview-section-heading">
            <div>
              <small>知识库</small>
              <strong>产品研发</strong>
            </div>
            <button type="button" aria-label="上传文档"><Upload /></button>
          </div>
          <div className="preview-search"><Search /><span>搜索文件与文件夹</span></div>
          <div className="folder-row is-selected">
            <span className="file-icon folder"><Folder /></span>
            <span><strong>产品规范</strong><small>12 个文档</small></span>
          </div>
          <div className="folder-row">
            <span className="file-icon"><FileText /></span>
            <span><strong>检索策略.md</strong><small>已索引 · 28 个切片</small></span>
          </div>
          <div className="folder-row">
            <span className="file-icon pdf">PDF</span>
            <span><strong>需求评审手册.pdf</strong><small>已索引 · 64 个切片</small></span>
          </div>
          <div className="index-state">
            <Database />
            <span><strong>104</strong><small>可检索切片</small></span>
            <span className="state-check"><Check /></span>
          </div>
        </section>

        <section className="preview-chat">
          <div className="chat-heading">
            <span><Sparkles />基于产品研发知识库</span>
            <small>3 个来源</small>
          </div>
          <div className="question-bubble">发布前，需求评审必须确认哪些内容？</div>
          <div className="answer-block">
            <div className="answer-avatar"><Sparkles /></div>
            <div>
              <p>发布前需要确认范围、验收标准与回滚方案，并完成权限和数据影响检查。</p>
              <div className="source-list">
                <span><FileText />需求评审手册.pdf · P.12</span>
                <span><FileText />发布流程.md · 第 3 节</span>
              </div>
            </div>
          </div>
          <div className="retrieval-note">
            <Network />
            <div><strong>检索链路可追溯</strong><small>问题向量化 · 权限过滤 · 片段召回 · 回答生成</small></div>
          </div>
        </section>
      </div>
    </div>
  );
}

const workflowSteps = [
  { index: "01", icon: Upload, title: "接入资料", detail: "上传团队文档并归入权限目录" },
  { index: "02", icon: GitPullRequestArrow, title: "解析切片", detail: "异步提取正文，形成可管理片段" },
  { index: "03", icon: Database, title: "向量索引", detail: "生成 Embedding 并写入 Qdrant" },
  { index: "04", icon: MessageSquareText, title: "可信回答", detail: "按权限召回来源，组合上下文回答" },
];

const capabilities = [
  {
    icon: Folder,
    title: "知识不是一堆附件",
    detail: "用文件夹树、文档状态和细粒度权限组织资料，原文件、正文与切片保持同一条追踪链路。",
    meta: "目录 · 文档 · 权限",
  },
  {
    icon: Search,
    title: "检索范围先于模型",
    detail: "先根据用户权限扩展可访问目录，再执行向量召回，让模型只看见当前用户有权使用的上下文。",
    meta: "范围 · 召回 · 来源",
  },
  {
    icon: Workflow,
    title: "耗时任务留在后台",
    detail: "文档解析、切片和向量化通过 Redis 队列异步执行，前台可以查看任务状态并重新处理失败文档。",
    meta: "队列 · 状态 · 重试",
  },
];

export default function HomePage() {
  return (
    <div className="landing-page">
      <SiteHeader />
      <main>
        <section className="hero-section">
          <div className="site-shell hero-grid">
            <div className="hero-copy">
              <div className="eyebrow"><span />SELF-HOSTED RAG WORKSPACE</div>
              <h1>SAG Wiki</h1>
              <p className="hero-statement">让团队资料真正参与每一次回答。</p>
              <p className="hero-description">
                从文档入库、权限管理到向量检索与来源追踪，把分散的团队知识变成可运营的问答工作台。
              </p>
              <div className="hero-actions">
                <a className="primary-button" href="./guide/getting-started">
                  查看快速开始
                  <ArrowRight aria-hidden="true" />
                </a>
                <a className="secondary-button" href={repoUrl}>
                  <Github aria-hidden="true" />
                  GitHub
                </a>
              </div>
              <div className="hero-proof" aria-label="技术栈">
                <span>React 19</span>
                <span>Go + Fiber</span>
                <span>Qdrant</span>
                <span>MinIO</span>
              </div>
              <a className="mobile-next-hint" href="#workflow">
                <span>下一步</span>
                从文档入库到可信回答
                <ArrowRight aria-hidden="true" />
              </a>
            </div>
            <ProductPreview />
          </div>
          <div className="hero-rule" aria-hidden="true"><span>DOCUMENTS IN</span><span>GROUNDED ANSWERS OUT</span></div>
        </section>

        <section className="workflow-section" id="workflow">
          <div className="site-shell section-heading-row">
            <div>
              <span className="section-kicker">KNOWLEDGE PIPELINE</span>
              <h2>一条能看见、能追踪的<span className="mobile-break"><br /></span>知识链路</h2>
            </div>
            <p>不是把文件直接塞给模型，而是明确处理过程、检索边界和回答来源。</p>
          </div>
          <div className="site-shell workflow-grid">
            {workflowSteps.map((step) => {
              const Icon = step.icon;
              return (
                <article className="workflow-step" key={step.index}>
                  <div className="step-top"><span>{step.index}</span><Icon aria-hidden="true" /></div>
                  <h3>{step.title}</h3>
                  <p>{step.detail}</p>
                </article>
              );
            })}
          </div>
        </section>

        <section className="capabilities-section" id="capabilities">
          <div className="site-shell">
            <div className="section-heading-row compact">
              <div>
                <span className="section-kicker">BUILT FOR OPERATIONS</span>
                <h2><span className="heading-line">知识库、检索、系统能力</span><span className="heading-line">放在同一个工作台</span></h2>
              </div>
            </div>
            <div className="capabilities-grid">
              {capabilities.map((capability) => {
                const Icon = capability.icon;
                return (
                  <article className="capability" key={capability.title}>
                    <div className="capability-icon"><Icon aria-hidden="true" /></div>
                    <span className="capability-meta">{capability.meta}</span>
                    <h3>{capability.title}</h3>
                    <p>{capability.detail}</p>
                  </article>
                );
              })}
            </div>
          </div>
        </section>

        <section className="architecture-section" id="architecture">
          <div className="site-shell architecture-grid">
            <div className="architecture-copy">
              <span className="section-kicker light">COMPOSABLE BY DESIGN</span>
              <h2><span className="heading-line">每类数据，</span><span className="heading-line">交给适合它的系统。</span></h2>
              <p>业务元数据、原始文件、任务状态和向量索引彼此独立，通过清晰的 Go 服务边界协作。</p>
              <a href="./guide/architecture">
                阅读系统架构
                <ArrowRight aria-hidden="true" />
              </a>
            </div>
            <div className="architecture-map" aria-label="SAG Wiki 系统组件关系">
              <div className="map-layer map-client"><span>01 / CLIENT</span><strong>React Web</strong><small>知识管理 · RAG Chat · 系统配置</small></div>
              <div className="map-connector"><span /><span /><span /></div>
              <div className="map-layer map-api"><span>02 / ORCHESTRATION</span><strong>Go + Fiber API</strong><small>鉴权 · 业务服务 · 权限边界</small></div>
              <div className="map-connector"><span /><span /><span /></div>
              <div className="map-storage-row">
                <div><Database /><strong>PostgreSQL</strong><small>业务数据</small></div>
                <div><Boxes /><strong>MinIO</strong><small>原始文档</small></div>
                <div><Workflow /><strong>Redis</strong><small>异步任务</small></div>
                <div><Network /><strong>Qdrant</strong><small>向量索引</small></div>
              </div>
            </div>
          </div>
        </section>

        <section className="closing-section">
          <div className="site-shell closing-inner">
            <div>
              <span className="section-kicker">RUN IT YOURSELF</span>
              <h2><span className="heading-line">从一份文档开始，</span><span className="heading-line">搭起你的知识工作台。</span></h2>
            </div>
            <div className="closing-actions">
              <a className="primary-button" href="./guide/getting-started">开始部署<ArrowRight /></a>
              <a className="text-link" href={repoUrl}><Github />查看源码</a>
            </div>
          </div>
        </section>
      </main>
      <footer className="site-footer">
        <div className="site-shell footer-inner">
          <Brand />
          <p>Self-hosted RAG workspace for team knowledge.</p>
          <a href={repoUrl}>GitHub</a>
        </div>
      </footer>
    </div>
  );
}
