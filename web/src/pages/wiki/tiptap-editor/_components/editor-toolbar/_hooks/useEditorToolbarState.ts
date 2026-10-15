import type { Editor } from "@tiptap/core";
import { computed, ref, watch, type Ref } from "vue";

import type { ToolbarContextValue } from "../../../_shared/const";
import { DEFAULT_RANGE_STYLE } from "../../../_shared/const";
import { readRangeStyle } from "../../../_shared/range-style";

export function useEditorToolbarState(
  editor: Ref<Editor | null | undefined>,
  docTitle: Ref<string>,
): ToolbarContextValue {
  const styleVersion = ref(0);

  const rangeStyle = computed(() => {
    void styleVersion.value;
    return editor.value ? readRangeStyle(editor.value) : DEFAULT_RANGE_STYLE;
  });

  watch(
    editor,
    (nextEditor, _prevEditor, onCleanup) => {
      if (!nextEditor) {
        return;
      }

      const updateStyle = () => {
        styleVersion.value += 1;
      };

      nextEditor.on?.("selectionUpdate", updateStyle);
      nextEditor.on?.("transaction", updateStyle);

      onCleanup(() => {
        nextEditor.off?.("selectionUpdate", updateStyle);
        nextEditor.off?.("transaction", updateStyle);
      });
    },
    { immediate: true },
  );

  return {
    editor,
    rangeStyle,
    docTitle,
  };
}
