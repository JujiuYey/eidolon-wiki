<script setup lang="ts">
import { BoldIcon, CodeIcon, ItalicIcon, StrikethroughIcon, UnderlineIcon } from "@lucide/vue";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import { FONT_FAMILIES, FONT_SIZES } from "../../../_shared/const";
import { useToolbarContext } from "../_context/toolbar-context";
import { useEditorState } from "../_hooks/useEditorState";
import ToolbarButton from "../_primitives/ToolbarButton.vue";

const { editor } = useToolbarContext();

const editorState = useEditorState(editor, (ctx) => {
  const currentFont = FONT_FAMILIES.find((f) =>
    ctx.editor?.isActive("textStyle", { fontFamily: f.value }),
  );
  const currentSize = FONT_SIZES.find((s) =>
    ctx.editor?.isActive("textStyle", { fontSize: `${s.value}px` }),
  );
  const bold = ctx.editor?.isActive("bold") ?? false;
  const italic = ctx.editor?.isActive("italic") ?? false;
  const underline = ctx.editor?.isActive("underline") ?? false;
  const strikethrough = ctx.editor?.isActive("strike") ?? false;
  const inlineCode = ctx.editor?.isActive("code") ?? false;
  return {
    currentFont: currentFont?.value ?? "Microsoft YaHei",
    currentSize: currentSize?.value ?? "16",
    bold,
    italic,
    underline,
    strikethrough,
    inlineCode: inlineCode,
  };
});

const setFontFamily = (value: string) => {
  editor.value?.chain().focus().setFontFamily(value).run();
};

const setFontSize = (value: string) => {
  editor.value?.chain().focus().setFontSize(`${value}px`).run();
};

const increaseFontSize = () => {
  const idx = FONT_SIZES.findIndex((s) => s.value === editorState.value.currentSize);
  const prev = FONT_SIZES[idx - 1];
  if (prev) editor.value?.chain().focus().setFontSize(`${prev.value}px`).run();
};

const decreaseFontSize = () => {
  const idx = FONT_SIZES.findIndex((s) => s.value === editorState.value.currentSize);
  const next = FONT_SIZES[idx + 1];
  if (next) editor.value?.chain().focus().setFontSize(`${next.value}px`).run();
};

const toggleBold = () => editor.value?.chain().focus().toggleBold().run();
const toggleItalic = () => editor.value?.chain().focus().toggleItalic().run();
const toggleUnderline = () => editor.value?.chain().focus().toggleUnderline().run();
const toggleStrike = () => editor.value?.chain().focus().toggleStrike().run();
const toggleCode = () => editor.value?.chain().focus().toggleCode().run();
</script>

<template>
  <Select
    :model-value="editorState.currentFont"
    :disabled="!editor"
    @update:model-value="setFontFamily($event as string)"
  >
    <SelectTrigger class="w-20 text-xs border-none bg-transparent shadow-none focus:ring-0 px-1">
      <SelectValue />
    </SelectTrigger>
    <SelectContent>
      <SelectItem
        v-for="f in FONT_FAMILIES"
        :key="f.value"
        :value="f.value"
        :style="{ fontFamily: f.value }"
      >
        {{ f.label }}
      </SelectItem>
    </SelectContent>
  </Select>

  <Select
    :model-value="editorState.currentSize"
    :disabled="!editor"
    @update:model-value="setFontSize($event as string)"
  >
    <SelectTrigger class="w-16 text-xs border-none bg-transparent shadow-none focus:ring-0 px-1">
      <SelectValue />
    </SelectTrigger>
    <SelectContent>
      <SelectItem v-for="s in FONT_SIZES" :key="s.value" :value="s.value">
        {{ s.label }}
      </SelectItem>
    </SelectContent>
  </Select>

  <ToolbarButton title="增大字号" :disabled="!editor" @click="increaseFontSize">
    <span class="text-xs">
      A<sup>+</sup>
    </span>
  </ToolbarButton>
  <ToolbarButton title="减小字号" :disabled="!editor" @click="decreaseFontSize">
    <span class="text-xs">
      A<sup>-</sup>
    </span>
  </ToolbarButton>

  <ToolbarButton title="粗体" :is-active="editorState.bold" :disabled="!editor" @click="toggleBold">
    <BoldIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton title="斜体" :is-active="editorState.italic" :disabled="!editor" @click="toggleItalic">
    <ItalicIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton
    title="下划线"
    :is-active="editorState.underline"
    :disabled="!editor"
    @click="toggleUnderline"
  >
    <UnderlineIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton
    title="删除线"
    :is-active="editorState.strikethrough"
    :disabled="!editor"
    @click="toggleStrike"
  >
    <StrikethroughIcon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton
    title="行内代码"
    :is-active="editorState.inlineCode"
    :disabled="!editor"
    @click="toggleCode"
  >
    <CodeIcon class="h-4 w-4" />
  </ToolbarButton>
</template>
