<script setup lang="ts">
import { cn } from "@/lib/utils";

defineOptions({ name: "SagPage" });

const props = defineProps<{
  class?: string;
  bodyClass?: string;
}>();
</script>

<template>
  <div :class="cn('flex min-h-0 flex-1 flex-col overflow-hidden p-6', props.class)">
    <div class="mb-6 flex shrink-0 items-start justify-between gap-4">
      <div class="min-w-0">
        <h1 class="mb-2 flex items-center gap-2 text-2xl font-bold">
          <slot name="icon" />
          <slot name="title" />
        </h1>
        <p v-if="$slots.description" class="text-muted-foreground">
          <slot name="description" />
        </p>
      </div>
      <div v-if="$slots.actions" class="shrink-0">
        <slot name="actions" />
      </div>
    </div>

    <div :class="cn('min-h-0 flex-1 overflow-hidden', props.bodyClass)">
      <slot />
    </div>
  </div>
</template>
