<script setup lang="ts">
import type { Editor } from "@tiptap/core";
import {
  AlignCenterIcon,
  AlignJustifyIcon,
  AlignLeftIcon,
  AlignRightIcon,
  ListIcon,
  ListOrderedIcon,
} from "@lucide/vue";

import { useToolbarContext } from "../_context/toolbar-context";
import { useEditorState } from "../_hooks/useEditorState";
import ToolbarButton from "../_primitives/ToolbarButton.vue";
import ToolbarSelect from "../_primitives/ToolbarSelect.vue";

const HEADING_OPTIONS = [
  { value: "paragraph", label: "正文" },
  { value: "h1", label: "标题 1" },
  { value: "h2", label: "标题 2" },
  { value: "h3", label: "标题 3" },
  { value: "h4", label: "标题 4" },
];

const HEADING_LEVELS = {
  h1: 1,
  h2: 2,
  h3: 3,
  h4: 4,
} as const;

function getCurrentHeading(editor: Editor | null | undefined) {
  if (editor?.isActive?.("heading", { level: 1 })) return "h1";
  if (editor?.isActive?.("heading", { level: 2 })) return "h2";
  if (editor?.isActive?.("heading", { level: 3 })) return "h3";
  if (editor?.isActive?.("heading", { level: 4 })) return "h4";
  return "paragraph";
}

const { editor } = useToolbarContext();

const state = useEditorState(editor, (ctx) => ({
  currentHeading: getCurrentHeading(ctx.editor),
  alignLeft: !!ctx.editor?.isActive?.({ textAlign: "left" }),
  alignCenter: !!ctx.editor?.isActive?.({ textAlign: "center" }),
  alignRight: !!ctx.editor?.isActive?.({ textAlign: "right" }),
  alignJustify: !!ctx.editor?.isActive?.({ textAlign: "justify" }),
  bulletList: !!ctx.editor?.isActive?.("bulletList"),
  orderedList: !!ctx.editor?.isActive?.("orderedList"),
}));

const handleHeadingChange = (value: string) => {
  if (!editor.value) {
    return;
  }

  if (value === "paragraph") {
    editor.value.chain().focus().setParagraph().run();
    return;
  }

  editor.value
    .chain()
    .focus()
    .toggleHeading({ level: HEADING_LEVELS[value as keyof typeof HEADING_LEVELS] })
    .run();
};

const setTextAlign = (alignment: "left" | "center" | "right" | "justify") => {
  editor.value?.chain().focus().setTextAlign(alignment).run();
};

const toggleBulletList = () => editor.value?.chain().focus().toggleBulletList().run();
const toggleOrderedList = () => editor.value?.chain().focus().toggleOrderedList().run();
</script>

<template>
  <ToolbarSelect
    ariaLabel="段落样式"
    :model-value="state.currentHeading"
    :options="HEADING_OPTIONS"
    @update:model-value="handleHeadingChange($event as string)"
  />
  <ToolbarButton
    title="左对齐"
    :is-active="state.alignLeft"
    :disabled="!editor"
    @click="setTextAlign('left')"
  >
    <AlignLeftIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton
    title="居中对齐"
    :is-active="state.alignCenter"
    :disabled="!editor"
    @click="setTextAlign('center')"
  >
    <AlignCenterIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton
    title="右对齐"
    :is-active="state.alignRight"
    :disabled="!editor"
    @click="setTextAlign('right')"
  >
    <AlignRightIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton
    title="两端对齐"
    :is-active="state.alignJustify"
    :disabled="!editor"
    @click="setTextAlign('justify')"
  >
    <AlignJustifyIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton
    title="无序列表"
    :is-active="state.bulletList"
    :disabled="!editor"
    @click="toggleBulletList"
  >
    <ListIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton
    title="有序列表"
    :is-active="state.orderedList"
    :disabled="!editor"
    @click="toggleOrderedList"
  >
    <ListOrderedIcon class="h-4 w-4" />
  </ToolbarButton>
</template>
