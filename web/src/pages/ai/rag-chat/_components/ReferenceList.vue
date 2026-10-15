<script setup lang="ts">
import { ChevronDownIcon, FileTextIcon } from "@lucide/vue";
import { computed, ref } from "vue";

import type { KnowledgeReference } from "@/api/ai/chat";
import { cn } from "@/lib/utils";

import CitationChunkDialog from "./CitationChunkDialog.vue";

const props = defineProps<{
  references: KnowledgeReference[];
}>();

interface ReferenceGroup {
  key: string;
  title: string;
  items: KnowledgeReference[];
}

const expandedKey = ref<string | null>(null);
const activeReference = ref<KnowledgeReference | null>(null);

// 回答下方的来源卡片：按文档分组，展开看各分块预览，点击看全文
const groups = computed<ReferenceGroup[]>(() => {
  const map = new Map<string, ReferenceGroup>();
  for (const ref of props.references) {
    const key = ref.knowledge_id || ref.id;
    const title = ref.knowledge_title || ref.knowledge_filename || "未知文档";
    const group = map.get(key) ?? { key, title, items: [] };
    group.items.push(ref);
    map.set(key, group);
  }
  return [...map.values()];
});
</script>

<template>
  <div v-if="props.references.length > 0" class="mt-3 border-t border-border/50 pt-3">
    <p class="mb-2 text-xs font-medium text-muted-foreground">
      参考来源（{{ props.references.length }} 个分块 / {{ groups.length }} 个文档）
    </p>
    <div class="space-y-1.5">
      <div v-for="(group, groupIndex) in groups" :key="group.key" class="rounded-md border bg-background/60">
        <button
          type="button"
          class="flex w-full items-center gap-2 px-2.5 py-1.5 text-left text-xs"
          @click="expandedKey = expandedKey === group.key ? null : group.key"
        >
          <FileTextIcon class="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
          <span class="min-w-0 flex-1 truncate font-medium">{{ group.title }}</span>
          <span class="shrink-0 text-muted-foreground">{{ group.items.length }} 块</span>
          <ChevronDownIcon
            :class="cn('h-3.5 w-3.5 shrink-0 transition-transform', expandedKey === group.key && 'rotate-180')"
          />
        </button>
        <div v-if="expandedKey === group.key" class="space-y-1.5 border-t px-2.5 py-2">
          <button
            v-for="ref in group.items"
            :key="ref.id"
            type="button"
            class="block w-full rounded bg-muted/50 p-2 text-left text-xs hover:bg-muted"
            @click="activeReference = ref"
          >
            <span class="mb-1 flex items-center gap-2 text-muted-foreground">
              <span>{{ "#" + (groupIndex + 1) + (ref.chunk_index != null ? ` · chunk ${ref.chunk_index}` : "") }}</span>
              <span v-if="ref.score">得分 {{ ref.score.toFixed(3) }}</span>
              <span v-if="ref.match_type">{{ ref.match_type }}</span>
            </span>
            <span class="line-clamp-2 text-foreground/80">{{ ref.content }}</span>
          </button>
        </div>
      </div>
    </div>

    <CitationChunkDialog :reference="activeReference" @close="activeReference = null" />
  </div>
</template>
