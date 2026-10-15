import { useMutation, useQuery, useQueryClient } from "@tanstack/vue-query";
import type { MaybeRefOrGetter } from "vue";
import { toValue } from "vue";
import { toast } from "vue-sonner";

import { request } from "@/utils/request";

// ---- 类型（对齐 WeKnora internal/types/knowledgebase.go 与官方前端 KB interface）----

// 切分配置（ChunkingConfig，:244-282）：单位是字符而非 token（token_limit 除外）；
// 0 值在后端有回退默认（chunk_size→512、overlap→80、parent→4096、child→384）
export interface ChunkingConfig {
  chunk_size?: number;
  chunk_overlap?: number;
  separators?: string[];
  parser_engine_rules?: Array<{ file_types?: string[]; engine?: string; xlsx_first_row_as_header?: boolean }>;
  enable_parent_child?: boolean;
  parent_chunk_size?: number;
  child_chunk_size?: number;
  /** legacy/auto/heading/heuristic/recursive；空 = 旧递归切分 */
  strategy?: string;
  /** 0 = 关闭（按字符切） */
  token_limit?: number;
  languages?: string[];
  table_metadata_instructions?: string;
}

export interface IndexingStrategy {
  vector_enabled?: boolean;
  keyword_enabled?: boolean;
  wiki_enabled?: boolean;
  graph_enabled?: boolean;
}

export interface VLMConfig {
  enabled?: boolean;
  model_id?: string;
  description_language?: string;
  custom_instructions?: string;
}

export interface ASRConfig {
  enabled?: boolean;
  model_id?: string;
  language?: string;
}

export interface ExtractConfig {
  enabled?: boolean;
  text?: string;
  tags?: string[];
  nodes?: unknown[];
  relations?: unknown[];
  custom_instructions?: string;
}

export interface QuestionGenerationConfig {
  enabled?: boolean;
  question_count?: number;
  custom_instructions?: string;
}

export interface KnowledgeBase {
  id: string;
  name: string;
  description?: string;
  type?: "document" | "faq";
  knowledge_count?: number;
  chunk_count?: number;
  processing_count?: number;
  is_pinned?: boolean;
  pinned_at?: string;
  embedding_model_id?: string;
  summary_model_id?: string;
  chunking_config?: ChunkingConfig;
  indexing_strategy?: IndexingStrategy;
  /** 以下高级配置在编辑模型配置时需原值回传（PUT initialization/config 为覆盖式语义） */
  vlm_config?: VLMConfig;
  asr_config?: ASRConfig;
  storage_provider_config?: { provider?: string };
  extract_config?: ExtractConfig;
  question_generation_config?: QuestionGenerationConfig;
  creator_id?: string;
  creator_name?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CreateKnowledgeBaseRequest {
  name: string;
  description?: string;
  type?: "document" | "faq";
  embedding_model_id?: string;
  summary_model_id?: string;
  chunking_config?: ChunkingConfig;
  indexing_strategy?: IndexingStrategy;
}

export interface UpdateKnowledgeBaseRequest {
  name: string;
  description?: string;
  /** PUT 的窄 DTO 通道：模型不在其中，改模型走 updateModelConfig */
  config?: {
    chunking_config?: ChunkingConfig;
    indexing_strategy?: IndexingStrategy;
  };
}

export interface KBModelConfig {
  llmModelId: string;
  embeddingModelId?: string;
  /** 覆盖式语义：编辑时必须原值回传以下字段，否则 VLM/ASR 被清空、存储回落 local */
  vlm_config?: VLMConfig;
  asr_config?: ASRConfig;
  multimodal?: { enabled: boolean };
  storage_provider_config?: { provider?: string };
  extract_config?: ExtractConfig;
  question_generation_config?: QuestionGenerationConfig;
  documentSplitting: {
    chunkSize?: number;
    chunkOverlap?: number;
    separators?: string[];
    enableParentChild?: boolean;
    parentChunkSize?: number;
    childChunkSize?: number;
    strategy?: string;
  };
}

// ---- API 函数 ----

export const knowledgeBaseApi = {
  // 列表无分页参数，后端全量返回
  list: () => request.get<{ success: boolean; data?: KnowledgeBase[]; message?: string }>("/api/v1/knowledge-bases"),

  detail: (id: string) =>
    request.get<{ success: boolean; data?: KnowledgeBase; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(id)}`,
    ),

  create: (data: CreateKnowledgeBaseRequest) =>
    request.post<{ success: boolean; data?: KnowledgeBase; message?: string }>("/api/v1/knowledge-bases", data),

  update: (id: string, data: UpdateKnowledgeBaseRequest) =>
    request.put<{ success: boolean; data?: KnowledgeBase; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(id)}`,
      data,
    ),

  /**
   * 模型 + 分块配置（PUT /initialization/config/:kbId，camelCase 请求体）。
   * 覆盖式语义：VLM/ASR/存储/图谱/问题生成不传会被清空或回落 local，
   * 调用方必须从 KB 详情实体原值回传（见 KBModelConfig 注释）。
   */
  updateModelConfig: (kbId: string, data: KBModelConfig) =>
    request.put<{ success: boolean; message?: string }>(
      `/api/v1/initialization/config/${encodeURIComponent(kbId)}`,
      data,
    ),

  delete: (id: string) =>
    request.delete<{ success: boolean; message?: string }>(`/api/v1/knowledge-bases/${encodeURIComponent(id)}`),

  // 置顶/取消置顶（切换语义）
  togglePin: (id: string) =>
    request.put<{ success: boolean; message?: string }>(`/api/v1/knowledge-bases/${encodeURIComponent(id)}/pin`),
};

// ---- Query keys ----

export const knowledgeBaseKeys = {
  all: ["knowledge-bases"] as const,
  lists: () => [...knowledgeBaseKeys.all, "list"] as const,
  details: () => [...knowledgeBaseKeys.all, "detail"] as const,
  detail: (id: string) => [...knowledgeBaseKeys.details(), id] as const,
};

// ---- Hooks ----

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: string } | null)?.message;
  return message || fallback;
}

export function useKnowledgeBaseList() {
  return useQuery({
    queryKey: knowledgeBaseKeys.lists(),
    queryFn: async () => {
      const res = await knowledgeBaseApi.list();
      return res.data ?? [];
    },
  });
}

export function useKnowledgeBase(id: MaybeRefOrGetter<string | undefined>) {
  return useQuery({
    queryKey: knowledgeBaseKeys.detail(toValue(id) ?? ""),
    queryFn: async () => {
      const res = await knowledgeBaseApi.detail(toValue(id)!);
      return res.data ?? null;
    },
    enabled: () => Boolean(toValue(id)),
  });
}

export function useCreateKnowledgeBase() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateKnowledgeBaseRequest) => knowledgeBaseApi.create(data),
    onSuccess: () => {
      toast.success("知识库已创建");
      queryClient.invalidateQueries({ queryKey: knowledgeBaseKeys.lists() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "创建知识库失败"));
    },
  });
}

export function useUpdateKnowledgeBase() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateKnowledgeBaseRequest }) =>
      knowledgeBaseApi.update(id, data),
    onSuccess: (_res, { id }) => {
      toast.success("知识库已更新");
      queryClient.invalidateQueries({ queryKey: knowledgeBaseKeys.lists() });
      queryClient.invalidateQueries({ queryKey: knowledgeBaseKeys.detail(id) });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "更新知识库失败"));
    },
  });
}

export function useUpdateKBModelConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ kbId, data }: { kbId: string; data: KBModelConfig }) =>
      knowledgeBaseApi.updateModelConfig(kbId, data),
    onSuccess: (_res, { kbId }) => {
      toast.success("模型与分块配置已更新");
      queryClient.invalidateQueries({ queryKey: knowledgeBaseKeys.lists() });
      queryClient.invalidateQueries({ queryKey: knowledgeBaseKeys.detail(kbId) });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "更新模型配置失败"));
    },
  });
}

export function useDeleteKnowledgeBase() {  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => knowledgeBaseApi.delete(id),
    onSuccess: () => {
      toast.success("知识库已删除");
      queryClient.invalidateQueries({ queryKey: knowledgeBaseKeys.lists() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "删除知识库失败"));
    },
  });
}

export function useTogglePinKnowledgeBase() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => knowledgeBaseApi.togglePin(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: knowledgeBaseKeys.lists() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "置顶操作失败"));
    },
  });
}
