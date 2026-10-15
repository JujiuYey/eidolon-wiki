<script setup lang="ts">
import { computed, ref } from "vue";
import { CopyIcon, LockIcon, MoreHorizontalIcon, PlusIcon, SquarePenIcon, Trash2Icon } from "@lucide/vue";
import { toast } from "vue-sonner";

import {
  isModelInUseError,
  modelApi,
  modelDisplayName,
  modelTypeBadgeClass,
  modelTypeLabels,
  providerLabel,
  useModelList,
  useModelProviders,
} from "@/api/ai/models";
import type { ModelInfo, ModelUsageDetails, ModelType } from "@/api/ai/models";
import { ConfirmDialog, SagPage } from "@/components/sag-ui";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { isSystemAdmin, useAuthStore } from "@/stores/auth";
import { cn } from "@/lib/utils";

import ModelEditorSheet from "./_components/ModelEditorSheet.vue";

const TYPE_TABS: Array<{ value: "all" | ModelType; label: string }> = [
  { value: "all", label: "全部" },
  { value: "KnowledgeQA", label: "对话" },
  { value: "Embedding", label: "向量" },
  { value: "Rerank", label: "重排" },
  { value: "VLLM", label: "多模态" },
  { value: "ASR", label: "语音" },
];

const bindingLabels: Record<string, string> = {
  embedding_model: "向量模型",
  summary_model: "摘要模型",
  image_processing_model: "图片处理模型",
  vlm_model: "视觉模型",
  asr_model: "语音模型",
  wiki_synthesis_model: "Wiki 合成模型",
  chat_model: "对话模型",
  rerank_model: "重排模型",
  query_understand_model: "查询理解模型",
  follow_up_model: "追问模型",
  extract_model: "抽取模型",
};

const { data: listData, isLoading: listLoading, refetch: refetchModels } = useModelList();
const { data: providersData } = useModelProviders();
const authStore = useAuthStore();

const typeTab = ref<"all" | ModelType>("all");
const editorOpen = ref(false);
const editingModel = ref<ModelInfo | null>(null);
const deleteTarget = ref<ModelInfo | null>(null);
const usageDetails = ref<ModelUsageDetails | null>(null);

const items = computed(() => {
  const all = listData.value ?? [];
  return typeTab.value === "all" ? all : all.filter((model) => model.type === typeTab.value);
});

const counts = computed(() => {
  const all = listData.value ?? [];
  const map = new Map<string, number>();
  for (const model of all) {
    map.set(model.type, (map.get(model.type) ?? 0) + 1);
  }
  return map;
});

const canManage = (model: ModelInfo) => (model.is_builtin ? isSystemAdmin(authStore) : true);

const openEditor = (model: ModelInfo | null) => {
  editingModel.value = model;
  editorOpen.value = true;
};

// 复制模型：深拷贝参数 + 名称去重（官方 copySuffix = " 副本" 递增），不复制密钥
const handleCopy = async (model: ModelInfo) => {
  if (model.is_builtin) {
    toast.warning("内置模型不能复制");
    return;
  }
  const baseName = modelDisplayName(model);
  const allNames = new Set((listData.value ?? []).map((item) => modelDisplayName(item)));
  let copyName = `${baseName} 副本`;
  let counter = 2;
  while (allNames.has(copyName)) {
    copyName = `${baseName} 副本 ${counter}`;
    counter++;
  }
  try {
    await modelApi.create({
      name: model.name,
      display_name: copyName,
      type: model.type,
      source: model.source === "local" ? "local" : "remote",
      description: model.description,
      parameters: JSON.parse(JSON.stringify(model.parameters)) as typeof model.parameters,
    });
    toast.success("模型已复制");
    await refetchModels();
  } catch (error) {
    toast.error((error as { message?: string }).message || "复制模型失败");
  }
};

const handleDelete = async () => {
  if (!deleteTarget.value) return;
  try {
    await modelApi.remove(deleteTarget.value.id);
    toast.success("模型已删除");
    await refetchModels();
  } catch (error) {
    const details = isModelInUseError(error);
    if (details) {
      usageDetails.value = details;
    } else {
      toast.error((error as { message?: string }).message || "删除模型失败");
    }
  }
};
</script>

<template>
  <SagPage>
    <template #title>模型配置</template>
    <template #description>
      配置对话、向量、重排等模型。对话模型（KnowledgeQA）用于 RAG 问答，向量模型（Embedding）用于知识库索引。
    </template>
    <template #actions>
      <Button @click="openEditor(null)">
        <PlusIcon />
        添加模型
      </Button>
    </template>

    <div class="flex h-full min-h-0 flex-col gap-4 overflow-y-auto pb-6">
      <div class="flex w-fit flex-wrap gap-1 rounded-lg border p-1">
        <button
          v-for="tab in TYPE_TABS"
          :key="tab.value"
          type="button"
          :class="
            cn(
              'rounded-md px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground',
              typeTab === tab.value && 'bg-muted font-medium text-foreground',
            )
          "
          @click="typeTab = tab.value"
        >
          {{ tab.label }}
          <span v-if="tab.value !== 'all' && counts.get(tab.value)" class="ml-1 text-xs">
            {{ counts.get(tab.value) }}
          </span>
        </button>
      </div>

      <div v-if="listLoading" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <Skeleton v-for="index in 6" :key="index" class="h-32 w-full" />
      </div>
      <Empty v-else-if="items.length === 0" class="rounded-lg border">
        <EmptyHeader>
          <EmptyTitle>还没有模型</EmptyTitle>
          <EmptyDescription>添加一个对话模型后，RAG 问答的模型选择器即可使用。</EmptyDescription>
        </EmptyHeader>
      </Empty>
      <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <Card v-for="model in items" :key="model.id" class="py-4">
          <CardHeader class="px-4">
            <CardTitle class="flex items-start justify-between gap-2 text-base font-semibold">
              <span class="flex min-w-0 items-center gap-2">
                <span class="truncate">{{ modelDisplayName(model) }}</span>
                <LockIcon v-if="model.is_builtin" class="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                <Badge v-if="model.is_default" variant="outline">默认</Badge>
              </span>
              <span v-if="canManage(model)" class="shrink-0" @click.stop>
                <DropdownMenu>
                  <DropdownMenuTrigger as-child>
                    <Button variant="ghost" size="icon-sm" :aria-label="`模型 ${modelDisplayName(model)} 操作菜单`">
                      <MoreHorizontalIcon />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem @click="openEditor(model)">
                      <SquarePenIcon />
                      编辑
                    </DropdownMenuItem>
                    <DropdownMenuItem @click="handleCopy(model)">
                      <CopyIcon />
                      复制
                    </DropdownMenuItem>
                    <DropdownMenuItem v-if="!model.is_builtin" variant="destructive" @click="deleteTarget = model">
                      <Trash2Icon />
                      删除
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </span>
            </CardTitle>
          </CardHeader>
          <CardContent class="space-y-2.5 px-4">
            <div class="flex flex-wrap items-center gap-1.5">
              <Badge variant="secondary" :class="modelTypeBadgeClass[model.type]">
                {{ modelTypeLabels[model.type] }}
              </Badge>
              <Badge variant="outline">{{ providerLabel(model, providersData ?? []) }}</Badge>
              <Badge
                v-if="model.type === 'Embedding' && model.parameters.embedding_parameters?.dimension"
                variant="outline"
              >
                维度 {{ model.parameters.embedding_parameters.dimension }}
              </Badge>
              <Badge v-if="model.type === 'KnowledgeQA' && model.parameters.supports_vision" variant="outline">
                多模态
              </Badge>
            </div>
            <p class="truncate font-mono text-xs text-muted-foreground" :title="model.name">
              {{ model.name }}{{ model.parameters.base_url ? ` · ${model.parameters.base_url}` : "" }}
            </p>
            <p v-if="model.description" class="line-clamp-2 text-sm text-muted-foreground">
              {{ model.description }}
            </p>
          </CardContent>
        </Card>
      </div>
    </div>

    <ModelEditorSheet
      :open="editorOpen"
      :providers="providersData ?? []"
      :model="editingModel"
      @update:open="editorOpen = $event"
    />

    <ConfirmDialog
      :open="Boolean(deleteTarget)"
      destructive
      title="删除模型"
      :description="`确定删除模型「${deleteTarget ? modelDisplayName(deleteTarget) : ''}」？被引用的模型会删除失败并提示占用详情。`"
      confirm-text="删除"
      @update:open="
        (open) => {
          if (!open) deleteTarget = null;
        }
      "
      @confirm="handleDelete"
    />

    <Dialog
      :open="Boolean(usageDetails)"
      @update:open="
        (open) => {
          if (!open) usageDetails = null;
        }
      "
    >
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>模型正在被使用，无法删除</DialogTitle>
          <DialogDescription>请先在以下资源中解除引用，再删除该模型。</DialogDescription>
        </DialogHeader>
        <div class="max-h-[50vh] space-y-4 overflow-y-auto">
          <div v-if="(usageDetails?.knowledge_bases?.length ?? 0) > 0">
            <p class="mb-1.5 text-sm font-semibold">
              知识库（{{ usageDetails?.knowledge_base_total ?? usageDetails?.knowledge_bases?.length }}）
            </p>
            <ul class="space-y-1">
              <li
                v-for="kb in usageDetails?.knowledge_bases"
                :key="kb.id"
                class="flex items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm"
              >
                <span class="truncate">{{ kb.name }}</span>
                <span class="shrink-0 text-xs text-muted-foreground">
                  {{ kb.bindings.map((binding) => bindingLabels[binding] ?? binding).join("、") }}
                </span>
              </li>
            </ul>
          </div>
          <div v-if="(usageDetails?.agents?.length ?? 0) > 0">
            <p class="mb-1.5 text-sm font-semibold">
              智能体（{{ usageDetails?.agent_total ?? usageDetails?.agents?.length }}）
            </p>
            <ul class="space-y-1">
              <li
                v-for="agent in usageDetails?.agents"
                :key="agent.id"
                class="flex items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm"
              >
                <span class="truncate">{{ agent.name }}</span>
                <span class="shrink-0 text-xs text-muted-foreground">
                  {{ agent.bindings.map((binding) => bindingLabels[binding] ?? binding).join("、") }}
                </span>
              </li>
            </ul>
          </div>
          <p v-if="usageDetails?.long_term_memory?.bindings?.length" class="text-sm text-muted-foreground">
            长期记忆配置引用：{{ usageDetails.long_term_memory.bindings.map((binding) => bindingLabels[binding] ?? binding).join("、") }}
          </p>
        </div>
      </DialogContent>
    </Dialog>
  </SagPage>
</template>
