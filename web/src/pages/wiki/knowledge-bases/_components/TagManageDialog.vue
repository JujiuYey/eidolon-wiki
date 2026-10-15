<script setup lang="ts">
import { CheckIcon, PlusIcon, Trash2Icon, XIcon } from "@lucide/vue";
import { ref, watch } from "vue";

import {
  tagBadgeStyle,
  useCreateTag,
  useDeleteTag,
  useUpdateTag,
  type KnowledgeTag,
} from "@/api/wiki/tag";
import { ConfirmDialog } from "@/components/sag-ui";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";

const props = defineProps<{
  open: boolean;
  kbId: string;
  tags: KnowledgeTag[];
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
}>();

const createTag = useCreateTag(() => props.kbId);
const updateTag = useUpdateTag(() => props.kbId);
const deleteTag = useDeleteTag(() => props.kbId);

const newName = ref("");
const editingId = ref<string | null>(null);
const editingName = ref("");
const deleteTarget = ref<KnowledgeTag | null>(null);

watch(
  () => props.open,
  (open) => {
    if (open) {
      newName.value = "";
      editingId.value = null;
      deleteTarget.value = null;
    }
  },
);

const handleCreate = async () => {
  const name = newName.value.trim();
  if (!name) return;
  await createTag.mutateAsync({ name });
  newName.value = "";
};

const startEdit = (tag: KnowledgeTag) => {
  editingId.value = tag.id;
  editingName.value = tag.name;
};

const handleRename = async () => {
  const tag = props.tags.find((t) => t.id === editingId.value);
  const name = editingName.value.trim();
  if (!tag || !name || name === tag.name) {
    editingId.value = null;
    return;
  }
  await updateTag.mutateAsync({ tagId: tag.id, data: { name } });
  editingId.value = null;
};

const handleDelete = async () => {
  if (!deleteTarget.value) return;
  await deleteTag.mutateAsync(deleteTarget.value.id);
  deleteTarget.value = null;
};
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>管理标签</DialogTitle>
        <DialogDescription>
          标签属于当前知识库，用于文档分类与检索过滤。删除标签会从所有文档上移除它。
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-3">
        <form class="flex gap-2" @submit.prevent="handleCreate">
          <Input v-model="newName" placeholder="新标签名称" class="flex-1" />
          <Button type="submit" size="sm" :disabled="!newName.trim() || createTag.isPending.value">
            <PlusIcon />
            新建
          </Button>
        </form>

        <div class="max-h-72 space-y-1 overflow-y-auto">
          <p v-if="tags.length === 0" class="py-6 text-center text-sm text-muted-foreground">
            还没有标签，先建一个。
          </p>
          <div
            v-for="tag in tags"
            v-else
            :key="tag.id"
            class="flex items-center gap-2 rounded-md border px-3 py-2"
          >
            <span
              class="size-2.5 shrink-0 rounded-full"
              :style="tagBadgeStyle(tag.color) ?? { backgroundColor: 'var(--muted-foreground)' }"
            />
            <template v-if="editingId === tag.id">
              <Input
                v-model="editingName"
                class="h-7 flex-1 text-sm"
                @keydown.enter="handleRename"
                @keydown.esc="editingId = null"
              />
              <Button variant="ghost" size="icon-xs" aria-label="保存" @click="handleRename">
                <CheckIcon />
              </Button>
              <Button variant="ghost" size="icon-xs" aria-label="取消" @click="editingId = null">
                <XIcon />
              </Button>
            </template>
            <template v-else>
              <span class="min-w-0 flex-1 truncate text-sm">{{ tag.name }}</span>
              <Button
                variant="ghost"
                size="icon-xs"
                aria-label="重命名"
                class="text-xs text-muted-foreground"
                @click="startEdit(tag)"
              >
                改名
              </Button>
              <Button
                variant="ghost"
                size="icon-xs"
                aria-label="删除标签"
                class="text-muted-foreground hover:text-destructive"
                @click="deleteTarget = tag"
              >
                <Trash2Icon class="h-3.5 w-3.5" />
              </Button>
            </template>
          </div>
        </div>
      </div>

      <ConfirmDialog
        :open="Boolean(deleteTarget)"
        destructive
        title="删除标签"
        :description="`确定删除标签「${deleteTarget?.name ?? ''}」？它会从所有文档上被移除。`"
        confirm-text="删除"
        @update:open="open => !open && (deleteTarget = null)"
        @confirm="handleDelete"
      />
    </DialogContent>
  </Dialog>
</template>
