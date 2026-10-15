<script setup lang="ts">
import { EraserIcon, LockIcon, PaintbrushIcon } from "@lucide/vue";

import { useToolbarContext } from "../_context/toolbar-context";
import { useFormatPainter } from "../_hooks/useFormatPainter";
import ToolbarButton from "../_primitives/ToolbarButton.vue";

const { editor } = useToolbarContext();
const { formatPainter, handleSingleClick, handleDoubleClick } = useFormatPainter(editor);

const clearFormat = () => {
  editor.value?.chain().focus().clearNodes().unsetAllMarks().run();
};
</script>

<template>
  <ToolbarButton
    :is-active="formatPainter.isActive"
    :disabled="!editor"
    :title="
      formatPainter.isLocked
        ? '格式刷(已锁定,点击或按ESC取消)'
        : formatPainter.isActive
          ? '格式刷(已激活,点击应用或按ESC取消)'
          : '格式刷'
    "
    @click="handleSingleClick"
    @dblclick="handleDoubleClick"
  >
    <LockIcon v-if="formatPainter.isLocked" class="h-4 w-4" />
    <PaintbrushIcon v-else class="h-4 w-4" />
  </ToolbarButton>
  <ToolbarButton title="清除格式" :disabled="!editor" @click="clearFormat">
    <EraserIcon />
  </ToolbarButton>
</template>
