<script setup lang="ts">
import { computed, ref, watch } from "vue";

import type { KnowledgeReference } from "@/api/ai/chat";
import { knowledgeApi } from "@/api/wiki/knowledge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";

/**
 * 引用分块全文弹窗：正文引用药丸与回答底部来源卡片共用。
 * 传入 reference 即打开，优先拉分块全文，失败退回检索片段（参考实现 ReferenceList.openChunk）。
 */
const props = defineProps<{
  reference: KnowledgeReference | null;
}>();

const emit = defineEmits<{
  close: [];
}>();

const content = ref("");
const loading = ref(false);

watch(
  () => props.reference,
  async (reference) => {
    content.value = "";
    if (!reference) return;
    loading.value = true;
    try {
      const res = await knowledgeApi.chunkById(reference.id);
      content.value = res.data?.content || reference.content;
    } catch {
      // 拉不到全文时退回检索片段
      content.value = reference.content;
    } finally {
      loading.value = false;
    }
  },
);

const title = computed(() => {
  if (!props.reference) return "";
  const doc = props.reference.knowledge_title || props.reference.knowledge_filename || "未知文档";
  return props.reference.chunk_index != null ? `${doc} · 分块 ${props.reference.chunk_index}` : doc;
});
</script>

<template>
  <Dialog :open="Boolean(reference)" @update:open="(open) => !open && emit('close')">
    <DialogContent class="sm:max-w-2xl">
      <DialogHeader>
        <DialogTitle class="pr-6 text-base">{{ title }}</DialogTitle>
        <DialogDescription>引用分块全文</DialogDescription>
      </DialogHeader>
      <div class="max-h-[60vh] overflow-y-auto rounded-md bg-muted/50 p-4">
        <div v-if="loading" class="flex items-center justify-center gap-2 py-8 text-sm text-muted-foreground">
          <Spinner class="h-4 w-4" />
          加载全文…
        </div>
        <p v-else class="whitespace-pre-wrap break-words text-sm leading-relaxed">{{ content }}</p>
      </div>
      <Button variant="outline" @click="emit('close')">
        关闭
      </Button>
    </DialogContent>
  </Dialog>
</template>
