<script setup lang="ts">
import { FileDownIcon, PrinterIcon, SearchIcon } from "@lucide/vue";
import { computed, ref } from "vue";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";

import { exportToDocx } from "../../../_shared/export-docx";
import { useToolbarContext } from "../_context/toolbar-context";
import ToolbarButton from "../_primitives/ToolbarButton.vue";

function getSearchMatches(text: string, query: string) {
  if (!query) {
    return [];
  }

  const matches: { from: number; to: number }[] = [];
  let index = text.indexOf(query);

  while (index >= 0) {
    matches.push({
      from: index + 1,
      to: index + query.length + 1,
    });
    index = text.indexOf(query, index + query.length);
  }

  return matches;
}

const { editor, docTitle } = useToolbarContext();
const open = ref(false);
const query = ref("");
const replaceValue = ref("");
const activeIndex = ref(-1);

const matches = computed(() =>
  getSearchMatches(editor.value?.state?.doc?.textContent || "", query.value),
);

const jumpToMatch = (index: number) => {
  const match = matches.value[index];
  if (!match) {
    return;
  }

  editor.value?.chain().focus().setTextSelection({ from: match.from, to: match.to }).run();
};

const handleQueryInput = (value: string | number) => {
  query.value = String(value);
  activeIndex.value = -1;
};

const handleNext = () => {
  if (matches.value.length === 0) {
    return;
  }

  const nextIndex = activeIndex.value >= matches.value.length - 1 ? 0 : activeIndex.value + 1;
  activeIndex.value = nextIndex;
  jumpToMatch(nextIndex);
};

const handlePrevious = () => {
  if (matches.value.length === 0) {
    return;
  }

  const previousIndex =
    activeIndex.value <= 0 ? matches.value.length - 1 : activeIndex.value - 1;
  activeIndex.value = previousIndex;
  jumpToMatch(previousIndex);
};

const handleReplaceCurrent = () => {
  if (!replaceValue.value || activeIndex.value < 0) {
    return;
  }

  const match = matches.value[activeIndex.value];
  if (!match) {
    return;
  }

  editor
    .value
    ?.chain()
    .focus()
    .setTextSelection({ from: match.from, to: match.to })
    .insertContent(replaceValue.value)
    .run();
};

const print = () => window.print();

const downloadDocx = () => {
  if (editor.value) exportToDocx(editor.value, docTitle.value);
};
</script>

<template>
  <ToolbarButton title="搜索" :disabled="!editor" @click="open = true">
    <SearchIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton title="打印" @click="print">
    <PrinterIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton title="导出 Word" :disabled="!editor" @click="downloadDocx">
    <FileDownIcon class="h-4 w-4" />
  </ToolbarButton>

  <Dialog :open="open" @update:open="open = $event">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>搜索</DialogTitle>
        <DialogDescription>在当前文档中搜索文本并跳转到下一个匹配项。</DialogDescription>
      </DialogHeader>

      <div class="space-y-3">
        <Input
          aria-label="搜索内容"
          :model-value="query"
          placeholder="输入搜索内容..."
          @update:model-value="handleQueryInput"
        />
        <Input
          v-model="replaceValue"
          aria-label="替换内容"
          placeholder="输入替换内容..."
        />
        <div class="flex items-center gap-2">
          <Button type="button" variant="outline" @click="handlePrevious">
            上一个
          </Button>
          <Button type="button" @click="handleNext">
            下一个
          </Button>
          <Button type="button" variant="outline" @click="handleReplaceCurrent">
            替换当前
          </Button>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
