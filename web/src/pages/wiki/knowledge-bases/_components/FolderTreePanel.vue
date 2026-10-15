<script setup lang="ts">
import { FolderIcon, FolderOpenIcon, SquarePenIcon } from "@lucide/vue";
import { computed, ref } from "vue";

import type { KnowledgeFolderNode } from "@/api/wiki/knowledge";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

interface FolderTreePanelProps {
  tree: { root_document_count: number; total_document_count: number; folders: KnowledgeFolderNode[] } | undefined;
  loading: boolean;
  /** 当前选中的 folder_path；undefined = 全部文档 */
  selectedPath: string | undefined;
}

const props = defineProps<FolderTreePanelProps>();

const emit = defineEmits<{
  select: [path: string | undefined];
  /** 请求打开文件夹重命名/移动对话框（携带完整路径） */
  rename: [path: string];
}>();

const keyword = ref("");

function flattenFolders(folders: KnowledgeFolderNode[], depth = 0): Array<{ node: KnowledgeFolderNode; depth: number }> {
  const rows: Array<{ node: KnowledgeFolderNode; depth: number }> = [];
  for (const folder of folders) {
    rows.push({ node: folder, depth });
    if (folder.children?.length) {
      rows.push(...flattenFolders(folder.children, depth + 1));
    }
  }
  return rows;
}

const rows = computed(() => {
  const all = flattenFolders(props.tree?.folders ?? []);
  const kw = keyword.value.trim().toLowerCase();
  if (!kw) return all;
  return all.filter((row) => row.node.path.toLowerCase().includes(kw));
});
</script>

<template>
  <div class="flex h-full min-h-0 flex-col gap-2 rounded-lg border p-2">
    <Input v-model="keyword" class="h-8 text-xs" placeholder="搜索文件夹" />
    <div class="min-h-0 flex-1 overflow-y-auto">
      <button
        type="button"
        :class="
          cn(
            'group flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-muted/50',
            selectedPath === undefined && 'bg-muted font-medium',
          )
        "
        @click="emit('select', undefined)"
      >
        <FolderOpenIcon class="h-4 w-4 shrink-0 text-muted-foreground" />
        <span class="flex-1 truncate">全部文档</span>
        <span class="text-xs text-muted-foreground">{{ tree?.total_document_count ?? 0 }}</span>
      </button>

      <div v-if="loading" class="space-y-2 p-2">
        <Skeleton class="h-6 w-full" />
        <Skeleton class="h-6 w-4/5" />
        <Skeleton class="h-6 w-3/5" />
      </div>
      <template v-else>
        <div
          v-for="{ node, depth } in rows"
          :key="node.path"
          type="button"
          :class="
            cn(
              'group flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-muted/50',
              selectedPath === node.path && 'bg-muted font-medium',
            )
          "
          :style="{ paddingLeft: `${depth * 14 + 8}px` }"
          :title="node.path"
          role="button"
          tabindex="0"
          @click="emit('select', node.path)"
          @keydown.enter="emit('select', node.path)"
        >
          <FolderIcon class="h-4 w-4 shrink-0 text-muted-foreground" />
          <span class="min-w-0 flex-1 truncate">{{ node.name }}</span>
          <span class="text-xs text-muted-foreground group-hover:hidden">{{ node.total_count }}</span>
          <button
            type="button"
            class="hidden shrink-0 rounded p-0.5 text-muted-foreground hover:text-foreground group-hover:block"
            :aria-label="`重命名文件夹 ${node.name}`"
            @click.stop="emit('rename', node.path)"
          >
            <SquarePenIcon class="h-3.5 w-3.5" />
          </button>
        </div>
      </template>

      <p v-if="!loading && rows.length === 0 && keyword" class="p-2 text-xs text-muted-foreground">
        没有匹配的文件夹
      </p>
    </div>
  </div>
</template>
