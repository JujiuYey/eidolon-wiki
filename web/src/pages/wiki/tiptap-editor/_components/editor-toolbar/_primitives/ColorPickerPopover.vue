<script setup lang="ts">
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

const props = defineProps<{
  color: string | null;
  colors: string[];
}>();

const emit = defineEmits<{
  change: [color: string];
}>();
</script>

<template>
  <Popover>
    <PopoverTrigger as-child>
      <slot />
    </PopoverTrigger>
    <PopoverContent class="w-auto p-2">
      <div class="flex max-w-40 flex-wrap gap-1">
        <button
          v-for="value in props.colors"
          :key="value"
          type="button"
          :aria-label="value"
          :class="
            cn('h-6 w-6 rounded border border-border', props.color === value && 'ring-2 ring-primary')
          "
          :style="{ backgroundColor: value }"
          @click="emit('change', value)"
        />
      </div>
    </PopoverContent>
  </Popover>
</template>
