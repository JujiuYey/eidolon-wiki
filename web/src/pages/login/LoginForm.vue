<script setup lang="ts">
import { EyeIcon, EyeOffIcon, Loader2Icon } from "@lucide/vue";
import { ref } from "vue";
import { toast } from "vue-sonner";
import { useRoute, useRouter } from "vue-router";

import { authApi } from "@/api/auth/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuthStore } from "@/stores/auth";

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

const EMAIL_PATTERN = /^[^@\s]+@[^@\s]+\.[^@\s]+$/;

const email = ref("");
const password = ref("");
const showPassword = ref(false);
const isLoading = ref(false);
const fieldErrors = ref<{ email?: string; password?: string }>({});

const handleLogin = async () => {
  if (isLoading.value) return;

  const nextErrors: { email?: string; password?: string } = {};
  if (!email.value.trim()) nextErrors.email = "请输入邮箱";
  else if (!EMAIL_PATTERN.test(email.value.trim())) nextErrors.email = "请输入有效的邮箱地址";
  if (!password.value) nextErrors.password = "请输入密码";
  fieldErrors.value = nextErrors;
  if (nextErrors.email || nextErrors.password) return;

  isLoading.value = true;
  try {
    const response = await authApi.login({ email: email.value.trim(), password: password.value });
    if (!response.success) throw new Error(response.message || "登录失败");

    const accessToken = response.token || response.access_token;
    if (!accessToken || !response.user) throw new Error("登录响应缺少会话信息");

    authStore.setTokens(accessToken, response.refresh_token || "");
    authStore.setSession({
      user: response.user,
      tenant: response.tenant ?? response.active_tenant ?? null,
      memberships: response.memberships ?? [],
      capabilities: response.capabilities ?? null,
    });
    toast.success("登录成功");
    // 带合法 redirect 参数时回跳原目标，否则落到知识问答（全员默认工作页）
    const redirect = route.query.redirect;
    const target = typeof redirect === "string" && redirect.startsWith("/") && !redirect.startsWith("//")
      ? redirect
      : "/rag-chat";
    router.push(target);
  } catch (error) {
    toast.error("登录失败", {
      description: error instanceof Error ? error.message : "请检查账号信息后重试",
    });
  } finally {
    isLoading.value = false;
  }
};
</script>

<template>
  <form novalidate class="mt-6 flex w-4/5 flex-col gap-4.5" @submit.prevent="handleLogin">
    <div class="flex flex-col gap-2">
      <Label for="login-email">邮箱</Label>
      <Input
        id="login-email"
        v-model="email"
        name="email"
        type="email"
        placeholder="团队邮箱"
        autocomplete="email"
        :aria-invalid="!!fieldErrors.email"
        :disabled="isLoading"
        class="h-10 border-transparent bg-muted/60"
      />
      <p v-if="fieldErrors.email" class="text-sm text-destructive">{{ fieldErrors.email }}</p>
    </div>

    <div class="flex flex-col gap-2">
      <Label for="login-password">密码</Label>
      <div class="relative">
        <Input
          id="login-password"
          v-model="password"
          name="password"
          :type="showPassword ? 'text' : 'password'"
          placeholder="输入密码"
          autocomplete="current-password"
          :aria-invalid="!!fieldErrors.password"
          :disabled="isLoading"
          class="h-10 border-transparent bg-muted/60 pr-12"
        />
        <button
          type="button"
          :aria-label="showPassword ? '隐藏密码' : '显示密码'"
          :aria-pressed="showPassword"
          :disabled="isLoading"
          class="absolute right-1.5 top-1/2 grid size-9 -translate-y-1/2 cursor-pointer place-items-center rounded-md text-muted-foreground transition-colors hover:text-foreground"
          @click="showPassword = !showPassword"
        >
          <EyeOffIcon v-if="showPassword" class="size-4.5" aria-hidden="true" />
          <EyeIcon v-else class="size-4.5" aria-hidden="true" />
        </button>
      </div>
      <p v-if="fieldErrors.password" class="text-sm text-destructive">
        {{ fieldErrors.password }}
      </p>
    </div>

    <Button type="submit" class="h-12 w-full font-medium" :disabled="isLoading">
      <Loader2Icon v-if="isLoading" class="animate-spin" aria-hidden="true" />
      <span>{{ isLoading ? "登录中…" : "登录" }}</span>
    </Button>
  </form>
</template>
