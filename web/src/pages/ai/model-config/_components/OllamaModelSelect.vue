<script setup lang="ts">
import { computed, onUnmounted, ref } from "vue";
import { CheckIcon, ChevronsUpDownIcon, DownloadIcon, RefreshCwIcon } from "@lucide/vue";
import { toast } from "vue-sonner";

import { modelApi } from "@/api/ai/models";
import { Button } from "@/components/ui/button";
import { Command, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

const props = defineProps<{
  value: string;
}>();

const emit = defineEmits<{
  "update:value": [name: string];
}>();

interface OllamaModel {
  name: string;
  size: number;
}

const DOWNLOAD_PREFIX = "__download__";

function formatModelSize(bytes: number): string {
  if (bytes >= 1024 ** 3) return `${(bytes / 1024 ** 3).toFixed(1)} GB`;
  return `${(bytes / 1024 ** 2).toFixed(0)} MB`;
}

const open = ref(false);
const models = ref<OllamaModel[]>([]);
const loading = ref(false);
const keyword = ref("");
const downloadTask = ref<{ id: string; name: string; progress: number } | null>(null);
let pollTimer: number | undefined;

const loadModels = async () => {
  loading.value = true;
  try {
    const res = await modelApi.ollamaModels();
    models.value = res.data?.models ?? [];
  } catch {
    models.value = [];
  } finally {
    loading.value = false;
  }
};

const stopPolling = () => {
  if (pollTimer) {
    window.clearInterval(pollTimer);
    pollTimer = undefined;
  }
};

onUnmounted(stopPolling);

const startDownload = async (modelName: string) => {
  try {
    const res = await modelApi.ollamaDownload(modelName);
    const taskId = res.data?.taskId;
    if (!taskId) {
      toast.error("下载任务创建失败");
      return;
    }
    downloadTask.value = { id: taskId, name: modelName, progress: 0 };
    stopPolling();
    pollTimer = window.setInterval(async () => {
      try {
        const progress = await modelApi.ollamaDownloadProgress(taskId);
        const status = progress.data?.status;
        const percent = progress.data?.progress ?? 0;
        if (status === "completed") {
          stopPolling();
          downloadTask.value = null;
          toast.success(`${modelName} 下载完成`);
          await loadModels();
          emit("update:value", modelName);
        } else if (status === "failed") {
          stopPolling();
          downloadTask.value = null;
          toast.error(progress.data?.message || `${modelName} 下载失败`);
        } else {
          downloadTask.value = downloadTask.value ? { ...downloadTask.value, progress: percent } : null;
        }
      } catch {
        // 单次轮询失败忽略，下一轮重试
      }
    }, 1000);
  } catch (error) {
    toast.error((error as { message?: string }).message || "下载任务创建失败");
  }
};

const handleSelect = (selected: string) => {
  if (selected.startsWith(DOWNLOAD_PREFIX)) {
    void startDownload(selected.slice(DOWNLOAD_PREFIX.length));
    return;
  }
  emit("update:value", selected);
  open.value = false;
};

const filtered = computed(() =>
  models.value.filter((model) => model.name.toLowerCase().includes(keyword.value.toLowerCase())),
);
const exactMatch = computed(() =>
  models.value.some((model) => model.name.toLowerCase() === keyword.value.toLowerCase()),
);
const showDownloadOption = computed(
  () => keyword.value.trim() !== "" && !exactMatch.value && !downloadTask.value,
);
</script>

<template>
  <!-- Ollama 本地模型可搜索下拉（官方语义）：
       打开时拉取已下载模型；搜索词无精确匹配时出现"下载: {keyword}"项，
       触发下载并每秒轮询进度，完成后自动选中新模型。 -->
  <div class="flex gap-1.5">
    <Popover
      :open="open"
      @update:open="
        (next) => {
          open = next;
          if (next) {
            void loadModels();
            keyword = '';
          }
        }
      "
    >
      <PopoverTrigger as-child>
        <Button variant="outline" class="flex-1 justify-between font-normal">
          <span class="truncate">{{ props.value || "搜索或选择本地模型" }}</span>
          <ChevronsUpDownIcon class="opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent class="w-(--reka-popover-trigger-width) p-0" align="start">
        <Command>
          <CommandInput
            placeholder="搜索模型..."
            :model-value="keyword"
            @update:model-value="(text: string) => (keyword = text)"
          />
          <CommandList>
            <div v-if="downloadTask" class="px-3 py-2 text-xs text-muted-foreground">
              正在下载 {{ downloadTask.name }}… {{ downloadTask.progress.toFixed(1) }}%
            </div>
            <template v-for="model in filtered" :key="model.name">
              <CommandItem :value="model.name" @select="handleSelect(model.name)">
                <CheckIcon :class="cn('mr-1 h-4 w-4', props.value === model.name ? 'opacity-100' : 'opacity-0')" />
                <span class="flex-1 truncate">{{ model.name }}</span>
                <span class="text-xs text-muted-foreground">{{ formatModelSize(model.size) }}</span>
              </CommandItem>
            </template>
            <CommandItem
              v-if="showDownloadOption"
              :value="`${DOWNLOAD_PREFIX}${keyword}`"
              class="text-primary"
              @select="handleSelect(`${DOWNLOAD_PREFIX}${keyword}`)"
            >
              <DownloadIcon class="mr-1 h-4 w-4" />
              下载: {{ keyword }}
            </CommandItem>
            <div
              v-if="!loading && filtered.length === 0 && !showDownloadOption && !downloadTask"
              class="px-3 py-4 text-center text-xs text-muted-foreground"
            >
              {{ keyword ? "没有匹配的本地模型" : "Ollama 上还没有已下载的模型" }}
            </div>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
    <Button
      variant="outline"
      size="icon"
      aria-label="刷新 Ollama 模型列表"
      @click="
        () => {
          void loadModels();
          toast.success('列表已刷新');
        }
      "
    >
      <RefreshCwIcon />
    </Button>
  </div>
</template>
