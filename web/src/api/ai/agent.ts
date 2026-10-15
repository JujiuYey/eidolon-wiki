import { request } from "@/utils/request";

/**
 * agent 体系只接入官方默认态（参考实现：WeKnora frontend/src/api/agent/index.ts）。
 *
 * 官方的 agent 选择器本质是问答管线切换：快速问答 → knowledge-chat，
 * 智能推理/自定义 agent → agent-chat。本前端问答管线固定 knowledge-chat，
 * 因此不提供选择器 UI，agent 仅作为推荐问题的生成主体走内置快速问答 agent
 * （后端 types/custom_agent.go BuiltinQuickAnswerID，注册表兜底、无需数据库记录）。
 */
export const BUILTIN_QUICK_ANSWER_ID = "builtin-quick-answer";

/** GET /agents/:id/suggested-questions 的返回项 */
export interface SuggestedQuestion {
  question: string;
  source: "faq" | "document" | "agent_config" | "wiki" | string;
  knowledge_base_id?: string;
}

interface Envelope<T> {
  success: boolean;
  data?: T;
  message?: string;
}

/**
 * GET /api/v1/agents/:agentId/suggested-questions
 * （internal/handler/custom_agent.go GetSuggestedQuestions；limit 缺省用 agent 配置的 6）
 */
export async function fetchSuggestedQuestions(
  agentId: string,
  params: { knowledgeBaseIds?: string[]; limit?: number } = {},
): Promise<SuggestedQuestion[]> {
  const search = new URLSearchParams();
  if (params.knowledgeBaseIds?.length) {
    search.set("knowledge_base_ids", params.knowledgeBaseIds.join(","));
  }
  if (params.limit) search.set("limit", String(params.limit));
  const qs = search.toString();
  const res = await request.get<Envelope<{ questions: SuggestedQuestion[] }>>(
    `/api/v1/agents/${encodeURIComponent(agentId)}/suggested-questions${qs ? `?${qs}` : ""}`,
  );
  return res.data?.questions ?? [];
}
