<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { CloudIcon, Loader2Icon, PlugZapIcon, ServerIcon } from "@lucide/vue";
import type { AcceptableValue } from "reka-ui";
import { toast } from "vue-sonner";

import {
  defaultThinkingControl,
  modelApi,
  modelTypeToUrlKey,
  thinkingControlOptions,
} from "@/api/ai/models";
import type { ModelCheckResult, ModelInfo, ModelType, ProviderInfo, ThinkingControl } from "@/api/ai/models";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

import CredentialField from "./CredentialField.vue";
import OllamaModelSelect from "./OllamaModelSelect.vue";

const ALL_TYPES: Array<{ value: ModelType; label: string }> = [
  { value: "KnowledgeQA", label: "对话" },
  { value: "Embedding", label: "Embedding" },
  { value: "Rerank", label: "ReRank" },
  { value: "VLLM", label: "视觉" },
  { value: "ASR", label: "语音" },
];

interface CustomHeader {
  key: string;
  value: string;
}

interface FormState {
  type: ModelType;
  source: "local" | "remote";
  provider: string;
  name: string;
  displayName: string;
  description: string;
  baseUrl: string;
  apiKey: string;
  appSecret: string;
  lkeapRegion: string;
  customHeaders: CustomHeader[];
  dimension: string;
  supportsDimensionOverride: boolean;
  supportsVision: boolean;
  thinkingControl: ThinkingControl;
  thinkingManual: boolean;
  maxConcurrency: string;
}

type WkcState = "loading" | "unconfigured" | "configured" | "expired";

function initialState(model: ModelInfo | null): FormState {
  if (!model) {
    return {
      type: "KnowledgeQA",
      source: "remote",
      provider: "",
      name: "",
      displayName: "",
      description: "",
      baseUrl: "",
      apiKey: "",
      appSecret: "",
      lkeapRegion: "ap-guangzhou",
      customHeaders: [],
      dimension: "",
      supportsDimensionOverride: false,
      supportsVision: false,
      thinkingControl: "none",
      thinkingManual: false,
      maxConcurrency: "",
    };
  }
  return {
    type: model.type,
    source: model.source === "local" ? "local" : "remote",
    provider: model.parameters.provider ?? "",
    name: model.name,
    displayName: model.display_name ?? "",
    description: model.description ?? "",
    baseUrl: model.parameters.base_url ?? "",
    apiKey: "",
    appSecret: "",
    lkeapRegion: model.parameters.extra_config?.region || "ap-guangzhou",
    customHeaders: Object.entries(model.parameters.custom_headers ?? {}).map(([key, value]) => ({
      key,
      value,
    })),
    dimension: model.parameters.embedding_parameters?.dimension
      ? String(model.parameters.embedding_parameters.dimension)
      : "",
    supportsDimensionOverride: model.parameters.embedding_parameters?.supports_dimension_override === true,
    supportsVision: model.parameters.supports_vision === true,
    // 存库值优先且视为"手动设置"，不再被自动推断覆盖
    thinkingControl:
      (model.parameters.extra_config?.thinking_control as ThinkingControl | undefined) ?? "none",
    thinkingManual: Boolean(model.parameters.extra_config?.thinking_control),
    maxConcurrency: model.parameters.max_concurrency ? String(model.parameters.max_concurrency) : "",
  };
}

function modelNamePlaceholder(state: FormState): string {
  if (state.source === "local") return "搜索并选择 Ollama 模型";
  if (state.type === "ASR") return "例如 sensevoice";
  if (state.type === "VLLM") return "例如 Qwen2.5-VL-7B-Instruct";
  return "例如 gpt-4o、glm-4.5";
}

function buildCustomHeadersMap(headers: CustomHeader[]): Record<string, string> | undefined {
  const map: Record<string, string> = {};
  for (const item of headers) {
    const key = item.key.trim();
    const value = item.value.trim();
    if (key && value) map[key] = value;
  }
  return Object.keys(map).length > 0 ? map : undefined;
}

const props = defineProps<{
  open: boolean;
  providers: ProviderInfo[];
  /** 传入则为编辑模式（type/source 不建议改） */
  model?: ModelInfo | null;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
}>();

const handleOpenChange = (open: boolean) => emit("update:open", open);

const isEdit = computed(() => Boolean(props.model));
const state = ref<FormState>(initialState(props.model ?? null));
const apiKeyConfigured = ref(false);
const appSecretConfigured = ref(false);
const ollamaAvailable = ref<boolean | null>(null);
const wkcState = ref<WkcState>("unconfigured");
const testing = ref(false);
const testResult = ref<{ ok: boolean; message: string } | null>(null);
const detecting = ref(false);
const saving = ref(false);

const patch = (partial: Partial<FormState>) => {
  state.value = { ...state.value, ...partial };
};

// 派生状态
const isSignedRerank = computed(
  () => state.value.type === "Rerank" && (state.value.provider === "lkeap" || state.value.provider === "volcengine"),
);
const isLkeapRerank = computed(() => state.value.type === "Rerank" && state.value.provider === "lkeap");
const isWeknora = computed(() => state.value.provider === "weknoracloud" && state.value.source === "remote");
const advancedVisible = computed(() => ["Embedding", "KnowledgeQA", "VLLM"].includes(state.value.type));

const availableProviders = computed(() => {
  const matching = props.providers.filter(
    (provider) =>
      !provider.modelTypes || provider.modelTypes.length === 0 || provider.modelTypes.includes(state.value.type),
  );
  return matching.length > 0 ? matching : props.providers;
});

const apiKeyLabel = computed(() =>
  isSignedRerank.value ? (state.value.provider === "lkeap" ? "SecretId" : "Access Key ID") : "API Key（可选）",
);
const appSecretLabel = computed(() => (state.value.provider === "lkeap" ? "SecretKey" : "Secret Access Key"));

// 打开时初始化
watch([() => props.open, () => props.model], ([open]) => {
  if (!open) return;
  state.value = initialState(props.model ?? null);
  apiKeyConfigured.value = props.model?.credentials?.api_key?.configured === true;
  appSecretConfigured.value = props.model?.credentials?.app_secret?.configured === true;
  testResult.value = null;
  // Ollama 可用性探测（决定"本地"来源是否可点）
  modelApi
    .ollamaStatus()
    .then((res) => (ollamaAvailable.value = res.data?.available === true))
    .catch(() => (ollamaAvailable.value = false));
});

// weknoracloud 凭证状态检查
watch([() => props.open, isWeknora], ([open, weknoraActive]) => {
  if (!open || !weknoraActive) return;
  wkcState.value = "loading";
  modelApi
    .weknoraCloudStatus()
    .then((res) => {
      const payload = res.data ?? res;
      if (payload.needs_reinit) wkcState.value = "expired";
      else if (payload.has_models) wkcState.value = "configured";
      else wkcState.value = "unconfigured";
    })
    .catch(() => (wkcState.value = "unconfigured"));
});

// 切换模型类型的联动（官方 selectModelType 语义）
const handleTypeChange = (type: ModelType) => {
  const next: Partial<FormState> = { type };
  if (type === "Rerank") next.source = "remote"; // Ollama 不支持 ReRank
  if (state.value.type === "Embedding" && type !== "Embedding") {
    next.dimension = "";
    next.supportsDimensionOverride = false;
  }
  if (state.value.type === "KnowledgeQA" && type !== "KnowledgeQA") {
    next.supportsVision = false;
    next.thinkingControl = "none";
    next.thinkingManual = false;
  }
  // 当前厂商不支持新类型 → 回退自定义并清空 Base URL
  const stillSupported = props.providers.find(
    (provider) =>
      provider.value === state.value.provider &&
      (!provider.modelTypes || provider.modelTypes.length === 0 || provider.modelTypes.includes(type)),
  );
  if (state.value.provider && !stillSupported) {
    next.provider = "generic";
    next.baseUrl = "";
  }
  if (type === "KnowledgeQA" && next.source !== "local" && !next.thinkingManual) {
    next.thinkingControl = defaultThinkingControl(next.provider ?? state.value.provider, state.value.name);
  }
  testResult.value = null;
  patch(next);
};

const handleSourceChange = (source: "local" | "remote") => {
  testResult.value = null;
  patch({ source });
};

// 选厂商联动：默认 URL、签名重排的模型名预填、思考模式重算
const handleProviderChange = (value: AcceptableValue) => {
  const providerId = typeof value === "string" ? value : "";
  const provider = props.providers.find((item) => item.value === providerId);
  const defaultUrl = provider?.defaultUrls?.[modelTypeToUrlKey(state.value.type)];
  const next: Partial<FormState> = { provider: providerId };
  if (defaultUrl) next.baseUrl = defaultUrl;
  if (state.value.type === "Rerank" && !state.value.name.trim()) {
    if (providerId === "lkeap") next.name = "lke-reranker-base";
    if (providerId === "volcengine") next.name = "doubao-seed-rerank";
  }
  if (state.value.type === "KnowledgeQA" && !state.value.thinkingManual) {
    next.thinkingControl = defaultThinkingControl(providerId, next.name ?? state.value.name);
  }
  testResult.value = null;
  patch(next);
};

const handleNameChange = (name: string) => {
  const next: Partial<FormState> = { name };
  if (state.value.type === "KnowledgeQA" && state.value.source === "remote" && !state.value.thinkingManual) {
    next.thinkingControl = defaultThinkingControl(state.value.provider, name);
  }
  patch(next);
};

// 表单校验（对齐官方三层校验的最严集）
const validate = (): string | null => {
  if (!state.value.name.trim()) return "请填写模型名称";
  if (state.value.name.trim().length > 100) return "模型名称不能超过 100 字符";
  if (state.value.displayName.trim().length > 100) return "显示名称不能超过 100 个字符";
  if (state.value.source === "remote" && !isWeknora.value) {
    if (!state.value.baseUrl.trim()) return "请填写 Base URL";
    try {
      new URL(state.value.baseUrl.trim());
    } catch {
      return "Base URL 格式不正确";
    }
  }
  if (state.value.type === "Embedding") {
    const dimension = Number(state.value.dimension);
    if (!state.value.dimension || Number.isNaN(dimension) || dimension < 128 || dimension > 4096) {
      return "Embedding 模型必须填写有效的向量维度（128-4096），可先测试连接自动检测";
    }
  }
  return null;
};

// 测试连通性（remote only；编辑态传 modelId 让后端用存储密钥兜底）
const handleTest = async () => {
  if (!state.value.name.trim() || (state.value.source === "remote" && !isWeknora.value && !state.value.baseUrl.trim())) {
    toast.warning("请先填写模型名称和 Base URL");
    return;
  }
  testing.value = true;
  testResult.value = null;
  try {
    const base = {
      modelName: state.value.name.trim(),
      baseUrl: state.value.baseUrl.trim(),
      apiKey: state.value.apiKey.trim() || "",
      provider: state.value.provider || "",
      modelId: isEdit.value && props.model ? props.model.id : undefined,
      custom_headers: buildCustomHeadersMap(state.value.customHeaders),
    };
    let result: ModelCheckResult;
    if (state.value.type === "Embedding") {
      result = await modelApi.checkEmbedding({
        ...base,
        source: "remote",
        dimension: Number(state.value.dimension) || undefined,
        supportsDimensionOverride: state.value.supportsDimensionOverride,
      });
    } else if (state.value.type === "Rerank") {
      result = await modelApi.checkRerank({
        ...base,
        appSecret: state.value.appSecret.trim() || undefined,
        extraConfig: isLkeapRerank.value ? { region: (state.value.lkeapRegion || "ap-guangzhou").trim() } : undefined,
      });
    } else if (state.value.type === "ASR") {
      result = await modelApi.checkAsr(base);
    } else {
      result = await modelApi.checkChat(base);
    }
    if (result.available) {
      testResult.value = { ok: true, message: result.message || "连接成功" };
      if (state.value.type === "Embedding" && result.dimension) {
        patch({ dimension: String(result.dimension) });
        toast.info(`检测到向量维度：${result.dimension}`);
      } else {
        toast.success(result.message || "连接成功");
      }
    } else {
      testResult.value = { ok: false, message: result.message || "连接失败" };
    }
  } catch (err) {
    testResult.value = {
      ok: false,
      message: (err as { message?: string }).message || "连接失败，请检查配置",
    };
  } finally {
    testing.value = false;
  }
};

// 本地 embedding 维度检测
const handleDetectDimension = async () => {
  if (!state.value.name.trim()) {
    toast.warning("请先选择本地模型");
    return;
  }
  detecting.value = true;
  try {
    const result = await modelApi.checkEmbedding({
      source: "local",
      modelName: state.value.name.trim(),
      supportsDimensionOverride: state.value.supportsDimensionOverride,
    });
    if (result.available && result.dimension) {
      patch({ dimension: String(result.dimension) });
      toast.success(`检测成功，向量维度：${result.dimension}`);
    } else {
      toast.warning("检测失败，请手动输入维度");
    }
  } catch {
    toast.warning("检测失败，请手动输入维度");
  } finally {
    detecting.value = false;
  }
};

const handleSubmit = async () => {
  const error = validate();
  if (error) {
    toast.error(error);
    return;
  }
  saving.value = true;
  try {
    const parameters: ModelInfo["parameters"] = {
      base_url: state.value.source === "remote" ? state.value.baseUrl.trim() : "",
      provider: state.value.provider || "",
    };
    // 明文密钥只在新建时进 parameters；编辑一律走 credentials 子资源
    if (!isEdit.value) {
      if (state.value.apiKey.trim()) parameters.api_key = state.value.apiKey.trim();
      if (isSignedRerank.value && state.value.appSecret.trim()) parameters.app_secret = state.value.appSecret.trim();
    }
    const extraConfig: Record<string, string> = {};
    if (isLkeapRerank.value) {
      extraConfig.region = (state.value.lkeapRegion || "ap-guangzhou").trim();
    }
    if (state.value.type === "KnowledgeQA" && state.value.source === "remote" && state.value.thinkingControl) {
      extraConfig.thinking_control = state.value.thinkingControl;
    }
    if (Object.keys(extraConfig).length > 0) parameters.extra_config = extraConfig;
    const headersMap = buildCustomHeadersMap(state.value.customHeaders);
    if (headersMap) parameters.custom_headers = headersMap;
    if (state.value.type === "Embedding" && state.value.dimension) {
      parameters.embedding_parameters = {
        dimension: Number(state.value.dimension),
        truncate_prompt_tokens: 0,
        supports_dimension_override: state.value.supportsDimensionOverride,
      };
    }
    // vllm 天生多模态硬编码 true；chat 跟随开关
    if (state.value.type === "VLLM") parameters.supports_vision = true;
    if (state.value.type === "KnowledgeQA") parameters.supports_vision = state.value.supportsVision;
    if (["KnowledgeQA", "Embedding", "VLLM"].includes(state.value.type) && Number(state.value.maxConcurrency) > 0) {
      parameters.max_concurrency = Number(state.value.maxConcurrency);
    }

    if (isEdit.value && props.model) {
      await modelApi.update(props.model.id, {
        name: state.value.name.trim(),
        display_name: state.value.displayName.trim() || "",
        description: state.value.description.trim() || undefined,
        source: state.value.source,
        parameters,
      });
      if (state.value.apiKey.trim() || (isSignedRerank.value && state.value.appSecret.trim())) {
        const credBody: Record<string, string> = {};
        if (state.value.apiKey.trim()) credBody.api_key = state.value.apiKey.trim();
        if (isSignedRerank.value && state.value.appSecret.trim()) credBody.app_secret = state.value.appSecret.trim();
        const res = await modelApi.putCredentials(props.model.id, credBody);
        apiKeyConfigured.value = res.fields?.api_key?.configured ?? apiKeyConfigured.value;
        appSecretConfigured.value = res.fields?.app_secret?.configured ?? appSecretConfigured.value;
      }
      toast.success("模型已更新");
    } else {
      await modelApi.create({
        name: state.value.name.trim(),
        display_name: state.value.displayName.trim() || undefined,
        type: state.value.type,
        source: state.value.source,
        description: state.value.description.trim() || undefined,
        parameters,
      });
      toast.success("模型已添加");
    }
    handleOpenChange(false);
  } catch (err) {
    toast.error((err as { message?: string }).message || "保存模型失败");
  } finally {
    saving.value = false;
  }
};

const saveDisabled = computed(() => isWeknora.value && wkcState.value !== "configured");
const testDisabled = computed(
  () => !state.value.name.trim() || (state.value.source === "remote" && !isWeknora.value && !state.value.baseUrl.trim()),
);
</script>

<template>
  <Sheet :open="props.open" @update:open="handleOpenChange">
    <SheetContent class="flex flex-col gap-0 sm:max-w-xl">
      <SheetHeader class="border-b">
        <SheetTitle>{{ isEdit ? `编辑模型：${model?.display_name || model?.name}` : "添加模型" }}</SheetTitle>
        <SheetDescription>
          {{ isEdit
            ? "密钥不会随表单回显，修改请使用下方密钥卡。"
            : "选择模型类型与来源，填写接入信息后可先测试连通性。" }}
        </SheetDescription>
      </SheetHeader>

      <div class="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-4">
        <div v-if="!isEdit" class="space-y-1.5">
          <Label>模型类型</Label>
          <div class="grid grid-cols-5 gap-1.5">
            <button
              v-for="type in ALL_TYPES"
              :key="type.value"
              type="button"
              :class="
                cn(
                  'rounded-md border px-2 py-2 text-xs font-medium transition-colors',
                  state.type === type.value
                    ? 'border-primary bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-muted',
                )
              "
              @click="handleTypeChange(type.value)"
            >
              {{ type.label }}
            </button>
          </div>
        </div>

        <!-- 模型来源 -->
        <div class="space-y-1.5">
          <Label>模型来源</Label>
          <div class="grid grid-cols-2 gap-1.5">
            <button
              type="button"
              :disabled="state.type === 'Rerank'"
              :class="
                cn(
                  'flex items-center justify-center gap-1.5 rounded-md border px-2 py-2 text-sm transition-colors',
                  state.source === 'remote' && 'border-primary bg-primary/10 font-medium text-primary',
                  'disabled:cursor-not-allowed disabled:opacity-50',
                )
              "
              @click="handleSourceChange('remote')"
            >
              <CloudIcon class="h-4 w-4" />
              API
            </button>
            <button
              type="button"
              :disabled="state.type === 'Rerank' || ollamaAvailable === false"
              :class="
                cn(
                  'flex items-center justify-center gap-1.5 rounded-md border px-2 py-2 text-sm transition-colors',
                  state.source === 'local' && 'border-primary bg-primary/10 font-medium text-primary',
                  'disabled:cursor-not-allowed disabled:opacity-50',
                )
              "
              @click="handleSourceChange('local')"
            >
              <ServerIcon class="h-4 w-4" />
              Ollama
            </button>
          </div>
          <p v-if="state.type === 'Rerank'" class="text-xs text-muted-foreground">
            Ollama 不支持 ReRank 模型，请使用远程接口配置
          </p>
          <p v-else-if="ollamaAvailable === false" class="text-xs text-muted-foreground">
            Ollama 服务不可用，无法选择本地模型
          </p>
        </div>

        <template v-if="state.source === 'local'">
          <div class="space-y-1.5">
            <Label>
              模型名称 <span class="text-destructive">*</span>
            </Label>
            <OllamaModelSelect :value="state.name" @update:value="handleNameChange" />
            <p v-if="state.type === 'Embedding' && state.name" class="text-xs text-muted-foreground">
              模型已选择，可点击下方"检测维度"自动获取向量维度
            </p>
          </div>
        </template>
        <template v-else>
          <div class="space-y-1.5">
            <Label>服务商</Label>
            <Select :model-value="state.provider || undefined" @update:model-value="handleProviderChange">
              <SelectTrigger class="w-full">
                <SelectValue placeholder="选择模型服务商" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="provider in availableProviders" :key="provider.value" :value="provider.value">
                  <span class="flex flex-col">
                    <span>{{ provider.label }}</span>
                    <span v-if="provider.description" class="text-xs text-muted-foreground">
                      {{ provider.description }}
                    </span>
                  </span>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div
            v-if="isWeknora"
            :class="
              cn(
                'rounded-md border px-3 py-2 text-xs',
                wkcState === 'configured' && 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300',
                wkcState === 'expired' && 'border-red-200 bg-red-50 text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300',
                (wkcState === 'unconfigured' || wkcState === 'loading') && 'bg-muted text-muted-foreground',
              )
            "
          >
            {{ wkcState === "loading"
              ? "正在检查凭证状态..."
              : wkcState === "configured"
                ? "WeKnoraCloud 凭证已配置。"
                : wkcState === "expired"
                  ? "凭证已失效，请重新配置。"
                  : "WeKnoraCloud 凭证未配置，请前往系统设置配置后使用。" }}
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-1.5">
              <Label>
                模型名称 <span class="text-destructive">*</span>
              </Label>
              <Input
                :model-value="state.name"
                :placeholder="modelNamePlaceholder(state)"
                :disabled="isWeknora && wkcState !== 'configured'"
                @update:model-value="(text) => handleNameChange(String(text))"
              />
            </div>
            <div class="space-y-1.5">
              <Label>显示名称（可选）</Label>
              <Input v-model="state.displayName" placeholder="例如：客服问答模型" />
              <p class="text-xs text-muted-foreground">仅用于界面展示，实际调用仍使用模型名称。</p>
            </div>
          </div>

          <div v-if="!isWeknora" class="space-y-1.5">
            <Label>
              Base URL <span class="text-destructive">*</span>
            </Label>
            <Input
              v-model="state.baseUrl"
              :placeholder="state.type === 'VLLM' ? 'http://localhost:11434/v1' : 'https://api.example.com/v1'"
            />
          </div>

          <div class="space-y-1.5">
            <Label>描述（可选）</Label>
            <Textarea v-model="state.description" :rows="2" />
          </div>

          <!-- 密钥：编辑态走三态卡，新增态明文输入（仅创建时提交） -->
          <div v-if="isEdit && model" class="space-y-3">
            <CredentialField
              :model-id="model.id"
              field="api_key"
              :label="apiKeyLabel"
              :configured="apiKeyConfigured"
              @update:configured="(configured) => (apiKeyConfigured = configured)"
            />
            <CredentialField
              v-if="isSignedRerank"
              :model-id="model.id"
              field="app_secret"
              :label="appSecretLabel"
              :configured="appSecretConfigured"
              @update:configured="(configured) => (appSecretConfigured = configured)"
            />
          </div>
          <template v-else>
            <div class="space-y-1.5">
              <Label>{{ apiKeyLabel }}</Label>
              <Input v-model="state.apiKey" type="password" :placeholder="isSignedRerank ? undefined : '输入 API Key'" />
            </div>
            <template v-if="isSignedRerank">
              <p class="text-xs text-muted-foreground">
                {{ state.provider === "lkeap"
                  ? "Rerank 使用腾讯云 API 签名（非 OpenAI API Key）。请在云 API 密钥控制台创建 SecretId/SecretKey。"
                  : "Rerank 使用 VikingDB AK/SK 签名（非方舟 API Key），模型建议填写 doubao-seed-rerank。" }}
              </p>
              <div class="space-y-1.5">
                <Label>
                  {{ appSecretLabel }} <span class="text-destructive">*</span>
                </Label>
                <Input v-model="state.appSecret" type="password" />
              </div>
            </template>
          </template>

          <div v-if="isLkeapRerank" class="space-y-1.5">
            <Label>地域</Label>
            <Input v-model="state.lkeapRegion" placeholder="ap-guangzhou" />
            <p class="text-xs text-muted-foreground">
              RunRerank 支持 ap-beijing、ap-guangzhou 等，默认 ap-guangzhou
            </p>
          </div>

          <div v-if="!isWeknora" class="space-y-1.5">
            <div class="flex items-center justify-between">
              <Label class="text-xs text-muted-foreground">自定义请求头（可选）</Label>
              <Button variant="ghost" size="sm" @click="patch({ customHeaders: [...state.customHeaders, { key: '', value: '' }] })">
                添加请求头
              </Button>
            </div>
            <div v-for="(header, index) in state.customHeaders" :key="index" class="flex gap-1.5">
              <Input
                class="flex-1"
                placeholder="Header 名称"
                :model-value="header.key"
                @update:model-value="
                  (text) =>
                    patch({
                      customHeaders: state.customHeaders.map((item, i) =>
                        i === index ? { ...item, key: String(text) } : item,
                      ),
                    })
                "
              />
              <Input
                class="flex-1"
                placeholder="Header 值"
                :model-value="header.value"
                @update:model-value="
                  (text) =>
                    patch({
                      customHeaders: state.customHeaders.map((item, i) =>
                        i === index ? { ...item, value: String(text) } : item,
                      ),
                    })
                "
              />
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label="删除请求头"
                @click="patch({ customHeaders: state.customHeaders.filter((_, i) => i !== index) })"
              >
                ×
              </Button>
            </div>
            <p v-if="state.customHeaders.length > 0" class="text-xs text-muted-foreground">
              Authorization、Content-Type 等保留头会被自动忽略。
            </p>
          </div>
        </template>

        <!-- 高级选项：embedding / chat / vllm -->
        <div v-if="advancedVisible" class="space-y-3 rounded-lg border p-3">
          <p class="text-sm font-medium">高级选项</p>

          <template v-if="state.type === 'Embedding'">
            <div class="grid grid-cols-2 items-end gap-3">
              <div class="space-y-1.5">
                <Label>向量维度</Label>
                <div class="flex gap-1.5">
                  <Input
                    v-model="state.dimension"
                    type="number"
                    min="128"
                    max="4096"
                    :disabled="!state.supportsDimensionOverride && !state.dimension"
                    placeholder="例如：1536"
                  />
                  <Button
                    v-if="state.source === 'local'"
                    variant="outline"
                    :disabled="detecting"
                    @click="handleDetectDimension"
                  >
                    <Loader2Icon v-if="detecting" class="animate-spin" />
                    检测维度
                  </Button>
                </div>
                <p v-if="!state.supportsDimensionOverride && !state.dimension" class="text-xs text-muted-foreground">
                  开启"自定义输出维度"可手动填写；否则通过测试连接 / 检测维度自动获取。
                </p>
              </div>
              <div class="space-y-1.5 pb-1">
                <div class="flex items-center gap-2">
                  <Switch
                    id="dim-override"
                    :model-value="state.supportsDimensionOverride"
                    @update:model-value="(checked) => patch({ supportsDimensionOverride: checked === true })"
                  />
                  <Label for="dim-override" class="text-xs text-muted-foreground">
                    自定义输出维度
                  </Label>
                </div>
                <p class="text-xs text-muted-foreground">
                  仅在确认该模型支持 dimensions 参数时开启。
                </p>
              </div>
            </div>
          </template>

          <template v-if="state.type === 'KnowledgeQA'">
            <div class="flex items-center gap-2">
              <Switch
                id="supports-vision"
                :model-value="state.supportsVision"
                @update:model-value="(checked) => patch({ supportsVision: checked === true })"
              />
              <Label for="supports-vision" class="text-xs text-muted-foreground">
                支持视觉/多模态
              </Label>
            </div>
            <div v-if="state.source === 'remote'" class="space-y-1.5">
              <Label class="text-xs text-muted-foreground">思考模式参数格式</Label>
              <Select
                :model-value="state.thinkingControl"
                @update:model-value="
                  (value) => patch({ thinkingControl: value as ThinkingControl, thinkingManual: true })
                "
              >
                <SelectTrigger class="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="option in thinkingControlOptions" :key="option.value" :value="option.value">
                    <span class="flex flex-col">
                      <span>{{ option.label }}</span>
                      <span class="text-xs text-muted-foreground">{{ option.hint }}</span>
                    </span>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </template>

          <div class="space-y-1.5">
            <Label class="text-xs text-muted-foreground">后台并发上限</Label>
            <Input
              v-model="state.maxConcurrency"
              type="number"
              min="0"
              max="4096"
              placeholder="0 表示使用全局默认"
            />
            <p class="text-xs text-muted-foreground">
              限制文档入库/富化等后台任务对该模型的并发调用数（按模型全副本共享）。0 或留空表示沿用全局默认；不影响交互式对话。
            </p>
          </div>
        </div>
      </div>

      <SheetFooter class="flex-row items-center border-t">
        <div v-if="state.source === 'remote'" class="mr-auto flex min-w-0 flex-col items-start gap-0.5">
          <Button variant="outline" :disabled="testing || testDisabled" @click="handleTest">
            <Loader2Icon v-if="testing" class="animate-spin" />
            <PlugZapIcon
              v-else
              :class="testResult?.ok ? 'text-emerald-500' : testResult ? 'text-red-500' : ''"
            />
            {{ testing ? "测试中..." : "测试连接" }}
          </Button>
          <span
            v-if="testResult"
            :class="cn('max-w-56 truncate text-xs', testResult.ok ? 'text-emerald-600' : 'text-red-500')"
            :title="testResult.message"
          >
            {{ testResult.message }}
          </span>
        </div>
        <Button variant="outline" @click="handleOpenChange(false)">
          取消
        </Button>
        <Button :disabled="saving || saveDisabled" @click="handleSubmit">
          {{ saving ? "保存中..." : isEdit ? "保存" : "创建" }}
        </Button>
      </SheetFooter>
    </SheetContent>
  </Sheet>
</template>
