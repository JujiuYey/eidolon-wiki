<script setup lang="ts">
import { computed, ref } from "vue";

import { Input } from "@/components/ui/input";

const props = defineProps<{
  label: string;
  modelValue: number;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: number];
}>();

const labelClass = computed(() => "text-xs text-muted-foreground");

const handleInput = (value: string | number) => {
  emit("update:modelValue", Number(value) || 0);
};
</script>

<template>
  <div class="space-y-1">
    <label :class="labelClass">{{ props.label }} (px)</label>
    <Input
      type="number"
      min="0"
      class="h-7 text-xs"
      :model-value="props.modelValue"
      @update:model-value="handleInput"
    />
  </div>
</template>
