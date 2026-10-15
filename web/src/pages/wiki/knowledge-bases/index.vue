<script setup lang="ts">
import {
  BookOpenIcon,
  MoreHorizontalIcon,
  PinIcon,
  PinOffIcon,
  PlusIcon,
  SearchIcon,
  SquarePenIcon,
  Trash2Icon,
} from "@lucide/vue";
import { computed, ref } from "vue";
import { useRouter } from "vue-router";

import {
  useDeleteKnowledgeBase,
  useKnowledgeBaseList,
  useTogglePinKnowledgeBase,
  type KnowledgeBase,
} from "@/api/wiki/knowledge-base";
import { ConfirmDialog, SagPage } from "@/components/sag-ui";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";

import KbEditorDialog from "./_components/KbEditorDialog.vue";

function formatDateTime(value: string | undefined): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", { hour12: false });
}

const router = useRouter();
const kbList = useKnowledgeBaseList();
const togglePin = useTogglePinKnowledgeBase();
const deleteKb = useDeleteKnowledgeBase();

const searchText = ref("");
const formOpen = ref(false);
const editingKb = ref<KnowledgeBase | null>(null);
const deleteTarget = ref<KnowledgeBase | null>(null);

const items = computed(() => {
  const all = kbList.data.value ?? [];
  const keyword = searchText.value.trim().toLowerCase();
  const filtered = keyword
    ? all.filter(
        (kb) =>
          kb.name.toLowerCase().includes(keyword) ||
          (kb.description ?? "").toLowerCase().includes(keyword),
      )
    : all;
  // 置顶优先，其余按更新时间倒序（与官方一致）
  return [...filtered].sort((a, b) => {
    if (Boolean(a.is_pinned) !== Boolean(b.is_pinned)) return a.is_pinned ? -1 : 1;
    return (b.updated_at ?? "").localeCompare(a.updated_at ?? "");
  });
});

const openDetail = (kb: KnowledgeBase) => {
  void router.push({ path: "/wiki/knowledge-bases/detail", query: { kbId: kb.id } });
};

const handleDelete = async () => {
  if (!deleteTarget.value) return;
  await deleteKb.mutateAsync(deleteTarget.value.id);
};

const openCreate = () => {
  editingKb.value = null;
  formOpen.value = true;
};
</script>

<template>
  <SagPage title="知识库" description="管理知识库与文档，上传后自动进入解析流水线">
    <template #actions>
      <Button @click="openCreate">
        <PlusIcon />
        新建知识库
      </Button>
    </template>

    <div class="flex h-full min-h-0 flex-col gap-4 overflow-y-auto pb-6">
      <div class="relative w-72">
        <SearchIcon class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input v-model="searchText" class="pl-8" placeholder="搜索知识库" />
      </div>

      <div v-if="kbList.isLoading.value" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <Skeleton v-for="index in 6" :key="index" class="h-36 w-full" />
      </div>
      <Empty v-else-if="items.length === 0" class="rounded-lg border">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <BookOpenIcon />
          </EmptyMedia>
          <EmptyTitle>{{ searchText ? "没有匹配的知识库" : "还没有知识库" }}</EmptyTitle>
          <EmptyDescription>
            {{ searchText ? "换个关键字试试。" : "创建第一个知识库，开始沉淀团队文档。" }}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
      <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <Card
          v-for="kb in items"
          :key="kb.id"
          class="cursor-pointer py-4 transition-shadow hover:shadow-md"
          @click="openDetail(kb)"
        >
          <CardHeader class="px-4">
            <CardTitle class="flex items-start justify-between gap-2 text-base font-semibold">
              <span class="flex min-w-0 items-center gap-2">
                <PinIcon v-if="kb.is_pinned" class="h-4 w-4 shrink-0 text-amber-500" />
                <span class="truncate">{{ kb.name }}</span>
                <Badge v-if="kb.type === 'faq'" variant="secondary">FAQ</Badge>
              </span>
              <span class="shrink-0" @click.stop>
                <DropdownMenu>
                  <DropdownMenuTrigger as-child>
                    <Button variant="ghost" size="icon-sm" :aria-label="`知识库 ${kb.name} 操作菜单`">
                      <MoreHorizontalIcon />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem
                      @click="
                        () => {
                          editingKb = kb;
                          formOpen = true;
                        }
                      "
                    >
                      <SquarePenIcon />
                      编辑
                    </DropdownMenuItem>
                    <DropdownMenuItem @click="togglePin.mutate(kb.id)">
                      <PinOffIcon v-if="kb.is_pinned" />
                      <PinIcon v-else />
                      {{ kb.is_pinned ? "取消置顶" : "置顶" }}
                    </DropdownMenuItem>
                    <DropdownMenuItem variant="destructive" @click="deleteTarget = kb">
                      <Trash2Icon />
                      删除
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </span>
            </CardTitle>
          </CardHeader>
          <CardContent class="space-y-3 px-4">
            <p class="line-clamp-2 min-h-8 text-sm text-muted-foreground">{{ kb.description || "暂无描述" }}</p>
            <div class="flex items-center gap-2 text-xs text-muted-foreground">
              <Badge variant="secondary">{{ kb.knowledge_count ?? 0 }} 文档</Badge>
              <Badge variant="secondary">{{ kb.chunk_count ?? 0 }} 分块</Badge>
              <Badge
                v-if="(kb.processing_count ?? 0) > 0"
                class="bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-300"
              >
                {{ kb.processing_count }} 解析中
              </Badge>
            </div>
            <p class="text-xs text-muted-foreground/80">更新于 {{ formatDateTime(kb.updated_at) }}</p>
          </CardContent>
        </Card>
      </div>
    </div>

    <KbEditorDialog v-model:open="formOpen" :kb="editingKb" />

    <ConfirmDialog
      :open="Boolean(deleteTarget)"
      destructive
      title="删除知识库"
      :description="`确定删除知识库「${deleteTarget?.name ?? ''}」？其中所有文档、分块与索引都会被删除，操作不可恢复。`"
      confirm-text="删除"
      @update:open="open => !open && (deleteTarget = null)"
      @confirm="handleDelete"
    />
  </SagPage>
</template>
