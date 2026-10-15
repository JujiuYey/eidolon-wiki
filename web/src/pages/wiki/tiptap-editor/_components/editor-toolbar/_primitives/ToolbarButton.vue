<script setup lang="ts">
import { computed } from "vue";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

const props = withDefaults(
  defineProps<{
    title: string;
    onClick?: () => void;
    disabled?: boolean;
    active?: boolean;
    isActive?: boolean;
  }>(),
  {},
);

const emit = defineEmits<{
  dblclick: [event: MouseEvent];
}>();

const buttonActive = computed(() => props.active ?? props.isActive ?? false);

const handleClick = () => {
  props.onClick?.();
};

const handleDoubleClick = (event: MouseEvent) => {
  emit("dblclick", event);
};
</script>

<template>
  <Tooltip>
    <TooltipTrigger as-child>
      <Button
        type="button"
        :variant="buttonActive ? 'secondary' : 'ghost'"
        size="icon-sm"
        :title="title"
        :aria-label="title"
        :data-active="String(buttonActive)"
        :disabled="disabled"
        @click="handleClick"
        @dblclick="handleDoubleClick"
      >
        <slot />
      </Button>
    </TooltipTrigger>
    <TooltipContent>
      <p>{{ title }}</p>
    </TooltipContent>
  </Tooltip>
</template>
