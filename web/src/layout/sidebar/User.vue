<script setup lang="ts">
import { CircleUserRoundIcon, EllipsisVerticalIcon, LogOutIcon } from "@lucide/vue";
import { computed } from "vue";
import { toast } from "vue-sonner";
import { RouterLink, useRouter } from "vue-router";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useSidebar } from "@/components/ui/sidebar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAuthStore } from "@/stores/auth";

const { isMobile, state } = useSidebar();
const isCollapsed = computed(() => state.value === "collapsed");
const router = useRouter();
const authStore = useAuthStore();

const name = computed(() => authStore.user?.username || authStore.user?.email || "未登录");
const subtitle = computed(() => authStore.user?.email || "知识库协作者");
const avatarUrl = computed(() => authStore.user?.avatar || "");

const fallback = computed(() => (name.value ? name.value.charAt(0).toUpperCase() : "U"));

const handleLogout = () => {
  authStore.clearAuth();
  toast.success("已退出登录");
  router.push("/login");
};
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <button
        type="button"
        :aria-label="isCollapsed ? name : undefined"
        class="bg-sidebar-accent ring-sidebar-border/60 hover:bg-sidebar-accent/80 focus-visible:ring-sidebar-ring flex items-center gap-3 rounded-lg p-2 ring-1 shadow-[0_4px_14px_rgb(15_23_42/0.06),0_1px_2px_rgb(15_23_42/0.04)] transition-colors focus-visible:ring-2 focus-visible:outline-none group-data-[collapsible=icon]:bg-transparent group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:p-0 group-data-[collapsible=icon]:shadow-none group-data-[collapsible=icon]:ring-0 group-data-[collapsible=icon]:hover:bg-transparent data-[state=open]:bg-sidebar-accent/80"
      >
        <Tooltip v-if="isCollapsed">
          <TooltipTrigger as-child>
            <span class="flex">
              <Avatar class="size-8 rounded-md">
                <AvatarImage :src="avatarUrl" :alt="name" />
                <AvatarFallback class="rounded-md">{{ fallback }}</AvatarFallback>
              </Avatar>
            </span>
          </TooltipTrigger>
          <TooltipContent side="right">{{ name }}</TooltipContent>
        </Tooltip>
        <Avatar v-else class="size-8 rounded-md">
          <AvatarImage :src="avatarUrl" :alt="name" />
          <AvatarFallback class="rounded-md">{{ fallback }}</AvatarFallback>
        </Avatar>
        <div class="flex flex-1 items-center gap-2 group-data-[collapsible=icon]:hidden">
          <div class="grid flex-1 text-left leading-tight">
            <span class="truncate text-sm font-medium">{{ name }}</span>
            <span class="text-muted-foreground truncate text-xs">{{ subtitle }}</span>
          </div>
          <EllipsisVerticalIcon class="text-muted-foreground size-4 shrink-0" />
        </div>
      </button>
    </DropdownMenuTrigger>
    <DropdownMenuContent
      class="w-(--reka-dropdown-menu-trigger-width) min-w-56 rounded-lg"
      :side="isMobile ? 'bottom' : 'right'"
      align="end"
      :side-offset="4"
    >
      <DropdownMenuLabel class="p-0 font-normal">
        <div class="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
          <Avatar class="size-8 rounded-md">
            <AvatarImage :src="avatarUrl" :alt="name" />
            <AvatarFallback class="rounded-md">{{ fallback }}</AvatarFallback>
          </Avatar>
          <div class="grid flex-1 text-left text-sm leading-tight">
            <span class="truncate font-medium">{{ name }}</span>
            <span class="text-muted-foreground truncate text-xs">{{ subtitle }}</span>
          </div>
        </div>
      </DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuGroup>
        <DropdownMenuItem as-child>
          <RouterLink to="/account">
            <CircleUserRoundIcon />
            个人设置
          </RouterLink>
        </DropdownMenuItem>
      </DropdownMenuGroup>
      <DropdownMenuSeparator />
      <DropdownMenuItem class="text-destructive" @click="handleLogout">
        <LogOutIcon />
        退出登录
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
