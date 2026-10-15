<script setup lang="ts">
import { onMounted } from "vue";

import { authApi } from "@/api/auth/auth";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { useAuthStore } from "@/stores/auth";

import AppSidebar from "./sidebar/AppSidebar.vue";

const authStore = useAuthStore();

// 如果有 token 但没有用户信息，则获取用户信息
onMounted(() => {
  if (authStore.accessToken && !authStore.user) {
    authApi
      .getCurrentUser()
      .then((response) => {
        if (response.success && response.data?.user) {
          authStore.setSession({
            user: response.data.user,
            tenant: response.data.tenant ?? null,
            memberships: response.data.memberships ?? [],
            capabilities: response.data.capabilities ?? null,
          });
        }
      })
      .catch(() => {
        // 获取用户信息失败，可能是 token 过期
        console.error("获取用户信息失败");
      });
  }
});
</script>

<template>
  <SidebarProvider :default-open="true">
    <AppSidebar />
    <SidebarInset>
      <RouterView />
    </SidebarInset>
  </SidebarProvider>
</template>
