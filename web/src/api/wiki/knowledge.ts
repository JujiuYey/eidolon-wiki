import { useMutation, useQuery, useQueryClient } from "@tanstack/vue-query";
import type { MaybeRefOrGetter } from "vue";
import { toValue, watch } from "vue";
import { toast } from "vue-sonner";

import { getDown, post, postUpload, put, request } from "@/utils/request";

// ---- 类型（对齐 WeKnora internal/types/knowledge.go、chunk.go、folder 接口）----

/** 解析状态完整枚举（internal/types/knowledge.go:42-70） */
export type KnowledgeParseStatus =
  | "pending"
  | "processing"
  | "finalizing"
  | "completed"
  | "failed"
  | "deleting"
  | "cancelled"
  | "draft";

export type KnowledgeSummaryStatus = "none" | "pending" | "processing" | "completed" | "failed";

export interface Knowledge {
  id: string;
  knowledge_base_id: string;
  title?: string;
  file_name?: string;
  folder_path?: string;
  file_type?: string;
  file_size?: number;
  type?: string;
  source?: string;
  channel?: string;
  parse_status: KnowledgeParseStatus;
  summary_status?: KnowledgeSummaryStatus;
  description?: string;
  error_message?: string;
  tags?: unknown[];
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  processed_at?: string;
  knowledge_base_name?: string;
}

export interface ListKnowledgeParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  parse_status?: string;
  file_type?: string;
  /** 逗号分隔的标签 id（OR 语义，后端 parseCommaSeparatedTagIDs） */
  tag_ids?: string;
  /** 空串 = 仅根目录；缺席 = 不过滤 */
  folder_path?: string;
  folder_recursive?: boolean;
}

export interface ListKnowledgeResponse {
  success: boolean;
  data?: Knowledge[];
  total?: number;
  page?: number;
  page_size?: number;
  message?: string;
}

export interface KnowledgeChunk {
  id: string;
  seq_id?: number;
  content: string;
  chunk_index?: number;
  chunk_type?: string;
  is_enabled?: boolean;
  /** 乐观锁：更新时作 expected_revision 传回 */
  content_revision?: number;
  index_status?: string;
  start_at?: number;
  end_at?: number;
  created_at?: string;
  updated_at?: string;
}

/** chunk 历史修订快照（GET /chunks/:kid/:cid/revisions） */
export interface ChunkRevision {
  id: string;
  chunk_id: string;
  revision: number;
  content: string;
  is_enabled?: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface UpdateChunkRequest {
  content?: string;
  is_enabled?: boolean;
  expected_revision?: number;
}

// ---- 解析过程 span 树（GET /knowledge/:id/spans，internal/types/knowledge_span.go）----

export interface KnowledgeProcessingSpan {
  span_id: string;
  parent_span_id?: string;
  name: string;
  /** root / stage / sub */
  kind: string;
  /** running / done / failed / pending */
  status: string;
  input?: Record<string, unknown>;
  output?: Record<string, unknown>;
  error_code?: string;
  error_message?: string;
  started_at?: string;
  finished_at?: string;
}

export interface SpanTreeNode extends KnowledgeProcessingSpan {
  children?: SpanTreeNode[];
}

export interface KnowledgeSpansResponse {
  knowledge_id: string;
  attempt: number;
  latest_attempt: number;
  parse_status: string;
  current_stage?: string;
  trace: SpanTreeNode | null;
  last_error?: {
    stage: string;
    code: string;
    message: string;
  };
}

export interface KnowledgeFolderNode {
  path: string;
  name: string;
  document_count: number;
  total_count: number;
  children?: KnowledgeFolderNode[];
}

export interface KnowledgeFolderTree {
  root_document_count: number;
  total_document_count: number;
  folders: KnowledgeFolderNode[];
}

export function isKnowledgeParseInFlight(status?: string): boolean {
  return status === "pending" || status === "processing" || status === "finalizing";
}

/** 官方规则：解析进行中，或解析完成但摘要仍在生成时需要轮询 */
export function knowledgeNeedsStatusPolling(
  item: Pick<Knowledge, "parse_status" | "summary_status">,
): boolean {
  if (isKnowledgeParseInFlight(item.parse_status)) return true;
  return (
    item.parse_status === "completed" &&
    (item.summary_status === "pending" || item.summary_status === "processing")
  );
}

// ---- API 函数 ----

function buildQuery(params: Record<string, string | number | boolean | undefined>): string {
  const u = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === "") continue;
    u.set(key, String(value));
  }
  const qs = u.toString();
  return qs ? `?${qs}` : "";
}

export const knowledgeApi = {
  list: (kbId: string, params: ListKnowledgeParams = {}) => {
    const search = new URLSearchParams();
    if (params.page) search.set("page", String(params.page));
    if (params.page_size) search.set("page_size", String(params.page_size));
    if (params.keyword) search.set("keyword", params.keyword);
    if (params.parse_status) search.set("parse_status", params.parse_status);
    if (params.file_type) search.set("file_type", params.file_type);
    if (params.tag_ids) search.set("tag_ids", params.tag_ids);
    if (params.folder_path !== undefined) search.set("folder_path", params.folder_path);
    if (params.folder_recursive) search.set("folder_recursive", "true");
    const qs = search.toString();
    return request.get<ListKnowledgeResponse>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/knowledge${qs ? `?${qs}` : ""}`,
    );
  },

  detail: (id: string) =>
    request.get<{ success: boolean; data?: Knowledge; message?: string }>(
      `/api/v1/knowledge/${encodeURIComponent(id)}`,
    ),

  chunks: (knowledgeId: string, page = 1, pageSize = 25) =>
    request.get<{
      success: boolean;
      data?: KnowledgeChunk[];
      total?: number;
      page?: number;
      page_size?: number;
      message?: string;
    }>(`/api/v1/chunks/${encodeURIComponent(knowledgeId)}${buildQuery({ page, page_size: pageSize })}`),

  /** 按 chunk id 取单条全文（引用悬停/点击查看用） */
  chunkById: (chunkId: string) =>
    request.get<{ success: boolean; data?: KnowledgeChunk; message?: string }>(
      `/api/v1/chunks/by-id/${encodeURIComponent(chunkId)}`,
    ),

  /**
   * 更新分块（内容/启用态），乐观锁：expected_revision 不匹配返回 409。
   * 只传要改的字段（后端指针语义，未传字段不动）。
   */
  updateChunk: (knowledgeId: string, chunkId: string, data: UpdateChunkRequest) =>
    request.put<{ success: boolean; data?: KnowledgeChunk; message?: string }>(
      `/api/v1/chunks/${encodeURIComponent(knowledgeId)}/${encodeURIComponent(chunkId)}`,
      data,
    ),

  /** 分块修订历史（当前内容在 chunk 本体上，这里只返回被替换的旧版本） */
  chunkRevisions: (knowledgeId: string, chunkId: string) =>
    request.get<{ success: boolean; data?: ChunkRevision[]; message?: string }>(
      `/api/v1/chunks/${encodeURIComponent(knowledgeId)}/${encodeURIComponent(chunkId)}/revisions`,
    ),

  /** 回滚到指定修订（revision 为目标修订号；expected_revision 同样校验并发） */
  revertChunk: (knowledgeId: string, chunkId: string, revision: number, expectedRevision?: number) =>
    request.post<{ success: boolean; data?: KnowledgeChunk; message?: string }>(
      `/api/v1/chunks/${encodeURIComponent(knowledgeId)}/${encodeURIComponent(chunkId)}/revert`,
      { revision, expected_revision: expectedRevision },
    ),

  /** 上传即自动进入解析流水线；fileName 含文件夹路径前缀时后端会拆出 folder_path */
  upload: (kbId: string, file: File, options: { fileName?: string } = {}) => {
    const form = new FormData();
    form.append("file", file);
    if (options.fileName) form.append("fileName", options.fileName);
    return postUpload<{ success: boolean; data?: Knowledge; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/knowledge/file`,
      form,
    );
  },

  /** 批量查询解析状态（轮询用）：GET /knowledge/batch?ids=a&ids=b */
  batchQuery: (ids: string[]) =>
    request.get<{ success: boolean; data?: Knowledge[]; message?: string }>(
      `/api/v1/knowledge/batch?${ids.map((id) => `ids=${encodeURIComponent(id)}`).join("&")}`,
    ),

  reparse: (id: string) =>
    post<{ success: boolean; message?: string }>(`/api/v1/knowledge/${encodeURIComponent(id)}/reparse`),

  cancelParse: (id: string) =>
    post<{ success: boolean; message?: string }>(`/api/v1/knowledge/${encodeURIComponent(id)}/cancel-parse`),

  delete: (id: string) =>
    request.delete<{ success: boolean; message?: string }>(`/api/v1/knowledge/${encodeURIComponent(id)}`),

  download: (id: string) => getDown(`/api/v1/knowledge/${encodeURIComponent(id)}/download`),

  preview: (id: string) => getDown(`/api/v1/knowledge/${encodeURIComponent(id)}/preview`),

  folderTree: (kbId: string) =>
    request.get<{ success: boolean; data?: KnowledgeFolderTree; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/knowledge/folders`,
    ),

  /** 移动文档到文件夹（路径不存在会隐式创建；不触发重解析） */
  moveToFolder: (kbId: string, knowledgeIds: string[], folderPath: string) =>
    post<{ success: boolean; message?: string }>(`/api/v1/knowledge/folder`, {
      kb_id: kbId,
      knowledge_ids: knowledgeIds,
      folder_path: folderPath,
    }),

  /** 重命名文件夹 */
  renameFolder: (kbId: string, from: string, to: string) =>
    put<{ success: boolean; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/knowledge/folders`,
      { from, to },
    ),

  /** 批量删除（POST /knowledge/batch-delete，上限 200/批） */
  batchDelete: (kbId: string, ids: string[]) =>
    post<{ success: boolean; message?: string }>(`/api/v1/knowledge/batch-delete`, {
      kb_id: kbId,
      ids,
    }),

  /** 批量重新解析（POST /knowledge/batch-reparse） */
  batchReparse: (kbId: string, ids: string[]) =>
    post<{ success: boolean; message?: string }>(`/api/v1/knowledge/batch-reparse`, {
      kb_id: kbId,
      ids,
    }),

  /**
   * 批量设置标签（PUT /knowledge/tags）。updates 是「每文档的最终标签集合」：
   * { [knowledge_id]: [tag_id...] }——覆盖语义，不是增量。
   */
  batchUpdateTags: (kbId: string, updates: Record<string, string[]>) =>
    put<{ success: boolean; message?: string }>(`/api/v1/knowledge/tags`, {
      updates,
      kb_id: kbId,
    }),

  /** 解析流水线 trace 树（固定 5 阶段，缺失补 pending；?attempt=N 看历史尝试） */
  spans: (id: string) =>
    request.get<{ success: boolean; data?: KnowledgeSpansResponse; message?: string }>(
      `/api/v1/knowledge/${encodeURIComponent(id)}/spans`,
    ),

  /** 重新生成摘要（摘要文本在 knowledge.description，summary_status 参与轮询） */
  regenerateSummary: (id: string) =>
    post<{ success: boolean; message?: string }>(
      `/api/v1/knowledge/${encodeURIComponent(id)}/regenerate-summary`,
    ),

  /**
   * 拉取全部分块（合并全文视图用）：循环翻页直到取完。
   * 上限 5000 块保护，避免异常大文档拖死前端。
   */
  allChunks: async (knowledgeId: string): Promise<KnowledgeChunk[]> => {
    const all: KnowledgeChunk[] = [];
    const pageSize = 200;
    let page = 1;
    let total = Number.POSITIVE_INFINITY;
    while (all.length < total && all.length < 5000) {
      const res = await knowledgeApi.chunks(knowledgeId, page, pageSize);
      total = res.total ?? 0;
      const items = res.data ?? [];
      all.push(...items);
      if (items.length === 0) break;
      page += 1;
    }
    return all;
  },
};

// ---- Query keys ----

export const knowledgeKeys = {
  all: ["knowledge"] as const,
  lists: () => [...knowledgeKeys.all, "list"] as const,
  list: (kbId: string, params: ListKnowledgeParams) => [...knowledgeKeys.lists(), kbId, params] as const,
  folders: (kbId: string) => [...knowledgeKeys.all, "folders", kbId] as const,
  chunks: (knowledgeId: string) => [...knowledgeKeys.all, "chunks", knowledgeId] as const,
};

// ---- Hooks ----

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: string } | null)?.message;
  return message || fallback;
}

export function useKnowledgeList(kbId: MaybeRefOrGetter<string | undefined>, params: () => ListKnowledgeParams) {
  return useQuery({
    queryKey: ["knowledge", "list", () => toValue(kbId) ?? "", () => params()],
    queryFn: async () => {
      const res = await knowledgeApi.list(toValue(kbId)!, params());
      return {
        items: res.data ?? [],
        total: res.total ?? 0,
      };
    },
    enabled: () => Boolean(toValue(kbId)),
    placeholderData: (prev) => prev,
  });
}

export function useKnowledgeFolders(kbId: MaybeRefOrGetter<string | undefined>) {
  return useQuery({
    queryKey: knowledgeKeys.folders(toValue(kbId) ?? ""),
    queryFn: async () => {
      const res = await knowledgeApi.folderTree(toValue(kbId)!);
      return res.data ?? { root_document_count: 0, total_document_count: 0, folders: [] };
    },
    enabled: () => Boolean(toValue(kbId)),
  });
}

export function useKnowledgeChunks(knowledgeId: MaybeRefOrGetter<string | undefined>, page: MaybeRefOrGetter<number>, pageSize = 25) {
  return useQuery({
    queryKey: [...knowledgeKeys.chunks(toValue(knowledgeId) ?? ""), () => toValue(page), pageSize],
    queryFn: async () => {
      const res = await knowledgeApi.chunks(toValue(knowledgeId)!, toValue(page), pageSize);
      return { items: res.data ?? [], total: res.total ?? 0 };
    },
    enabled: () => Boolean(toValue(knowledgeId)),
    placeholderData: (prev) => prev,
  });
}

// ---- Chunk 编辑（乐观锁）----

function chunkConflictToast(error: unknown, fallback: string) {
  const status = (error as { status?: number } | null)?.status;
  toast.error(status === 409 ? "分块已被他人修改，请刷新后重试" : errorMessage(error, fallback));
}

function invalidateChunks(queryClient: ReturnType<typeof useQueryClient>, knowledgeId: string) {
  queryClient.invalidateQueries({ queryKey: knowledgeKeys.chunks(knowledgeId) });
}

export function useUpdateChunk(knowledgeId: MaybeRefOrGetter<string | undefined>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ chunkId, data }: { chunkId: string; data: UpdateChunkRequest }) =>
      knowledgeApi.updateChunk(toValue(knowledgeId)!, chunkId, data),
    onSuccess: () => {
      toast.success("分块已保存");
      const id = toValue(knowledgeId);
      if (id) invalidateChunks(queryClient, id);
    },
    onError: (error) => chunkConflictToast(error, "保存分块失败"),
  });
}

export function useRevertChunk(knowledgeId: MaybeRefOrGetter<string | undefined>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      chunkId,
      revision,
      expectedRevision,
    }: {
      chunkId: string;
      revision: number;
      expectedRevision?: number;
    }) => knowledgeApi.revertChunk(toValue(knowledgeId)!, chunkId, revision, expectedRevision),
    onSuccess: (_res, { revision }) => {
      toast.success(`已回滚到修订 ${revision}`);
      const id = toValue(knowledgeId);
      if (id) invalidateChunks(queryClient, id);
    },
    onError: (error) => chunkConflictToast(error, "回滚失败"),
  });
}

export function useChunkRevisions(
  knowledgeId: MaybeRefOrGetter<string | undefined>,
  chunkId: MaybeRefOrGetter<string | undefined>,
  enabled: MaybeRefOrGetter<boolean>,
) {
  return useQuery({
    queryKey: [...knowledgeKeys.chunks(toValue(chunkId) ?? ""), "revisions"],
    queryFn: async () => {
      const res = await knowledgeApi.chunkRevisions(toValue(knowledgeId)!, toValue(chunkId)!);
      return res.data ?? [];
    },
    enabled: () => Boolean(toValue(knowledgeId) && toValue(chunkId)) && toValue(enabled),
  });
}

export function useReparseKnowledge(_kbId: string) {  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => knowledgeApi.reparse(id),
    onSuccess: () => {
      toast.success("已重新提交解析");
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.lists() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "重新解析失败"));
    },
  });
}

export function useCancelKnowledgeParse(_kbId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => knowledgeApi.cancelParse(id),
    onSuccess: () => {
      toast.success("已取消解析（已生成的分块会保留）");
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.lists() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "取消解析失败"));
    },
  });
}

export function useDeleteKnowledge(kbId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => knowledgeApi.delete(id),
    onSuccess: () => {
      toast.success("文档已删除");
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.lists() });
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.folders(kbId) });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "删除文档失败"));
    },
  });
}

export function useMoveKnowledgeToFolder(kbId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ knowledgeIds, folderPath }: { knowledgeIds: string[]; folderPath: string }) =>
      knowledgeApi.moveToFolder(kbId, knowledgeIds, folderPath),
    onSuccess: () => {
      toast.success("文档已移动");
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.lists() });
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.folders(kbId) });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "移动文档失败"));
    },
  });
}

export function useRenameKnowledgeFolder(kbId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ from, to }: { from: string; to: string }) => knowledgeApi.renameFolder(kbId, from, to),
    onSuccess: () => {
      toast.success("文件夹已更新");
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.folders(kbId) });
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.lists() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "重命名文件夹失败"));
    },
  });
}

export function useBatchDeleteKnowledge(kbId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => knowledgeApi.batchDelete(kbId, ids),
    onSuccess: () => {
      toast.success("选中文档已删除");
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.lists() });
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.folders(kbId) });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "批量删除失败"));
    },
  });
}

export function useBatchReparseKnowledge(kbId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => knowledgeApi.batchReparse(kbId, ids),
    onSuccess: () => {
      toast.success("已批量提交重新解析");
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.lists() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "批量重新解析失败"));
    },
  });
}

export function useBatchUpdateTags(kbId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (updates: Record<string, string[]>) => knowledgeApi.batchUpdateTags(kbId, updates),
    onSuccess: () => {
      toast.success("标签已更新");
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.lists() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "批量设置标签失败"));
    },
  });
}

export function useRegenerateSummary() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => knowledgeApi.regenerateSummary(id),
    onSuccess: () => {
      toast.success("已提交摘要重新生成");
      queryClient.invalidateQueries({ queryKey: knowledgeKeys.lists() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "提交摘要生成失败"));
    },
  });
}

/**
 * 解析状态轮询：1.5s 间隔只查 in-flight 的文档（与官方一致）。
 * 有状态变化时把结果合并回所有文档列表缓存，避免整页刷新闪烁。
 */
export function useKnowledgeStatusPolling(kbId: MaybeRefOrGetter<string | undefined>, items: () => Knowledge[] | undefined) {
  const queryClient = useQueryClient();

  watch(
    () => {
      const inFlightIds = (items() ?? []).filter(knowledgeNeedsStatusPolling).map((item) => item.id);
      return [toValue(kbId), inFlightIds.join(",")] as const;
    },
    ([currentKbId, idsKey], _prev, onCleanup) => {
      if (!currentKbId || !idsKey) return;
      let cancelled = false;
      let timer: number | undefined;

      const poll = async () => {
        try {
          const res = await knowledgeApi.batchQuery(idsKey.split(","));
          const updates = res.data ?? [];
          if (!cancelled && updates.length > 0) {
            const updateMap = new Map(updates.map((item) => [item.id, item]));
            queryClient.setQueriesData<{ items: Knowledge[]; total: number }>(
              { queryKey: knowledgeKeys.lists() },
              (old) => {
                if (!old) return old;
                let changed = false;
                const nextItems = old.items.map((item) => {
                  const update = updateMap.get(item.id);
                  if (!update) return item;
                  if (
                    update.parse_status !== item.parse_status ||
                    update.summary_status !== item.summary_status ||
                    update.error_message !== item.error_message
                  ) {
                    changed = true;
                  }
                  return { ...item, ...update };
                });
                return changed ? { ...old, items: nextItems } : old;
              },
            );
          }
        } catch {
          // 轮询失败静默，下一轮重试
        }
        if (!cancelled) {
          timer = window.setTimeout(() => void poll(), 1500);
        }
      };

      timer = window.setTimeout(() => void poll(), 1500);
      onCleanup(() => {
        cancelled = true;
        if (timer) window.clearTimeout(timer);
      });
    },
    { immediate: true },
  );
}

// ---- 展示辅助 ----

export const parseStatusLabels: Record<KnowledgeParseStatus, string> = {
  pending: "排队中",
  processing: "解析中",
  finalizing: "收尾中",
  completed: "已完成",
  failed: "失败",
  deleting: "删除中",
  cancelled: "已取消",
  draft: "草稿",
};

export function parseStatusBadgeClass(status: string): string {
  switch (status) {
    case "completed":
      return "bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300";
    case "failed":
      return "bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300";
    case "pending":
      return "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300";
    case "processing":
    case "finalizing":
      return "bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-300";
    default:
      return "bg-muted text-muted-foreground";
  }
}
