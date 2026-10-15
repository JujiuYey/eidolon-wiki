<script setup lang="ts">
import { CheckIcon, EditIcon, PinIcon, PinOffIcon, Trash2Icon, XIcon } from "@lucide/vue";
import { ref, useTemplateRef } from "vue";
import type { ComponentPublicInstance } from "vue";

import { useDeleteSession, useRenameSession, useTogglePinSession } from "@/api/ai/chat";
import type { ChatSession } from "@/api/ai/chat";
import { ConfirmDialog } from "@/components/sag-ui";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

const props = defineProps<{
  items: ChatSession[];
  loading: boolean;
  selectedId?: string;
  /** 批量管理模式：点击行切换勾选而非选中会话 */
  batchMode?: boolean;
  checkedIds?: string[];
}>();

const emit = defineEmits<{
  select: [id: string];
  /** 删除成功后回调（父组件据此清空选中态） */
  deleted: [id: string];
  toggleChecked: [id: string];
}>();

const renameSession = useRenameSession();
const deleteSession = useDeleteSession();
const togglePinSession = useTogglePinSession();

const editingId = ref<string | null>(null);
const editingTitle = ref("");
const deleteTarget = ref<ChatSession | null>(null);
const inputRef = useTemplateRef<ComponentPublicInstance>("inputRef");

const isChecked = (id: string) => Boolean(props.checkedIds?.includes(id));

const handleRowClick = (session: ChatSession) => {
  if (props.batchMode) {
    emit("toggleChecked", session.id);
  } else if (editingId.value !== session.id) {
    emit("select", session.id);
  }
};

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) return dateStr;
  const diffMs = Date.now() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return "刚刚";
  if (diffMin < 60) return `${diffMin} 分钟前`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour} 小时前`;
  const diffDay = Math.floor(diffHour / 24);
  if (diffDay < 30) return `${diffDay} 天前`;
  const diffMonth = Math.floor(diffDay / 30);
  if (diffMonth < 12) return `${diffMonth} 个月前`;
  return `${Math.floor(diffDay / 365)} 年前`;
}

const startRename = (session: ChatSession) => {
  editingId.value = session.id;
  editingTitle.value = session.title;
  requestAnimationFrame(() => inputRef.value?.$el?.select());
};

const saveRename = async () => {
  if (!editingId.value) return;
  const title = editingTitle.value.trim();
  if (title) {
    await renameSession.mutateAsync({ id: editingId.value, title });
  }
  editingId.value = null;
};

const handleDeleteConfirm = async () => {
  if (deleteTarget.value) {
    await deleteSession.mutateAsync(deleteTarget.value.id);
    emit("deleted", deleteTarget.value.id);
  }
};
</script>

<template>
  <div v-if="props.loading" class="flex-1 space-y-2 overflow-y-auto p-3">
    <Skeleton v-for="index in 5" :key="index" class="h-16 w-full" />
  </div>

  <div v-else-if="props.items.length === 0" class="flex flex-1 items-center justify-center p-6">
    <p class="text-sm text-muted-foreground">暂无对话，发送第一条消息开始</p>
  </div>

  <div v-else class="flex-1 space-y-1 overflow-y-auto p-3">
    <div
      v-for="session in props.items"
      :key="session.id"
      role="button"
      tabindex="0"
      :class="
        cn(
          'group cursor-pointer rounded-lg border p-3 transition-colors hover:bg-muted/50',
          props.batchMode && isChecked(session.id) && 'border-primary bg-primary/5',
          !props.batchMode && props.selectedId === session.id && 'border-primary/40 bg-muted',
        )
      "
      @click="handleRowClick(session)"
      @keydown.enter="handleRowClick(session)"
    >
      <div v-if="editingId === session.id" class="flex items-center gap-1">
        <Input
          ref="inputRef"
          v-model="editingTitle"
          class="h-7 text-sm"
          @keydown.enter="saveRename()"
          @keydown.escape="editingId = null"
          @click.stop
        />
        <Button
          size="icon-sm"
          variant="ghost"
          aria-label="保存标题"
          @click.stop="saveRename()"
        >
          <CheckIcon class="h-4 w-4" />
        </Button>
        <Button
          size="icon-sm"
          variant="ghost"
          aria-label="取消重命名"
          @click.stop="editingId = null"
        >
          <XIcon class="h-4 w-4" />
        </Button>
      </div>
      <template v-else>
        <div class="flex items-center justify-between gap-2">
          <span
            v-if="props.batchMode"
            class="flex h-4 w-4 shrink-0 items-center justify-center rounded-full border"
            :class="isChecked(session.id) ? 'border-primary bg-primary text-primary-foreground' : 'border-muted-foreground/40'"
          >
            <CheckIcon v-if="isChecked(session.id)" class="h-3 w-3" />
          </span>
          <span class="min-w-0 flex-1 truncate text-sm font-medium">
            {{ session.title || "未命名对话" }}
          </span>
          <PinIcon
            v-if="session.is_pinned && !props.batchMode"
            class="h-3.5 w-3.5 shrink-0 text-amber-500"
          />
          <span v-if="!props.batchMode" class="hidden shrink-0 gap-0.5 group-hover:flex">
            <Button
              size="icon-sm"
              variant="ghost"
              :aria-label="session.is_pinned ? `取消置顶 ${session.title || '会话'}` : `置顶 ${session.title || '会话'}`"
              @click.stop="togglePinSession.mutate({ id: session.id, pinned: Boolean(session.is_pinned) })"
            >
              <PinOffIcon v-if="session.is_pinned" class="h-3.5 w-3.5" />
              <PinIcon v-else class="h-3.5 w-3.5" />
            </Button>
            <Button
              size="icon-sm"
              variant="ghost"
              :aria-label="`重命名 ${session.title || '会话'}`"
              @click.stop="startRename(session)"
            >
              <EditIcon class="h-3.5 w-3.5" />
            </Button>
            <Button
              size="icon-sm"
              variant="ghost"
              :aria-label="`删除 ${session.title || '会话'}`"
              @click.stop="deleteTarget = session"
            >
              <Trash2Icon class="h-3.5 w-3.5" />
            </Button>
          </span>
        </div>
        <p class="mt-1 text-xs text-muted-foreground">
          {{ formatRelativeTime(session.updated_at) }}
        </p>
      </template>
    </div>

    <ConfirmDialog
      :open="Boolean(deleteTarget)"
      destructive
      title="删除对话"
      :description="`确定删除「${deleteTarget?.title || '未命名对话'}」？会话中的所有消息都会被删除。`"
      confirm-text="删除"
      @update:open="(open) => !open && (deleteTarget = null)"
      @confirm="handleDeleteConfirm"
    />
  </div>
</template>
