<script setup lang="ts">
import { RotateCcwIcon } from "@lucide/vue";
import { computed, ref, watch } from "vue";

import {
  hasSettingOverride,
  HIGH_RISK_SETTING_KEYS,
  settingCategoryLabel,
  settingDescription,
  settingEnumLabel,
  settingLabel,
  settingModifiedAt,
  useResetSystemSetting,
  useSystemSettings,
  useUpdateSystemSetting,
} from "@/api/system/settings";
import type { SystemSettingItem } from "@/api/system/settings";
import { ConfirmDialog, SagPage } from "@/components/sag-ui";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";

function formatModifiedMeta(item: SystemSettingItem): string {
  const at = settingModifiedAt(item);
  if (!at) return "";
  const time = new Date(at);
  const timeText = Number.isNaN(time.getTime())
    ? at
    : time.toLocaleString("zh-CN", { hour12: false });
  const who = item.last_modified_by_name || item.last_modified_by.slice(0, 8);
  return `上次修改：${timeText} · ${who}`;
}

const settingsQuery = useSystemSettings();
const updateSetting = useUpdateSystemSetting();
const resetSetting = useResetSystemSetting();

const { data: settingsData, isLoading: settingsLoading, isError: settingsIsError, error: settingsError } =
  settingsQuery;

const items = computed(() => settingsData.value ?? []);

const editValues = ref<Record<string, string>>({});
const savedKeys = ref<Set<string>>(new Set());
const confirmTarget = ref<{ item: SystemSettingItem; value: unknown } | null>(null);
const resetTarget = ref<SystemSettingItem | null>(null);

const categories = computed(() => {
  const order: string[] = [];
  for (const item of items.value) {
    if (!order.includes(item.category)) order.push(item.category);
  }
  return order;
});

watch(savedKeys, (set, _prev, onCleanup) => {
  if (set.size === 0) return;
  const timer = window.setTimeout(() => (savedKeys.value = new Set()), 2000);
  onCleanup(() => window.clearTimeout(timer));
});

const applyUpdate = (item: SystemSettingItem, value: unknown) => {
  updateSetting.mutate(
    { key: item.key, value },
    {
      onSuccess: () => {
        const next = { ...editValues.value };
        delete next[item.key];
        editValues.value = next;
        savedKeys.value = new Set(savedKeys.value).add(item.key);
      },
      onError: () => {
        // 失败后丢弃本地编辑，回显服务端规范值
        const next = { ...editValues.value };
        delete next[item.key];
        editValues.value = next;
      },
    },
  );
};

const handleCommit = (item: SystemSettingItem, value: unknown) => {
  if (HIGH_RISK_SETTING_KEYS.has(item.key)) {
    confirmTarget.value = { item, value };
    return;
  }
  applyUpdate(item, value);
};

const handleHighRiskConfirm = async () => {
  if (confirmTarget.value) applyUpdate(confirmTarget.value.item, confirmTarget.value.value);
};

const handleResetConfirm = async () => {
  if (resetTarget.value) await resetSetting.mutateAsync(resetTarget.value.key);
};

// ---- SettingRow / SettingControl 内联实现 ----

const categoryItems = (category: string) => items.value.filter((item) => item.category === category);

const showSaved = (item: SystemSettingItem) => savedKeys.value.has(item.key);

const intValue = (item: SystemSettingItem) =>
  editValues.value[item.key] ?? (item.value == null ? "" : String(item.value));

const handleIntBlur = (item: SystemSettingItem) => {
  const value = intValue(item);
  if (value === "" || value === String(item.value)) return;
  const parsed = Number(value);
  if (Number.isNaN(parsed)) return;
  handleCommit(item, parsed);
};

const stringListValue = (item: SystemSettingItem) =>
  editValues.value[item.key] ?? (Array.isArray(item.value) ? item.value.join("\n") : "");

const handleStringListBlur = (item: SystemSettingItem) => {
  const value = stringListValue(item);
  const next = value
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
  const current = Array.isArray(item.value) ? item.value.map(String) : [];
  if (JSON.stringify(next) === JSON.stringify(current)) return;
  handleCommit(item, next);
};

const stringValue = (item: SystemSettingItem) =>
  editValues.value[item.key] ?? (item.value == null ? "" : String(item.value));

const handleStringBlur = (item: SystemSettingItem) => {
  const value = stringValue(item);
  if (value === String(item.value ?? "")) return;
  handleCommit(item, value);
};

const handleEditValue = (key: string, value: string | number) => {
  editValues.value = { ...editValues.value, [key]: String(value) };
};
</script>

<template>
  <SagPage>
    <template #title>系统设置</template>
    <template #description>
      平台级运行时配置。数据库覆盖优先于环境变量与内置默认值，修改后立即生效。
    </template>

    <div v-if="settingsLoading" class="space-y-4">
      <Skeleton v-for="index in 5" :key="index" class="h-16 w-full" />
    </div>
    <p v-else-if="settingsIsError" class="text-sm text-destructive">
      设置加载失败：{{ (settingsError as Error)?.message ?? "请确认当前账号是系统管理员" }}
    </p>
    <Tabs v-else :default-value="categories[0] ?? 'other'" class="flex h-full min-h-0 flex-col gap-4">
      <TabsList class="w-fit flex-wrap">
        <TabsTrigger v-for="category in categories" :key="category" :value="category">
          {{ settingCategoryLabel(category) }}
        </TabsTrigger>
      </TabsList>
      <TabsContent
        v-for="category in categories"
        :key="category"
        :value="category"
        class="min-h-0 flex-1 overflow-y-auto pb-6"
      >
        <div class="rounded-lg border px-5">
          <div
            v-for="item in categoryItems(category)"
            :key="item.key"
            class="flex items-start justify-between gap-6 border-b py-5 last:border-b-0"
          >
            <div class="min-w-0 flex-1">
              <div class="mb-1 flex flex-wrap items-center gap-2">
                <Label class="text-sm font-semibold">{{ settingLabel(item) }}</Label>
                <Badge
                  v-if="hasSettingOverride(item)"
                  variant="secondary"
                  class="bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300"
                >
                  已覆盖
                </Badge>
                <Badge
                  v-if="HIGH_RISK_SETTING_KEYS.has(item.key)"
                  variant="secondary"
                  class="bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300"
                >
                  高危
                </Badge>
                <Badge
                  v-if="showSaved(item)"
                  variant="secondary"
                  class="bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300"
                >
                  已保存
                </Badge>
              </div>
              <p v-if="settingDescription(item)" class="max-w-2xl text-xs leading-relaxed text-muted-foreground">
                {{ settingDescription(item) }}
              </p>
              <p v-if="formatModifiedMeta(item)" class="mt-1 text-xs text-muted-foreground/80">
                {{ formatModifiedMeta(item) }}
              </p>
            </div>

            <div class="w-64 shrink-0">
              <!-- SettingControl -->
              <Select
                v-if="item.enum && item.enum.length > 0"
                :model-value="typeof item.value === 'string' ? item.value : String(item.value ?? '')"
                @update:model-value="(next) => handleCommit(item, next)"
              >
                <SelectTrigger class="w-full">
                  <SelectValue placeholder="选择值" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="option in item.enum" :key="option" :value="option">
                    {{ settingEnumLabel(item, option) }}
                  </SelectItem>
                </SelectContent>
              </Select>

              <div v-else-if="item.value_type === 'bool'" class="flex justify-end">
                <Switch
                  :checked="item.value === true"
                  @update:checked="(checked: boolean) => handleCommit(item, checked)"
                />
              </div>

              <Input
                v-else-if="item.value_type === 'int'"
                type="number"
                :model-value="intValue(item)"
                @update:model-value="(value) => handleEditValue(item.key, value)"
                @blur="handleIntBlur(item)"
              />

              <Textarea
                v-else-if="item.value_type === 'string_list'"
                :rows="3"
                class="font-mono text-xs"
                placeholder="每行一条"
                :model-value="stringListValue(item)"
                @update:model-value="(value) => handleEditValue(item.key, value)"
                @blur="handleStringListBlur(item)"
              />

              <Input
                v-else
                :model-value="stringValue(item)"
                @update:model-value="(value) => handleEditValue(item.key, value)"
                @blur="handleStringBlur(item)"
              />
            </div>

            <div class="shrink-0 pt-1">
              <button
                v-if="hasSettingOverride(item)"
                type="button"
                class="inline-flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
                @click="resetTarget = item"
              >
                <RotateCcwIcon class="h-3.5 w-3.5" />
                重置
              </button>
            </div>
          </div>
        </div>
      </TabsContent>
    </Tabs>

    <ConfirmDialog
      :open="Boolean(confirmTarget)"
      title="确认修改高危设置"
      :description="`「${confirmTarget ? settingLabel(confirmTarget.item) : ''}」是高风险设置，确定将值修改为「${confirmTarget ? settingEnumLabel(confirmTarget.item, String(confirmTarget.value)) : ''}」？`"
      confirm-text="确认修改"
      @update:open="(open) => !open && (confirmTarget = null)"
      @confirm="handleHighRiskConfirm"
    />

    <ConfirmDialog
      :open="Boolean(resetTarget)"
      destructive
      title="重置设置"
      :description="`确定将「${resetTarget ? settingLabel(resetTarget) : ''}」恢复为环境变量或内置默认值？`"
      confirm-text="重置"
      @update:open="(open) => !open && (resetTarget = null)"
      @confirm="handleResetConfirm"
    />
  </SagPage>
</template>
