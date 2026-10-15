<script setup lang="ts" generic="T extends { id: string; children?: T[] }">
import { ref, watch } from "vue";

import { cn } from "@/lib/utils";

import TreeRow from "./TreeRow.vue";

const props = withDefaults(
  defineProps<{
    data: T[];
    selectedId?: string;
    hasChildren?: (node: T) => boolean;
    class?: string;
    rootLabel?: string;
    loading?: boolean;
    defaultExpandedAll?: boolean;
  }>(),
  {
    defaultExpandedAll: false,
  },
);

const emit = defineEmits<{
  select: [node: T | null];
}>();

// 递归收集所有节点 ID
function collectAllNodeIds(nodes: T[]): Set<string> {
  const ids = new Set<string>();
  const collect = (nodeList: T[]) => {
    for (const node of nodeList) {
      ids.add(node.id);
      if (node.children && node.children.length > 0) {
        collect(node.children as T[]);
      }
    }
  };
  collect(nodes);
  return ids;
}

const expandedIds = ref<Set<string>>(props.defaultExpandedAll ? collectAllNodeIds(props.data) : new Set<string>());

// 当 defaultExpandedAll 为 true 且 data 更新时，重新展开所有节点
watch(
  () => [props.defaultExpandedAll, props.data] as const,
  ([expandedAll, data]) => {
    if (expandedAll && data.length > 0) {
      expandedIds.value = collectAllNodeIds(data);
    }
  },
);

const handleToggleExpand = (node: T) => {
  const next = new Set(expandedIds.value);
  if (next.has(node.id)) {
    next.delete(node.id);
  } else {
    next.add(node.id);
  }
  expandedIds.value = next;
};
</script>

<template>
  <div :class="cn('flex flex-col p-2', props.class)">
    <div
      v-if="rootLabel"
      :class="
        cn(
          'group flex items-center gap-2 rounded-md px-2 py-1.5 cursor-pointer transition-colors',
          'hover:bg-muted/50',
          !selectedId && 'bg-muted',
        )
      "
      @click="emit('select', null)"
    >
      <slot name="rootIcon" />
      <span class="text-sm">{{ rootLabel }}</span>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-4">
      <span class="text-sm text-muted-foreground">加载中...</span>
    </div>
    <template v-else>
      <TreeRow
        v-for="node in data"
        :key="node.id"
        :node="node as any"
        :level="0"
        :selected-id="selectedId"
        :expanded-ids="expandedIds"
        :has-children="hasChildren as any"
        @select="n => emit('select', n as T)"
        @toggle-expand="handleToggleExpand as any"
      >
        <template #node="{ node: childNode, isExpanded: childExpanded, isSelected: childSelected }">
          <slot name="node" :node="childNode as T" :is-expanded="childExpanded" :is-selected="childSelected" />
        </template>
      </TreeRow>
    </template>
  </div>
</template>
