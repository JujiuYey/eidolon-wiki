import { getCurrentScope, onScopeDispose, watch } from "vue";

import { applyTheme, type ThemeMode } from "@/lib/theme";
import { useAppStore } from "@/stores/app";

function resolveTheme(theme: "system" | ThemeMode): ThemeMode {
  if (theme !== "system") {
    return theme;
  }

  if (typeof window !== "undefined" && window.matchMedia("(prefers-color-scheme: dark)").matches) {
    return "dark";
  }

  return "light";
}

export function useThemeSync() {
  const appStore = useAppStore();

  let media: MediaQueryList | null = null;
  let syncTheme: () => void = () => {};

  watch(
    () => appStore.settings.theme,
    (theme) => {
      // 主题切换时重建 system 模式的监听
      if (media) media.removeEventListener("change", syncTheme);
      media = null;

      applyTheme(resolveTheme(theme));
      if (theme !== "system") return;

      media = window.matchMedia("(prefers-color-scheme: dark)");
      syncTheme = () => applyTheme(resolveTheme("system"));
      media.addEventListener("change", syncTheme);
    },
    { immediate: true },
  );

  if (getCurrentScope()) {
    onScopeDispose(() => media?.removeEventListener("change", syncTheme));
  }
}

export function initializeTheme() {
  applyTheme(resolveTheme(useAppStore().settings.theme));
}
