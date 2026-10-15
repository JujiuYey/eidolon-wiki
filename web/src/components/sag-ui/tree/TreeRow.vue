<script setup lang="ts">
import { ChevronRightIcon } from "@lucide/vue";
import { computed } from "vue";

import { cn } from "@/lib/utils";

import type { TreeNode } from "./types";

// 递归行组件内部按 TreeNode 处理（递归泛型自引用会导致 vue-tsc 推断死循环，
// 泛型由外层 Tree.vue 承担并做断言转发）
const props = defineProps<{
  node: TreeNode;
  level: number;
  selectedId?: string;
  expandedIds: Set<string>;
  hasChildren?: (node: TreeNode) => boolean;
}>();

const emit = defineEmits<{
  select: [node: TreeNode];
  "toggle-expand": [node: TreeNode];
}>();

// 显式声明插槽类型，避免递归组件的隐式类型自引用
defineSlots<{
  node?: (props: { node: TreeNode; isExpanded: boolean; isSelected: boolean }) => any;
  default?: () => any;
}>();

const isExpanded = computed(() => props.expandedIds.has(props.node.id));
const isSelected = computed(() => props.selectedId === props.node.id);
const nodeHasChildren = computed(() =>
  props.hasChildren
    ? props.hasChildren(props.node)
    : Boolean(props.node.children && props.node.children.length > 0),
);
</script>

<template>
  <div>
    <div
      :class="
        cn(
          'group flex items-center gap-1 rounded-md px-2 py-1.5 cursor-pointer transition-colors',
          'hover:bg-muted/50',
          isSelected && 'bg-muted',
        )
      "
      :style="{ paddingLeft: `${level * 16 + 8}px` }"
      @click="emit('select', node)"
    >
      <button
        v-if="nodeHasChildren"
        :class="cn('p-0 border-none bg-transparent shrink-0 transition-transform', isExpanded && 'rotate-90')"
        @click.stop="emit('toggle-expand', node)"
      >
        <ChevronRightIcon class="size-4 text-muted-foreground" />
      </button>
      <span v-else class="size-4 shrink-0" />

      <div class="flex items-center gap-2 flex-1">
        <slot
          name="node"
          :node="node"
          :is-expanded="isExpanded"
          :is-selected="isSelected"
        />
      </div>
    </div>

    <template v-if="nodeHasChildren && isExpanded">
      <TreeRow
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :level="level + 1"
        :selected-id="selectedId"
        :expanded-ids="expandedIds"
        :has-children="hasChildren"
        @select="n => emit('select', n)"
        @toggle-expand="n => emit('toggle-expand', n)"
      >
        <template #node="{ node: childNode, isExpanded: childExpanded, isSelected: childSelected }">
          <slot name="node" :node="childNode" :is-expanded="childExpanded" :is-selected="childSelected" />
        </template>
      </TreeRow>
    </template>
  </div>
</template>
