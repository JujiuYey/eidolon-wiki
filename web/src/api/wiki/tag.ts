import { useMutation, useQuery, useQueryClient } from "@tanstack/vue-query";
import type { MaybeRefOrGetter } from "vue";
import { toValue } from "vue";
import { toast } from "vue-sonner";

import { request } from "@/utils/request";

// ---- 类型（对齐 WeKnora internal/types/tag.go）----

export interface KnowledgeTag {
  id: string;
  seq_id?: number;
  knowledge_base_id?: string;
  /** KB 域内唯一 */
  name: string;
  /** 可选展示色（hex） */
  color?: string;
  sort_order?: number;
  created_at?: string;
  updated_at?: string;
}

export interface CreateTagRequest {
  name: string;
  color?: string;
  sort_order?: number;
}

export interface UpdateTagRequest {
  name?: string;
  color?: string;
  sort_order?: number;
}

// ---- API（端点见 routes_knowledge.go:271-281，全部 {success, data} 包装）----

export const tagApi = {
  list: (kbId: string) =>
    request.get<{ success: boolean; data?: KnowledgeTag[]; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/tags`,
    ),

  create: (kbId: string, data: CreateTagRequest) =>
    request.post<{ success: boolean; data?: KnowledgeTag; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/tags`,
      data,
    ),

  update: (kbId: string, tagId: string, data: UpdateTagRequest) =>
    request.put<{ success: boolean; data?: KnowledgeTag; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/tags/${encodeURIComponent(tagId)}`,
      data,
    ),

  delete: (kbId: string, tagId: string) =>
    request.delete<{ success: boolean; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/tags/${encodeURIComponent(tagId)}`,
    ),
};

// ---- Query keys ----

export const tagKeys = {
  all: ["knowledge-tags"] as const,
  list: (kbId: string) => [...tagKeys.all, kbId] as const,
};

// ---- Hooks ----

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: string } | null)?.message;
  return message || fallback;
}

export function useKnowledgeTags(kbId: MaybeRefOrGetter<string | undefined>) {
  return useQuery({
    queryKey: tagKeys.list(toValue(kbId) ?? ""),
    queryFn: async () => (await tagApi.list(toValue(kbId)!)).data ?? [],
    enabled: () => Boolean(toValue(kbId)),
  });
}

export function useCreateTag(kbId: MaybeRefOrGetter<string | undefined>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateTagRequest) => tagApi.create(toValue(kbId)!, data),
    onSuccess: (res) => {
      toast.success(`标签「${res.data?.name ?? ""}」已创建`);
      queryClient.invalidateQueries({ queryKey: tagKeys.all });
    },
    onError: (error) => toast.error(errorMessage(error, "创建标签失败")),
  });
}

export function useUpdateTag(kbId: MaybeRefOrGetter<string | undefined>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ tagId, data }: { tagId: string; data: UpdateTagRequest }) =>
      tagApi.update(toValue(kbId)!, tagId, data),
    onSuccess: () => {
      toast.success("标签已更新");
      queryClient.invalidateQueries({ queryKey: tagKeys.all });
    },
    onError: (error) => toast.error(errorMessage(error, "更新标签失败")),
  });
}

export function useDeleteTag(kbId: MaybeRefOrGetter<string | undefined>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (tagId: string) => tagApi.delete(toValue(kbId)!, tagId),
    onSuccess: () => {
      toast.success("标签已删除");
      queryClient.invalidateQueries({ queryKey: tagKeys.all });
    },
    onError: (error) => toast.error(errorMessage(error, "删除标签失败")),
  });
}

// ---- 展示辅助 ----

/** color 存的是 hex 时直接用作 badge 背景，否则回退默认色 */
export function tagBadgeStyle(color: string | undefined): Record<string, string> | undefined {
  if (color && color.startsWith("#")) {
    return { backgroundColor: `${color}1a`, color };
  }
  return undefined;
}
