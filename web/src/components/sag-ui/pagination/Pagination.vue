<script setup lang="ts">
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronsLeftIcon,
  ChevronsRightIcon,
} from "@lucide/vue";
import { computed } from "vue";

import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const props = withDefaults(
  defineProps<{
    page: number;
    pageSize: number;
    total: number;
    pageSizeOptions?: number[];
  }>(),
  {
    pageSizeOptions: () => [10, 20, 30, 40, 50],
  },
);

const emit = defineEmits<{
  "page-change": [page: number];
  "page-size-change": [pageSize: number];
}>();

const totalPages = computed(() => Math.ceil(props.total / props.pageSize));
</script>

<template>
  <div class="flex items-center justify-between px-4">
    <div class="text-muted-foreground hidden flex-1 text-sm lg:flex">共 {{ total }} 条</div>
    <div class="flex w-full items-center gap-8 lg:w-fit">
      <div class="hidden items-center gap-2 lg:flex">
        <Label for="rows-per-page" class="text-sm font-medium">每页条数</Label>
        <Select
          :model-value="`${pageSize}`"
          @update:model-value="(value) => emit('page-size-change', Number(value))"
        >
          <SelectTrigger size="sm" class="w-20" id="rows-per-page">
            <SelectValue :placeholder="`${pageSize}`" />
          </SelectTrigger>
          <SelectContent side="top">
            <SelectItem v-for="size in pageSizeOptions" :key="size" :value="`${size}`">
              {{ size }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="flex w-fit items-center justify-center text-sm font-medium">
        第 {{ page }} 页，共 {{ totalPages }} 页
      </div>
      <div class="ml-auto flex items-center gap-2 lg:ml-0">
        <Button
          variant="outline"
          class="hidden h-8 w-8 p-0 lg:flex"
          :disabled="page <= 1"
          aria-label="第一页"
          @click="emit('page-change', 1)"
        >
          <ChevronsLeftIcon />
        </Button>
        <Button
          variant="outline"
          size="icon"
          class="size-8"
          :disabled="page <= 1"
          aria-label="上一页"
          @click="emit('page-change', page - 1)"
        >
          <ChevronLeftIcon />
        </Button>
        <Button
          variant="outline"
          size="icon"
          class="size-8"
          :disabled="page >= totalPages"
          aria-label="下一页"
          @click="emit('page-change', page + 1)"
        >
          <ChevronRightIcon />
        </Button>
        <Button
          variant="outline"
          size="icon"
          class="hidden size-8 lg:flex"
          :disabled="page >= totalPages"
          aria-label="最后一页"
          @click="emit('page-change', totalPages)"
        >
          <ChevronsRightIcon />
        </Button>
      </div>
    </div>
  </div>
</template>
