<script setup lang="ts">
import { MoonIcon, SunIcon } from "@lucide/vue";
import { computed } from "vue";

import { SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import { useAppStore } from "@/stores/app";

const appStore = useAppStore();
const isDark = computed(() => appStore.settings.theme === "dark");
</script>

<template>
  <SidebarMenuItem>
    <SidebarMenuButton
      :tooltip="isDark ? '切换到浅色' : '切换到深色'"
      @click="appStore.updateSettings({ theme: isDark ? 'light' : 'dark' })"
    >
      <MoonIcon v-if="isDark" />
      <SunIcon v-else />
      <span>{{ isDark ? "深色" : "浅色" }}</span>
    </SidebarMenuButton>
  </SidebarMenuItem>
</template>
