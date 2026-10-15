<script setup lang="ts">
import { ListChecksIcon, MessageCirclePlusIcon, SearchIcon } from "@lucide/vue";
import { useQueryClient } from "@tanstack/vue-query";
import { computed, ref, watch } from "vue";

import { chatKeys, useBatchDeleteSessions, useSessionList } from "@/api/ai/chat";
import type { ChatSession } from "@/api/ai/chat";
import { ConfirmDialog } from "@/components/sag-ui";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";

import ChatPanel from "./_components/ChatPanel.vue";
import SessionList from "./_components/SessionList.vue";

const queryClient = useQueryClient();
const selectedSession = ref<string>();
const searchValue = ref("");

const batchMode = ref(false);
const checkedIds = ref<string[]>([]);
const batchConfirmOpen = ref(false);
const batchDelete = useBatchDeleteSessions();

const { data: sessionsData, isLoading: sessionsLoading } = useSessionList();

// 置顶优先，其余按更新时间倒序（对齐官方会话列表语义）
const filteredSessions = computed(() => {
  const keyword = searchValue.value.trim().toLowerCase();
  const items = sessionsData.value ?? [];
  const list = keyword
    ? items.filter((session) => (session.title || "").toLowerCase().includes(keyword))
    : [...items];
  return list.sort(
    (a, b) =>
      Number(Boolean(b.is_pinned)) - Number(Boolean(a.is_pinned)) ||
      (b.updated_at || "").localeCompare(a.updated_at || ""),
  );
});

// 默认选中第一个会话
watch(
  [sessionsData, selectedSession],
  () => {
    const items = sessionsData.value;
    if (items && items.length > 0 && !selectedSession.value) {
      selectedSession.value = items[0].id;
    }
  },
  { immediate: true },
);

const handleNewSession = () => {
  selectedSession.value = undefined;
};

const handleSessionCreated = (sessionId: string) => {
  selectedSession.value = sessionId;
  queryClient.invalidateQueries({ queryKey: chatKeys.sessions() });
};

const handleSessionTitle = (sessionId: string, title: string) => {
  // 新会话首问后，SSE session_title 事件带来的新标题，就地更新会话缓存
  queryClient.setQueriesData<ChatSession[]>({ queryKey: chatKeys.sessions() }, (old) =>
    old ? old.map((session) => (session.id === sessionId ? { ...session, title } : session)) : old,
  );
};

const handleTurnComplete = () => {
  queryClient.invalidateQueries({ queryKey: chatKeys.sessions() });
};

const handleDeleteSession = () => {
  selectedSession.value = undefined;
};

const exitBatchMode = () => {
  batchMode.value = false;
  checkedIds.value = [];
};

const handleBatchDelete = async () => {
  const ids = [...checkedIds.value];
  await batchDelete.mutateAsync(ids);
  batchConfirmOpen.value = false;
  if (ids.includes(selectedSession.value ?? "")) {
    selectedSession.value = undefined;
  }
  exitBatchMode();
};
</script>

<template>
  <div class="flex h-[calc(100vh-1rem)] overflow-hidden">
    <!-- 左：会话列表 -->
    <div class="flex w-[320px] shrink-0 flex-col overflow-hidden border-r">
      <div class="flex items-center justify-between px-4 py-2">
        <h1 class="text-xl font-bold">对话历史</h1>
        <div class="flex items-center gap-1">
          <Button
            size="icon-sm"
            variant="ghost"
            :class="batchMode && 'bg-muted text-foreground'"
            :aria-label="batchMode ? '退出批量管理' : '批量管理会话'"
            :title="batchMode ? '退出批量管理' : '批量管理'"
            @click="batchMode ? exitBatchMode() : (batchMode = true)"
          >
            <ListChecksIcon class="h-4 w-4" />
          </Button>
          <Button size="sm" @click="handleNewSession">
            <MessageCirclePlusIcon class="mr-1 h-4 w-4" />
            新建对话
          </Button>
        </div>
      </div>
      <div
        v-if="batchMode"
        class="flex items-center justify-between border-b bg-muted/40 px-4 py-1.5 text-xs"
      >
        <span class="text-muted-foreground">
          已选 <strong class="text-foreground">{{ checkedIds.length }}</strong> 个会话
        </span>
        <div class="flex items-center gap-1">
          <Button
            size="sm"
            variant="destructive"
            class="h-7 px-2 text-xs"
            :disabled="checkedIds.length === 0 || batchDelete.isPending.value"
            @click="batchConfirmOpen = true"
          >
            删除
          </Button>
          <Button size="sm" variant="ghost" class="h-7 px-2 text-xs" @click="exitBatchMode">
            取消
          </Button>
        </div>
      </div>
      <Separator v-if="!batchMode" />
      <div class="bg-background/95 p-4 backdrop-blur supports-backdrop-filter:bg-background/60">
        <div class="relative">
          <SearchIcon class="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input v-model="searchValue" placeholder="搜索" class="pl-8" />
        </div>
      </div>
      <SessionList
        :items="filteredSessions"
        :loading="sessionsLoading"
        :selected-id="selectedSession"
        :batch-mode="batchMode"
        :checked-ids="checkedIds"
        @select="(id) => (selectedSession = id)"
        @deleted="handleDeleteSession"
        @toggle-checked="
          (id) =>
            (checkedIds = checkedIds.includes(id)
              ? checkedIds.filter((checked) => checked !== id)
              : [...checkedIds, id])
        "
      />
    </div>

    <!-- 右：聊天面板 -->
    <div class="min-w-0 flex-1 overflow-hidden">
      <ChatPanel
        :session-id="selectedSession"
        @session-created="handleSessionCreated"
        @session-title="handleSessionTitle"
        @turn-complete="handleTurnComplete"
      />
    </div>

    <ConfirmDialog
      :open="batchConfirmOpen"
      destructive
      title="批量删除对话"
      :description="`确定删除选中的 ${checkedIds.length} 个对话？会话中的所有消息都会被删除。`"
      confirm-text="删除"
      @update:open="(open) => !open && (batchConfirmOpen = false)"
      @confirm="handleBatchDelete"
    />
  </div>
</template>
