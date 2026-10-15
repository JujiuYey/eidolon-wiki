<script setup lang="ts">
import { ref, watch } from "vue";

import type { KnowledgeTag } from "@/api/wiki/tag";
import { tagBadgeStyle } from "@/api/wiki/tag";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

const props = defineProps<{
  open: boolean;
  tags: KnowledgeTag[];
  /** 选中的文档数（用于文案） */
  count: number;
  loading?: boolean;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
  /** 应用：携带所选标签 id 集合（覆盖所选文档的标签） */
  apply: [tagIds: string[]];
}>();

const selected = ref<Set<string>>(new Set());

watch(
  () => props.open,
  (open) => {
    if (open) selected.value = new Set();
  },
);

const toggle = (tagId: string, checked: boolean) => {
  const next = new Set(selected.value);
  if (checked) next.add(tagId);
  else next.delete(tagId);
  selected.value = next;
};

const handleApply = () => {
  emit("apply", [...selected.value]);
};
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>设置标签</DialogTitle>
        <DialogDescription>
          为选中的 {{ count }} 个文档设置标签集合（覆盖原有标签）。到「管理标签」里可以新建或删除标签。
        </DialogDescription>
      </DialogHeader>

      <div class="max-h-64 space-y-1 overflow-y-auto">
        <p v-if="tags.length === 0" class="py-6 text-center text-sm text-muted-foreground">
          该知识库还没有标签。
        </p>
        <label
          v-for="tag in tags"
          :key="tag.id"
          class="flex cursor-pointer items-center gap-2 rounded-md border px-3 py-2 hover:bg-muted/50"
        >
          <input
            type="checkbox"
            class="accent-[var(--primary)]"
            :checked="selected.has(tag.id)"
            @change="toggle(tag.id, ($event.target as HTMLInputElement).checked)"
          />
          <span
            class="size-2.5 shrink-0 rounded-full"
            :style="tagBadgeStyle(tag.color) ?? { backgroundColor: 'var(--muted-foreground)' }"
          />
          <span class="min-w-0 flex-1 truncate text-sm">{{ tag.name }}</span>
        </label>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)">取消</Button>
        <Button :disabled="loading" @click="handleApply">
          {{ loading ? "应用中..." : `应用到 ${count} 个文档` }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
