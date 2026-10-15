<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";

import { useModelList } from "@/api/ai/models";
import type { ModelInfo } from "@/api/ai/models";
import {
  useCreateKnowledgeBase,
  useUpdateKBModelConfig,
  useUpdateKnowledgeBase,
  type ChunkingConfig,
  type KnowledgeBase,
} from "@/api/wiki/knowledge-base";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";

// 结构参考官方 KnowledgeBaseEditorModal（basic/models/chunking section）；
// 编辑态双通道保存：PUT /knowledge-bases/:id（基本+分块）+ PUT /initialization/config/:kbId（模型+分块）。
const props = defineProps<{
  open: boolean;
  /** 传入则为编辑模式 */
  kb?: KnowledgeBase | null;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
}>();

const createKb = useCreateKnowledgeBase();
const updateKb = useUpdateKnowledgeBase();
const updateModelConfig = useUpdateKBModelConfig();
const isEdit = computed(() => Boolean(props.kb));
const pending = computed(
  () => createKb.isPending.value || updateKb.isPending.value || updateModelConfig.isPending.value,
);

// ---- 模型列表（GET /models 全量，前端按 type 过滤；总结模型即 KnowledgeQA 类型）----

const modelsQuery = useModelList();
const llmModels = computed(() =>
  (modelsQuery.data.value ?? []).filter((m) => m.type === "KnowledgeQA"),
);
const embeddingModels = computed(() =>
  (modelsQuery.data.value ?? []).filter((m) => m.type === "Embedding"),
);

const modelLabel = (m: ModelInfo) =>
  `${m.display_name || m.name}${m.is_default ? "（默认）" : ""}`;
const defaultModelId = (list: ModelInfo[]) => list.find((m) => m.is_default)?.id ?? "";

// ---- 表单状态 ----

const name = ref("");
const description = ref("");
// 编辑时锁定（后端不允许改型）；faq 型 KB 不走文档切分
const kbType = ref<"document" | "faq">("document");
// 解析管线 Wiki 自动生成页（indexing_strategy.wiki_enabled）
const wikiEnabled = ref(false);
const embeddingModelId = ref("");
const llmModelId = ref("");

// 新建默认值（对齐官方：auto 策略、512/80、父子分块开）
const defaultChunking = () => ({
  strategy: "auto",
  chunk_size: 512,
  chunk_overlap: 80,
  enable_parent_child: true,
  parent_chunk_size: 4096,
  child_chunk_size: 384,
});

const chunking = reactive(defaultChunking());
// 未在 UI 暴露的 chunking 字段（separators/token_limit 等）编辑时原样带回，避免被覆盖丢失
let rawChunking: ChunkingConfig | undefined;
// 覆盖式端点的保底回传字段（编辑态从实体取原值）
let preserve: {
  vlm_config?: KnowledgeBase["vlm_config"];
  asr_config?: KnowledgeBase["asr_config"];
  storage_provider_config?: KnowledgeBase["storage_provider_config"];
  extract_config?: KnowledgeBase["extract_config"];
  question_generation_config?: KnowledgeBase["question_generation_config"];
} = {};

const hasFiles = computed(() => (props.kb?.knowledge_count ?? 0) > 0);
const embeddingLocked = computed(() => isEdit.value && hasFiles.value);

watch(
  () => [props.open, props.kb] as const,
  ([open, kb]) => {
    if (!open) return;
    name.value = kb?.name ?? "";
    description.value = kb?.description ?? "";
    kbType.value = kb?.type === "faq" ? "faq" : "document";
    wikiEnabled.value = kb?.indexing_strategy?.wiki_enabled === true;
    llmModelId.value = kb?.summary_model_id ?? "";
    embeddingModelId.value = kb?.embedding_model_id ?? "";
    rawChunking = kb?.chunking_config ? { ...kb.chunking_config } : undefined;
    Object.assign(chunking, defaultChunking(), {
      strategy: kb?.chunking_config?.strategy ?? "auto",
      chunk_size: kb?.chunking_config?.chunk_size ?? 512,
      chunk_overlap: kb?.chunking_config?.chunk_overlap ?? 80,
      enable_parent_child: kb?.chunking_config?.enable_parent_child ?? true,
      parent_chunk_size: kb?.chunking_config?.parent_chunk_size ?? 4096,
      child_chunk_size: kb?.chunking_config?.child_chunk_size ?? 384,
    });
    preserve = {
      vlm_config: kb?.vlm_config,
      asr_config: kb?.asr_config,
      storage_provider_config: kb?.storage_provider_config,
      extract_config: kb?.extract_config,
      question_generation_config: kb?.question_generation_config,
    };
    // 创建模式预选默认模型
    if (!kb) {
      llmModelId.value = defaultModelId(llmModels.value);
      embeddingModelId.value = defaultModelId(embeddingModels.value);
    }
  },
  { immediate: true },
);

// 模型列表异步到达后，创建模式仍无选择时补默认值
watch(modelsQuery.data, (models) => {
  if (!props.open || props.kb) return;
  const list = models ?? [];
  if (!llmModelId.value) llmModelId.value = defaultModelId(list.filter((m) => m.type === "KnowledgeQA"));
  if (!embeddingModelId.value) {
    embeddingModelId.value = defaultModelId(list.filter((m) => m.type === "Embedding"));
  }
});

const chunkingPayload = (): ChunkingConfig => ({
  ...rawChunking,
  strategy: chunking.strategy,
  chunk_size: Number(chunking.chunk_size) || 0,
  chunk_overlap: Number(chunking.chunk_overlap) || 0,
  enable_parent_child: chunking.enable_parent_child,
  parent_chunk_size: Number(chunking.parent_chunk_size) || 0,
  child_chunk_size: Number(chunking.child_chunk_size) || 0,
});

const validationError = (): string | null => {
  if (!name.value.trim()) return "请填写知识库名称";
  if (!llmModelId.value) return "请选择 LLM 大语言模型";
  if (!embeddingModelId.value) return "请选择 Embedding 嵌入模型";
  return null;
};

const formError = ref("");
const overlapWarning = computed(
  () => Number(chunking.chunk_overlap) >= Number(chunking.chunk_size) / 2,
);

const handleSubmit = async () => {
  const error = validationError();
  formError.value = error ?? "";
  if (error) return;
  const payload = {
    llmModelId: llmModelId.value,
    embeddingModelId: embeddingModelId.value,
    documentSplitting: {
      chunkSize: Number(chunking.chunk_size) || 0,
      chunkOverlap: Number(chunking.chunk_overlap) || 0,
      enableParentChild: chunking.enable_parent_child,
      parentChunkSize: Number(chunking.parent_chunk_size) || 0,
      childChunkSize: Number(chunking.child_chunk_size) || 0,
      strategy: chunking.strategy,
    },
    ...preserve,
  };
  if (props.kb) {
    await updateKb.mutateAsync({
      id: props.kb.id,
      data: {
        name: name.value.trim(),
        description: description.value.trim(),
        config: {
          chunking_config: chunkingPayload(),
          // indexing_strategy 是整体覆盖语义：必须带上原值，否则 vector/keyword 索引会被关掉
          indexing_strategy: {
            // FAQ 库的向量索引是检索路由的硬前提（后端按此决定是否查询 FAQ 索引），
            // 历史 bug 可能落库为 false，保存时强制拉起
            vector_enabled:
              kbType.value === "faq" ? true : (props.kb.indexing_strategy?.vector_enabled ?? true),
            keyword_enabled: props.kb.indexing_strategy?.keyword_enabled ?? true,
            graph_enabled: props.kb.indexing_strategy?.graph_enabled ?? false,
            wiki_enabled: wikiEnabled.value,
          },
        },
      },
    });
    await updateModelConfig.mutateAsync({ kbId: props.kb.id, data: payload });
  } else {
    await createKb.mutateAsync({
      name: name.value.trim(),
      description: description.value.trim() || undefined,
      type: kbType.value,
      embedding_model_id: embeddingModelId.value,
      summary_model_id: llmModelId.value,
      chunking_config: kbType.value === "document" ? chunkingPayload() : undefined,
      indexing_strategy:
        kbType.value === "document"
          ? { vector_enabled: true, keyword_enabled: true, wiki_enabled: wikiEnabled.value }
          : // FAQ 库必须显式带 vector_enabled:true：后端 EnsureDefaults 对 FAQ 型
            // 在请求未含 faq_config 时会跳过索引策略默认值，落库全 false 会导致
            // 聊天/FAQ 检索路由把整个库静默跳过
            { vector_enabled: true, keyword_enabled: false, wiki_enabled: false },
    });
  }
  emit("update:open", false);
};
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{{ isEdit ? "编辑知识库" : "新建知识库" }}</DialogTitle>
          <DialogDescription>
            {{
              isEdit
                ? "修改基本信息、模型与切分配置。已有文件的知识库不能更换 Embedding 模型。"
                : kbType === "faq"
                  ? "创建 FAQ 知识库：用「标准问题 + 答案」的条目组织知识，适合客服、制度类问答。"
                  : "创建文档型知识库，上传的文档会自动进入解析流水线。"
            }}
          </DialogDescription>
        </DialogHeader>

      <div class="space-y-5">
        <!-- 基本信息 -->
        <div class="space-y-4">
          <h3 class="text-sm font-semibold">基本信息</h3>
          <div class="grid gap-4 sm:grid-cols-[1fr_200px]">
            <div class="space-y-2">
              <Label for="kb-name">名称</Label>
              <Input id="kb-name" v-model="name" placeholder="例如：产品文档库" />
            </div>
            <div class="space-y-2">
              <Label for="kb-type">类型</Label>
              <Select
                v-model="kbType"
                :disabled="isEdit"
              >
                <SelectTrigger id="kb-type" class="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="document">文档库</SelectItem>
                  <SelectItem value="faq">FAQ 问答库</SelectItem>
                </SelectContent>
              </Select>
              <p v-if="isEdit" class="text-xs text-muted-foreground">类型创建后不可更改。</p>
            </div>
          </div>
          <div class="space-y-2">
            <Label for="kb-description">描述（可选）</Label>
            <Textarea id="kb-description" v-model="description" :rows="2" placeholder="这个知识库用来放什么" />
          </div>
          <div v-if="kbType === 'document'" class="flex items-center justify-between rounded-lg border px-3 py-2.5">
            <div>
              <Label>解析时生成 Wiki 页面</Label>
              <p class="text-xs text-muted-foreground">
                开启后，解析管线会自动把文档内容整理成相互链接的 Wiki 页面（影响之后解析的文档）。详情页会出现「Wiki」页签。
              </p>
            </div>
            <Switch v-model:checked="wikiEnabled" />
          </div>
        </div>

        <Separator />

        <!-- 模型 -->
        <div class="space-y-4">
          <div>
            <h3 class="text-sm font-semibold">模型</h3>
            <p class="mt-1 text-xs text-muted-foreground">
              创建后可在模型配置页添加更多模型。Embedding 模型决定检索向量化方式，导入文件后不可更换。
            </p>
          </div>
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-2">
              <Label>LLM 大语言模型 <span class="text-destructive">*</span></Label>
              <Select v-model="llmModelId">
                <SelectTrigger class="w-full">
                  <SelectValue placeholder="用于总结与摘要" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="m in llmModels" :key="m.id" :value="m.id">
                    {{ modelLabel(m) }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-2">
              <Label>Embedding 嵌入模型 <span class="text-destructive">*</span></Label>
              <Select v-model="embeddingModelId" :disabled="embeddingLocked">
                <SelectTrigger class="w-full">
                  <SelectValue placeholder="用于向量化检索" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="m in embeddingModels" :key="m.id" :value="m.id">
                    {{ modelLabel(m) }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <p v-if="embeddingLocked" class="text-xs text-amber-600 dark:text-amber-400">
                知识库中已有文件，无法修改 Embedding 模型。
              </p>
            </div>
          </div>
        </div>

        <Separator />

        <!-- 切分配置（仅文档型；FAQ 条目不走文档切分） -->
        <div v-if="kbType === 'document'" class="space-y-4">
          <div>
            <h3 class="text-sm font-semibold">切分配置</h3>
            <p class="mt-1 text-xs text-muted-foreground">控制文档解析后的分块方式，只影响之后解析的文档。</p>
          </div>
          <div class="grid gap-4 sm:grid-cols-3">
            <div class="space-y-2">
              <Label>分块策略</Label>
              <Select v-model="chunking.strategy">
                <SelectTrigger class="w-full">
                  <SelectValue placeholder="按长度切分" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="auto">自动</SelectItem>
                  <SelectItem value="heading">按标题切分</SelectItem>
                  <SelectItem value="heuristic">结构感知</SelectItem>
                  <SelectItem value="recursive">按长度切分</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-2">
              <Label for="kb-chunk-size">分块大小（字符）</Label>
              <Input id="kb-chunk-size" v-model="chunking.chunk_size" type="number" :min="100" :max="4000" />
            </div>
            <div class="space-y-2">
              <Label for="kb-chunk-overlap">分块重叠（字符）</Label>
              <Input id="kb-chunk-overlap" v-model="chunking.chunk_overlap" type="number" :min="0" :max="500" />
            </div>
          </div>
          <div class="flex items-center justify-between rounded-lg border px-3 py-2.5">
            <div>
              <Label>父子分块</Label>
              <p class="text-xs text-muted-foreground">用大块提供上下文、小块做精确检索，提升问答效果。</p>
            </div>
            <Switch v-model:checked="chunking.enable_parent_child" />
          </div>
          <div v-if="chunking.enable_parent_child" class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-2">
              <Label for="kb-parent-size">父块大小（字符）</Label>
              <Input id="kb-parent-size" v-model="chunking.parent_chunk_size" type="number" :min="512" :max="8192" />
            </div>
            <div class="space-y-2">
              <Label for="kb-child-size">子块大小（字符）</Label>
              <Input id="kb-child-size" v-model="chunking.child_chunk_size" type="number" :min="64" :max="2048" />
            </div>
          </div>
        </div>

        <p v-if="overlapWarning" class="text-xs text-amber-600 dark:text-amber-400">
          分块重叠建议小于分块大小的一半，过大会被后端自动截断。
        </p>
        <p v-if="formError" class="text-sm text-destructive">{{ formError }}</p>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)">取消</Button>
        <Button :disabled="pending" @click="handleSubmit()">
          {{ pending ? "保存中..." : isEdit ? "保存" : "创建" }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
