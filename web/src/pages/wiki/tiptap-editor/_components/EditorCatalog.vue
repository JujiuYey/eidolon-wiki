<script setup lang="ts">
import type { Editor } from "@tiptap/core";
import { ListTreeIcon } from "@lucide/vue";
import { ref, watch } from "vue";

import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";

interface CatalogItem {
  id: string;
  text: string;
  level: number;
  pos: number;
}

const props = defineProps<{
  open: boolean;
  editor: Editor | null | undefined;
}>();

const emit = defineEmits<{
  toggle: [];
}>();

const items = ref<CatalogItem[]>([]);

function getCatalogItems(editor: Editor | null | undefined): CatalogItem[] {
  const result: CatalogItem[] = [];
  let index = 0;

  editor?.state?.doc?.descendants?.((node, pos) => {
    if (node.type?.name === "heading") {
      result.push({
        id: `heading-${index}`,
        text: node.textContent || "无标题",
        level: Number(node.attrs?.level) || 1,
        pos,
      });
      index += 1;
    }
  });

  return result;
}

watch(
  () => props.editor,
  (editor, _prevEditor, onCleanup) => {
    if (!editor) {
      items.value = [];
      return;
    }

    const syncItems = () => {
      items.value = getCatalogItems(editor);
    };

    syncItems();
    editor.on?.("update", syncItems);
    editor.on?.("selectionUpdate", syncItems);

    onCleanup(() => {
      editor.off?.("update", syncItems);
      editor.off?.("selectionUpdate", syncItems);
    });
  },
  { immediate: true },
);

const jumpToHeading = (pos: number) => {
  props.editor?.chain?.().focus().setTextSelection(pos).run();
};
</script>

<template>
  <aside
    v-if="open"
    class="w-64 shrink-0 border-r bg-white/80 backdrop-blur supports-backdrop-filter:bg-white/65 h-full flex flex-col"
  >
    <div class="flex items-center justify-between border-b px-4 py-3">
      <div class="flex items-center gap-2 text-sm font-medium text-slate-700">
        <ListTreeIcon class="h-4 w-4" />
        目录
      </div>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        aria-label="切换目录"
        @click="emit('toggle')"
      >
        <ListTreeIcon class="h-3.5 w-3.5" />
      </Button>
    </div>

    <ScrollArea class="min-h-0 flex-1 overflow-auto px-2 py-3">
      <template v-if="items.length > 0">
        <button
          v-for="item in items"
          :key="item.id"
          type="button"
          class="flex w-full rounded-md px-2 py-1.5 text-left text-sm text-slate-700 transition hover:bg-slate-100"
          :style="{ paddingLeft: `${(item.level - 1) * 8}px` }"
          @click="jumpToHeading(item.pos)"
        >
          {{ item.text }}
        </button>
      </template>
      <p v-else class="px-2 text-xs text-slate-500">暂无目录</p>
    </ScrollArea>
  </aside>
</template>
