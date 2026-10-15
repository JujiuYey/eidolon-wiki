<script setup lang="ts">
import { cn } from "@/lib/utils";

const props = defineProps<{
  ariaLabel: string;
  class?: string;
  options: { value: string; label: string }[];
  modelValue: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const handleChange = (event: Event) => {
  emit("update:modelValue", (event.target as HTMLSelectElement).value);
};
</script>

<template>
  <select
    :aria-label="props.ariaLabel"
    :class="
      cn(
        'h-8 rounded-md border border-input bg-background px-2 text-sm shadow-xs outline-none',
        props.class,
      )
    "
    :value="props.modelValue"
    @change="handleChange"
  >
    <option v-for="option in props.options" :key="option.value" :value="option.value">
      {{ option.label }}
    </option>
  </select>
</template>
