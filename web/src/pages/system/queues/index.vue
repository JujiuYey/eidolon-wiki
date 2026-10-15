<script setup lang="ts">
import { DatabaseIcon, RefreshCwIcon } from "@lucide/vue";
import type { ColumnDef } from "@tanstack/vue-table";
import { computed, h, ref } from "vue";

import {
  queueStateLabel,
  runtimeTaskStateLabels,
  useRuntimeQueues,
  type QueueStat,
} from "@/api/system/queues";
import { DataTable, SagPage } from "@/components/sag-ui";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Label } from "@/components/ui/label";
import { Progress } from "@/components/ui/progress";
import { Switch } from "@/components/ui/switch";
import RuntimeTasksSheet from "./_components/RuntimeTasksSheet.vue";

function formatBytes(bytes: number): string {
  if (!bytes) return "-";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${value.toFixed(value >= 10 || unit === 0 ? 0 : 1)} ${units[unit]}`;
}

const autoRefresh = ref(true);
const queuesQuery = useRuntimeQueues(() => autoRefresh.value);
const taskQueue = ref<QueueStat | null>(null);

const data = computed(() => queuesQuery.data.value);

const totals = computed(() => {
  const queues = data.value?.queues ?? [];
  return {
    active: queues.reduce((sum, q) => sum + q.active, 0),
    pending: queues.reduce((sum, q) => sum + q.pending + q.scheduled, 0),
    retry: queues.reduce((sum, q) => sum + q.retry, 0),
    archived: queues.reduce((sum, q) => sum + q.archived, 0),
  };
});

// 计数单元格：点击打开该队列的任务抽屉
function CountCell(queue: QueueStat, field: "active" | "pending" | "retry" | "archived" | "completed", extra?: number) {
  return h(
    "button",
    {
      type: "button",
      class:
        "inline-flex items-center gap-1 rounded px-1 text-sm tabular-nums transition-colors hover:text-foreground hover:underline",
      title: `查看${runtimeTaskStateLabels[field]}任务`,
      onClick: () => (taskQueue.value = queue),
    },
    [
      String(queue[field]),
      extra ? h("span", { class: "text-xs text-muted-foreground" }, `(+${extra})`) : null,
    ],
  );
}

const columns: ColumnDef<QueueStat, any>[] = [
  {
    accessorKey: "name",
    header: "队列",
    cell: ({ row }) => {
      const state = queueStateLabel(row.original);
      return h("div", { class: "flex flex-col gap-1" }, [
        h("div", { class: "flex items-center gap-2" }, [
          h("span", { class: "font-mono text-sm font-medium" }, row.original.name),
          h(Badge, { variant: "secondary", class: state.className }, () => state.text),
        ]),
        h(
          "span",
          { class: "text-xs text-muted-foreground" },
          `${row.original.pool} 池 · 权重 ${row.original.weight} · ${formatBytes(row.original.memory_usage_bytes)}`,
        ),
      ]);
    },
  },
  {
    accessorKey: "active",
    header: "执行中",
    cell: ({ row }) => CountCell(row.original, "active"),
  },
  {
    accessorKey: "pending",
    header: "等待(+排期)",
    cell: ({ row }) => CountCell(row.original, "pending", row.original.scheduled),
  },
  {
    accessorKey: "retry",
    header: "重试",
    cell: ({ row }) => CountCell(row.original, "retry"),
  },
  {
    accessorKey: "archived",
    header: "已归档",
    cell: ({ row }) => CountCell(row.original, "archived"),
  },
  {
    accessorKey: "completed",
    header: "已完成",
    cell: ({ row }) => CountCell(row.original, "completed"),
  },
  {
    accessorKey: "latency_ms",
    header: "延迟",
    cell: ({ row }) => h("span", { class: "text-sm text-muted-foreground" }, `${row.original.latency_ms}ms`),
  },
];
</script>

<template>
  <SagPage
    v-if="data && !data.available"
    title="运行队列"
    description="后台任务队列的实时负载（每 5 秒自动刷新）"
  >
    <Empty class="border rounded-lg">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <DatabaseIcon />
        </EmptyMedia>
        <EmptyTitle>任务队列不可用</EmptyTitle>
        <EmptyDescription>当前部署未启用 Redis / asynq 队列（Lite 模式），无队列可展示。</EmptyDescription>
      </EmptyHeader>
    </Empty>
  </SagPage>

  <SagPage
    v-else
    title="运行队列"
    description="后台任务队列的实时负载，以及各独立 worker 池的并发配置"
  >
    <template #actions>
      <div class="flex items-center gap-4">
        <div class="flex items-center gap-2">
          <Switch id="queue-auto-refresh" :model-value="autoRefresh" @update:model-value="v => (autoRefresh = v === true)" />
          <Label for="queue-auto-refresh" class="text-sm text-muted-foreground">自动刷新（每 5 秒）</Label>
        </div>
        <Button variant="outline" size="sm" :disabled="queuesQuery.isFetching.value" @click="queuesQuery.refetch()">
          <RefreshCwIcon :class="queuesQuery.isFetching.value ? 'animate-spin' : ''" />
          刷新
        </Button>
      </div>
    </template>

    <div class="flex h-full min-h-0 flex-col gap-5 overflow-y-auto pb-6">
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <Card class="py-4">
          <CardHeader class="px-4">
            <CardTitle class="text-xs font-medium text-muted-foreground">执行中</CardTitle>
          </CardHeader>
          <CardContent class="px-4">
            <span class="text-2xl font-bold tabular-nums">{{ totals.active }}</span>
          </CardContent>
        </Card>
        <Card class="py-4">
          <CardHeader class="px-4">
            <CardTitle class="text-xs font-medium text-muted-foreground">等待 / 排期</CardTitle>
          </CardHeader>
          <CardContent class="px-4">
            <span class="text-2xl font-bold tabular-nums">{{ totals.pending }}</span>
          </CardContent>
        </Card>
        <Card class="py-4">
          <CardHeader class="px-4">
            <CardTitle class="text-xs font-medium text-muted-foreground">待重试</CardTitle>
          </CardHeader>
          <CardContent class="px-4">
            <span class="text-2xl font-bold tabular-nums">{{ totals.retry }}</span>
          </CardContent>
        </Card>
        <Card class="py-4">
          <CardHeader class="px-4">
            <CardTitle class="text-xs font-medium text-muted-foreground">失败归档</CardTitle>
          </CardHeader>
          <CardContent class="px-4">
            <span class="text-2xl font-bold tabular-nums">{{ totals.archived }}</span>
          </CardContent>
        </Card>
      </div>

      <div v-if="(data?.pools.length ?? 0) > 0" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <Card v-for="pool in data?.pools ?? []" :key="pool.name" class="py-4">
          <CardHeader class="px-4">
            <CardTitle class="flex items-center justify-between text-sm font-semibold">
              <span>{{ pool.name }}</span>
              <span class="text-xs font-normal text-muted-foreground">{{ pool.active }}/{{ pool.cluster_capacity || "-" }}</span>
            </CardTitle>
          </CardHeader>
          <CardContent class="space-y-2 px-4">
            <Progress :model-value="Math.min(pool.utilization, 100)" />
            <p class="text-xs text-muted-foreground">
              每实例并发 {{ pool.concurrency }} · 实例 {{ pool.instances }} · 队列 {{ pool.queue_count }} · 利用率
              {{ Math.round(pool.utilization) }}%
            </p>
          </CardContent>
        </Card>
      </div>

      <div class="flex flex-col gap-3">
        <h2 class="text-base font-semibold">队列</h2>
        <DataTable :columns="columns" :data="data?.queues ?? []" :loading="queuesQuery.isLoading.value" empty-text="暂无队列" />
      </div>

      <div v-if="(data?.models.length ?? 0) > 0" class="flex flex-col gap-3">
        <h2 class="text-base font-semibold">模型限流</h2>
        <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          <Card v-for="model in data?.models ?? []" :key="model.model_id" class="py-4">
            <CardHeader class="px-4">
              <CardTitle class="flex items-center justify-between text-sm font-semibold">
                <span class="truncate">{{ model.name || model.model_id }}</span>
                <span class="text-xs font-normal text-muted-foreground">{{ model.active }}/{{ model.limit || "∞" }}</span>
              </CardTitle>
            </CardHeader>
            <CardContent class="space-y-2 px-4">
              <Progress :model-value="model.limit > 0 ? Math.min((model.active / model.limit) * 100, 100) : 0" />
              <p class="text-xs text-muted-foreground">执行 {{ model.active }} · 等待 {{ model.waiting }}</p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>

    <RuntimeTasksSheet :open="Boolean(taskQueue)" :queue="taskQueue" @update:open="open => !open && (taskQueue = null)" />
  </SagPage>
</template>
