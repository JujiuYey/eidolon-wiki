import type { Editor } from "@tiptap/core";
import { computed, ref, watch, type ComputedRef, type Ref } from "vue";

/**
 * 等价于 `@tiptap/react` 的 `useEditorState`（`@tiptap/vue-3` 未提供）：
 * 订阅编辑器事务/选区变化，在每次变化后重算 selector 的返回值。
 */
export function useEditorState<T>(
  editor: Ref<Editor | null | undefined>,
  selector: (ctx: { editor: Editor | null | undefined }) => T,
): ComputedRef<T> {
  const styleVersion = ref(0);

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

  return computed(() => {
    void styleVersion.value;
    return selector({ editor: editor.value });
  });
}
