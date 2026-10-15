<script setup lang="ts">
import { EyeIcon, EyeOffIcon, TriangleAlertIcon } from "@lucide/vue";
import { computed, ref } from "vue";
import { z } from "zod";
import { toast } from "vue-sonner";

import { useAuthConfig, useResetUserPassword } from "@/api/system/admin-user";
import { buildPasswordSchema, passwordHintText } from "@/utils/password-policy";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuthStore } from "@/stores/auth";

const authStore = useAuthStore();
const resetPassword = useResetUserPassword();
const { isPending } = resetPassword;
const authConfig = useAuthConfig();
// 配置拉取失败时退回简单策略，最终以后端校验为准
const complexPassword = computed(() => authConfig.data.value?.complex_password_enabled === true);

// 未引入表单库：本地状态 + 提交时 zod 校验（校验失败 toast）
const email = ref("");
const newPassword = ref("");
const confirmPassword = ref("");
const showPassword = ref(false);

const resetForm = () => {
  email.value = "";
  newPassword.value = "";
  confirmPassword.value = "";
  showPassword.value = false;
};

const handleSubmit = async () => {
  if (email.value.trim().toLowerCase() === authStore.user?.email?.toLowerCase()) {
    toast.error("不能在这里重置自己的密码，请到「账号设置」修改");
    return;
  }
  const schema = z
    .object({
      email: z.email("请输入有效的邮箱地址"),
      newPassword: buildPasswordSchema(complexPassword.value),
    })
    .refine((data) => data.newPassword === confirmPassword.value, {
      message: "两次输入的密码不一致",
      path: ["confirmPassword"],
    });
  const parsed = schema.safeParse({
    email: email.value,
    newPassword: newPassword.value,
  });
  if (!parsed.success) {
    toast.error(parsed.error.issues[0]?.message ?? "请检查表单填写");
    return;
  }
  try {
    await resetPassword.mutateAsync({
      email: parsed.data.email.trim().toLowerCase(),
      new_password: parsed.data.newPassword,
    });
    resetForm();
  } catch {
    // 错误 toast 已在 mutation onError 中处理
  }
};
</script>

<template>
  <div class="rounded-lg border px-5 py-5">
    <form class="max-w-xl space-y-4" @submit.prevent="handleSubmit">
      <Alert variant="destructive">
        <TriangleAlertIcon class="h-4 w-4" />
        <AlertDescription>重置成功后，该用户的所有登录会话将立即失效。</AlertDescription>
      </Alert>

      <div class="space-y-2">
        <Label for="reset-password-email">用户邮箱</Label>
        <Input
          id="reset-password-email"
          v-model="email"
          type="email"
          placeholder="name@company.com"
        />
      </div>

      <div class="space-y-2">
        <Label for="reset-password-new">新密码</Label>
        <div class="relative">
          <Input
            id="reset-password-new"
            v-model="newPassword"
            :type="showPassword ? 'text' : 'password'"
            placeholder="请输入新密码"
            class="pr-10"
            autocomplete="new-password"
          />
          <button
            type="button"
            class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
            @click="showPassword = !showPassword"
          >
            <EyeOffIcon v-if="showPassword" class="h-4 w-4" />
            <EyeIcon v-else class="h-4 w-4" />
          </button>
        </div>
        <p class="text-xs text-muted-foreground">{{ passwordHintText(complexPassword) }}</p>
      </div>

      <div class="space-y-2">
        <Label for="reset-password-confirm">确认新密码</Label>
        <Input
          id="reset-password-confirm"
          v-model="confirmPassword"
          type="password"
          placeholder="请再次输入新密码"
          autocomplete="new-password"
        />
      </div>

      <div class="flex items-center justify-between pt-2">
        <p class="text-xs text-muted-foreground">
          平台没有邮件找回，新密码需通过其他渠道交付给用户。
        </p>
        <Button type="submit" variant="destructive" :disabled="isPending">
          {{ isPending ? "重置中..." : "重置密码" }}
        </Button>
      </div>
    </form>
  </div>
</template>
