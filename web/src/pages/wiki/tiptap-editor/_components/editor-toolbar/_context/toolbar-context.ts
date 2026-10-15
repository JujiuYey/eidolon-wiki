import { inject, provide, type InjectionKey } from "vue";

import type { ToolbarContextValue } from "../../../_shared/const";

const toolbarContextKey: InjectionKey<ToolbarContextValue> = Symbol("ToolbarContext");

export function provideToolbarContext(value: ToolbarContextValue) {
  provide(toolbarContextKey, value);
}

export function useToolbarContext() {
  const context = inject(toolbarContextKey);

  if (!context) {
    throw new Error("useToolbarContext must be used within ToolbarContext.Provider");
  }

  return context;
}
