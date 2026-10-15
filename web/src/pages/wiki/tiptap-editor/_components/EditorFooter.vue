<script setup lang="ts">
import { CheckIcon, Loader2Icon, RulerIcon, SaveIcon, ZoomInIcon, ZoomOutIcon } from "@lucide/vue";
import { computed, ref } from "vue";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

import type { EditorPageSetup } from "../_hooks/useEditorPageSetup";

import MarginInput from "./MarginInput.vue";

const props = defineProps<{
  saveStatus: "saved" | "saving" | "unsaved";
  wordCount: number;
  onSave: () => void | Promise<void>;
  pageSetup: EditorPageSetup;
}>();

const marginOpen = ref(false);
const zoomLabel = computed(() => `${Math.round(props.pageSetup.zoom * 100)}%`);

const statusClass = computed(() =>
  cn(
    "flex items-center gap-1 text-xs",
    props.saveStatus === "saved" && "text-green-600",
    props.saveStatus === "saving" && "text-amber-600",
    props.saveStatus === "unsaved" && "text-orange-600",
  ),
);

const handleSave = () => {
  void props.onSave();
};
</script>

<template>
  <div class="flex items-center justify-between border-t bg-background px-4 py-2 text-sm">
    <span class="text-muted-foreground">
      字数:
      {{ wordCount }}
    </span>

    <div class="flex items-center gap-2">
      <Button
        type="button"
        variant="ghost"
        size="icon"
        class="h-7 w-7"
        title="缩小"
        @click="props.pageSetup.zoomOut"
      >
        <ZoomOutIcon class="h-4 w-4" />
      </Button>

      <Button
        type="button"
        variant="ghost"
        size="sm"
        class="h-7 min-w-[3rem] text-xs"
        title="重置缩放"
        @click="props.pageSetup.zoomReset"
      >
        {{ zoomLabel }}
      </Button>

      <Button
        type="button"
        variant="ghost"
        size="icon"
        class="h-7 w-7"
        title="放大"
        @click="props.pageSetup.zoomIn"
      >
        <ZoomInIcon class="h-4 w-4" />
      </Button>
      <Popover :open="marginOpen" @update:open="marginOpen = $event">
        <PopoverTrigger as-child>
          <Button type="button" variant="ghost" size="sm" class="h-7 text-xs" title="页边距">
            <RulerIcon class="mr-1 h-3 w-3" />
            边距
          </Button>
        </PopoverTrigger>
        <PopoverContent class="w-64" align="end">
          <div class="grid grid-cols-2 gap-3">
            <MarginInput
              label="上"
              :model-value="props.pageSetup.margins.top"
              @update:model-value="(v) => props.pageSetup.setMargins({ top: v })"
            />
            <MarginInput
              label="下"
              :model-value="props.pageSetup.margins.bottom"
              @update:model-value="(v) => props.pageSetup.setMargins({ bottom: v })"
            />
            <MarginInput
              label="左"
              :model-value="props.pageSetup.margins.left"
              @update:model-value="(v) => props.pageSetup.setMargins({ left: v })"
            />
            <MarginInput
              label="右"
              :model-value="props.pageSetup.margins.right"
              @update:model-value="(v) => props.pageSetup.setMargins({ right: v })"
            />
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="mt-3 w-full text-xs"
            @click="props.pageSetup.resetMargins"
          >
            重置边距
          </Button>
        </PopoverContent>
      </Popover>

      <span :class="statusClass">
        <template v-if="props.saveStatus === 'saved'">
          <CheckIcon class="h-3 w-3" />
          已保存
        </template>
        <template v-else-if="props.saveStatus === 'saving'">
          <Loader2Icon class="h-3 w-3 animate-spin" />
          保存中...
        </template>
        <template v-else-if="props.saveStatus === 'unsaved'">未保存</template>
      </span>

      <Button type="button" size="sm" variant="outline" @click="handleSave">
        <SaveIcon class="h-3 w-3" />
        保存
      </Button>
    </div>
  </div>
</template>
