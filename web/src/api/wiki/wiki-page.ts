import { useMutation, useQuery, useQueryClient } from "@tanstack/vue-query";
import type { MaybeRefOrGetter } from "vue";
import { toValue } from "vue";
import { toast } from "vue-sonner";

import { request } from "@/utils/request";

// ---- 类型（对齐 WeKnora frontend/src/api/wiki/index.ts）----

export interface WikiPage {
  id: string;
  knowledge_base_id: string;
  slug: string;
  title: string;
  page_type: string;
  status: string;
  /** markdown 源文本 */
  content: string;
  summary?: string;
  version: number;
  /** 当前版本的编辑来源：pipeline=管线 / agent=智能体 / user=人工 / revert=回滚 */
  last_edit_source?: "pipeline" | "agent" | "user" | "revert" | string;
  updated_at: string;
  created_at: string;
}

export interface WikiPageUpdatePayload {
  title?: string;
  content?: string;
  /** 乐观锁：不匹配时后端 409 */
  version?: number;
}

/** 历史修订快照；content 仅查单条时返回（列表省略） */
export interface WikiPageRevision {
  id: string;
  slug: string;
  version: number;
  title: string;
  status: string;
  content?: string;
  updated_at?: string;
  created_at?: string;
}

export interface WikiRevisionsResponse {
  success: boolean;
  message?: string;
  data?: {
    /** 新版在前；当前版本没有修订行（当前内容就是页面本体） */
    revisions: WikiPageRevision[];
    total: number;
    current_version: number;
  };
}

interface WikiListResponse {
  success: boolean;
  data?: WikiPage[];
  total?: number;
  page?: number;
  page_size?: number;
  message?: string;
}

// slug 可能是 "a/b/c" 形式，逐段编码后拼回路径（与官方 encodeSlugPath 一致）
function encodeSlugPath(slug: string): string {
  return slug.split("/").map((segment) => encodeURIComponent(segment)).join("/");
}

export const wikiPageApi = {
  list: (kbId: string) =>
    request.get<WikiListResponse>(
      `/api/v1/knowledgebase/${encodeURIComponent(kbId)}/wiki/pages`,
    ),

  create: (kbId: string, data: { title: string; content?: string }) =>
    request.post<{ success: boolean; data?: WikiPage; message?: string }>(
      `/api/v1/knowledgebase/${encodeURIComponent(kbId)}/wiki/pages`,
      data,
    ),

  get: (kbId: string, slug: string) =>
    request.get<{ success: boolean; data?: WikiPage; message?: string }>(
      `/api/v1/knowledgebase/${encodeURIComponent(kbId)}/wiki/pages/${encodeSlugPath(slug)}`,
    ),

  update: (kbId: string, slug: string, data: WikiPageUpdatePayload) =>
    request.put<{ success: boolean; data?: WikiPage; message?: string }>(
      `/api/v1/knowledgebase/${encodeURIComponent(kbId)}/wiki/pages/${encodeSlugPath(slug)}`,
      data,
    ),

  delete: (kbId: string, slug: string) =>
    request.delete<{ success: boolean; message?: string }>(
      `/api/v1/knowledgebase/${encodeURIComponent(kbId)}/wiki/pages/${encodeSlugPath(slug)}`,
    ),

  /** 历史修订列表（新版在前，content 省略）；当前版本 = 页面本体，无修订行 */
  revisions: (kbId: string, slug: string, params: { limit?: number; offset?: number } = {}) => {
    const search = new URLSearchParams();
    if (params.limit) search.set("limit", String(params.limit));
    if (params.offset) search.set("offset", String(params.offset));
    const qs = search.toString();
    return request.get<WikiRevisionsResponse>(
      `/api/v1/knowledgebase/${encodeURIComponent(kbId)}/wiki/revisions/${encodeSlugPath(slug)}${qs ? `?${qs}` : ""}`,
    );
  },

  /** 单条修订全文 */
  getRevision: (kbId: string, slug: string, version: number) =>
    request.get<{ success: boolean; data?: WikiPageRevision; message?: string }>(
      `/api/v1/knowledgebase/${encodeURIComponent(kbId)}/wiki/revisions/${encodeSlugPath(slug)}?version=${version}`,
    ),

  /**
   * 回滚到指定修订：作为一次普通编辑应用（回滚前状态先存快照，version 前进），
   * 所以回滚不会丢「回滚前」的内容。
   */
  revert: (kbId: string, slug: string, version: number) =>
    request.post<{ success: boolean; data?: WikiPage; message?: string }>(
      `/api/v1/knowledgebase/${encodeURIComponent(kbId)}/wiki/revert`,
      { slug, version },
    ),
};

// ---- Query keys ----

export const wikiPageKeys = {
  all: ["wiki-pages"] as const,
  lists: () => [...wikiPageKeys.all, "list"] as const,
  list: (kbId: string) => [...wikiPageKeys.lists(), kbId] as const,
};

// ---- Hooks ----

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: string } | null)?.message;
  return message || fallback;
}

export function useWikiPages(kbId: MaybeRefOrGetter<string | undefined>) {
  return useQuery({
    queryKey: ["wiki-pages", "list", () => toValue(kbId) ?? ""],
    queryFn: async () => {
      const res = await wikiPageApi.list(toValue(kbId)!);
      return res.data ?? [];
    },
    enabled: () => Boolean(toValue(kbId)),
  });
}

export function useCreateWikiPage(kbId: MaybeRefOrGetter<string>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { title: string; content?: string }) =>
      wikiPageApi.create(toValue(kbId), data),
    onSuccess: () => {
      toast.success("页面已创建");
      queryClient.invalidateQueries({ queryKey: wikiPageKeys.list(toValue(kbId)) });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "创建页面失败"));
    },
  });
}

export function useDeleteWikiPage(kbId: MaybeRefOrGetter<string>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (slug: string) => wikiPageApi.delete(toValue(kbId), slug),
    onSuccess: () => {
      toast.success("页面已删除");
      queryClient.invalidateQueries({ queryKey: wikiPageKeys.list(toValue(kbId)) });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "删除页面失败"));
    },
  });
}

export function useWikiRevisions(
  kbId: MaybeRefOrGetter<string>,
  slug: MaybeRefOrGetter<string | undefined>,
  enabled: MaybeRefOrGetter<boolean>,
) {
  return useQuery({
    queryKey: [...wikiPageKeys.list(toValue(kbId)), "revisions", () => toValue(slug) ?? ""],
    queryFn: async () => {
      const res = await wikiPageApi.revisions(toValue(kbId)!, toValue(slug)!, { limit: 50 });
      return res.data ?? { revisions: [], total: 0, current_version: 0 };
    },
    enabled: () => Boolean(toValue(slug)) && toValue(enabled),
  });
}

export function useRevertWikiPage(kbId: MaybeRefOrGetter<string>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ slug, version }: { slug: string; version: number }) =>
      wikiPageApi.revert(toValue(kbId), slug, version),
    onSuccess: (_res, { version }) => {
      toast.success(`已回滚到版本 ${version}（回滚前的内容已另存为历史版本）`);
      queryClient.invalidateQueries({ queryKey: wikiPageKeys.list(toValue(kbId)) });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "回滚失败"));
    },
  });
}
