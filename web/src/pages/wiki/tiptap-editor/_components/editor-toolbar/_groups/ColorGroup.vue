<script setup lang="ts">
import { HighlighterIcon, TypeIcon } from "@lucide/vue";

import { HIGHLIGHT_COLORS, TEXT_COLORS } from "../../../_shared/const";
import { useToolbarContext } from "../_context/toolbar-context";
import ColorPickerPopover from "../_primitives/ColorPickerPopover.vue";
import ToolbarButton from "../_primitives/ToolbarButton.vue";

const { editor, rangeStyle } = useToolbarContext();

const setTextColor = (color: string) => {
  editor.value?.chain().focus().setColor(color).run();
};

const toggleHighlight = (color: string) => {
  editor.value?.chain().focus().toggleHighlight({ color }).run();
};
</script>

<template>
  <ColorPickerPopover
    :color="rangeStyle.textColor"
    :colors="TEXT_COLORS"
    @change="setTextColor"
  >
    <div>
      <ToolbarButton title="文字颜色" :disabled="!editor">
        <TypeIcon class="h-4 w-4" />
      </ToolbarButton>
    </div>
  </ColorPickerPopover>

  <ColorPickerPopover
    :color="rangeStyle.highlight"
    :colors="HIGHLIGHT_COLORS"
    @change="toggleHighlight"
  >
    <div>
      <ToolbarButton title="高亮颜色" :disabled="!editor">
        <HighlighterIcon class="h-4 w-4" />
      </ToolbarButton>
    </div>
  </ColorPickerPopover>
</template>
