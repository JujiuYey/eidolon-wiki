import { defineStore } from "pinia";

export interface AppSettings {
  theme: "system" | "light" | "dark";
}

interface AppState {
  settings: AppSettings;
}

const defaultSettings: AppSettings = {
  theme: "system",
};

export const useAppStore = defineStore("app", {
  state: (): AppState => ({
    settings: { ...defaultSettings },
  }),
  actions: {
    updateSettings(partial: Partial<AppSettings>) {
      this.settings = { ...this.settings, ...partial };
    },
    resetSettings() {
      this.settings = { ...defaultSettings };
    },
  },
  persist: {
    key: "app-setting",
    pick: ["settings"],
  },
});
