import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/vue-query";
import { toValue } from "vue";
import type { MaybeRefOrGetter } from "vue";
import { toast } from "vue-sonner";

import type { HybridSearchResult } from "@/api/wiki/search";
import { request } from "@/utils/request";
import { sseRequest } from "@/utils/request/sse";

// ---- 类型（对齐 WeKnora internal/types/session.go、message.go、chat.go）----

export interface ChatSession {
  id: string;
  title: string;
  description?: string;
  is_pinned?: boolean;
  created_at: string;
  updated_at: string;
}

export interface TokenUsage {
  prompt_tokens?: number;
  completion_tokens?: number;
  total_tokens?: number;
  cached_tokens?: number;
}

/** 引用即检索结果（internal/types/search.go SearchResult） */
export type KnowledgeReference = HybridSearchResult;

export interface ChatMessage {
  id: string;
  session_id: string;
  content: string;
  role: "user" | "assistant" | "system";
  knowledge_references?: KnowledgeReference[];
  is_completed?: boolean;
  is_fallback?: boolean;
  usage?: TokenUsage;
  model_id?: string;
  created_at: string;
  updated_at: string;
}

/**
 * SSE 事件：事件名恒为 `message`，data 为 StreamResponse JSON。
 * 按 response_type 分发（internal/types/chat.go:203-249）。
 */
export type StreamResponseType =
  | "agent_query"
  | "thinking"
  | "tool_call"
  | "tool_result"
  | "references"
  | "answer"
  | "reflection"
  | "session_title"
  | "complete"
  | "stop"
  | "error"
  | string;

export interface StreamResponse {
  id?: string;
  response_type: StreamResponseType;
  content?: string;
  done?: boolean;
  knowledge_references?: KnowledgeReference[];
  session_id?: string;
  assistant_message_id?: string;
  data?: {
    references?: KnowledgeReference[];
    event_id?: string;
    duration_ms?: number;
    usage?: TokenUsage;
    session_id?: string;
    title?: string;
    [key: string]: unknown;
  };
  usage?: TokenUsage;
  finish_reason?: string;
}

/** GET /api/v1/models 的模型条目（只取聊天所需字段） */
export interface ChatModel {
  id?: string;
  name: string;
  display_name?: string;
  type?: "KnowledgeQA" | "Embedding" | "Rerank" | "VLLM" | "ASR" | string;
  is_default?: boolean;
}

/** 追问建议（internal/handler/message_suggestion.go + 官方前端 message-suggestion.ts） */
export interface MessageSuggestionItem {
  id: string;
  text: string;
  category?: "clarify" | "deepen" | "action";
  source?: "model" | "faq" | "document" | "wiki" | string;
  knowledge_base_ids?: string[];
}

export interface MessageSuggestionSet {
  id: string;
  session_id: string;
  assistant_message_id: string;
  status: "generating" | "ready" | "suppressed" | "failed";
  allow_regenerate?: boolean;
  questions: MessageSuggestionItem[];
}

interface Envelope<T> {
  success: boolean;
  data?: T;
  total?: number;
  page?: number;
  page_size?: number;
  message?: string;
}

// ---- API 函数 ----

export const chatApi = {
  listSessions: (page = 1, pageSize = 100, keyword?: string) => {
    const search = new URLSearchParams();
    search.set("page", String(page));
    search.set("page_size", String(pageSize));
    if (keyword) search.set("keyword", keyword);
    return request.get<Envelope<ChatSession[]>>(`/api/v1/sessions?${search.toString()}`);
  },

  createSession: (data: { title?: string } = {}) =>
    request.post<Envelope<ChatSession>>("/api/v1/sessions", data),

  renameSession: (id: string, title: string) =>
    request.put<Envelope<ChatSession>>(`/api/v1/sessions/${encodeURIComponent(id)}`, { title }),

  deleteSession: (id: string) =>
    request.delete<Envelope<null>>(`/api/v1/sessions/${encodeURIComponent(id)}`),

  /** 历史消息游标分页：before_time 传上一批最早一条的 created_at（RFC3339Nano） */
  loadMessages: (sessionId: string, options: { limit?: number; beforeTime?: string } = {}) => {
    const search = new URLSearchParams();
    search.set("limit", String(options.limit ?? 20));
    if (options.beforeTime) search.set("before_time", options.beforeTime);
    return request.get<Envelope<ChatMessage[]>>(
      `/api/v1/messages/${encodeURIComponent(sessionId)}/load?${search.toString()}`,
    );
  },

  listChatModels: () => request.get<Envelope<ChatModel[]>>("/api/v1/models"),

  /** 停止生成 */
  stopGeneration: (sessionId: string, messageId: string) =>
    request.post<Envelope<null>>(`/api/v1/sessions/${encodeURIComponent(sessionId)}/stop`, {
      message_id: messageId,
    }),

  pinSession: (id: string) =>
    request.post<Envelope<null>>(`/api/v1/sessions/${encodeURIComponent(id)}/pin`, {}),

  unpinSession: (id: string) =>
    request.delete<Envelope<null>>(`/api/v1/sessions/${encodeURIComponent(id)}/pin`),

  /** 批量删除会话（routes_chat.go /sessions/batch，body {ids}） */
  batchDeleteSessions: (ids: string[]) =>
    request.delete<Envelope<null>>(`/api/v1/sessions/batch`, { ids }),

  /** 追问建议：ensure 触发生成（幂等），GET 轮询至 ready/suppressed/failed（参考实现 message-suggestion.ts） */
  ensureSuggestions: (sessionId: string, messageId: string, regenerate = false) =>
    request.post<Envelope<MessageSuggestionSet>>(
      `/api/v1/sessions/${encodeURIComponent(sessionId)}/messages/${encodeURIComponent(messageId)}/suggestions`,
      { regenerate },
    ),

  getSuggestions: (sessionId: string, messageId: string) =>
    request.get<Envelope<MessageSuggestionSet>>(
      `/api/v1/sessions/${encodeURIComponent(sessionId)}/messages/${encodeURIComponent(messageId)}/suggestions`,
    ),
};

// ---- RAG 流式问答（SSE）----

export interface KnowledgeChatParams {
  query: string;
  knowledge_base_ids?: string[];
  summary_model_id?: string;
}

/**
 * POST /api/v1/knowledge-chat/{session_id}
 * 返回取消函数；AbortError 会被 sseRequest 归为 onComplete。
 */
export function startKnowledgeChat(
  sessionId: string,
  params: KnowledgeChatParams,
  onEvent: (event: StreamResponse) => void,
  onComplete?: () => void,
  onError?: (error: Error) => void,
): () => void {
  return sseRequest({
    url: `/api/v1/knowledge-chat/${encodeURIComponent(sessionId)}`,
    method: "POST",
    body: {
      query: params.query,
      knowledge_base_ids: params.knowledge_base_ids ?? [],
      agent_enabled: false,
      summary_model_id: params.summary_model_id ?? "",
      channel: "web",
    },
    onMessage: (raw) => {
      try {
        onEvent(JSON.parse(raw) as StreamResponse);
      } catch {
        // 非 JSON 心跳/杂散行忽略
      }
    },
    onComplete,
    onError,
  });
}

// ---- Query keys ----

export const chatKeys = {
  sessions: () => ["chat-sessions"] as const,
  messages: (sessionId: string) => ["chat-messages", sessionId] as const,
  models: () => ["chat-models"] as const,
};

// ---- Hooks ----

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: string } | null)?.message;
  return message || fallback;
}

export const HISTORY_PAGE_SIZE = 20;

export function useSessionList(keyword?: MaybeRefOrGetter<string | undefined>) {
  return useQuery({
    queryKey: [...chatKeys.sessions(), () => toValue(keyword) ?? ""],
    queryFn: async () => {
      const res = await chatApi.listSessions(1, 100, toValue(keyword));
      return res.data ?? [];
    },
  });
}

export function useCreateSession() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { title?: string }) => chatApi.createSession(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: chatKeys.sessions() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "创建会话失败"));
    },
  });
}

export function useRenameSession() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, title }: { id: string; title: string }) => chatApi.renameSession(id, title),
    onSuccess: () => {
      toast.success("标题已更新");
      queryClient.invalidateQueries({ queryKey: chatKeys.sessions() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "重命名失败"));
    },
  });
}

export function useDeleteSession() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => chatApi.deleteSession(id),
    onSuccess: () => {
      toast.success("会话已删除");
      queryClient.invalidateQueries({ queryKey: chatKeys.sessions() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "删除会话失败"));
    },
  });
}

export function useTogglePinSession() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, pinned }: { id: string; pinned: boolean }) =>
      pinned ? chatApi.unpinSession(id) : chatApi.pinSession(id),
    onSuccess: (_result, vars) => {
      toast.success(vars.pinned ? "已取消置顶" : "已置顶");
      queryClient.invalidateQueries({ queryKey: chatKeys.sessions() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "置顶操作失败"));
    },
  });
}

export function useBatchDeleteSessions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) => chatApi.batchDeleteSessions(ids),
    onSuccess: (_result, ids) => {
      toast.success(`已删除 ${ids.length} 个会话`);
      queryClient.invalidateQueries({ queryKey: chatKeys.sessions() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "批量删除失败"));
    },
  });
}

/** 历史消息：游标分页，pages 按获取顺序（最新一批在前），展示时需反转 */
export function useMessageHistory(sessionId: MaybeRefOrGetter<string | undefined>) {
  return useInfiniteQuery({
    queryKey: [...chatKeys.messages(""), () => toValue(sessionId)],
    queryFn: async ({ pageParam }: { pageParam: string | undefined }) => {
      const res = await chatApi.loadMessages(toValue(sessionId)!, {
        limit: HISTORY_PAGE_SIZE,
        beforeTime: pageParam,
      });
      return res.data ?? [];
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => {
      if (lastPage.length < HISTORY_PAGE_SIZE || lastPage.length === 0) return undefined;
      return lastPage[0]?.created_at;
    },
    enabled: () => Boolean(toValue(sessionId)),
  });
}

export function useChatModels() {
  return useQuery({
    queryKey: chatKeys.models(),
    queryFn: async () => {
      const res = await chatApi.listChatModels();
      return (res.data ?? []).filter((model) => model.type === "KnowledgeQA");
    },
  });
}
