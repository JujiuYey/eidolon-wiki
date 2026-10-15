<script setup lang="ts">
import { Loader2Icon, SearchIcon } from "@lucide/vue";
import { ref } from "vue";

import { knowledgeApi, type KnowledgeChunk } from "@/api/wiki/knowledge";
import {
  matchTypeBadgeClass,
  matchTypeLabels,
  searchApi,
  type HybridSearchResult,
} from "@/api/wiki/search";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";

const props = defineProps<{
  kbId: string;
}>();

const query = ref("");
const submittedQuery = ref("");
const matchCount = ref(5);
const vectorThreshold = ref<string>("");
const keywordThreshold = ref<string>("");
const disableVector = ref(false);
const disableKeywords = ref(false);
const results = ref<HybridSearchResult[] | null>(null);
const searching = ref(false);
const error = ref<string | null>(null);

// 查看全文（chunkById 拉完整内容）
const viewTarget = ref<HybridSearchResult | null>(null);
const viewChunk = ref<KnowledgeChunk | null>(null);
const viewLoading = ref(false);

const parseThreshold = (value: string): number | undefined => {
  const parsed = Number(value);
  return value.trim() !== "" && !Number.isNaN(parsed) && parsed >= 0 && parsed <= 1 ? parsed : undefined;
};

const handleSearch = async () => {
  const keyword = query.value.trim();
  if (!keyword) return;
  searching.value = true;
  error.value = null;
  try {
    const res = await searchApi.hybridSearch(props.kbId, {
      query_text: keyword,
      match_count: matchCount.value,
      vector_threshold: parseThreshold(vectorThreshold.value),
      keyword_threshold: parseThreshold(keywordThreshold.value),
      disable_vector_match: disableVector.value,
      disable_keywords_match: disableKeywords.value,
    });
    results.value = res.data ?? [];
    submittedQuery.value = keyword;
  } catch (err) {
    results.value = null;
    error.value = (err as { message?: string }).message || "检索失败";
  } finally {
    searching.value = false;
  }
};

const openFullText = async (result: HybridSearchResult) => {
  viewTarget.value = result;
  viewChunk.value = null;
  viewLoading.value = true;
  try {
    const res = await knowledgeApi.chunkById(result.id);
    viewChunk.value = res.data ?? null;
  } catch (err) {
    // 拉不到全文时回退展示检索内容
    viewChunk.value = { id: result.id, content: result.content };
    void err;
  } finally {
    viewLoading.value = false;
  }
};
</script>

<template>
  <div class="flex h-full min-h-0 flex-col gap-4 overflow-y-auto pb-6">
    <div class="rounded-lg border p-4">
      <div class="flex gap-2">
        <Input
          v-model="query"
          placeholder="输入检索语句，回车或点击按钮执行"
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
          <Label class="text-xs text-muted-foreground">Top K</Label>
          <Input
            v-model.number="matchCount"
            type="number"
            min="1"
            max="50"
            class="h-8 w-20"
            @change="matchCount = Math.max(1, Number(matchCount) || 5)"
          />
        </div>
        <div class="flex items-center gap-2">
          <Label class="text-xs text-muted-foreground">向量阈值</Label>
          <Input
            v-model="vectorThreshold"
            type="number"
            min="0"
            max="1"
            step="0.05"
            placeholder="不设"
            class="h-8 w-20"
          />
        </div>
        <div class="flex items-center gap-2">
          <Label class="text-xs text-muted-foreground">关键词阈值</Label>
          <Input
            v-model="keywordThreshold"
            type="number"
            min="0"
            max="1"
            step="0.05"
            placeholder="不设"
            class="h-8 w-20"
          />
        </div>
        <div class="flex items-center gap-2">
          <Switch id="disable-vector" :model-value="disableVector" @update:model-value="v => (disableVector = v === true)" />
          <Label for="disable-vector" class="text-xs text-muted-foreground">纯关键词（关闭向量）</Label>
        </div>
        <div class="flex items-center gap-2">
          <Switch id="disable-keywords" :model-value="disableKeywords" @update:model-value="v => (disableKeywords = v === true)" />
          <Label for="disable-keywords" class="text-xs text-muted-foreground">纯向量（关闭关键词）</Label>
        </div>
      </div>
    </div>

    <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
    <div v-else-if="results" class="space-y-3">
      <p class="text-sm text-muted-foreground">「{{ submittedQuery }}」命中 {{ results.length }} 条</p>
      <p v-if="results.length === 0" class="py-8 text-center text-sm text-muted-foreground">没有检索到相关内容</p>
      <div v-for="result in results" v-else :key="result.id" class="rounded-lg border p-4">
        <div class="mb-2 flex flex-wrap items-center gap-2 text-xs">
          <Badge variant="secondary" :class="matchTypeBadgeClass(result.match_type)">
            {{ matchTypeLabels[result.match_type] ?? result.match_type }}
          </Badge>
          <Badge variant="outline">得分 {{ result.score.toFixed(4) }}</Badge>
          <span class="truncate text-muted-foreground">
            {{ result.knowledge_title || result.knowledge_filename || result.knowledge_id }}
            {{ result.chunk_index != null ? ` · chunk ${result.chunk_index}` : "" }}
          </span>
          <Button variant="ghost" size="xs" class="ml-auto" @click="openFullText(result)">
            查看全文
          </Button>
        </div>
        <div
          v-if="result.matched_content && result.matched_content !== result.content"
          class="mb-2 rounded-md bg-amber-50 px-3 py-2 dark:bg-amber-950/40"
        >
          <p class="mb-1 text-xs font-medium text-amber-700 dark:text-amber-400">命中片段</p>
          <p class="whitespace-pre-wrap break-words text-xs leading-relaxed">{{ result.matched_content }}</p>
        </div>
        <p class="whitespace-pre-wrap break-words text-sm leading-relaxed">{{ result.content }}</p>
      </div>
    </div>
    <p v-else class="py-10 text-center text-sm text-muted-foreground">
      输入检索语句，测试该知识库的向量 + 关键词混合检索效果
    </p>

    <!-- 查看全文 -->
    <Dialog :open="Boolean(viewTarget)" @update:open="open => !open && (viewTarget = null)">
      <DialogContent class="max-h-[80vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle class="pr-8">
            {{ viewTarget?.knowledge_title || viewTarget?.knowledge_filename || "分块全文" }}
          </DialogTitle>
          <DialogDescription v-if="viewTarget">
            chunk {{ viewTarget.chunk_index ?? "-" }} · 得分 {{ viewTarget.score.toFixed(4) }}
          </DialogDescription>
        </DialogHeader>
        <Skeleton v-if="viewLoading" class="h-40 w-full" />
        <p v-else class="whitespace-pre-wrap break-words text-sm leading-relaxed">
          {{ viewChunk?.content ?? viewTarget?.content }}
        </p>
      </DialogContent>
    </Dialog>
  </div>
</template>
