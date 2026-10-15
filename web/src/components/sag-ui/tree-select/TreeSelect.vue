<script setup lang="ts">
import { CheckIcon, ChevronDownIcon } from "@lucide/vue";
import { computed, ref } from "vue";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

import TreeSelectBranch from "./TreeSelectBranch.vue";
import type { TreeSelectItem, TreeSelectNodeLike } from "./types";

const props = withDefaults(
  defineProps<{
    items: TreeSelectItem[];
    value?: string;
    placeholder?: string;
    allowClear?: boolean;
    clearLabel?: string;
    disabled?: boolean;
    class?: string;
    size?: "sm" | "default";
    emptyMessage?: string;
  }>(),
  {
    placeholder: "选择...",
    allowClear: false,
    clearLabel: "清空选择",
    size: "default",
    emptyMessage: "暂无数据",
  },
);

const emit = defineEmits<{
  "update:value": [value: string, node: TreeSelectNodeLike | null];
}>();

const open = ref(false);
const expandedIds = ref<Set<string>>(new Set());

const selectedItem = computed(() => findItemByValue(props.items, props.value));

function findItemByValue(items: TreeSelectItem[], value?: string): TreeSelectItem | undefined {
  if (!value) return undefined;
  for (const item of items) {
    if (item.value === value) return item;
    if (item.children) {
      const found = findItemByValue(item.children, value);
      if (found) return found;
    }
  }
  return undefined;
}

const handleSelect = (selectedValue: string, node: TreeSelectNodeLike) => {
  emit("update:value", selectedValue, node);
  open.value = false;
};

const handleClear = () => {
  emit("update:value", "", null);
  open.value = false;
};

const toggleExpand = (id: string) => {
  const next = new Set(expandedIds.value);
  if (next.has(id)) {
    next.delete(id);
  } else {
    next.add(id);
  }
  expandedIds.value = next;
};
</script>

<template>
  <Popover
    :open="open"
    @update:open="nextOpen => {
      if (!disabled) open = nextOpen;
    }"
  >
    <PopoverTrigger as-child>
      <button
        type="button"
        role="combobox"
        :aria-expanded="open"
        :aria-disabled="disabled"
        :data-placeholder="selectedItem ? undefined : ''"
        :data-size="size"
        :disabled="disabled"
        :class="
          cn(
            'border-input data-[placeholder]:text-muted-foreground [&_svg:not([class*=\'text-\'])]:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive dark:bg-input/30 dark:hover:bg-input/50 flex w-fit items-center justify-between gap-2 rounded-md border bg-transparent px-3 py-2 text-sm whitespace-nowrap shadow-xs transition-[color,box-shadow] outline-none focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50 data-[size=default]:h-9 data-[size=sm]:h-8 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*=\'size-\'])]:size-4',
            'w-full',
            props.class,
          )
        "
      >
        <span class="line-clamp-1 flex items-center gap-2">
          {{ selectedItem?.label || placeholder }}
        </span>
        <ChevronDownIcon class="size-4 opacity-50" />
      </button>
    </PopoverTrigger>
    <PopoverContent align="start" :side-offset="4" class="max-h-80 overflow-y-auto p-1" style="width: var(--reka-popover-trigger-width)">
      <template v-if="allowClear">
        <div
          role="button"
          tabindex="0"
          :aria-pressed="!value"
          :class="
            cn(
              'relative flex items-center gap-1 rounded-sm py-1.5 pr-8 pl-2 text-sm cursor-pointer',
              'hover:bg-accent hover:text-accent-foreground',
              'outline-none focus-visible:bg-accent focus-visible:text-accent-foreground',
              !value && 'bg-accent text-accent-foreground',
            )
          "
          @click="handleClear"
          @keydown.enter.prevent="handleClear"
          @keydown.space.prevent="handleClear"
        >
          <span class="truncate">{{ clearLabel }}</span>
          <span v-if="!value" class="absolute right-2 flex size-3.5 items-center justify-center">
            <CheckIcon class="size-4" />
          </span>
        </div>
        <div v-if="items.length > 0" class="bg-border my-1 h-px" />
      </template>
      <div v-if="items.length === 0" class="py-6 text-center text-sm text-muted-foreground">
        {{ emptyMessage }}
      </div>
      <div v-else role="tree" :aria-label="placeholder">
        <TreeSelectBranch
          :items="items"
          :level="0"
          :selected-value="value"
          :expanded-ids="expandedIds"
          @select="handleSelect"
          @toggle-expand="toggleExpand"
        />
      </div>
    </PopoverContent>
  </Popover>
</template>
