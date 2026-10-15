<script setup lang="ts">
import { Sidebar, SidebarFooter, SidebarMenu } from "@/components/ui/sidebar";

import Logo from "./Logo.vue";
import NavSystem from "./NavSystem.vue";
import ThemeToggle from "./ThemeToggle.vue";
import UserMenu from "./User.vue";
</script>

<template>
  <Sidebar collapsible="icon">
    <Logo />
    <NavSystem />
    <SidebarFooter class="mt-auto">
      <SidebarMenu>
        <ThemeToggle />
      </SidebarMenu>
      <UserMenu />
    </SidebarFooter>
  </Sidebar>
</template>
