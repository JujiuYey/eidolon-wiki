<script setup lang="ts">
import { Redo2Icon, Undo2Icon } from "@lucide/vue";

import { useToolbarContext } from "../_context/toolbar-context";
import ToolbarButton from "../_primitives/ToolbarButton.vue";

const { editor } = useToolbarContext();
</script>

<template>
  <ToolbarButton
    title="撤销"
    :disabled="!editor"
    @click="editor?.chain().focus().undo().run()"
  >
    <Undo2Icon class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton
    title="重做"
    :disabled="!editor"
    @click="editor?.chain().focus().redo().run()"
  >
    <Redo2Icon class="h-4 w-4" />
  </ToolbarButton>
</template>
