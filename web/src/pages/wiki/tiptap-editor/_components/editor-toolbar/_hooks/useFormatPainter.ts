import type { Editor } from "@tiptap/core";
import { reactive, watch, type Ref } from "vue";

import type { RangeStyle } from "../../../_shared/const";
import { applyRangeStyle, readRangeStyle } from "../../../_shared/range-style";

export interface FormatPainterState {
  isActive: boolean;
  isLocked: boolean;
  savedStyle: Partial<RangeStyle> | null;
}

const DEFAULT_FORMAT_PAINTER: FormatPainterState = {
  isActive: false,
  isLocked: false,
  savedStyle: null,
};

export function useFormatPainter(editor: Ref<Editor | null | undefined>) {
  const formatPainter = reactive<FormatPainterState>({ ...DEFAULT_FORMAT_PAINTER });

  const captureStyle = () => {
    if (!editor.value) {
      return;
    }

    formatPainter.savedStyle = readRangeStyle(editor.value);
  };

  const applyCapturedStyle = () => {
    if (!formatPainter.savedStyle) {
      return;
    }

    applyRangeStyle(editor.value, formatPainter.savedStyle);
  };

  const activateFormatPainter = () => {
    formatPainter.isActive = true;
  };

  const deactivateFormatPainter = () => {
    formatPainter.isActive = false;
    formatPainter.isLocked = false;
  };

  const toggleFormatPainterLock = () => {
    formatPainter.isLocked = !formatPainter.isLocked;
  };

  const handleEditorClick = () => {
    if (!formatPainter.isActive || !formatPainter.savedStyle) {
      return;
    }

    applyCapturedStyle();

    if (!formatPainter.isLocked) {
      deactivateFormatPainter();
    }
  };

  watch(
    () => [editor.value, formatPainter.isActive] as const,
    ([nextEditor, isActive], _prev, onCleanup) => {
      if (!nextEditor || !isActive) {
        return;
      }

      const editorElement = nextEditor.view?.dom;
      if (!editorElement) {
        return;
      }

      editorElement.addEventListener("click", handleEditorClick);

      onCleanup(() => {
        editorElement.removeEventListener("click", handleEditorClick);
      });
    },
    { immediate: true },
  );

  watch(
    () => formatPainter.isActive,
    (isActive, _prev, onCleanup) => {
      if (!isActive) {
        return;
      }

      const handleKeyDown = (event: KeyboardEvent) => {
        if (event.key === "Escape") {
          deactivateFormatPainter();
        }
      };

      document.addEventListener("keydown", handleKeyDown);

      onCleanup(() => {
        document.removeEventListener("keydown", handleKeyDown);
      });
    },
    { immediate: true },
  );

  const handleSingleClick = () => {
    if (formatPainter.isActive) {
      deactivateFormatPainter();
      return;
    }

    captureStyle();
    activateFormatPainter();
  };

  const handleDoubleClick = (event: MouseEvent) => {
    event.preventDefault();

    if (!formatPainter.isActive) {
      captureStyle();
      activateFormatPainter();
    }

    toggleFormatPainterLock();
  };

  return {
    formatPainter,
    handleSingleClick,
    handleDoubleClick,
  };
}
