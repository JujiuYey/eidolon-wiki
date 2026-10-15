<script setup lang="ts">
import MarkdownIt from "markdown-it";
import { computed } from "vue";
import type { HTMLAttributes } from "vue";

import { cn } from "@/lib/utils";

import {
  extractCitationPlaceholders,
  joinCitationTagsToPreviousLine,
  restoreCitationHtml,
  stripIncompleteCitationTag,
  type CitationRef,
} from "./citations";

/**
 * 流式 Markdown 渲染（对标 React 生态的 Streamdown，Vue 近似实现）。
 *
 * - 基于 markdown-it（CommonMark + GFM 表格/删除线/linkify 自动链接）；
 * - 对未闭合语法天然容忍：未配对的 **、` 等按字面渲染，未闭合 ``` 围栏把余下内容
 *   当作代码块 —— 与 Streamdown 流式渲染行为一致；
 * - `streaming` 时在最后一个块级元素后叠加闪烁光标，近似 Streamdown isAnimating 的打字观感；
 * - 传入 `citations`（该消息的检索引用）时，正文里的 <kb/>/<web/> 引用标签渲染为
 *   行内药丸，点击药丸 emit citationClick（参考实现 WeKnora chatMarkdownRenderer）；
 * - 与 Streamdown 的差异（生态依赖不可用）：代码块无 shiki 高亮/复制按钮，数学公式与 mermaid
 *   不渲染为专用图（按原文展示），任务列表 `- [ ]` 按普通列表展示。
 */
const props = withDefaults(
  defineProps<{
    content: string;
    /** 流式进行中：渲染闪烁光标 */
    streaming?: boolean;
    /** 消息附带的检索引用，用于把 <kb/> 标签解析成可点击药丸 */
    citations?: CitationRef[];
    class?: HTMLAttributes["class"];
  }>(),
  {
    streaming: false,
  },
);

const emit = defineEmits<{
  citationClick: [chunkId: string];
}>();

const md = new MarkdownIt({
  // 模型输出不信任原始 HTML（Streamdown 侧等价于 rehype 消毒后的安全行为）
  html: false,
  linkify: true,
  breaks: false,
});

// 链接一律新窗口打开（对齐 Streamdown）
const defaultLinkOpen = md.renderer.rules.link_open;
md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  tokens[idx].attrSet("target", "_blank");
  tokens[idx].attrSet("rel", "noopener noreferrer nofollow");
  return defaultLinkOpen
    ? defaultLinkOpen(tokens, idx, options, env, self)
    : self.renderToken(tokens, idx, options);
};

const html = computed(() => {
  let source = props.content ?? "";
  if (props.streaming) source = stripIncompleteCitationTag(source);
  if (props.citations?.length || /<(?:kb|web)\b/.test(source)) {
    source = joinCitationTagsToPreviousLine(source);
    const { content, snippets } = extractCitationPlaceholders(source, props.citations);
    return restoreCitationHtml(md.render(content), snippets);
  }
  return md.render(source);
});

const activateCitation = (event: MouseEvent | KeyboardEvent) => {
  const target = event.target as HTMLElement | null;
  const pill = target?.closest?.("[data-cite-chunk-id]");
  if (pill) emit("citationClick", pill.getAttribute("data-cite-chunk-id") ?? "");
};
</script>

<template>
  <div
    :class="
      cn(
        'stream-markdown size-full [&>*:first-child]:mt-0 [&>*:last-child]:mb-0',
        props.class,
        props.streaming && 'sd-animating',
      )
    "
    v-html="html"
    @click="activateCitation"
    @keydown.enter="activateCitation"
  />
</template>

<style scoped>
/* 引用药丸（v-html 注入，需 :deep）：点击 kb 药丸由父组件打开分块全文 */
.stream-markdown :deep(.cite-pill) {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  margin: 0 2px;
  max-width: 12rem;
  padding: 1px 7px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--muted);
  color: var(--foreground);
  font-size: 0.75rem;
  line-height: 1.5;
  vertical-align: baseline;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.stream-markdown :deep(.cite-pill svg) {
  flex-shrink: 0;
  color: var(--muted-foreground);
}
.stream-markdown :deep(.cite-pill--kb) {
  cursor: pointer;
}
.stream-markdown :deep(.cite-pill--kb:hover),
.stream-markdown :deep(.cite-pill--kb:focus-visible) {
  border-color: var(--primary);
  color: var(--primary);
}
.stream-markdown :deep(.cite-pill--web) {
  text-decoration: none;
  color: var(--primary);
}
.stream-markdown :deep(.cite-pill--web:hover) {
  text-decoration: underline;
}

/* 近似 Streamdown 默认排版（项目未引入 @tailwindcss/typography，走主题变量） */
.stream-markdown :deep(p) {
  margin: 0.75rem 0;
  line-height: 1.7;
}
.stream-markdown :deep(h1) {
  margin: 1.5rem 0 0.75rem;
  font-size: 1.5rem;
  font-weight: 600;
  line-height: 1.3;
}
.stream-markdown :deep(h2) {
  margin: 1.25rem 0 0.75rem;
  font-size: 1.25rem;
  font-weight: 600;
  line-height: 1.3;
}
.stream-markdown :deep(h3) {
  margin: 1rem 0 0.5rem;
  font-size: 1.125rem;
  font-weight: 600;
}
.stream-markdown :deep(h4),
.stream-markdown :deep(h5),
.stream-markdown :deep(h6) {
  margin: 1rem 0 0.5rem;
  font-size: 1rem;
  font-weight: 600;
}
.stream-markdown :deep(ul) {
  margin: 0.75rem 0;
  padding-left: 1.5rem;
  list-style: disc;
}
.stream-markdown :deep(ol) {
  margin: 0.75rem 0;
  padding-left: 1.5rem;
  list-style: decimal;
}
.stream-markdown :deep(li) {
  margin: 0.25rem 0;
}
.stream-markdown :deep(blockquote) {
  margin: 0.75rem 0;
  border-left: 4px solid var(--border);
  padding-left: 1rem;
  color: var(--muted-foreground);
}
.stream-markdown :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.875em;
  background: var(--muted);
  border-radius: 0.25rem;
  padding: 0.125rem 0.375rem;
}
.stream-markdown :deep(pre) {
  margin: 0.75rem 0;
  overflow-x: auto;
  border-radius: 0.5rem;
  background: var(--muted);
  padding: 1rem;
}
.stream-markdown :deep(pre code) {
  background: transparent;
  padding: 0;
  font-size: 0.875rem;
}
.stream-markdown :deep(a) {
  color: var(--primary);
  text-decoration: underline;
  text-underline-offset: 2px;
}
.stream-markdown :deep(hr) {
  margin: 1.5rem 0;
  border-color: var(--border);
}
.stream-markdown :deep(table) {
  margin: 0.75rem 0;
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}
.stream-markdown :deep(th) {
  border: 1px solid var(--border);
  padding: 0.375rem 0.75rem;
  font-weight: 600;
  text-align: left;
}
.stream-markdown :deep(td) {
  border: 1px solid var(--border);
  padding: 0.375rem 0.75rem;
}
.stream-markdown :deep(img) {
  max-width: 100%;
  border-radius: 0.5rem;
}

/* 流式打字光标（isAnimating 近似） */
.stream-markdown.sd-animating :deep(> :last-child::after) {
  content: "";
  display: inline-block;
  margin-left: 2px;
  width: 0.5rem;
  height: 1em;
  vertical-align: text-bottom;
  background: var(--foreground);
  opacity: 0.75;
  animation: sd-caret-blink 1s step-end infinite;
}
@keyframes sd-caret-blink {
  50% {
    opacity: 0;
  }
}
</style>
