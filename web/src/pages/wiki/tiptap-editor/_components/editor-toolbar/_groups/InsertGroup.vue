<script setup lang="ts">
import { CheckSquareIcon, Code2Icon, ImagePlusIcon, LinkIcon, Table2Icon } from "@lucide/vue";
import { ref } from "vue";

import { useToolbarContext } from "../_context/toolbar-context";
import ToolbarButton from "../_primitives/ToolbarButton.vue";

const { editor } = useToolbarContext();
const fileInputRef = ref<HTMLInputElement | null>(null);

const insertLink = () => {
  const href = window.prompt("输入链接地址");
  if (href) {
    editor.value?.chain().focus().setLink({ href }).run();
  }
};

const insertTable = () => {
  editor
    .value
    ?.chain()
    .focus()
    .insertTable({
      rows: 3,
      cols: 3,
      withHeaderRow: true,
    })
    .run();
};

const insertImage = () => {
  fileInputRef.value?.click();
};

const toggleCodeBlock = () => editor.value?.chain().focus().toggleCodeBlock().run();
const toggleTaskList = () => editor.value?.chain().focus().toggleTaskList().run();

const handleImageChange = (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) {
    return;
  }

  const reader = new FileReader();
  reader.onload = () => {
    const src = typeof reader.result === "string" ? reader.result : "";
    if (src) {
      editor.value?.chain().focus().setImage({ src }).run();
    }
  };
  reader.readAsDataURL(file);
  target.value = "";
};
</script>

<template>
  <ToolbarButton title="插入链接" :disabled="!editor" @click="insertLink">
    <LinkIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton title="插入表格" :disabled="!editor" @click="insertTable">
    <Table2Icon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton title="插入图片" :disabled="!editor" @click="insertImage">
    <ImagePlusIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton title="代码块" :disabled="!editor" @click="toggleCodeBlock">
    <Code2Icon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton title="任务列表" :disabled="!editor" @click="toggleTaskList">
    <CheckSquareIcon class="h-4 w-4" />
  </ToolbarButton>
  <input
    ref="fileInputRef"
    type="file"
    accept="image/*"
    class="hidden"
    @change="handleImageChange"
  />
</template>
