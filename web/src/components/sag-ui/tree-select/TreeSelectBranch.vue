<script setup lang="ts" generic="T extends { id: string; children?: T[] }">
import { CheckIcon, ChevronDownIcon, ChevronRightIcon } from "@lucide/vue";
import { cn } from "@/lib/utils";

const props = withDefaults(
  defineProps<{
    items: Array<{
      value: string;
      label: string;
      node: T;
      children?: Array<{ value: string; label: string; node: T; children?: any[] }>;
    }>;
    level?: number;
    selectedValue?: string;
    expandedIds: Set<string>;
  }>(),
  {
    level: 0,
  },
);

const emit = defineEmits<{
  select: [value: string, node: T];
  "toggle-expand": [id: string];
}>();
</script>

<template>
  <template v-for="item in items" :key="item.value">
    <div
      role="button"
      tabindex="0"
      :aria-pressed="selectedValue === item.value"
      :class="
        cn(
          'relative flex items-center gap-1 py-1.5 pr-8 text-sm rounded-sm cursor-pointer',
          'hover:bg-accent hover:text-accent-foreground',
          'outline-none',
          'focus-visible:bg-accent focus-visible:text-accent-foreground',
          selectedValue === item.value && 'bg-accent text-accent-foreground',
        )
      "
      :style="{ paddingLeft: `${level * 16 + 8}px` }"
      @click="emit('select', item.value, item.node)"
      @keydown.enter.prevent="emit('select', item.value, item.node)"
      @keydown.space.prevent="emit('select', item.value, item.node)"
    >
      <button
        v-if="item.children && item.children.length > 0"
        type="button"
        class="p-0.5 hover:bg-accent rounded-sm"
        @click.stop="emit('toggle-expand', item.value)"
      >
        <ChevronDownIcon v-if="expandedIds.has(item.value)" class="size-3.5" />
        <ChevronRightIcon v-else class="size-3.5" />
      </button>
      <span v-else class="size-3.5" />
      <span class="truncate">{{ item.label }}</span>
      <span v-if="selectedValue === item.value" class="absolute right-2 flex size-3.5 items-center justify-center">
        <CheckIcon class="size-4" />
      </span>
    </div>
    <TreeSelectBranch
      v-if="item.children && item.children.length > 0 && expandedIds.has(item.value)"
      :items="item.children"
      :level="level + 1"
      :selected-value="selectedValue"
      :expanded-ids="expandedIds"
      @select="(value, node) => emit('select', value, node)"
      @toggle-expand="id => emit('toggle-expand', id)"
    />
  </template>
</template>
