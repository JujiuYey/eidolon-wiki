<script setup lang="ts">
import type { Editor } from "@tiptap/core";
import { computed, toRef } from "vue";

import { TooltipProvider } from "@/components/ui/tooltip";

import { provideToolbarContext } from "./_context/toolbar-context";
import ColorGroup from "./_groups/ColorGroup.vue";
import FormatGroup from "./_groups/FormatGroup.vue";
import HistoryGroup from "./_groups/HistoryGroup.vue";
import InsertGroup from "./_groups/InsertGroup.vue";
import ParagraphGroup from "./_groups/ParagraphGroup.vue";
import SearchPrintGroup from "./_groups/SearchPrintGroup.vue";
import TextStyleGroup from "./_groups/TextStyleGroup.vue";
import { useEditorToolbarState } from "./_hooks/useEditorToolbarState";

const props = defineProps<{
  editor: Editor | null | undefined;
  docTitle: string;
  onSave: () => void | Promise<void>;
}>();

const contextValue = useEditorToolbarState(
  computed(() => props.editor),
  toRef(() => props.docTitle),
);

provideToolbarContext(contextValue);
</script>

<template>
  <TooltipProvider>
    <div class="flex flex-wrap justify-center items-center gap-2 border-b bg-background px-4 py-2">
      <HistoryGroup />
      <div class="h-6 w-px bg-border" />
      <FormatGroup />
      <div class="h-6 w-px bg-border" />
      <TextStyleGroup />
      <div class="h-6 w-px bg-border" />
      <ColorGroup />
      <div class="h-6 w-px bg-border" />
      <ParagraphGroup />
      <div class="h-6 w-px bg-border" />
      <InsertGroup />
      <div class="h-6 w-px bg-border" />
      <SearchPrintGroup />
    </div>
  </TooltipProvider>
</template>
