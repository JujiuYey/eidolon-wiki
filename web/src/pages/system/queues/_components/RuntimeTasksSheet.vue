<script setup lang="ts">
import { Loader2Icon, PlayIcon, Trash2Icon, XIcon } from "@lucide/vue";
import { computed, ref, watch } from "vue";

import {
  RUNTIME_TASK_PAGE_SIZE,
  runtimeTaskStateLabels,
  runtimeTaskTypeLabel,
  usePurgeArchivedTasks,
  useRuntimeTaskAction,
  useRuntimeTasks,
  type QueueStat,
  type RuntimeTask,
  type RuntimeTaskState,
} from "@/api/system/queues";
import { ConfirmDialog } from "@/components/sag-ui";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

const TASK_STATES: RuntimeTaskState[] = ["pending", "active", "scheduled", "retry", "archived", "completed"];

const TASK_ACTION_LABELS: Record<string, string> = {
  cancel: "终止任务",
  run_now: "立即执行",
  delete: "删除记录",
};

const props = defineProps<{
  open: boolean;
  queue: QueueStat | null;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
}>();

const state = ref<RuntimeTaskState>("pending");
const actionTarget = ref<{ task: RuntimeTask; action: string } | null>(null);
const purgeOpen = ref(false);

watch(
  () => [props.open, props.queue] as const,
  ([open, queue]) => {
    if (open && queue) {
      // 默认打开有内容的 tab：归档 > 重试 > 等待
      state.value = queue.archived > 0 ? "archived" : queue.retry > 0 ? "retry" : "pending";
    }
  },
  { immediate: true },
);

const queueName = computed(() => props.queue?.name ?? "");

const tasksQuery = useRuntimeTasks(
  () => (props.open ? queueName.value : null),
  () => state.value,
);
const taskAction = useRuntimeTaskAction(
  queueName,
  () => state.value,
);
const purgeArchived = usePurgeArchivedTasks(queueName);

const tasks = computed(() => tasksQuery.data.value?.pages.flatMap((page) => page.tasks) ?? []);

const counts = computed<Record<RuntimeTaskState, number>>(() => ({
  pending: props.queue?.pending ?? 0,
  active: props.queue?.active ?? 0,
  scheduled: props.queue?.scheduled ?? 0,
  retry: props.queue?.retry ?? 0,
  archived: props.queue?.archived ?? 0,
  completed: props.queue?.completed ?? 0,
}));

const handleConfirmAction = async () => {
  if (!actionTarget.value || !props.queue) return;
  await taskAction.mutateAsync({
    taskId: actionTarget.value.task.id,
    action: actionTarget.value.action as never,
  });
};
</script>

<template>
  <Sheet :open="open" @update:open="emit('update:open', $event)">
    <SheetContent class="flex flex-col gap-0 sm:max-w-2xl">
      <SheetHeader class="pb-2">
        <SheetTitle>队列任务：{{ queueName }}</SheetTitle>
        <SheetDescription>
          {{ queue ? `${queue.pool} 池 · 权重 ${queue.weight} · 延迟 ${queue.latency_ms}ms` : "" }}
        </SheetDescription>
      </SheetHeader>

      <div class="flex min-h-0 flex-1 flex-col gap-3 px-4 pb-4">
        <Tabs v-model="state" class="gap-0">
          <TabsList class="w-full flex-wrap">
            <TabsTrigger v-for="item in TASK_STATES" :key="item" :value="item" class="gap-1">
              {{ runtimeTaskStateLabels[item] }}
              <span class="text-xs text-muted-foreground">{{ counts[item] }}</span>
            </TabsTrigger>
          </TabsList>
        </Tabs>

        <div v-if="state === 'archived' && (queue?.archived ?? 0) > 0" class="flex justify-end">
          <Button variant="outline" size="sm" :disabled="purgeArchived.isPending.value" @click="purgeOpen = true">
            <Trash2Icon />
            清空失败任务
          </Button>
        </div>

        <div class="min-h-0 flex-1 space-y-3 overflow-y-auto">
          <div v-if="tasksQuery.isLoading.value" class="flex justify-center py-10">
            <Loader2Icon class="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
          <Empty v-else-if="tasks.length === 0">
            <EmptyHeader>
              <EmptyTitle>没有{{ runtimeTaskStateLabels[state] }}任务</EmptyTitle>
              <EmptyDescription>该状态下当前队列没有任务。</EmptyDescription>
            </EmptyHeader>
          </Empty>
          <template v-else>
            <div v-for="task in tasks" :key="task.id" class="rounded-lg border p-3">
              <div class="mb-2 flex items-center justify-between gap-2">
                <div class="flex min-w-0 items-center gap-2">
                  <Badge variant="secondary">{{ runtimeTaskTypeLabel(task.type) }}</Badge>
                  <span class="truncate font-mono text-xs text-muted-foreground">{{ task.id }}</span>
                  <Badge
                    v-if="task.is_orphaned"
                    variant="secondary"
                    class="bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300"
                  >
                    孤儿任务
                  </Badge>
                </div>
                <div class="flex shrink-0 gap-1">
                  <Button
                    v-for="action in task.allowed_actions"
                    :key="action"
                    variant="ghost"
                    size="sm"
                    :disabled="taskAction.isPending.value"
                    @click="actionTarget = { task, action }"
                  >
                    <PlayIcon v-if="action === 'run_now'" />
                    <XIcon v-else-if="action === 'cancel'" />
                    <Trash2Icon v-else />
                    {{ TASK_ACTION_LABELS[action] ?? action }}
                  </Button>
                </div>
              </div>
              <div class="grid grid-cols-[72px_1fr] gap-x-2 gap-y-1 text-xs">
                <span class="text-muted-foreground">重试</span>
                <span>{{ task.retried + 1 }} / {{ task.max_retry + 1 }}</span>
                <template v-if="task.worker">
                  <span class="text-muted-foreground">Worker</span>
                  <span class="truncate font-mono">{{ task.worker }}</span>
                </template>
                <template v-if="task.next_process_at">
                  <span class="text-muted-foreground">下次执行</span>
                  <span class="font-mono">{{ new Date(task.next_process_at).toLocaleString("zh-CN", { hour12: false }) }}</span>
                </template>
                <template v-if="task.completed_at">
                  <span class="text-muted-foreground">完成时间</span>
                  <span class="font-mono">{{ new Date(task.completed_at).toLocaleString("zh-CN", { hour12: false }) }}</span>
                </template>
                <template v-if="task.knowledge_base_id">
                  <span class="text-muted-foreground">知识库</span>
                  <span class="truncate font-mono">{{ task.knowledge_base_id }}</span>
                </template>
                <template v-if="task.tenant_id != null">
                  <span class="text-muted-foreground">空间</span>
                  <span>{{ task.tenant_id }}</span>
                </template>
                <template v-if="task.last_error">
                  <span class="text-destructive">错误</span>
                  <span class="whitespace-pre-wrap break-all text-destructive">{{ task.last_error }}</span>
                </template>
              </div>
            </div>
          </template>

          <div v-if="tasksQuery.hasNextPage.value" class="flex justify-center pb-2">
            <Button
              variant="outline"
              size="sm"
              :disabled="tasksQuery.isFetchingNextPage.value"
              @click="tasksQuery.fetchNextPage()"
            >
              {{ tasksQuery.isFetchingNextPage.value ? "加载中..." : `加载更多（每页 ${RUNTIME_TASK_PAGE_SIZE} 条）` }}
            </Button>
          </div>
        </div>
      </div>

      <ConfirmDialog
        :open="Boolean(actionTarget)"
        :destructive="actionTarget?.action !== 'run_now'"
        :title="actionTarget ? TASK_ACTION_LABELS[actionTarget.action] ?? '确认操作' : '确认操作'"
        :description="`确定对任务 ${actionTarget?.task.id.slice(0, 16)}… 执行「${actionTarget ? TASK_ACTION_LABELS[actionTarget.action] : ''}」？`"
        @update:open="open => !open && (actionTarget = null)"
        @confirm="handleConfirmAction"
      />

      <ConfirmDialog
        :open="purgeOpen"
        destructive
        title="清空失败任务"
        :description="`确定清除队列 ${queueName} 的全部已归档失败任务记录？该操作不可恢复。`"
        confirm-text="清空"
        @update:open="purgeOpen = $event"
        @confirm="() => { void purgeArchived.mutateAsync() }"
      />
    </SheetContent>
  </Sheet>
</template>
