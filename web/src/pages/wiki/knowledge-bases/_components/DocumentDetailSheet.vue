<script setup lang="ts">
import { DownloadIcon, ExternalLinkIcon, HistoryIcon, Loader2Icon, RefreshCwIcon, SquarePenIcon } from "@lucide/vue";
import { useQuery } from "@tanstack/vue-query";
import { computed, ref, watch } from "vue";
import { toast } from "vue-sonner";

import {
  knowledgeApi,
  knowledgeKeys,
  useChunkRevisions,
  useKnowledgeChunks,
  useRegenerateSummary,
  useRevertChunk,
  useUpdateChunk,
  type Knowledge,
  type KnowledgeChunk,
  type SpanTreeNode,
} from "@/api/wiki/knowledge";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";

const CHUNK_PAGE_SIZE = 25;

function formatBytes(bytes: number | undefined): string {
  if (!bytes) return "-";
  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${value.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function formatDateTime(value: string | undefined): string {
  if (!value) return "-";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString("zh-CN", { hour12: false });
}

/** 按 blob 类型决定预览方式；复杂格式（docx/xlsx 等）提示下载 */
function previewKind(
  fileName: string | undefined,
  contentType: string,
): "image" | "pdf" | "text" | "markdown" | "unsupported" {
  const ext = (fileName ?? "").split(".").pop()?.toLowerCase() ?? "";
  if (contentType.startsWith("image/") || ["png", "jpg", "jpeg", "gif", "webp", "svg", "bmp"].includes(ext)) {
    return "image";
  }
  if (contentType.includes("pdf") || ext === "pdf") return "pdf";
  if (contentType.startsWith("text/") || ["txt", "log", "csv", "json", "xml", "html"].includes(ext)) return "text";
  if (["md", "markdown"].includes(ext) || contentType.includes("markdown")) return "markdown";
  return "unsupported";
}

/**
 * 合并全文：按 seq_id 顺序拼接，位置重叠时裁剪前 overlap 字符
 * （近似官方 appendChunkContent；间隔大时插入空行）。
 */
function mergeChunkContents(chunks: KnowledgeChunk[]): string {
  const sorted = [...chunks]
    .filter((c) => c.is_enabled !== false)
    .sort((a, b) => (a.seq_id ?? 0) - (b.seq_id ?? 0));
  if (sorted.length === 0) return "";
  let merged = sorted[0].content || "";
  let mergedEnd = sorted[0].end_at ?? 0;
  for (const chunk of sorted.slice(1)) {
    const content = chunk.content || "";
    const startAt = chunk.start_at ?? 0;
    if (startAt > mergedEnd && mergedEnd > 0) {
      merged += "\n\n" + content;
    } else if (mergedEnd > startAt) {
      merged += content.slice(Math.min(mergedEnd - startAt, content.length));
    } else {
      merged += content;
    }
    mergedEnd = Math.max(mergedEnd, chunk.end_at ?? mergedEnd);
  }
  return merged;
}

const props = defineProps<{
  open: boolean;
  knowledge: Knowledge | null;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
}>();

const chunkPage = ref(1);
const previewUrl = ref<string | null>(null);
const previewText = ref<string | null>(null);
const previewLoading = ref(false);

const chunksQuery = useKnowledgeChunks(
  () => (props.open ? props.knowledge?.id : undefined),
  chunkPage,
  CHUNK_PAGE_SIZE,
);

watch(
  () => props.open,
  (open) => {
    if (!open) {
      chunkPage.value = 1;
      previewUrl.value = null;
      previewText.value = null;
    }
  },
);

const kind = computed(() => previewKind(props.knowledge?.file_name ?? undefined, props.knowledge?.type ?? ""));

const loadPreview = async () => {
  if (!props.knowledge) return;
  previewLoading.value = true;
  try {
    const blob = await knowledgeApi.preview(props.knowledge.id);
    const url = URL.createObjectURL(blob);
    previewUrl.value = url;
    if (kind.value === "text" || kind.value === "markdown") {
      previewText.value = await blob.text();
    }
  } catch (error) {
    toast.error((error as { message?: string }).message || "预览加载失败");
  } finally {
    previewLoading.value = false;
  }
};

const handleDownload = async () => {
  if (!props.knowledge) return;
  try {
    const blob = await knowledgeApi.download(props.knowledge.id);
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = props.knowledge.file_name || `${props.knowledge.id}`;
    link.click();
    URL.revokeObjectURL(url);
  } catch (error) {
    toast.error((error as { message?: string }).message || "下载失败");
  }
};

const total = computed(() => chunksQuery.data.value?.total ?? 0);
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / CHUNK_PAGE_SIZE)));

// ---- 摘要（knowledge.description + summary_status）----

const regenerateSummary = useRegenerateSummary();

// ---- 合并全文（切到该页签时懒加载全部分块后拼接）----

const detailTab = ref("chunks");
const mergedText = ref<string | null>(null);
const mergedLoading = ref(false);

watch(detailTab, (tab) => {
  if (tab === "merged" && mergedText.value === null) void loadMerged();
});

const loadMerged = async () => {
  if (!props.knowledge || mergedLoading.value) return;
  mergedLoading.value = true;
  try {
    const all = await knowledgeApi.allChunks(props.knowledge.id);
    mergedText.value = mergeChunkContents(all);
  } catch (error) {
    toast.error((error as { message?: string }).message || "全文加载失败");
  } finally {
    mergedLoading.value = false;
  }
};

// ---- 解析过程时间线（解析中每 2s 刷新）----

const spansQuery = useQuery({
  queryKey: [...knowledgeKeys.all, "spans", () => props.knowledge?.id ?? ""],
  queryFn: async () => {
    const res = await knowledgeApi.spans(props.knowledge!.id);
    return res.data ?? null;
  },
  enabled: () => Boolean(props.open && props.knowledge),
  refetchInterval: (query) =>
    query.state.data && ["pending", "processing", "finalizing"].includes(query.state.data.parse_status)
      ? 2000
      : false,
});

const spanStatusBadge: Record<string, string> = {
  done: "bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300",
  failed: "bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300",
  running: "bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-300",
  pending: "bg-muted text-muted-foreground",
};

const spanStatusLabels: Record<string, string> = {
  done: "完成",
  failed: "失败",
  running: "进行中",
  pending: "等待",
};

const spanDuration = (node: SpanTreeNode): string => {
  if (!node.started_at || !node.finished_at) return "-";
  const ms = new Date(node.finished_at).getTime() - new Date(node.started_at).getTime();
  if (Number.isNaN(ms) || ms < 0) return "-";
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`;
};

// ---- Chunk 编辑 / 启用禁用 / 修订历史 ----

const knowledgeId = computed(() => (props.open ? (props.knowledge?.id ?? undefined) : undefined));
const updateChunk = useUpdateChunk(knowledgeId);
const revertChunk = useRevertChunk(knowledgeId);

const editingChunk = ref<KnowledgeChunk | null>(null);
const editContent = ref("");
const historyChunk = ref<KnowledgeChunk | null>(null);

const revisionsQuery = useChunkRevisions(
  knowledgeId,
  () => historyChunk.value?.id,
  () => Boolean(historyChunk.value),
);

watch(
  () => props.open,
  (open) => {
    if (!open) {
      editingChunk.value = null;
      historyChunk.value = null;
    }
  },
);

const openEdit = (chunk: KnowledgeChunk) => {
  editingChunk.value = chunk;
  editContent.value = chunk.content;
};

const handleEditSave = async () => {
  const chunk = editingChunk.value;
  if (!chunk) return;
  if (!editContent.value.trim()) {
    toast.error("分块内容不能为空");
    return;
  }
  await updateChunk.mutateAsync({
    chunkId: chunk.id,
    data: { content: editContent.value, expected_revision: chunk.content_revision },
  });
  mergedText.value = null; // 内容变了，全文缓存失效
  editingChunk.value = null;
};

const handleToggleEnabled = async (chunk: KnowledgeChunk, enabled: boolean) => {
  await updateChunk.mutateAsync({
    chunkId: chunk.id,
    data: { is_enabled: enabled, expected_revision: chunk.content_revision },
  });
  mergedText.value = null;
};

const handleRevert = async (revision: number) => {
  const chunk = historyChunk.value;
  if (!chunk) return;
  await revertChunk.mutateAsync({
    chunkId: chunk.id,
    revision,
    expectedRevision: chunk.content_revision,
  });
};

const revisionTime = (value: string | undefined): string => formatDateTime(value);
</script>

<template>
  <Sheet :open="open" @update:open="emit('update:open', $event)">
    <SheetContent class="flex flex-col gap-0 sm:max-w-3xl">
      <SheetHeader class="pb-2">
        <SheetTitle class="truncate pr-8">{{ knowledge?.file_name || knowledge?.title || "文档详情" }}</SheetTitle>
        <SheetDescription class="flex items-center gap-2">
          <template v-if="knowledge">
            <span>{{ formatBytes(knowledge.file_size) }}</span>
            <Badge v-if="knowledge.file_type" variant="secondary">{{ knowledge.file_type }}</Badge>
            <Badge v-if="knowledge.folder_path" variant="outline">{{ knowledge.folder_path }}</Badge>
          </template>
        </SheetDescription>
      </SheetHeader>

      <!-- 摘要区（有摘要或摘要生成中时显示） -->
      <div
        v-if="knowledge && (knowledge.description || knowledge.summary_status === 'processing' || knowledge.summary_status === 'pending')"
        class="mx-4 mb-2 rounded-lg border bg-muted/40 px-3 py-2.5"
      >
        <div class="mb-1 flex items-center justify-between">
          <span class="text-xs font-medium text-muted-foreground">
            AI 摘要
            <Badge
              v-if="knowledge.summary_status === 'processing' || knowledge.summary_status === 'pending'"
              variant="secondary"
              class="ml-1"
            >
              生成中…
            </Badge>
          </span>
          <Button
            variant="ghost"
            size="xs"
            :disabled="regenerateSummary.isPending.value"
            @click="knowledge && regenerateSummary.mutate(knowledge.id)"
          >
            <RefreshCwIcon />
            重新生成
          </Button>
        </div>
        <p class="whitespace-pre-wrap break-words text-sm leading-relaxed">
          {{ knowledge.description || "摘要生成中…" }}
        </p>
      </div>

      <Tabs v-model="detailTab" default-value="chunks" class="flex min-h-0 flex-1 flex-col gap-3 px-4 pb-4">
        <div class="flex items-center justify-between">
          <TabsList>
            <TabsTrigger value="chunks">分块（{{ total }}）</TabsTrigger>
            <TabsTrigger value="merged">全文</TabsTrigger>
            <TabsTrigger value="preview">原文件</TabsTrigger>
            <TabsTrigger value="trace">解析过程</TabsTrigger>
          </TabsList>
          <Button variant="outline" size="sm" @click="handleDownload()">
            <DownloadIcon />
            下载
          </Button>
        </div>

        <TabsContent value="chunks" class="min-h-0 flex-1 overflow-y-auto">
          <div v-if="chunksQuery.isLoading.value" class="space-y-3">
            <Skeleton v-for="index in 4" :key="index" class="h-20 w-full" />
          </div>
          <div v-else-if="chunksQuery.data.value?.items.length" class="space-y-3">
            <div
              v-for="(chunk, index) in chunksQuery.data.value.items"
              :key="chunk.id"
              class="rounded-lg border p-3"
              :class="chunk.is_enabled === false && 'opacity-60'"
            >
              <div class="mb-1 flex items-center justify-between gap-2 text-xs text-muted-foreground">
                <span class="flex items-center gap-2">
                  #{{ (chunkPage - 1) * CHUNK_PAGE_SIZE + index + 1 }}
                  {{ chunk.chunk_type ? ` · ${chunk.chunk_type}` : "" }}
                  <Badge v-if="chunk.is_enabled === false" variant="outline">已禁用</Badge>
                </span>
                <span class="flex items-center gap-2">
                  <span>chunk {{ chunk.chunk_index ?? "-" }}</span>
                  <Switch
                    :checked="chunk.is_enabled !== false"
                    aria-label="启用/禁用分块"
                    @update:checked="(checked: boolean) => handleToggleEnabled(chunk, checked)"
                  />
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    aria-label="编辑分块"
                    @click="openEdit(chunk)"
                  >
                    <SquarePenIcon />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon-xs"
                    aria-label="修订历史"
                    @click="historyChunk = chunk"
                  >
                    <HistoryIcon />
                  </Button>
                </span>
              </div>
              <p class="whitespace-pre-wrap break-words text-sm leading-relaxed">{{ chunk.content }}</p>
            </div>
            <div v-if="totalPages > 1" class="flex items-center justify-center gap-3 py-2">
              <Button
                variant="outline"
                size="sm"
                :disabled="chunkPage <= 1 || chunksQuery.isFetching.value"
                @click="chunkPage--"
              >
                上一页
              </Button>
              <span class="text-sm text-muted-foreground">{{ chunkPage }} / {{ totalPages }}</span>
              <Button
                variant="outline"
                size="sm"
                :disabled="chunkPage >= totalPages || chunksQuery.isFetching.value"
                @click="chunkPage++"
              >
                下一页
              </Button>
            </div>
          </div>
          <p v-else class="py-10 text-center text-sm text-muted-foreground">该文档还没有分块数据</p>
        </TabsContent>

        <TabsContent value="merged" class="min-h-0 flex-1 overflow-y-auto">
          <div v-if="mergedLoading" class="space-y-3 py-4">
            <Skeleton v-for="index in 6" :key="index" class="h-8 w-full" />
          </div>
          <template v-else-if="mergedText !== null">
            <p class="mb-3 text-xs text-muted-foreground">
              按解析顺序合并的分块全文（已按位置去重；禁用的分块不包含在内）。
            </p>
            <p class="whitespace-pre-wrap break-words rounded-lg border bg-muted/30 p-4 text-sm leading-relaxed">
              {{ mergedText || "（无内容）" }}
            </p>
          </template>
          <div v-else class="flex h-full flex-col items-center justify-center gap-3 py-10">
            <p class="text-sm text-muted-foreground">拉取全部分块并合并阅读。</p>
            <Button @click="loadMerged()">
              <Loader2Icon v-if="mergedLoading" class="animate-spin" />
              加载全文
            </Button>
          </div>
        </TabsContent>

        <TabsContent value="preview" class="min-h-0 flex-1 overflow-y-auto">
          <div v-if="!previewUrl && !previewLoading" class="flex h-full flex-col items-center justify-center gap-3 py-10">
            <p class="text-sm text-muted-foreground">
              {{ kind === "unsupported" ? "该格式暂不支持在线预览，可下载后查看。" : "加载原文件预览。" }}
            </p>
            <Button @click="loadPreview()">
              <ExternalLinkIcon />
              {{ kind === "unsupported" ? "下载文件" : "加载预览" }}
            </Button>
          </div>
          <Skeleton v-else-if="previewLoading" class="h-64 w-full" />
          <img
            v-else-if="previewUrl && kind === 'image'"
            :src="previewUrl"
            :alt="knowledge?.file_name ?? '预览'"
            class="mx-auto max-w-full"
          />
          <iframe
            v-else-if="previewUrl && kind === 'pdf'"
            :src="previewUrl"
            title="PDF 预览"
            class="h-full min-h-[480px] w-full rounded-md border"
          />
          <pre
            v-else-if="previewText"
            class="whitespace-pre-wrap break-words rounded-md bg-muted p-4 font-mono text-xs leading-relaxed"
            >{{ previewText }}</pre
          >
          <p v-else-if="previewUrl" class="py-10 text-center text-sm text-muted-foreground">
            该格式暂不支持在线预览，可下载后查看。
          </p>
        </TabsContent>

        <TabsContent value="trace" class="min-h-0 flex-1 overflow-y-auto">
          <div v-if="spansQuery.isLoading.value" class="space-y-3 py-2">
            <Skeleton v-for="index in 5" :key="index" class="h-12 w-full" />
          </div>
          <template v-else-if="spansQuery.data.value">
            <div class="mb-3 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
              <span>第 {{ spansQuery.data.value.attempt }} / {{ Math.max(1, spansQuery.data.value.latest_attempt) }} 次解析</span>
              <Button
                v-if="spansQuery.data.value.latest_attempt > spansQuery.data.value.attempt"
                variant="ghost"
                size="xs"
                @click="spansQuery.refetch()"
              >
                刷新
              </Button>
            </div>

            <div
              v-if="spansQuery.data.value.last_error"
              class="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2.5 dark:border-red-900 dark:bg-red-950/40"
            >
              <p class="text-xs font-medium text-red-700 dark:text-red-400">
                解析失败：{{ spansQuery.data.value.last_error.stage }}
                <span v-if="spansQuery.data.value.last_error.code">（{{ spansQuery.data.value.last_error.code }}）</span>
              </p>
              <p class="mt-1 break-words text-sm">{{ spansQuery.data.value.last_error.message }}</p>
            </div>

            <div v-if="spansQuery.data.value.trace" class="space-y-2">
              <div
                v-for="stage in spansQuery.data.value.trace.children ?? []"
                :key="stage.span_id"
                class="rounded-lg border p-3"
                :class="stage.status === 'failed' && 'border-red-200 dark:border-red-900'"
              >
                <div class="flex items-center justify-between gap-2">
                  <span class="flex items-center gap-2 text-sm font-medium">
                    {{ stage.name }}
                    <Badge variant="secondary" :class="spanStatusBadge[stage.status]">
                      {{ spanStatusLabels[stage.status] ?? stage.status }}
                    </Badge>
                  </span>
                  <span class="text-xs text-muted-foreground">{{ spanDuration(stage) }}</span>
                </div>
                <p v-if="stage.error_message" class="mt-1 break-words text-xs text-red-600 dark:text-red-400">
                  {{ stage.error_message }}
                </p>
                <div v-if="stage.children?.length" class="mt-2 space-y-1 border-l pl-3">
                  <div
                    v-for="child in stage.children"
                    :key="child.span_id"
                    class="flex items-center justify-between gap-2 text-xs text-muted-foreground"
                  >
                    <span class="flex items-center gap-2">
                      {{ child.name }}
                      <Badge variant="outline" :class="spanStatusBadge[child.status]">
                        {{ spanStatusLabels[child.status] ?? child.status }}
                      </Badge>
                    </span>
                    <span>{{ spanDuration(child) }}</span>
                  </div>
                </div>
              </div>
            </div>
            <p v-else class="py-10 text-center text-sm text-muted-foreground">暂无解析过程数据</p>
          </template>
          <p v-else class="py-10 text-center text-sm text-muted-foreground">暂无解析过程数据</p>
        </TabsContent>
      </Tabs>

      <!-- 编辑分块 -->
      <Dialog :open="Boolean(editingChunk)" @update:open="open => !open && (editingChunk = null)">
        <DialogContent class="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>编辑分块</DialogTitle>
            <DialogDescription>
              修改后重新建立索引；原内容会保存为历史修订，可随时回滚。
            </DialogDescription>
          </DialogHeader>
          <div class="space-y-2">
            <Label for="chunk-edit-content">分块内容</Label>
            <Textarea
              id="chunk-edit-content"
              v-model="editContent"
              :rows="10"
              class="font-mono text-xs"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" @click="editingChunk = null">取消</Button>
            <Button :disabled="updateChunk.isPending.value" @click="handleEditSave">
              {{ updateChunk.isPending.value ? "保存中..." : "保存" }}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <!-- 修订历史 -->
      <Dialog :open="Boolean(historyChunk)" @update:open="open => !open && (historyChunk = null)">
        <DialogContent class="max-h-[80vh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>修订历史</DialogTitle>
            <DialogDescription>
              当前为修订 {{ historyChunk?.content_revision ?? 0 }}。回滚会把历史内容恢复为当前内容。
            </DialogDescription>
          </DialogHeader>
          <div v-if="revisionsQuery.isLoading.value" class="space-y-2 py-2">
            <Skeleton v-for="i in 3" :key="i" class="h-16 w-full" />
          </div>
          <p v-else-if="!revisionsQuery.data.value?.length" class="py-6 text-center text-sm text-muted-foreground">
            该分块还没有历史修订（编辑一次后生成）。
          </p>
          <div v-else class="space-y-3">
            <div v-for="rev in revisionsQuery.data.value" :key="rev.id" class="rounded-lg border p-3">
              <div class="mb-1 flex items-center justify-between gap-2 text-xs text-muted-foreground">
                <span>修订 {{ rev.revision }} · {{ revisionTime(rev.created_at) }}</span>
                <Button
                  variant="outline"
                  size="xs"
                  :disabled="revertChunk.isPending.value"
                  @click="handleRevert(rev.revision)"
                >
                  回滚到此版本
                </Button>
              </div>
              <p class="line-clamp-3 whitespace-pre-wrap break-words text-xs leading-relaxed text-muted-foreground">
                {{ rev.content }}
              </p>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </SheetContent>
  </Sheet>
</template>
