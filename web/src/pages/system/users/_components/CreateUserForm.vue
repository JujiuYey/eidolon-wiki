<script setup lang="ts">
import { CheckIcon, CopyIcon, EyeIcon, EyeOffIcon, TriangleAlertIcon } from "@lucide/vue";
import { computed, ref } from "vue";
import { z } from "zod";
import { toast } from "vue-sonner";

import { useAuthConfig, useCreateSystemUser } from "@/api/system/admin-user";
import { buildPasswordSchema, passwordHintText } from "@/utils/password-policy";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const createUser = useCreateSystemUser();
const { isPending } = createUser;
const authConfig = useAuthConfig();
// 配置拉取失败时退回简单策略，最终以后端校验为准
const complexPassword = computed(() => authConfig.data.value?.complex_password_enabled === true);

// 未引入表单库：本地状态 + 提交时 zod 校验（校验失败 toast）
const username = ref("");
const email = ref("");
const password = ref("");
const showPassword = ref(false);

// 服务端自动生成的密码只在此处一次性展示，离开面板即不可再取
const generatedPassword = ref<string | null>(null);

const resetForm = () => {
  username.value = "";
  email.value = "";
  password.value = "";
  showPassword.value = false;
};

const handleCopy = async () => {
  if (!generatedPassword.value) return;
  try {
    await navigator.clipboard.writeText(generatedPassword.value);
    toast.success("已复制到剪贴板");
  } catch {
    toast.error("复制失败，请手动选中复制");
  }
};

const handleCreateAnother = () => {
  generatedPassword.value = null;
  resetForm();
};

const handleSubmit = async () => {
  const passwordSchema = buildPasswordSchema(complexPassword.value);
  const schema = z.object({
    username: z.string().trim().min(2, "用户名至少 2 个字符").max(50, "用户名不能超过 50 个字符"),
    email: z.email("请输入有效的邮箱地址"),
    password: password.value ? passwordSchema : z.string().optional(),
  });
  const parsed = schema.safeParse({
    username: username.value,
    email: email.value,
    password: password.value || undefined,
  });
  if (!parsed.success) {
    toast.error(parsed.error.issues[0]?.message ?? "请检查表单填写");
    return;
  }
  try {
    const response = await createUser.mutateAsync({
      username: parsed.data.username,
      email: parsed.data.email.trim().toLowerCase(),
      password: parsed.data.password,
    });
    if (response.generated_password) {
      generatedPassword.value = response.generated_password;
      password.value = "";
    } else {
      // 幂等命中已有账号，「该用户已存在」toast 在 mutation onSuccess 里
      resetForm();
    }
  } catch {
    // 错误 toast 已在 mutation onError 中处理
  }
};
</script>

<template>
  <div class="rounded-lg border px-5 py-5">
    <template v-if="generatedPassword">
      <div class="max-w-xl space-y-4">
        <div>
          <h3 class="text-sm font-semibold">用户创建成功</h3>
          <p class="mt-1 text-xs leading-relaxed text-muted-foreground">
            以下密码由系统生成。平台没有邮件找回，请立即复制并交付给用户。
          </p>
        </div>

        <Alert variant="destructive">
          <TriangleAlertIcon class="h-4 w-4" />
          <AlertDescription>
            仅此一次展示，切换或刷新后无法再次查看，请确认已保存。
          </AlertDescription>
        </Alert>

        <div class="flex items-center gap-2 rounded-md border bg-muted/50 px-3 py-2.5">
          <span class="min-w-0 flex-1 break-all font-mono text-sm select-all">
            {{ generatedPassword }}
          </span>
          <Button type="button" variant="outline" size="sm" class="shrink-0" @click="handleCopy">
            <CopyIcon class="h-3.5 w-3.5" />
            复制
          </Button>
        </div>

        <div class="flex justify-end">
          <Button type="button" @click="handleCreateAnother">
            <CheckIcon class="h-4 w-4" />
            我已保存，创建下一个
          </Button>
        </div>
      </div>
    </template>

    <form v-else class="max-w-xl space-y-4" @submit.prevent="handleSubmit">
      <div class="space-y-2">
        <Label for="create-user-username">用户名</Label>
        <Input
          id="create-user-username"
          v-model="username"
          placeholder="2-50 个字符"
          autocomplete="off"
        />
      </div>

      <div class="space-y-2">
        <Label for="create-user-email">邮箱</Label>
        <Input
          id="create-user-email"
          v-model="email"
          type="email"
          placeholder="name@company.com"
        />
      </div>

      <div class="space-y-2">
        <Label for="create-user-password">初始密码（可选）</Label>
        <div class="relative">
          <Input
            id="create-user-password"
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            placeholder="留空则自动生成"
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

      <div class="flex justify-end pt-2">
        <Button type="submit" :disabled="isPending">
          <span v-if="isPending" class="flex items-center gap-2">创建中...</span>
          <span v-else>创建用户</span>
        </Button>
      </div>
    </form>
  </div>
</template>
