/**
 * 聊天引用标签预处理（参考实现：WeKnora frontend/src/utils/citationMarkdown.ts）。
 *
 * 模型会在回答正文里输出 <kb doc="…" chunk_id="…" kb_id="…"/> / <web url="…" title="…"/>
 * 引用标签。这里把它们替换为占位符，经 markdown-it 渲染后再还原为行内引用药丸，
 * 避免标签被 html:false 转义成纯文本或被 markdown 语法吞掉。
 */

export interface CitationRef {
  id?: string;
  knowledge_id?: string;
  knowledge_title?: string;
  knowledge_filename?: string;
  chunk_index?: number;
  chunk_type?: string;
  knowledge_base_id?: string;
}

const KB_WEB_TAG_RE = /<(?:kb|web)\b[^>]*?\s*\/?>/g;
const KB_TAG_ATTR_RE = /<kb\b([^>]*?)\s*\/?>/g;
const WEB_TAG_ATTR_RE = /<web\b([^>]*?)\s*\/?>/g;
const ATTRIBUTE_RE = /([\w-]+)\s*=\s*"([^"]*)"/g;
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
/** 私有区字符做占位符，markdown-it 不会赋予特殊含义 */
const PLACEHOLDER_RE = /\uE000(\d+)\uE001/g;

/** 流式期间标签只输出了一半时藏起残缺尾巴，避免 `<` 以普通文本闪现（参考实现同名函数） */
export function stripIncompleteCitationTag(content: string): string {
  if (!content) return content;
  const start = content.lastIndexOf("<");
  if (start < 0) return content;
  const tail = content.slice(start);
  if (tail.includes(">")) return content;
  const isCitationPrefix =
    tail === "<" ||
    /^<k(?:b(?:\s[\s\S]*)?)?$/i.test(tail) ||
    /^<w(?:e(?:b(?:\s[\s\S]*)?)?)?$/i.test(tail);
  return isCitationPrefix ? content.slice(0, start) : content;
}

function parseAttrs(attrString: string): Record<string, string> {
  const attributes: Record<string, string> = {};
  if (!attrString) return attributes;
  ATTRIBUTE_RE.lastIndex = 0;
  let match: RegExpExecArray | null;
  while ((match = ATTRIBUTE_RE.exec(attrString)) !== null) {
    attributes[match[1]] = match[2];
  }
  return attributes;
}

function escapeHtml(text: string): string {
  return String(text)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function truncateMiddle(text: string, maxLength = 13): string {
  if (!text) return "";
  if (text.length <= maxLength) return text;
  const half = Math.floor((maxLength - 3) / 2);
  return `${text.slice(0, half + ((maxLength - 3) % 2))}...${text.slice(-half)}`;
}

function normalizeTitle(title: string): string {
  return title.trim().toLowerCase();
}

function docTitlesMatch(a: string, b: string): boolean {
  if (!a || !b) return false;
  const na = normalizeTitle(a);
  const nb = normalizeTitle(b);
  return na === nb || na.includes(nb) || nb.includes(na);
}

/**
 * 把模型上下文里的编号（1、FAQ-1、DOC-2）或文档名解析成真实分块 UUID
 * （参考实现 resolveCitationChunkId；未命中时原样返回）
 */
export function resolveCitationChunkId(
  rawChunkId: string,
  attrs: { doc?: string; kbId?: string },
  refs?: CitationRef[] | null,
): string {
  const raw = String(rawChunkId || "").trim();
  if (!raw || UUID_RE.test(raw)) return raw;
  const list = (refs ?? []).filter((r) => r && r.chunk_type !== "web_search");
  if (!list.length) return raw;
  const doc = (attrs.doc || "").trim();
  const kbId = (attrs.kbId || "").trim();

  if (doc) {
    const byDoc = list.find(
      (r) =>
        docTitlesMatch(doc, r.knowledge_title || "") ||
        docTitlesMatch(doc, r.knowledge_filename || ""),
    );
    if (byDoc?.id) return byDoc.id;
  }

  const faqMatch = raw.match(/^FAQ-(\d+)$/i);
  if (faqMatch) {
    const hit = list.filter((r) => r.chunk_type === "faq")[Number(faqMatch[1]) - 1];
    if (hit?.id) return hit.id;
  }

  const docMatch = raw.match(/^DOC-(\d+)$/i);
  if (docMatch) {
    const hit = list.filter((r) => r.chunk_type !== "faq")[Number(docMatch[1]) - 1];
    if (hit?.id) return hit.id;
  }

  const num = Number.parseInt(raw, 10);
  if (!Number.isNaN(num) && String(num) === raw) {
    const byPos = list[num - 1];
    if (byPos?.id) return byPos.id;
    const byChunkIndex = list.find((r) => r.chunk_index === num || r.chunk_index === num - 1);
    if (byChunkIndex?.id) return byChunkIndex.id;
  }

  if (kbId) {
    const scoped = list.filter((r) => r.knowledge_base_id === kbId);
    if (doc) {
      const byDoc = scoped.find(
        (r) =>
          docTitlesMatch(doc, r.knowledge_title || "") ||
          docTitlesMatch(doc, r.knowledge_filename || ""),
      );
      if (byDoc?.id) return byDoc.id;
    }
    if (scoped.length === 1 && scoped[0].id) return scoped[0].id;
  }

  return raw;
}

const KB_ICON_SVG =
  '<svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>';
const WEB_ICON_SVG =
  '<svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="10"/><path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"/><path d="M2 12h20"/></svg>';

/** <kb/>/<web/> → 引用药丸 HTML（参考实现 preprocessCitationTags；wiki [[..]] 链接本前端未实现，不处理） */
export function preprocessCitationTags(content: string, refs?: CitationRef[] | null): string {
  if (!content.trim()) return "";
  return content
    .replace(WEB_TAG_ATTR_RE, (_match, attrString: string) => {
      const attrs = parseAttrs(attrString);
      const url = attrs.url || "";
      const title = attrs.title || "";
      if (!url) return "";
      let domain = url;
      try {
        const u = new URL(url);
        const parts = u.hostname.split(".");
        domain = parts.length >= 2 ? parts.slice(-2).join(".") : u.hostname || url;
      } catch {
        // 非法 URL 原样展示
      }
      return `<a class="cite-pill cite-pill--web" href="${escapeHtml(url)}" target="_blank" rel="noopener noreferrer nofollow" title="${escapeHtml(title)}">${WEB_ICON_SVG}${escapeHtml(domain)}</a>`;
    })
    .replace(KB_TAG_ATTR_RE, (_match, attrString: string) => {
      const attrs = parseAttrs(attrString);
      const doc = attrs.doc || "";
      const chunkId = resolveCitationChunkId(
        attrs.chunk_id || attrs.chunkId || "",
        { doc, kbId: attrs.kb_id || attrs.kbId || "" },
        refs,
      );
      if (!doc || !chunkId) return "";
      return `<span class="cite-pill cite-pill--kb" role="button" tabindex="0" data-cite-chunk-id="${escapeHtml(chunkId)}" title="${escapeHtml(doc)}">${KB_ICON_SVG}${escapeHtml(truncateMiddle(doc))}</span>`;
    });
}

/** 标签换成占位符交给 markdown-it，渲染完用 restoreCitationHtml 还原成药丸 */
export function extractCitationPlaceholders(
  content: string,
  refs?: CitationRef[] | null,
): { content: string; snippets: string[] } {
  const snippets: string[] = [];
  const result = content.replace(KB_WEB_TAG_RE, (match) => {
    snippets.push(preprocessCitationTags(match, refs));
    return `\uE000${snippets.length - 1}\uE001`;
  });
  return { content: result, snippets };
}

export function restoreCitationHtml(html: string, snippets: string[]): string {
  if (!snippets.length) return html;
  return html.replace(PLACEHOLDER_RE, (_match, idx: string) => snippets[Number(idx)] ?? "");
}

const FENCED_CODE_DELIMITER_RE = /^ {0,3}(`{3,}|~{3,})(\s*\S.*)?\s*$/;

/**
 * 独占一行的引用标签会被 markdown 拆成独立段落，把标签拼回上一行保持行内
 * （参考实现 joinCitationTagsToPreviousLine，围栏代码块行除外）
 */
export function joinCitationTagsToPreviousLine(content: string): string {
  if (!content) return content;

  let result = content;
  let prev = "";
  while (result !== prev) {
    prev = result;
    result = result.replace(
      /(<(?:kb|web)\b[^>]*?\s*\/?>)\s*\n+\s*(<(?:kb|web)\b)/g,
      "$1 $2",
    );
  }

  result = result.replace(
    /\n[ \t]*\n+([ \t]*<(?:kb|web)\b)/g,
    (match: string, kbStart: string, offset: number, full: string) => {
      const before = full.slice(0, offset);
      const lastLine = before.split("\n").filter((line) => line.trim()).pop() || "";
      if (FENCED_CODE_DELIMITER_RE.test(lastLine)) {
        return `\n\n${kbStart}`;
      }
      return ` ${kbStart.trimStart()}`;
    },
  );

  result = result.replace(
    /(?<!\n)(<(?:kb|web)\b[^>]*?\s*\/?>|[ \t]*\S[^\n]*?)\n([ \t]*<(?:kb|web)\b)/g,
    (match: string, beforePart: string, kbStart: string, offset: number, full: string) => {
      const lineStart = full.lastIndexOf("\n", offset - 1) + 1;
      const fullPrevLine = full.slice(lineStart, offset + beforePart.length);
      if (FENCED_CODE_DELIMITER_RE.test(fullPrevLine)) {
        return match;
      }
      return `${beforePart} ${kbStart.trimStart()}`;
    },
  );

  return result;
}
