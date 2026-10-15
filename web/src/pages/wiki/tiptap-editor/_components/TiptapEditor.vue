<script setup lang="ts">
import type { Editor } from "@tiptap/core";
import CodeBlockLowlight from "@tiptap/extension-code-block-lowlight";
import Color from "@tiptap/extension-color";
import Highlight from "@tiptap/extension-highlight";
import Image from "@tiptap/extension-image";
import Link from "@tiptap/extension-link";
import Placeholder from "@tiptap/extension-placeholder";
import { Table, TableCell, TableHeader, TableRow } from "@tiptap/extension-table";
import TaskItem from "@tiptap/extension-task-item";
import TaskList from "@tiptap/extension-task-list";
import TextAlign from "@tiptap/extension-text-align";
import { TextStyle, TextStyleKit } from "@tiptap/extension-text-style";
import Typography from "@tiptap/extension-typography";
import Underline from "@tiptap/extension-underline";
import StarterKit from "@tiptap/starter-kit";
import { EditorContent, useEditor } from "@tiptap/vue-3";
import { common, createLowlight } from "lowlight";
import { watch } from "vue";
import { Markdown } from "tiptap-markdown";

const props = defineProps<{
  content: string;
}>();

const emit = defineEmits<{
  change: [content: string];
  editorReady: [editor: Editor | null | undefined];
}>();

const lowlight = createLowlight(common);

const editor = useEditor({
  extensions: [
    StarterKit.configure({
      codeBlock: false,
    }),
    Markdown,
    Underline,
    TextStyle,
    TextStyleKit,
    Color,
    TextAlign.configure({
      types: ["heading", "paragraph"],
    }),
    TaskList,
    TaskItem.configure({
      nested: true,
    }),
    Highlight.configure({
      multicolor: true,
    }),
    Typography,
    Image,
    Link.configure({
      openOnClick: false,
    }),
    Table.configure({
      resizable: true,
    }),
    TableRow,
    TableHeader,
    TableCell,
    CodeBlockLowlight.configure({
      lowlight,
    }),
    Placeholder.configure({
      placeholder: "开始输入内容，或按 / 插入块...",
    }),
  ],
  content: props.content,
  onUpdate: ({ editor: nextEditor }) => {
    const markdown = (nextEditor.storage as Record<string, any>).markdown?.getMarkdown?.();
    emit("change", markdown ?? nextEditor.getHTML());
  },
  editorProps: {
    attributes: {
      class: "min-h-[500px] focus:outline-none",
    },
  },
  injectCSS: true,
});

watch(
  editor,
  (nextEditor) => {
    emit("editorReady", nextEditor);
  },
  { immediate: true },
);

watch([() => props.content, editor], ([content, nextEditor]) => {
  if (!nextEditor) {
    return;
  }

  const currentMarkdown = (nextEditor.storage as Record<string, any>).markdown?.getMarkdown?.();
  if (currentMarkdown !== content) {
    nextEditor.commands.setContent(content, { emitUpdate: false });
  }
});
</script>

<template>
  <EditorContent v-if="editor" :editor="editor" />
</template>
