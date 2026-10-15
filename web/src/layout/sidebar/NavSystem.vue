<script setup lang="ts">
import { computed } from "vue";
import { RouterLink, useRoute } from "vue-router";

import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { hasTenantRole, isSystemAdmin, useAuthStore } from "@/stores/auth";

import { adminNav, workNav, type NavItem } from "./menu";

const route = useRoute();
const authState = useAuthStore();

const activeClass =
  "bg-sidebar-accent text-sidebar-foreground font-medium shadow-[0_1px_3px_rgb(15_23_42_/_0.08),0_1px_2px_rgb(15_23_42_/_0.04)]";

// 仅做 UI 显隐，越权访问由后端 RBAC 拦截
function filterByAccess(items: NavItem[]): NavItem[] {
  return items.filter((item) => {
    if (item.access === "system-admin") return isSystemAdmin(authState);
    if (item.access === "tenant-admin") return hasTenantRole(authState, "admin");
    return true;
  });
}

const visibleAdminNav = computed(() => filterByAccess(adminNav));

const groups = computed<Array<{ label: string; items: NavItem[] }>>(() => {
  const list: Array<{ label: string; items: NavItem[] }> = [
    { label: "工作台", items: filterByAccess(workNav) },
  ];
  // 管理组对无权限成员整组隐藏
  if (visibleAdminNav.value.length > 0) {
    list.push({ label: "管理", items: visibleAdminNav.value });
  }
  return list;
});

function isActive(url: string) {
  return route.path === url || route.path.startsWith(`${url}/`);
}
</script>

<template>
  <SidebarGroup v-for="group in groups" :key="group.label">
    <SidebarGroupLabel>{{ group.label }}</SidebarGroupLabel>
    <SidebarGroupContent class="flex flex-col gap-2">
      <SidebarMenu>
        <SidebarMenuItem v-for="item in group.items" :key="item.title">
          <SidebarMenuButton :tooltip="item.title" as-child>
            <RouterLink :to="item.url" class="group/nav-link" :class="isActive(item.url) ? activeClass : ''">
              <component :is="item.icon" />
              <span>{{ item.title }}</span>
            </RouterLink>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarGroupContent>
  </SidebarGroup>
</template>
