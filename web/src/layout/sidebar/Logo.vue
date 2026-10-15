<script setup lang="ts">
import { PanelLeftCloseIcon } from "@lucide/vue";

import logo from "@/assets/logo.svg";
import { Button } from "@/components/ui/button";
import { SidebarHeader, useSidebar } from "@/components/ui/sidebar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

const { state, toggleSidebar } = useSidebar();
const isCollapsed = state.value === "collapsed";
</script>

<template>
  <SidebarHeader>
    <div
      class="bg-sidebar-accent ring-sidebar-border/60 flex items-center gap-3 rounded-lg p-2 ring-1 shadow-[0_4px_14px_rgb(15_23_42/0.06),0_1px_2px_rgb(15_23_42/0.04)] group-data-[collapsible=icon]:bg-transparent group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:p-0 group-data-[collapsible=icon]:shadow-none group-data-[collapsible=icon]:ring-0"
    >
      <Tooltip v-if="isCollapsed">
        <TooltipTrigger as-child>
          <button
            type="button"
            aria-label="展开侧边栏"
            class="focus-visible:ring-sidebar-ring rounded-md focus-visible:ring-2 focus-visible:outline-none"
            @click="toggleSidebar"
          >
            <img :src="logo" alt="SAG Wiki" class="size-9 rounded-md object-contain" />
          </button>
        </TooltipTrigger>
        <TooltipContent side="right">展开侧边栏</TooltipContent>
      </Tooltip>
      <img v-else :src="logo" alt="SAG Wiki" class="size-9 shrink-0 rounded-md object-contain" />
      <div class="flex flex-1 items-center gap-2 group-data-[collapsible=icon]:hidden">
        <div class="grid flex-1 text-left leading-tight">
          <span class="truncate text-[15px] font-semibold tracking-tight">SAG Wiki</span>
          <span class="text-muted-foreground truncate text-xs">智能知识运营中枢</span>
        </div>
        <Button
          variant="ghost"
          size="icon"
          class="text-muted-foreground hover:text-foreground size-7 shrink-0"
          aria-label="折叠侧边栏"
          @click="toggleSidebar"
        >
          <PanelLeftCloseIcon class="size-4" />
        </Button>
      </div>
    </div>
  </SidebarHeader>
</template>
