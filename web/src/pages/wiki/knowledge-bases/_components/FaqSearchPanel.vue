<script setup lang="ts">
import { Loader2Icon, SearchIcon } from "@lucide/vue";
import { ref } from "vue";

import {
  faqApi,
  faqMatchTypeBadgeClass,
  faqMatchTypeLabels,
  type FAQEntry,
} from "@/api/wiki/faq";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";

const props = defineProps<{
  kbId: string;
}>();

const query = ref("");
const vectorThreshold = ref(0.7);
const matchCount = ref(10);
const onlyRecommended = ref(false);
const results = ref<FAQEntry[] | null>(null);
const searching = ref(false);
const error = ref<string | null>(null);

const handleSearch = async () => {
  const keyword = query.value.trim();
  if (!keyword) return;
  searching.value = true;
  error.value = null;
  try {
    const res = await faqApi.search(props.kbId, {
      query_text: keyword,
      vector_threshold: vectorThreshold.value,
      match_count: Math.min(50, Math.max(1, matchCount.value)),
      only_recommended: onlyRecommended.value,
    });
    results.value = res.data ?? [];
  } catch (err) {
    results.value = null;
    error.value = (err as { message?: string }).message || "检索失败";
  } finally {
    searching.value = false;
  }
};
</script>

<template>
  <div class="flex h-full min-h-0 flex-col gap-4 overflow-y-auto pb-6">
    <div class="rounded-lg border p-4">
      <div class="flex gap-2">
        <Input
          v-model="query"
          placeholder="用用户的问法提问，测试 FAQ 命中效果"
          @keydown.enter="handleSearch()"
        />
        <Button :disabled="searching || !query.trim()" @click="handleSearch()">
          <Loader2Icon v-if="searching" class="animate-spin" />
          <SearchIcon v-else />
          {{ searching ? "检索中" : "检索" }}
        </Button>
      </div>
      <div class="mt-3 flex flex-wrap items-center gap-x-6 gap-y-3">
        <div class="flex items-center gap-2">
          <Label class="text-xs text-muted-foreground">向量阈值</Label>
          <Input v-model.number="vectorThreshold" type="number" min="0" max="1" step="0.05" class="h-8 w-24" />
        </div>
        <div class="flex items-center gap-2">
          <Label class="text-xs text-muted-foreground">Top K（上限 50）</Label>
          <Input v-model.number="matchCount" type="number" min="1" max="50" class="h-8 w-20" />
        </div>
        <div class="flex items-center gap-2">
          <Switch id="faq-only-recommended" :model-value="onlyRecommended" @update:model-value="v => (onlyRecommended = v === true)" />
          <Label for="faq-only-recommended" class="text-xs text-muted-foreground">仅推荐位条目</Label>
        </div>
      </div>
    </div>

    <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
    <div v-else-if="results" class="space-y-3">
      <p class="text-sm text-muted-foreground">「{{ query.trim() }}」命中 {{ results.length }} 条</p>
      <p v-if="results.length === 0" class="py-8 text-center text-sm text-muted-foreground">
        没有命中的 FAQ 条目
      </p>
      <div v-for="entry in results" v-else :key="entry.id" class="rounded-lg border p-4">
        <div class="mb-2 flex flex-wrap items-center gap-2 text-xs">
          <Badge variant="secondary" :class="faqMatchTypeBadgeClass(entry.match_type)">
            {{ faqMatchTypeLabels[entry.match_type ?? 0] ?? "向量" }}
          </Badge>
          <Badge variant="outline">得分 {{ (entry.score ?? 0).toFixed(4) }}</Badge>
          <Badge v-if="entry.tag_name" variant="outline">{{ entry.tag_name }}</Badge>
          <span v-if="entry.matched_question" class="truncate text-muted-foreground">
            命中：{{ entry.matched_question }}
          </span>
        </div>
        <p class="font-medium">{{ entry.standard_question }}</p>
        <div class="mt-2 space-y-1.5">
          <p
            v-for="(answer, index) in entry.answers"
            :key="index"
            class="whitespace-pre-wrap break-words text-sm leading-relaxed text-muted-foreground"
          >
            {{ answer }}
          </p>
        </div>
      </div>
    </div>
    <p v-else class="py-10 text-center text-sm text-muted-foreground">
      FAQ 检索按问题向量匹配（当前禁用关键词匹配），返回启用的条目
    </p>
  </div>
</template>
