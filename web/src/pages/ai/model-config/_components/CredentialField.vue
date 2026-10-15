<script setup lang="ts">
import { ref } from "vue";
import { CheckCircle2Icon, CircleDashedIcon, EyeIcon, EyeOffIcon, KeyRoundIcon } from "@lucide/vue";
import { toast } from "vue-sonner";

import { modelApi } from "@/api/ai/models";
import type { ModelCredentialField } from "@/api/ai/models";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const props = defineProps<{
  modelId: string;
  field: ModelCredentialField;
  label: string;
  /** 主响应 credentials 里的 configured 状态；更新后需回传给父级 */
  configured: boolean;
}>();

const emit = defineEmits<{
  "update:configured": [configured: boolean];
}>();

const editing = ref(false);
const value = ref("");
const showValue = ref(false);
const saving = ref(false);
const removing = ref(false);

const handleSave = async () => {
  const trimmed = value.value.trim();
  if (!trimmed) return;
  saving.value = true;
  try {
    const res = await modelApi.putCredentials(props.modelId, { [props.field]: trimmed });
    emit("update:configured", res.fields?.[props.field]?.configured ?? true);
    toast.success(`${props.label}已保存`);
    editing.value = false;
    value.value = "";
  } catch (error) {
    toast.error((error as { message?: string }).message || "保存密钥失败");
  } finally {
    saving.value = false;
  }
};

const handleRemove = async () => {
  if (!removing.value) {
    removing.value = true;
    return;
  }
  try {
    await modelApi.removeCredentialField(props.modelId, props.field);
    emit("update:configured", false);
    toast.success(`${props.label}已移除`);
  } catch (error) {
    toast.error((error as { message?: string }).message || "移除密钥失败");
  } finally {
    removing.value = false;
  }
};
</script>

<template>
  <!-- 密钥三态卡（官方 CredentialResource 语义）：
       已配置（可更换/移除）→ 未配置（点击配置）→ 编辑中（保存走 PUT credentials）。
       后端没有 GET 密钥的端点，configured 布尔即真相来源。 -->
  <div v-if="editing" class="space-y-1.5">
    <Label class="text-xs text-muted-foreground">{{ props.label }}</Label>
    <div class="flex gap-1.5">
      <div class="relative flex-1">
        <Input
          v-model="value"
          :type="showValue ? 'text' : 'password'"
          placeholder="粘贴密钥"
          class="pr-9"
        />
        <button
          type="button"
          :aria-label="showValue ? '隐藏密钥' : '显示密钥'"
          class="text-muted-foreground hover:text-foreground absolute top-1/2 right-2.5 -translate-y-1/2"
          @click="showValue = !showValue"
        >
          <EyeOffIcon v-if="showValue" class="h-4 w-4" />
          <EyeIcon v-else class="h-4 w-4" />
        </button>
      </div>
      <Button size="sm" :disabled="saving || !value.trim()" @click="handleSave">
        {{ saving ? "保存中" : "保存" }}
      </Button>
      <Button
        size="sm"
        variant="outline"
        @click="
          () => {
            editing = false;
            value = '';
          }
        "
      >
        取消
      </Button>
    </div>
  </div>

  <div v-else class="space-y-1.5">
    <Label class="text-xs text-muted-foreground">{{ props.label }}</Label>
    <div class="flex items-center gap-2 rounded-md border px-3 py-2">
      <CheckCircle2Icon v-if="props.configured" class="h-4 w-4 text-emerald-500" />
      <CircleDashedIcon v-else class="h-4 w-4 text-muted-foreground" />
      <span class="flex-1 text-sm">
        <template v-if="props.configured">已配置</template>
        <span v-else class="text-muted-foreground">未配置</span>
      </span>
      <Button size="sm" variant="outline" @click="editing = true">
        {{ props.configured ? "更换" : "配置" }}
      </Button>
      <Button
        v-if="props.configured"
        size="sm"
        variant="ghost"
        :class="removing ? 'text-destructive' : 'text-muted-foreground'"
        @click="handleRemove"
      >
        <KeyRoundIcon class="mr-1 h-3.5 w-3.5" />
        {{ removing ? "确认移除" : "移除" }}
      </Button>
    </div>
  </div>
</template>
