import { useMutation, useQuery, useQueryClient } from "@tanstack/vue-query";
import type { MaybeRefOrGetter } from "vue";
import { toValue } from "vue";
import { toast } from "vue-sonner";

import { request } from "@/utils/request";

// ---- 类型（对齐 WeKnora internal/types/faq.go）----
// FAQ 路由前缀：/api/v1/knowledge-bases/:id/faq/*；entry_id = int64 seq_id（起始 100000000）

export interface FAQEntry {
  id: number;
  chunk_id?: string;
  knowledge_base_id?: string;
  /** 标签 seq_id（0 = 未分类）；注意与 KB 标签的 UUID 是两套 ID */
  tag_id: number;
  tag_name?: string;
  is_enabled: boolean;
  /** 新建默认 true（存储在 chunk flags） */
  is_recommended: boolean;
  standard_question: string;
  similar_questions?: string[];
  /** 命中即被过滤的反例 */
  negative_questions?: string[];
  answers: string[];
  answer_strategy?: "all" | "random";
  /** 只读：冗余自 KB faq_config.index_mode */
  index_mode?: string;
  created_at?: string;
  updated_at?: string;
  /** 仅检索响应 */
  score?: number;
  /** 0 = 向量，1 = 关键词（int 枚举，官方前端当字符串比较是坏的） */
  match_type?: number;
  matched_question?: string;
}

export interface FAQEntryPayload {
  standard_question: string;
  similar_questions?: string[];
  negative_questions?: string[];
  answers: string[];
  answer_strategy?: "all" | "random";
  /** 0/缺省 = 清空标签 */
  tag_id?: number;
  is_enabled?: boolean;
  is_recommended?: boolean;
}

export interface ListFAQEntriesResponse {
  success: boolean;
  message?: string;
  /** 双层 data：PageResult 包在 data 里 */
  data?: {
    total: number;
    page: number;
    page_size: number;
    data: FAQEntry[];
  };
}

export interface ListFAQEntriesParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  /** 标签 seq_id（旧版单选） */
  tag_id?: number;
  search_field?: "standard_question" | "similar_questions" | "answers";
  sort_order?: "asc";
}

export interface FAQSearchRequest {
  query_text: string;
  /** 默认 0.7 */
  vector_threshold?: number;
  /** 服务层上限 50 */
  match_count?: number;
  only_recommended?: boolean;
}

// ---- API ----

export const faqApi = {
  listEntries: (kbId: string, params: ListFAQEntriesParams = {}) => {
    const search = new URLSearchParams();
    if (params.page) search.set("page", String(params.page));
    if (params.page_size) search.set("page_size", String(params.page_size));
    if (params.keyword) search.set("keyword", params.keyword);
    if (params.tag_id) search.set("tag_id", String(params.tag_id));
    if (params.search_field) search.set("search_field", params.search_field);
    if (params.sort_order) search.set("sort_order", params.sort_order);
    const qs = search.toString();
    return request.get<ListFAQEntriesResponse>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/faq/entries${qs ? `?${qs}` : ""}`,
    );
  },

  createEntry: (kbId: string, data: FAQEntryPayload) =>
    request.post<{ success: boolean; data?: FAQEntry; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/faq/entry`,
      data,
    ),

  /** 全量替换语义：standard/similar/negative/answers 整组覆盖 */
  updateEntry: (kbId: string, entryId: number, data: FAQEntryPayload) =>
    request.put<{ success: boolean; data?: FAQEntry; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/faq/entries/${entryId}`,
      data,
    ),

  /** 批量启停/推荐：{by_id: {entry_seq_id: {is_enabled?, is_recommended?}}} */
  batchUpdateFields: (
    kbId: string,
    byId: Record<string, { is_enabled?: boolean; is_recommended?: boolean }>,
  ) =>
    request.put<{ success: boolean; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/faq/entries/fields`,
      { by_id: byId },
    ),

  batchDelete: (kbId: string, ids: number[]) =>
    request.delete<{ success: boolean; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/faq/entries`,
      // del(url, data) 第二参就是请求体；后端绑定 {ids}（faqDeleteRequest）
      { ids },
    ),

  /** FAQ 检索：data 直接是 FAQEntry 数组（附 score/match_type/matched_question） */
  search: (kbId: string, data: FAQSearchRequest) =>
    request.post<{ success: boolean; data?: FAQEntry[]; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/faq/search`,
      data,
    ),
};

// ---- Query keys ----

export const faqKeys = {
  all: ["faq-entries"] as const,
  list: (kbId: string) => [...faqKeys.all, kbId] as const,
};

// ---- Hooks ----

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: string } | null)?.message;
  return message || fallback;
}

export function useFAQEntries(
  kbId: MaybeRefOrGetter<string | undefined>,
  params: () => ListFAQEntriesParams,
) {
  return useQuery({
    queryKey: [...faqKeys.list(toValue(kbId) ?? ""), () => params()],
    queryFn: async () => {
      const res = await faqApi.listEntries(toValue(kbId)!, params());
      return res.data ?? { total: 0, page: 1, page_size: 20, data: [] };
    },
    enabled: () => Boolean(toValue(kbId)),
    placeholderData: (prev) => prev,
  });
}

/** 单建/更新是同步的：请求返回即向量化完成（后端 5s 预算，超时回滚） */
export function useCreateFAQEntry(kbId: MaybeRefOrGetter<string | undefined>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: FAQEntryPayload) => faqApi.createEntry(toValue(kbId)!, data),
    onSuccess: () => {
      toast.success("FAQ 条目已创建");
      queryClient.invalidateQueries({ queryKey: faqKeys.all });
    },
    onError: (error) => toast.error(errorMessage(error, "创建 FAQ 条目失败")),
  });
}

export function useUpdateFAQEntry(kbId: MaybeRefOrGetter<string | undefined>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ entryId, data }: { entryId: number; data: FAQEntryPayload }) =>
      faqApi.updateEntry(toValue(kbId)!, entryId, data),
    onSuccess: () => {
      toast.success("FAQ 条目已保存");
      queryClient.invalidateQueries({ queryKey: faqKeys.all });
    },
    onError: (error) => toast.error(errorMessage(error, "保存 FAQ 条目失败")),
  });
}

export function useBatchUpdateFAQFields(kbId: MaybeRefOrGetter<string | undefined>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (
      byId: Record<string, { is_enabled?: boolean; is_recommended?: boolean }>,
    ) => faqApi.batchUpdateFields(toValue(kbId)!, byId),
    onSuccess: () => {
      toast.success("已批量更新");
      queryClient.invalidateQueries({ queryKey: faqKeys.all });
    },
    onError: (error) => toast.error(errorMessage(error, "批量更新失败")),
  });
}

export function useBatchDeleteFAQEntries(kbId: MaybeRefOrGetter<string | undefined>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: number[]) => faqApi.batchDelete(toValue(kbId)!, ids),
    onSuccess: () => {
      toast.success("已删除选中条目");
      queryClient.invalidateQueries({ queryKey: faqKeys.all });
    },
    onError: (error) => toast.error(errorMessage(error, "批量删除失败")),
  });
}

// ---- 展示辅助 ----

export const faqMatchTypeLabels: Record<number, string> = {
  0: "向量",
  1: "关键词",
};

export function faqMatchTypeBadgeClass(matchType: number | undefined): string {
  if (matchType === 1) return "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300";
  return "bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-300";
}
