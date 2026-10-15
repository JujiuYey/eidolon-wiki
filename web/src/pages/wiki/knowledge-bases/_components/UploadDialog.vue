<script setup lang="ts">
import { CheckCircle2Icon, CircleIcon, Loader2Icon, UploadIcon } from "@lucide/vue";
import { computed, ref } from "vue";
import { toast } from "vue-sonner";

import { knowledgeApi } from "@/api/wiki/knowledge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

interface UploadTask {
  name: string;
  size: number;
  status: "pending" | "uploading" | "done" | "failed";
  error?: string;
}

const props = defineProps<{
  open: boolean;
  kbId: string;
  /** 上传目标文件夹（含路径前缀），根目录为空串 */
  folderPath: string;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
  uploaded: [];
}>();

const inputRef = ref<HTMLInputElement | null>(null);
const files = ref<File[]>([]);
const tasks = ref<UploadTask[]>([]);
const uploading = ref(false);

const pickFiles = () => inputRef.value?.click();

const handleFilesChosen = (event: Event) => {
  const input = event.target as HTMLInputElement;
  const chosen = input.files;
  if (!chosen || chosen.length === 0) return;
  const incoming = Array.from(chosen);
  // 追加而非替换（去重同名文件）
  const existing = new Set(files.value.map((file) => `${file.name}:${file.size}`));
  const fresh = incoming.filter((file) => !existing.has(`${file.name}:${file.size}`));
  if (fresh.length === 0) return;
  files.value = [...files.value, ...fresh];
  tasks.value = [
    ...tasks.value,
    ...fresh.map((file) => ({ name: file.name, size: file.size, status: "pending" as const })),
  ];
  input.value = "";
};

const handleUpload = async () => {
  uploading.value = true;
  let successCount = 0;
  for (let index = 0; index < files.value.length; index++) {
    const file = files.value[index];
    if (!file) continue;
    tasks.value = tasks.value.map((task, i) => (i === index ? { ...task, status: "uploading" } : task));
    try {
      // fileName 携带文件夹前缀，后端拆出 folder_path（官方 buildUploadFileName 语义）
      const fileName = props.folderPath ? `${props.folderPath}/${file.name}` : file.name;
      await knowledgeApi.upload(props.kbId, file, { fileName });
      tasks.value = tasks.value.map((task, i) => (i === index ? { ...task, status: "done" } : task));
      successCount++;
    } catch (error) {
      const message = (error as { message?: string }).message || "上传失败";
      tasks.value = tasks.value.map((task, i) => (i === index ? { ...task, status: "failed", error: message } : task));
    }
  }
  uploading.value = false;
  if (successCount > 0) {
    toast.success(`已上传 ${successCount} 个文档，解析已自动开始`);
    emit("uploaded");
  }
};

const handleClose = (next: boolean) => {
  if (!next) {
    tasks.value = [];
    files.value = [];
  }
  emit("update:open", next);
};

const hasTasks = computed(() => tasks.value.length > 0);
const allDone = computed(() => tasks.value.every((task) => task.status === "done" || task.status === "failed"));
</script>

<template>
  <Dialog :open="open" @update:open="handleClose">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>上传文档</DialogTitle>
        <DialogDescription>上传到「{{ folderPath || "根目录" }}」，上传成功后自动开始解析。</DialogDescription>
      </DialogHeader>

      <div class="space-y-3">
        <input ref="inputRef" type="file" multiple class="hidden" @change="handleFilesChosen" />

        <div v-if="hasTasks" class="max-h-64 space-y-2 overflow-y-auto rounded-md border p-2">
          <div v-for="(task, index) in tasks" :key="`${task.name}-${index}`" class="flex items-center gap-2 text-sm">
            <CheckCircle2Icon v-if="task.status === 'done'" class="h-4 w-4 shrink-0 text-emerald-500" />
            <Loader2Icon v-else-if="task.status === 'uploading'" class="h-4 w-4 shrink-0 animate-spin text-blue-500" />
            <CircleIcon
              :class="cn('h-4 w-4 shrink-0', task.status === 'failed' ? 'text-red-500' : 'text-muted-foreground')"
            />
            <span class="min-w-0 flex-1 truncate">{{ task.name }}</span>
            <span class="shrink-0 text-xs text-muted-foreground">
              {{
                task.status === "uploading"
                  ? "上传中"
                  : task.status === "done"
                    ? "已上传"
                    : task.status === "failed"
                      ? (task.error ?? "失败")
                      : `${(task.size / 1024).toFixed(0)} KB`
              }}
            </span>
          </div>
        </div>
        <button
          v-else
          type="button"
          class="flex w-full flex-col items-center gap-2 rounded-lg border border-dashed p-8 text-sm text-muted-foreground transition-colors hover:border-primary hover:text-foreground"
          @click="pickFiles"
        >
          <UploadIcon class="h-6 w-6" />
          点击选择文件（支持多选）
        </button>
      </div>

      <DialogFooter>
        <template v-if="hasTasks">
          <Button variant="outline" :disabled="uploading" @click="pickFiles">继续添加</Button>
          <Button :disabled="uploading || allDone" @click="handleUpload()">
            {{ uploading ? "上传中..." : allDone ? "已完成" : `上传 ${tasks.length} 个文件` }}
          </Button>
        </template>
        <Button v-else variant="outline" @click="handleClose(false)">取消</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
