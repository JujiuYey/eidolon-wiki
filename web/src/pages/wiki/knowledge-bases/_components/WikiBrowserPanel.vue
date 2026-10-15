<script setup lang="ts">
import { HistoryIcon, PlusIcon, SquarePenIcon, Trash2Icon } from "@lucide/vue";
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { toast } from "vue-sonner";

import {
  useDeleteWikiPage,
  useRevertWikiPage,
  useWikiPages,
  useWikiRevisions,
  wikiPageApi,
  type WikiPage,
} from "@/api/wiki/wiki-page";
import { ConfirmDialog } from "@/components/sag-ui";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";

const props = defineProps<{
  kbId: string;
}>();

const router = useRouter();

const pagesQuery = useWikiPages(() => props.kbId);
const pages = computed(() => pagesQuery.data.value ?? []);
// 更新时间倒序
const sortedPages = computed(() =>
  [...pages.value].sort((a, b) => (b.updated_at ?? "").localeCompare(a.updated_at ?? "")),
);

const deleteWikiPage = useDeleteWikiPage(() => props.kbId);
const revertWikiPage = useRevertWikiPage(() => props.kbId);

// ---- 新建页面 ----

const createOpen = ref(false);
const newTitle = ref("");
const creating = ref(false);
const deleteTarget = ref<WikiPage | null>(null);

const handleDelete = async () => {
  if (!deleteTarget.value) return;
  await deleteWikiPage.mutateAsync(deleteTarget.value.slug);
  deleteTarget.value = null;
};

// 先创建拿到 slug，再带 slug 进编辑器（编辑器按 kbId+slug 加载）
const handleCreate = async () => {
  const title = newTitle.value.trim();
  if (!title) return;
  creating.value = true;
  try {
    const res = await wikiPageApi.create(props.kbId, { title });
    const slug = res.data?.slug;
    createOpen.value = false;
    newTitle.value = "";
    if (slug) {
      void router.push({ path: "/wiki/editor", query: { kbId: props.kbId, slug } });
    }
  } catch (error) {
    toast.error((error as { message?: string }).message || "创建页面失败");
  } finally {
    creating.value = false;
  }
};

// ---- 版本历史 / 回滚 ----

const historyPage = ref<WikiPage | null>(null);
const expandedVersion = ref<number | null>(null);
const expandedContent = ref("");
const expandedLoading = ref(false);

const revisionsQuery = useWikiRevisions(
  () => props.kbId,
  () => historyPage.value?.slug,
  () => Boolean(historyPage.value),
);

watch(historyPage, () => {
  expandedVersion.value = null;
  expandedContent.value = "";
});

const sourceLabels: Record<string, string> = {
  pipeline: "管线生成",
  agent: "智能体",
  user: "人工编辑",
  revert: "回滚",
};

const sourceBadgeClass: Record<string, string> = {
  pipeline: "bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-300",
  agent: "bg-violet-100 text-violet-800 dark:bg-violet-950 dark:text-violet-300",
  user: "bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300",
  revert: "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300",
};

const formatDateTime = (value: string | undefined): string => {
  if (!value) return "-";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString("zh-CN", { hour12: false });
};

const openEditor = (page: WikiPage) => {
  void router.push({ path: "/wiki/editor", query: { kbId: props.kbId, slug: page.slug } });
};

const toggleExpand = async (version: number) => {
  if (!historyPage.value) return;
  if (expandedVersion.value === version) {
    expandedVersion.value = null;
    expandedContent.value = "";
    return;
  }
  expandedVersion.value = version;
  expandedContent.value = "";
  expandedLoading.value = true;
  try {
    const res = await wikiPageApi.getRevision(props.kbId, historyPage.value.slug, version);
    expandedContent.value = res.data?.content ?? "（该版本没有正文内容）";
  } catch {
    expandedContent.value = "（内容加载失败）";
  } finally {
    expandedLoading.value = false;
  }
};

const handleRevert = async (version: number) => {
  const page = historyPage.value;
  if (!page) return;
  await revertWikiPage.mutateAsync({ slug: page.slug, version });
  expandedVersion.value = null;
  expandedContent.value = "";
};
</script>

<template>
  <div class="flex h-full min-h-0 flex-col gap-3 overflow-y-auto pb-6">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <p class="text-sm text-muted-foreground">
        解析管线自动生成的 Wiki 页面，也可以手动创建、编辑；页面内容会随文档解析自动更新。
      </p>
      <Button @click="createOpen = true">
        <PlusIcon />
        新建页面
      </Button>
    </div>

    <div v-if="pagesQuery.isLoading.value" class="space-y-3">
      <Skeleton v-for="index in 4" :key="index" class="h-16 w-full" />
    </div>
    <Empty v-else-if="sortedPages.length === 0" class="rounded-lg border">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <SquarePenIcon />
        </EmptyMedia>
        <EmptyTitle>还没有 Wiki 页面</EmptyTitle>
        <EmptyDescription>
          在知识库设置里开启「解析时生成 Wiki 页面」，解析后的文档会自动整理成页面；也可以手动新建。
        </EmptyDescription>
      </EmptyHeader>
    </Empty>
    <div v-else class="space-y-3">
      <div v-for="page in sortedPages" :key="page.slug" class="rounded-lg border p-4">
        <div class="flex flex-wrap items-start justify-between gap-2">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-medium">{{ page.title || page.slug }}</span>
              <Badge
                v-if="page.last_edit_source"
                variant="secondary"
                :class="sourceBadgeClass[page.last_edit_source]"
              >
                {{ sourceLabels[page.last_edit_source] ?? page.last_edit_source }}
              </Badge>
              <Badge variant="outline">v{{ page.version }}</Badge>
            </div>
            <p class="mt-1 truncate text-xs text-muted-foreground">{{ page.slug }}</p>
            <p v-if="page.summary" class="mt-1 line-clamp-2 text-sm text-muted-foreground">
              {{ page.summary }}
            </p>
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <span class="text-xs text-muted-foreground">{{ formatDateTime(page.updated_at) }}</span>
            <Button variant="ghost" size="icon-xs" aria-label="版本历史" @click="historyPage = page">
              <HistoryIcon />
            </Button>
            <Button
              variant="ghost"
              size="icon-xs"
              class="text-muted-foreground hover:text-destructive"
              aria-label="删除页面"
              @click="deleteTarget = page"
            >
              <Trash2Icon class="h-3.5 w-3.5" />
            </Button>
            <Button variant="outline" size="sm" @click="openEditor(page)">
              <SquarePenIcon />
              编辑
            </Button>
          </div>
        </div>
      </div>
    </div>

    <!-- 新建页面 -->
    <Dialog :open="createOpen" @update:open="createOpen = $event">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>新建 Wiki 页面</DialogTitle>
          <DialogDescription>填写标题后进入编辑器开始写作，保存时生成页面地址。</DialogDescription>
        </DialogHeader>
        <div class="space-y-2">
          <Label for="wiki-new-title">页面标题</Label>
          <Input
            id="wiki-new-title"
            v-model="newTitle"
            placeholder="例如：产品概述"
            @keydown.enter="handleCreate"
          />
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="createOpen = false">取消</Button>
          <Button :disabled="!newTitle.trim() || creating" @click="handleCreate">
            {{ creating ? "创建中..." : "创建并编辑" }}
          </Button>
        </div>
      </DialogContent>
    </Dialog>

    <!-- 版本历史 -->
    <Dialog :open="Boolean(historyPage)" @update:open="open => !open && (historyPage = null)">
      <DialogContent class="max-h-[80vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>版本历史</DialogTitle>
          <DialogDescription>
            当前为版本 v{{ historyPage?.version ?? "-" }}。回滚会作为一次新编辑应用，回滚前的内容自动另存。
          </DialogDescription>
        </DialogHeader>

        <div v-if="revisionsQuery.isLoading.value" class="space-y-2 py-2">
          <Skeleton v-for="i in 3" :key="i" class="h-14 w-full" />
        </div>
        <p v-else-if="!revisionsQuery.data.value?.revisions.length" class="py-6 text-center text-sm text-muted-foreground">
          还没有历史版本（页面被修改过一次后生成）。
        </p>
        <div v-else class="space-y-2">
          <div v-for="rev in revisionsQuery.data.value.revisions" :key="rev.id" class="rounded-lg border p-3">
            <div class="flex items-center justify-between gap-2">
              <button
                type="button"
                class="flex min-w-0 items-center gap-2 text-left text-sm hover:underline"
                @click="toggleExpand(rev.version)"
              >
                <span class="font-medium">v{{ rev.version }}</span>
                <span class="truncate text-muted-foreground">{{ rev.title }}</span>
              </button>
              <div class="flex shrink-0 items-center gap-2">
                <span class="text-xs text-muted-foreground">{{ formatDateTime(rev.updated_at) }}</span>
                <Button variant="outline" size="xs" :disabled="revertWikiPage.isPending.value" @click="handleRevert(rev.version)">
                  回滚到此版本
                </Button>
              </div>
            </div>
            <div v-if="expandedVersion === rev.version" class="mt-2">
              <Skeleton v-if="expandedLoading" class="h-20 w-full" />
              <pre v-else class="max-h-60 overflow-y-auto whitespace-pre-wrap break-words rounded-md bg-muted p-3 font-mono text-xs leading-relaxed">{{ expandedContent }}</pre>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>

    <ConfirmDialog
      :open="Boolean(deleteTarget)"
      destructive
      title="删除 Wiki 页面"
      :description="`确定删除「${deleteTarget?.title || deleteTarget?.slug || '该页面'}」？其历史版本会一并删除。`"
      confirm-text="删除"
      @update:open="open => !open && (deleteTarget = null)"
      @confirm="handleDelete"
    />
  </div>
</template>
