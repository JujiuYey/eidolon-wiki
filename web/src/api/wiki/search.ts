import { request } from "@/utils/request";

// ---- 类型（对齐 WeKnora internal/types/search.go）----

export interface HybridSearchParams {
  query_text: string;
  /** top_k */
  match_count?: number;
  vector_threshold?: number;
  keyword_threshold?: number;
  /** 纯关键词检索（关闭向量） */
  disable_vector_match?: boolean;
  /** 纯向量检索（关闭关键词） */
  disable_keywords_match?: boolean;
  knowledge_ids?: string[];
}

export interface HybridSearchResult {
  id: string;
  content: string;
  knowledge_id: string;
  chunk_index?: number;
  knowledge_title?: string;
  knowledge_filename?: string;
  knowledge_source?: string;
  score: number;
  match_type: "keyword" | "vector" | "hybrid" | string;
  chunk_type?: string;
  start_at?: number;
  end_at?: number;
  matched_content?: string;
}

// ---- API 函数 ----

export const searchApi = {
  /** KB 内混合检索测试：POST /api/v1/knowledge-bases/:id/hybrid-search */
  hybridSearch: (kbId: string, params: HybridSearchParams) =>
    request.post<{ success: boolean; data?: HybridSearchResult[]; message?: string }>(
      `/api/v1/knowledge-bases/${encodeURIComponent(kbId)}/hybrid-search`,
      params,
    ),
};

// ---- 展示辅助 ----

export const matchTypeLabels: Record<string, string> = {
  hybrid: "混合",
  vector: "向量",
  keyword: "关键词",
};

export function matchTypeBadgeClass(matchType: string): string {
  switch (matchType) {
    case "hybrid":
      return "bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300";
    case "vector":
      return "bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-300";
    case "keyword":
      return "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300";
    default:
      return "bg-muted text-muted-foreground";
  }
}
