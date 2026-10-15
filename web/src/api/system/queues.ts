import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/vue-query";
import { toValue } from "vue";
import type { MaybeRefOrGetter } from "vue";
import { toast } from "vue-sonner";

import { request } from "@/utils/request";

// ---- 类型（对齐 WeKnora frontend/src/api/system/index.ts 运行时队列部分）----

export interface QueueStat {
  name: string;
  pool: string;
  weight: number;
  size: number;
  pending: number;
  active: number;
  scheduled: number;
  retry: number;
  archived: number;
  completed: number;
  processed: number;
  failed: number;
  paused: boolean;
  latency_ms: number;
  memory_usage_bytes: number;
}

export interface RuntimeWorkerPool {
  name: string;
  concurrency: number;
  queue_count: number;
  instances: number;
  cluster_capacity: number;
  active: number;
  utilization: number;
}

export interface ModelRuntimeStat {
  model_id: string;
  name: string;
  active: number;
  waiting: number;
  limit: number;
}

/** Lite 模式（无 Redis/asynq）available=false */
export interface RuntimeQueuesResponse {
  available: boolean;
  upstream_concurrency: number;
  parse_concurrency: number;
  wiki_concurrency: number;
  pools: RuntimeWorkerPool[];
  queues: QueueStat[];
  model_limiter_available: boolean;
  models: ModelRuntimeStat[];
  timestamp: number;
}

export type RuntimeTaskState = "pending" | "active" | "scheduled" | "retry" | "archived" | "completed";
export type RuntimeTaskAction = "cancel" | "run_now" | "delete";

export interface RuntimeTask {
  id: string;
  queue: string;
  type: string;
  state: RuntimeTaskState;
  allowed_actions: RuntimeTaskAction[];
  last_error?: string;
  last_failed_at?: string;
  next_process_at?: string;
  started_at?: string;
  completed_at?: string;
  deadline?: string;
  enqueued_at?: string;
  retried: number;
  max_retry: number;
  is_orphaned?: boolean;
  worker?: string;
  tenant_id?: number;
  knowledge_base_id?: string;
  knowledge_id?: string;
  task_id?: string;
  source_id?: string;
  target_id?: string;
  source_kb_id?: string;
  target_kb_id?: string;
  data_source_id?: string;
  sync_log_id?: string;
  knowledge_count?: number;
}

export interface RuntimeTasksResponse {
  available: boolean;
  tasks: RuntimeTask[];
  page_size: number;
  has_more: boolean;
  next_cursor?: string;
}

// ---- API 函数（裸对象响应，无 {data} 包裹）----

export const queueApi = {
  getRuntimeQueues: () => request.get<RuntimeQueuesResponse>("/api/v1/system/admin/runtime/queues"),

  listTasks: (queue: string, params: { state: RuntimeTaskState; cursor?: string; page_size?: number }) => {
    const qs = new URLSearchParams();
    qs.set("state", params.state);
    if (params.cursor) qs.set("cursor", params.cursor);
    if (params.page_size) qs.set("page_size", String(params.page_size));
    return request.get<RuntimeTasksResponse>(
      `/api/v1/system/admin/runtime/queues/${encodeURIComponent(queue)}/tasks?${qs.toString()}`,
    );
  },

  mutateTask: (queue: string, taskId: string, action: RuntimeTaskAction) =>
    request.post<{ success: boolean; message?: string }>(
      `/api/v1/system/admin/runtime/queues/${encodeURIComponent(queue)}/tasks/${encodeURIComponent(taskId)}/actions/${action}`,
    ),

  purgeArchived: (queue: string) =>
    request.delete<{ success: boolean; deleted?: number }>(
      `/api/v1/system/admin/runtime/queues/${encodeURIComponent(queue)}/archived`,
    ),
};

// ---- Query keys ----

export const queueKeys = {
  all: ["runtime-queues"] as const,
  overview: () => [...queueKeys.all, "overview"] as const,
  tasks: (queue: string, state: RuntimeTaskState) => [...queueKeys.all, "tasks", queue, state] as const,
};

// ---- Hooks ----

export const RUNTIME_POLL_INTERVAL_MS = 5000;
export const RUNTIME_TASK_PAGE_SIZE = 20;

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: string } | null)?.message;
  return message || fallback;
}

export function useRuntimeQueues(autoRefresh: MaybeRefOrGetter<boolean>) {
  return useQuery(() => ({
    queryKey: queueKeys.overview(),
    queryFn: async () => queueApi.getRuntimeQueues(),
    // 显式断言阻止字面量加宽（false → boolean），vue-query 不接受 true
    refetchInterval: (toValue(autoRefresh) ? RUNTIME_POLL_INTERVAL_MS : false) as number | false,
  }));
}

export function useRuntimeTasks(queue: MaybeRefOrGetter<string | null>, state: MaybeRefOrGetter<RuntimeTaskState>) {
  return useInfiniteQuery({
    // queryKey 形状与 queueKeys.tasks(queue, state) 一致（getter 由 vue-query 解析）
    queryKey: [...queueKeys.all, "tasks", () => toValue(queue) ?? "", () => toValue(state)],
    queryFn: async ({ pageParam }: { pageParam: string | undefined }) =>
      queueApi.listTasks(toValue(queue) ?? "", {
        state: toValue(state),
        cursor: pageParam,
        page_size: RUNTIME_TASK_PAGE_SIZE,
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => (lastPage.has_more ? (lastPage.next_cursor ?? undefined) : undefined),
    enabled: () => Boolean(toValue(queue)),
  });
}

export function useRuntimeTaskAction(queue: MaybeRefOrGetter<string>, state: MaybeRefOrGetter<RuntimeTaskState>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ taskId, action }: { taskId: string; action: RuntimeTaskAction }) =>
      queueApi.mutateTask(toValue(queue), taskId, action),
    onSuccess: (_res, { action }) => {
      const labels: Record<RuntimeTaskAction, string> = {
        cancel: "任务已终止",
        run_now: "已安排立即执行",
        delete: "任务记录已删除",
      };
      toast.success(labels[action]);
      queryClient.invalidateQueries({ queryKey: queueKeys.tasks(toValue(queue), toValue(state)) });
      queryClient.invalidateQueries({ queryKey: queueKeys.overview() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "操作失败"));
    },
  });
}

export function usePurgeArchivedTasks(queue: MaybeRefOrGetter<string>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => queueApi.purgeArchived(toValue(queue)),
    onSuccess: (res) => {
      toast.success(`已清除 ${res.deleted ?? 0} 条失败任务记录`);
      queryClient.invalidateQueries({ queryKey: queueKeys.tasks(toValue(queue), "archived") });
      queryClient.invalidateQueries({ queryKey: queueKeys.overview() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "清除失败任务记录出错"));
    },
  });
}

// ---- 展示辅助 ----

export function queueStateLabel(queue: QueueStat): { text: string; className: string } {
  if (queue.paused) return { text: "已暂停", className: "bg-muted text-muted-foreground" };
  if (queue.archived > 0) return { text: "需处理", className: "bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300" };
  if (queue.retry > 0) return { text: "重试中", className: "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300" };
  if (queue.active > 0) return { text: "执行中", className: "bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-300" };
  if (queue.pending + queue.scheduled > 0) return { text: "排队中", className: "bg-cyan-100 text-cyan-800 dark:bg-cyan-950 dark:text-cyan-300" };
  return { text: "空闲", className: "bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300" };
}

export const runtimeTaskStateLabels: Record<RuntimeTaskState, string> = {
  pending: "等待中",
  active: "执行中",
  scheduled: "已排期",
  retry: "重试",
  archived: "已归档",
  completed: "已完成",
};

const taskTypeLabels: Record<string, string> = {
  "document:process": "文档解析",
  "document:reprocess": "文档重解析",
  "document:summary": "文档摘要",
  "knowledge:index": "知识索引",
  "knowledge:enrich": "内容富化",
  "datasource:sync": "数据源同步",
  "wiki:process": "Wiki 处理",
  "maintenance:cleanup": "维护清理",
};

export function runtimeTaskTypeLabel(type: string): string {
  return taskTypeLabels[type] ?? type;
}
