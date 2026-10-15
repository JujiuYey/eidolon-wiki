<script setup lang="ts">
import { EyeIcon, EyeOffIcon } from "@lucide/vue";
import { storeToRefs } from "pinia";
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { z } from "zod";
import { toast } from "vue-sonner";

import { authApi } from "@/api/auth/auth";
import { useAuthConfig } from "@/api/system/admin-user";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SagPage } from "@/components/sag-ui";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuthStore, isSystemAdmin, tenantRoleLabels, type TenantRole } from "@/stores/auth";
import { buildPasswordSchema, passwordHintText } from "@/utils/password-policy";

const router = useRouter();
const authStore = useAuthStore();
const { user, tenant } = storeToRefs(authStore);

// ---- 账号信息（只读；WeKnora 无资料修改端点，身份字段以登录账号为准）----

const displayName = computed(() => user.value?.username || "未命名用户");
const initial = computed(() => displayName.value.charAt(0).toUpperCase());

const currentRole = computed<TenantRole | undefined>(() => {
  const tenantId = tenant.value?.id;
  if (!tenantId) return undefined;
  return authStore.memberships.find((m) => String(m.tenant_id) === String(tenantId))?.role as
    | TenantRole
    | undefined;
});

// ---- 修改密码（POST /auth/change-password；策略从免鉴权的 /auth/config 读取）----

const authConfig = useAuthConfig();
// 配置拉取失败时退回简单策略，最终以后端校验为准
const complexPassword = computed(() => authConfig.data.value?.complex_password_enabled === true);

const formData = reactive({
  oldPassword: "",
  newPassword: "",
  confirmPassword: "",
});
const showNewPassword = ref(false);
const loading = ref(false);

const resetForm = () => {
  formData.oldPassword = "";
  formData.newPassword = "";
  formData.confirmPassword = "";
  showNewPassword.value = false;
};

const handleChangePassword = async () => {
  if (!formData.oldPassword) {
    toast.error("请输入原始密码");
    return;
  }
  const schema = z
    .object({ newPassword: buildPasswordSchema(complexPassword.value) })
    .refine((data) => data.newPassword === formData.confirmPassword, {
      message: "两次输入的密码不一致",
    })
    .refine((data) => data.newPassword !== formData.oldPassword, {
      message: "新密码不能与原密码相同",
    });
  const parsed = schema.safeParse({ newPassword: formData.newPassword });
  if (!parsed.success) {
    toast.error(parsed.error.issues[0]?.message ?? "请检查表单填写");
    return;
  }

  loading.value = true;
  try {
    await authApi.changePassword({
      old_password: formData.oldPassword,
      new_password: parsed.data.newPassword,
    });
    toast.success("密码修改成功，即将退出登录");
    resetForm();
    setTimeout(() => {
      authStore.clearAuth();
      void router.push("/login");
    }, 1500);
  } catch (error) {
    toast.error((error as { message?: string }).message || "密码修改失败");
  } finally {
    loading.value = false;
  }
};
</script>

<template>
  <SagPage>
    <template #title>个人设置</template>
    <template #description>查看当前账号信息与修改登录密码。</template>

    <div class="max-w-2xl space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>账号信息</CardTitle>
          <CardDescription>身份字段由账号体系管理，如需变更请联系系统管理员。</CardDescription>
        </CardHeader>
        <CardContent class="space-y-5">
          <div class="flex items-center gap-4">
            <Avatar class="h-12 w-12">
              <AvatarFallback class="text-lg">{{ initial }}</AvatarFallback>
            </Avatar>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span class="truncate font-medium">{{ displayName }}</span>
                <Badge
                  v-if="isSystemAdmin(authStore)"
                  variant="secondary"
                  class="bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300"
                >
                  平台超管
                </Badge>
              </div>
              <p class="text-sm text-muted-foreground">{{ user?.email || "-" }}</p>
            </div>
          </div>

          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <p class="text-xs text-muted-foreground">当前空间</p>
              <p class="mt-1 text-sm">{{ tenant?.name || "-" }}</p>
            </div>
            <div>
              <p class="text-xs text-muted-foreground">空间角色</p>
              <p class="mt-1">
                <Badge v-if="currentRole" variant="secondary">
                  {{ tenantRoleLabels[currentRole] }}
                </Badge>
                <span v-else class="text-sm text-muted-foreground">-</span>
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>修改密码</CardTitle>
          <CardDescription>修改成功后将退出登录，请使用新密码重新登录。</CardDescription>
        </CardHeader>
        <CardContent>
          <form class="max-w-md space-y-4" @submit.prevent="handleChangePassword">
            <div class="space-y-2">
              <Label for="account-old-password">原始密码</Label>
              <Input
                id="account-old-password"
                v-model="formData.oldPassword"
                type="password"
                placeholder="请输入原始密码"
                autocomplete="current-password"
              />
            </div>

            <div class="space-y-2">
              <Label for="account-new-password">新密码</Label>
              <div class="relative">
                <Input
                  id="account-new-password"
                  v-model="formData.newPassword"
                  :type="showNewPassword ? 'text' : 'password'"
                  placeholder="请输入新密码"
                  class="pr-10"
                  autocomplete="new-password"
                />
                <button
                  type="button"
                  class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  @click="showNewPassword = !showNewPassword"
                >
                  <EyeOffIcon v-if="showNewPassword" class="h-4 w-4" />
                  <EyeIcon v-else class="h-4 w-4" />
                </button>
              </div>
              <p class="text-xs text-muted-foreground">{{ passwordHintText(complexPassword) }}</p>
            </div>

            <div class="space-y-2">
              <Label for="account-confirm-password">确认新密码</Label>
              <Input
                id="account-confirm-password"
                v-model="formData.confirmPassword"
                type="password"
                placeholder="请再次输入新密码"
                autocomplete="new-password"
              />
            </div>

            <div class="flex justify-end pt-2">
              <Button type="submit" :disabled="loading">
                {{ loading ? "保存中..." : "修改密码" }}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  </SagPage>
</template>
