import { useMutation, useQuery, useQueryClient } from "@tanstack/vue-query";
import { toast } from "vue-sonner";

import { post, put, request } from "@/utils/request";

// ---- 类型（对齐 WeKnora internal/types/model.go 与 handler/dto/model.go）----

export type ModelType = "KnowledgeQA" | "Embedding" | "Rerank" | "VLLM" | "ASR";

export type ModelCredentialField = "api_key" | "app_secret";

export interface EmbeddingParameters {
  dimension?: number;
  truncate_prompt_tokens?: number;
  supports_dimension_override?: boolean;
}

/**
 * 模型参数。注意：主响应里永远没有 api_key/app_secret 的值——
 * 只有 credentials.{field}.configured 布尔（DTO 层已剔除）；
 * 明文密钥仅允许在"新建"时进 parameters，编辑一律走 credentials 子资源。
 */
export interface ModelParameters {
  base_url?: string;
  provider?: string;
  api_key?: string;
  app_secret?: string;
  interface_type?: "ollama" | "openai";
  parameter_size?: string;
  embedding_parameters?: EmbeddingParameters;
  extra_config?: Record<string, string>;
  custom_headers?: Record<string, string>;
  supports_vision?: boolean;
  max_concurrency?: number;
}

export interface ModelInfo {
  id: string;
  name: string;
  display_name?: string;
  type: ModelType;
  source: "local" | "remote" | string;
  description?: string;
  parameters: ModelParameters;
  is_default?: boolean;
  is_builtin?: boolean;
  status?: "active" | "downloading" | "download_failed" | string;
  credentials?: Partial<Record<ModelCredentialField, { configured: boolean }>>;
  created_at?: string;
  updated_at?: string;
}

export interface CreateModelRequest {
  name: string;
  display_name?: string;
  type: ModelType;
  source: "local" | "remote";
  description?: string;
  parameters: ModelParameters;
}

export interface UpdateModelRequest {
  name?: string;
  display_name?: string;
  description?: string;
  source?: "local" | "remote";
  parameters?: ModelParameters;
}

export interface ProviderInfo {
  value: string;
  label: string;
  description?: string;
  /** 各模型类型的默认 Base URL */
  defaultUrls?: Partial<Record<"chat" | "embedding" | "rerank" | "vllm" | "asr", string>>;
  modelTypes?: string[];
}

export interface ModelCheckResult {
  available: boolean;
  message?: string;
  dimension?: number;
}

interface Envelope<T> {
  success: boolean;
  data?: T;
  message?: string;
}

// ---- API 函数 ----

export const modelApi = {
  /** 后端不分页不过滤，一次全量返回；按 type 过滤由前端做 */
  list: () => request.get<Envelope<ModelInfo[]>>("/api/v1/models"),

  create: (data: CreateModelRequest) => request.post<Envelope<ModelInfo>>("/api/v1/models", data),

  update: (id: string, data: UpdateModelRequest) =>
    put<Envelope<ModelInfo>>(`/api/v1/models/${encodeURIComponent(id)}`, data),

  remove: (id: string) =>
    request.delete<Envelope<null>>(`/api/v1/models/${encodeURIComponent(id)}`),

  /** 保存密钥；返回各字段的 configured 状态 */
  putCredentials: (id: string, body: Partial<Record<ModelCredentialField, string>>) =>
    put<{ success: boolean; fields?: Record<string, { configured: boolean }> }>(
      `/api/v1/models/${encodeURIComponent(id)}/credentials`,
      body,
    ),

  removeCredentialField: (id: string, field: ModelCredentialField) =>
    request.delete<null>(`/api/v1/models/${encodeURIComponent(id)}/credentials/${field}`),

  /** 厂商注册表（带各类型默认 URL），失败时调用方用 FALLBACK_PROVIDERS 兜底 */
  listProviders: () => request.get<Envelope<ProviderInfo[]>>("/api/v1/models/providers"),

  /** 连通性测试（未持久化的裸参数；编辑态传 modelId 时后端用存储的解密密钥兜底） */
  checkChat: (body: {
    modelName: string;
    baseUrl: string;
    apiKey?: string;
    provider?: string;
    modelId?: string;
    custom_headers?: Record<string, string>;
  }) => post<ModelCheckResult>("/api/v1/initialization/remote/check", body),

  checkEmbedding: (body: {
    modelName: string;
    baseUrl?: string;
    apiKey?: string;
    provider?: string;
    source?: string;
    dimension?: number;
    supportsDimensionOverride?: boolean;
    modelId?: string;
    custom_headers?: Record<string, string>;
  }) => post<ModelCheckResult>("/api/v1/initialization/embedding/test", body),

  checkRerank: (body: {
    modelName: string;
    baseUrl: string;
    apiKey?: string;
    provider?: string;
    modelId?: string;
    custom_headers?: Record<string, string>;
    appSecret?: string;
    extraConfig?: Record<string, string>;
  }) => post<ModelCheckResult>("/api/v1/initialization/rerank/check", body),

  checkAsr: (body: {
    modelName: string;
    baseUrl: string;
    apiKey?: string;
    provider?: string;
    modelId?: string;
    custom_headers?: Record<string, string>;
  }) => post<ModelCheckResult>("/api/v1/initialization/asr/check", body),

  // ---- Ollama 本地模型（source=local）----

  ollamaStatus: () =>
    request.get<Envelope<{ available: boolean; version?: string; error?: string; baseUrl?: string }>>(
      "/api/v1/initialization/ollama/status",
    ),

  ollamaModels: () =>
    request.get<
      Envelope<{ models?: Array<{ name: string; size: number; digest?: string; modified_at?: string }> }>
    >("/api/v1/initialization/ollama/models"),

  ollamaDownload: (modelName: string) =>
    post<Envelope<{ taskId: string; modelName?: string; status?: string; progress?: number }>>(
      "/api/v1/initialization/ollama/models/download",
      { modelName },
    ),

  ollamaDownloadProgress: (taskId: string) =>
    request.get<
      Envelope<{
        id?: string;
        modelName?: string;
        status: "pending" | "downloading" | "completed" | "failed" | string;
        progress?: number;
        message?: string;
      }>
    >(`/api/v1/initialization/ollama/download/progress/${encodeURIComponent(taskId)}`),

  // ---- WeKnoraCloud（provider=weknoracloud 的凭证门控）----
  // 官方兼容两种包法：裸对象或 {success, data}，字段拍平返回

  weknoraCloudStatus: () =>
    request.get<{
      success?: boolean;
      data?: { has_models?: boolean; needs_reinit?: boolean; reason?: string };
      has_models?: boolean;
      needs_reinit?: boolean;
      reason?: string;
    }>("/api/v1/models/weknoracloud/status"),
};

// providers 接口不可用时的兜底（精简版官方列表）
export const FALLBACK_PROVIDERS: ProviderInfo[] = [
  { value: "zhipu", label: "智谱 AI" },
  { value: "openai", label: "OpenAI" },
  { value: "aliyun", label: "阿里云百炼" },
  { value: "siliconflow", label: "硅基流动" },
  { value: "deepseek", label: "DeepSeek" },
  { value: "gemini", label: "Google Gemini" },
  { value: "openrouter", label: "OpenRouter" },
  { value: "generic", label: "自定义" },
];

/** 删除冲突（后端 400 + error.code=2300）的结构化详情 */
export interface ModelUsageDetails {
  knowledge_bases?: Array<{ id: string; name: string; bindings: string[] }>;
  agents?: Array<{ id: string; name: string; bindings: string[] }>;
  long_term_memory?: { bindings?: string[] };
  knowledge_base_total?: number;
  agent_total?: number;
}

export function isModelInUseError(error: unknown): ModelUsageDetails | null {
  const err = error as { error?: { code?: number; details?: ModelUsageDetails } } | null;
  if (err?.error?.code === 2300) return err.error.details ?? {};
  return null;
}

// ---- Query keys ----

export const modelKeys = {
  all: ["models"] as const,
  list: () => [...modelKeys.all, "list"] as const,
  providers: () => [...modelKeys.all, "providers"] as const,
};

// ---- Hooks ----

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: string } | null)?.message;
  return message || fallback;
}

export function useModelList() {
  return useQuery({
    queryKey: modelKeys.list(),
    queryFn: async () => (await modelApi.list()).data ?? [],
  });
}

export function useModelProviders() {
  return useQuery({
    queryKey: modelKeys.providers(),
    queryFn: async () => {
      try {
        const res = await modelApi.listProviders();
        return res.data && res.data.length > 0 ? res.data : FALLBACK_PROVIDERS;
      } catch {
        return FALLBACK_PROVIDERS;
      }
    },
    staleTime: 10 * 60 * 1000,
  });
}

export function useCreateModel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateModelRequest) => modelApi.create(data),
    onSuccess: () => {
      toast.success("模型已创建");
      queryClient.invalidateQueries({ queryKey: modelKeys.list() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "创建模型失败"));
    },
  });
}

export function useUpdateModel() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateModelRequest }) => modelApi.update(id, data),
    onSuccess: () => {
      toast.success("模型已更新");
      queryClient.invalidateQueries({ queryKey: modelKeys.list() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "更新模型失败"));
    },
  });
}

// ---- 展示辅助 ----

export const modelTypeLabels: Record<ModelType, string> = {
  KnowledgeQA: "对话",
  Embedding: "向量",
  Rerank: "重排",
  VLLM: "多模态",
  ASR: "语音",
};

export const modelTypeBadgeClass: Record<ModelType, string> = {
  KnowledgeQA: "bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-300",
  Embedding: "bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300",
  Rerank: "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300",
  VLLM: "bg-violet-100 text-violet-800 dark:bg-violet-950 dark:text-violet-300",
  ASR: "bg-cyan-100 text-cyan-800 dark:bg-cyan-950 dark:text-cyan-300",
};

/** 官方语义：模型显示名 = display_name 优先，回退 name */
export function modelDisplayName(model: ModelInfo): string {
  return model.display_name || model.name;
}

/** provider 展示名：local→Ollama，generic→自定义，其余取厂商注册表 label（vllm/asr 回退"OpenAI兼容"，镜像官方） */
export function providerLabel(model: ModelInfo, providers: ProviderInfo[]): string {
  if (model.source === "local") return "Ollama";
  const providerId = model.parameters.provider ?? "";
  if (providerId === "generic") return "自定义";
  const label = providers.find((item) => item.value === providerId)?.label;
  if (label) return label;
  return model.type === "VLLM" || model.type === "ASR" ? "OpenAI兼容" : providerId || "Remote";
}

// ---- 思考模式参数格式（extra_config.thinking_control，官方 thinkingControl.ts 语义）----

export type ThinkingControl = "none" | "chat_template_kwargs" | "enable_thinking" | "thinking_type";

export const thinkingControlOptions: Array<{ value: ThinkingControl; label: string; hint: string }> = [
  { value: "none", label: "不写入思考参数", hint: "标准 OpenAI 接口，不携带思考模式参数" },
  {
    value: "chat_template_kwargs",
    label: "chat_template_kwargs",
    hint: "自定义 OpenAI 兼容服务、NVIDIA NIM、vLLM / 本地 Qwen 部署",
  },
  {
    value: "enable_thinking",
    label: "enable_thinking",
    hint: "阿里云 DashScope（qwen3 / qwen-plus / qwen-max / qwen-turbo）",
  },
  {
    value: "thinking_type",
    label: "thinking_type",
    hint: "火山引擎 Ark；腾讯云 LKEAP 等",
  },
];

function isQwenThinkingModel(name: string): boolean {
  const lower = name.toLowerCase();
  return lower.startsWith("qwen3") || ["qwen-plus", "qwen-max", "qwen-turbo"].some((m) => lower.startsWith(m));
}

/** 按厂商 + 模型名推断默认思考参数格式（用户手动改过则不覆盖） */
export function defaultThinkingControl(provider: string, modelName: string): ThinkingControl {
  const lower = modelName.toLowerCase();
  switch (provider) {
    case "aliyun":
      return isQwenThinkingModel(lower) ? "enable_thinking" : "none";
    case "lkeap":
      return lower.startsWith("deepseek-r1") ? "none" : "thinking_type";
    case "generic":
    case "nvidia":
    case "litellm":
      return "chat_template_kwargs";
    case "volcengine":
      return "thinking_type";
    default:
      return "none";
  }
}

/** 前端模型类型 → providers.defaultUrls 的 key（官方映射） */
export function modelTypeToUrlKey(type: ModelType): "chat" | "embedding" | "rerank" | "vllm" | "asr" {
  return type === "KnowledgeQA" ? "chat" : (type.toLowerCase() as "embedding" | "rerank" | "vllm" | "asr");
}
