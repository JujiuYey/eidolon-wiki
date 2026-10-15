<script setup lang="ts">
import {
  ArrowLeftIcon,
  BanIcon,
  CheckIcon,
  DownloadIcon,
  EyeIcon,
  FolderPlusIcon,
  RotateCcwIcon,
  SearchIcon,
  Settings2Icon,
  TagIcon,
  Trash2Icon,
  UploadIcon,
} from "@lucide/vue";
import type { ColumnDef, RowSelectionState } from "@tanstack/vue-table";
import { computed, h, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { toast } from "vue-sonner";

import {
  isKnowledgeParseInFlight,
  knowledgeApi,
  parseStatusBadgeClass,
  parseStatusLabels,
  useBatchDeleteKnowledge,
  useBatchReparseKnowledge,
  useBatchUpdateTags,
  useCancelKnowledgeParse,
  useDeleteKnowledge,
  useKnowledgeFolders,
  useKnowledgeList,
  useKnowledgeStatusPolling,
  useMoveKnowledgeToFolder,
  useRenameKnowledgeFolder,
  useReparseKnowledge,
  type Knowledge,
  type KnowledgeFolderNode,
  type KnowledgeParseStatus,
} from "@/api/wiki/knowledge";
import { useKnowledgeBase } from "@/api/wiki/knowledge-base";
import { tagBadgeStyle, useKnowledgeTags, type KnowledgeTag } from "@/api/wiki/tag";
import { ConfirmDialog, DataTable, SagPage } from "@/components/sag-ui";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { getDocumentIconAsset } from "./_shared/file-icons";

import DocumentDetailSheet from "./_components/DocumentDetailSheet.vue";
import FolderTreePanel from "./_components/FolderTreePanel.vue";
import BatchTagDialog from "./_components/BatchTagDialog.vue";
import FaqEntryPanel from "./_components/FaqEntryPanel.vue";
import KbEditorDialog from "./_components/KbEditorDialog.vue";
import SearchPanel from "./_components/SearchPanel.vue";
import TagManageDialog from "./_components/TagManageDialog.vue";
import UploadDialog from "./_components/UploadDialog.vue";
import WikiBrowserPanel from "./_components/WikiBrowserPanel.vue";

const PAGE_SIZE = 20;
const editorOpen = ref(false);

function formatDateTime(value: string | undefined): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", { hour12: false });
}

function formatBytes(bytes: number | undefined): string {
  if (!bytes) return "-";
  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${value.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

const PARSE_STATUS_FILTERS: Array<{ value: string; label: string }> = [
  { value: "all", label: "全部状态" },
  ...(Object.keys(parseStatusLabels) as KnowledgeParseStatus[]).map((status) => ({
    value: status,
    label: parseStatusLabels[status],
  })),
];

const route = useRoute();
const router = useRouter();

const kbId = computed(() => (route.query.kbId as string | undefined) ?? undefined);

const kbQuery = useKnowledgeBase(kbId);
const foldersQuery = useKnowledgeFolders(kbId);
const tagsQuery = useKnowledgeTags(kbId);

const tab = ref<"documents" | "search">("documents");
// FAQ 型知识库不走文档管理（后端禁止文件上传），整页替换为 FAQ 条目管理
const isFaq = computed(() => kbQuery.data.value?.type === "faq");
// 开了 wiki 管线（indexing_strategy.wiki_enabled）的 KB 才显示 Wiki 页签（与官方一致）
const isWiki = computed(() => kbQuery.data.value?.indexing_strategy?.wiki_enabled === true);
const page = ref(1);
const keywordInput = ref("");
const keyword = ref("");
const statusFilter = ref("all");
const selectedFolder = ref<string | undefined>(undefined);
const selectedTagIds = ref<string[]>([]);
const uploadOpen = ref(false);
const detailTarget = ref<Knowledge | null>(null);
const deleteTarget = ref<Knowledge | null>(null);
const manageTagsOpen = ref(false);
const batchTagOpen = ref(false);

// 标签被删除后从筛选条件里剔除
watch(tagsQuery.data, (tags) => {
  const ids = new Set((tags ?? []).map((t) => t.id));
  if (selectedTagIds.value.some((id) => !ids.has(id))) {
    selectedTagIds.value = selectedTagIds.value.filter((id) => ids.has(id));
  }
});

const listQuery = useKnowledgeList(kbId, () => ({
  page: page.value,
  page_size: PAGE_SIZE,
  keyword: keyword.value || undefined,
  parse_status: statusFilter.value === "all" ? undefined : statusFilter.value,
  tag_ids: selectedTagIds.value.join(",") || undefined,
  folder_path: selectedFolder.value,
}));
const items = computed(() => listQuery.data.value?.items);

const reparse = useReparseKnowledge(kbId.value ?? "");
const cancelParse = useCancelKnowledgeParse(kbId.value ?? "");
const deleteDoc = useDeleteKnowledge(kbId.value ?? "");
const batchDelete = useBatchDeleteKnowledge(kbId.value ?? "");
const batchReparse = useBatchReparseKnowledge(kbId.value ?? "");
const moveToFolder = useMoveKnowledgeToFolder(kbId.value ?? "");
const renameFolder = useRenameKnowledgeFolder(kbId.value ?? "");
const batchUpdateTags = useBatchUpdateTags(kbId.value ?? "");

// ---- 行选择与批量操作 ----

const rowSelection = reactive<RowSelectionState>({});
const selectedIds = computed(() =>
  Object.keys(rowSelection).filter((key) => rowSelection[key]),
);

const clearSelection = () => {
  for (const key of Object.keys(rowSelection)) delete rowSelection[key];
};

// 翻页/筛选/文件夹切换后清空选择，避免跨视图误操作
watch([page, keyword, statusFilter, selectedFolder, selectedTagIds], clearSelection);

const tableOptions = {
  enableRowSelection: true,
  getRowId: (row: Knowledge) => row.id,
  state: { rowSelection },
  onRowSelectionChange: (updater: RowSelectionState | ((old: RowSelectionState) => RowSelectionState)) => {
    const next = typeof updater === "function" ? updater(rowSelection) : updater;
    clearSelection();
    Object.assign(rowSelection, next);
  },
};

/** 自绘选择框（项目无 Checkbox 组件） */
const renderCheckbox = (checked: boolean, onToggle: () => void, label: string) =>
  h(
    "button",
    {
      type: "button",
      role: "checkbox",
      "aria-checked": checked,
      "aria-label": label,
      class: `flex size-4 shrink-0 items-center justify-center rounded border transition-colors ${
        checked
          ? "border-primary bg-primary text-primary-foreground"
          : "border-muted-foreground/40 hover:border-muted-foreground/70"
      }`,
      onClick: (event: Event) => {
        event.stopPropagation();
        onToggle();
      },
    },
    checked ? [h(CheckIcon, { class: "h-3 w-3" })] : [],
  );

const flattenFolderPaths = (folders: KnowledgeFolderNode[]): string[] =>
  folders.flatMap((f) => [f.path, ...flattenFolderPaths(f.children ?? [])]);
const folderPaths = computed(() => flattenFolderPaths(foldersQuery.data.value?.folders ?? []));

const batchDeleteOpen = ref(false);
const moveNewOpen = ref(false);
const moveNewPath = ref("");
const renameOpen = ref(false);
const renameFrom = ref("");
const renameTo = ref("");

const handleBatchDelete = async () => {
  if (selectedIds.value.length === 0) return;
  await batchDelete.mutateAsync(selectedIds.value);
  batchDeleteOpen.value = false;
  clearSelection();
};

const handleBatchReparse = async () => {
  if (selectedIds.value.length === 0) return;
  await batchReparse.mutateAsync(selectedIds.value);
  clearSelection();
};

const handleMoveTo = async (folderPath?: string) => {
  if (selectedIds.value.length === 0) return;
  await moveToFolder.mutateAsync({ knowledgeIds: selectedIds.value, folderPath: folderPath ?? "" });
  clearSelection();
};

const handleMoveToNew = async () => {
  const path = moveNewPath.value.trim().replace(/^\/+|\/+$/g, "");
  if (!path) {
    toast.error("请输入文件夹路径");
    return;
  }
  moveNewOpen.value = false;
  await handleMoveTo(path);
  moveNewPath.value = "";
};

const openFolderRename = (path: string) => {
  renameFrom.value = path;
  renameTo.value = path;
  renameOpen.value = true;
};

const handleFolderRename = async () => {
  const to = renameTo.value.trim().replace(/^\/+|\/+$/g, "");
  if (!to || to === renameFrom.value) return;
  await renameFolder.mutateAsync({ from: renameFrom.value, to });
  renameOpen.value = false;
  if (selectedFolder.value === renameFrom.value) selectedFolder.value = to;
};

const handleBatchApplyTags = async (tagIds: string[]) => {
  const updates: Record<string, string[]> = {};
  for (const id of selectedIds.value) updates[id] = tagIds;
  await batchUpdateTags.mutateAsync(updates);
  batchTagOpen.value = false;
};

const toggleTagFilter = (tagId: string, checked: boolean) => {
  selectedTagIds.value = checked
    ? [...selectedTagIds.value, tagId]
    : selectedTagIds.value.filter((id) => id !== tagId);
  page.value = 1;
};

// 解析状态轮询（1.5s，只查 in-flight 的文档）
useKnowledgeStatusPolling(kbId, () => items.value);

const handleDownload = (doc: Knowledge) => {
  void knowledgeApi
    .download(doc.id)
    .then((blob) => {
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = doc.file_name || doc.id;
      link.click();
      URL.revokeObjectURL(url);
    })
    .catch((error: { message?: string }) => toast.error(error.message || "下载失败"));
};

const handleDelete = async () => {
  if (deleteTarget.value) await deleteDoc.mutateAsync(deleteTarget.value.id);
};

const columns: ColumnDef<Knowledge, any>[] = [
  {
    id: "select",
    header: ({ table }) =>
      renderCheckbox(
        table.getIsAllPageRowsSelected(),
        () => table.toggleAllPageRowsSelected(),
        "全选本页",
      ),
    cell: ({ row }) => renderCheckbox(row.getIsSelected(), () => row.toggleSelected(), "选择文档"),
  },
  {
    accessorKey: "file_name",
    header: "名称",
    cell: ({ row }) => {
      const doc = row.original;
      const icon = getDocumentIconAsset({ contentType: doc.type ?? "", filename: doc.file_name ?? "" });
      return h("div", { class: "flex items-center gap-2" }, [
        h("img", { src: icon.src, alt: icon.alt, class: "h-5 w-5 shrink-0" }),
        h("div", { class: "min-w-0" }, [
          h("p", { class: "truncate font-medium" }, doc.file_name || doc.title || doc.id),
          doc.folder_path ? h("p", { class: "truncate text-xs text-muted-foreground" }, doc.folder_path) : null,
        ]),
      ]);
    },
  },
  {
    accessorKey: "file_size",
    header: "大小",
    cell: ({ row }) => h("span", { class: "text-sm text-muted-foreground" }, formatBytes(row.original.file_size)),
  },
  {
    accessorKey: "parse_status",
    header: "解析状态",
    cell: ({ row }) => {
      const doc = row.original;
      const inFlight = isKnowledgeParseInFlight(doc.parse_status);
      return h(
        "span",
        { title: doc.error_message || undefined, class: "inline-flex" },
        h(
          Badge,
          { variant: "secondary", class: parseStatusBadgeClass(doc.parse_status) },
          () => `${parseStatusLabels[doc.parse_status] ?? doc.parse_status}${inFlight ? "…" : ""}`,
        ),
      );
    },
  },
  {
    id: "tags",
    header: "标签",
    cell: ({ row }) => {
      const tags = (row.original.tags ?? []) as KnowledgeTag[];
      if (tags.length === 0) return h("span", { class: "text-sm text-muted-foreground" }, "-");
      const visible = tags.slice(0, 2);
      return h(
        "div",
        { class: "flex flex-wrap gap-1" },
        visible.map((tag) =>
          h(
            Badge,
            {
              key: tag.id,
              variant: "secondary",
              style: tagBadgeStyle(tag.color),
              title: tags.length > 2 ? tags.slice(2).map((t) => t.name).join("、") : undefined,
            },
            () => (tags.length > 2 && tag === visible[1] ? `${tag.name} +${tags.length - 2}` : tag.name),
          ),
        ),
      );
    },
  },
  {
    accessorKey: "updated_at",
    header: "更新时间",
    cell: ({ row }) =>
      h("span", { class: "text-sm text-muted-foreground" }, formatDateTime(row.original.updated_at)),
  },
  {
    id: "actions",
    header: () => h("span", { class: "sr-only" }, "操作"),
    cell: ({ row }) => {
      const doc = row.original;
      const inFlight = isKnowledgeParseInFlight(doc.parse_status);
      return h("div", { class: "flex justify-end" }, [
        h(DropdownMenu, null, {
          default: () => [
            h(DropdownMenuTrigger, { asChild: true }, () =>
              h(
                Button,
                { variant: "ghost", size: "icon-sm", "aria-label": `文档 ${doc.file_name || doc.id} 操作菜单` },
                () => h("span", { class: "text-lg leading-none" }, "⋯"),
              ),
            ),
            h(DropdownMenuContent, { align: "end" }, () => [
              h(DropdownMenuItem, { onClick: () => (detailTarget.value = doc) }, () => [
                h(EyeIcon),
                "分块与预览",
              ]),
              inFlight
                ? h(DropdownMenuItem, { onClick: () => cancelParse.mutate(doc.id) }, () => [h(BanIcon), "取消解析"])
                : h(DropdownMenuItem, { onClick: () => reparse.mutate(doc.id) }, () => [
                    h(RotateCcwIcon),
                    "重新解析",
                  ]),
              h(DropdownMenuItem, { onClick: () => handleDownload(doc) }, () => [h(DownloadIcon), "下载"]),
              h(
                DropdownMenuItem,
                { variant: "destructive", onClick: () => (deleteTarget.value = doc) },
                () => [h(Trash2Icon), "删除"],
              ),
            ]),
          ],
        }),
      ]);
    },
  },
];

const total = computed(() => listQuery.data.value?.total ?? 0);
</script>

<template>
  <SagPage v-if="!kbId" title="知识库详情" description="未指定知识库">
    <Empty class="rounded-lg border">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <ArrowLeftIcon />
        </EmptyMedia>
        <EmptyTitle>未指定知识库</EmptyTitle>
        <EmptyDescription>请从知识库列表进入。</EmptyDescription>
      </EmptyHeader>
    </Empty>
  </SagPage>

  <SagPage
    v-else
    :title="kbQuery.data.value?.name ?? '知识库'"
    :description="kbQuery.data.value?.description || '知识库文档管理'"
  >
    <template #actions>
      <div class="flex gap-2">
        <Button variant="ghost" @click="router.push('/wiki/knowledge-bases')">
          <ArrowLeftIcon />
          返回列表
        </Button>
        <Button variant="outline" @click="editorOpen = true">
          <Settings2Icon />
          设置
        </Button>
        <Button v-if="!isFaq" @click="uploadOpen = true">
          <UploadIcon />
          上传文档
        </Button>
      </div>
    </template>

    <!-- FAQ 型知识库：条目管理 + FAQ 检索测试 -->
    <FaqEntryPanel v-if="isFaq" :kb-id="kbId" />

    <Tabs v-else v-model="tab" class="flex h-full min-h-0 flex-col gap-4">
      <TabsList class="w-fit">
        <TabsTrigger value="documents">文档管理（{{ total }}）</TabsTrigger>
        <TabsTrigger v-if="isWiki" value="wiki">Wiki</TabsTrigger>
        <TabsTrigger value="search">检索测试</TabsTrigger>
      </TabsList>

      <TabsContent value="documents" class="min-h-0 flex-1">
        <div class="flex h-full min-h-0 gap-4">
          <div class="hidden w-60 shrink-0 lg:block">
            <FolderTreePanel
              :tree="foldersQuery.data.value"
              :loading="foldersQuery.isLoading.value"
              :selected-path="selectedFolder"
              @rename="openFolderRename"
              @select="
                path => {
                  selectedFolder = path;
                  page = 1;
                }
              "
            />
          </div>

          <div class="flex min-h-0 min-w-0 flex-1 flex-col gap-3 overflow-y-auto pb-6">
            <div class="flex flex-wrap items-center gap-2">
              <div class="relative w-64">
                <SearchIcon class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  v-model="keywordInput"
                  class="pl-8"
                  placeholder="搜索文件名"
                  @keydown.enter="
                    () => {
                      page = 1;
                      keyword = keywordInput.trim();
                    }
                  "
                />
              </div>
              <Select
                :model-value="statusFilter"
                @update:model-value="
                  value => {
                    statusFilter = String(value);
                    page = 1;
                  }
                "
              >
                <SelectTrigger class="w-32">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="option in PARSE_STATUS_FILTERS" :key="option.value" :value="option.value">
                    {{ option.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <Badge v-if="selectedFolder !== undefined" variant="outline" class="gap-1">
                文件夹：{{ selectedFolder || "/" }}
              </Badge>

              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="outline" size="sm">
                    <TagIcon />
                    标签
                    <Badge v-if="selectedTagIds.length" variant="secondary" class="ml-1 px-1.5">
                      {{ selectedTagIds.length }}
                    </Badge>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" class="max-h-72 w-56 overflow-y-auto">
                  <div class="flex items-center justify-between px-2 py-1.5">
                    <span class="text-xs text-muted-foreground">按标签筛选（可多选）</span>
                    <button
                      type="button"
                      class="text-xs text-muted-foreground hover:text-foreground"
                      @click="manageTagsOpen = true"
                    >
                      管理
                    </button>
                  </div>
                  <p v-if="(tagsQuery.data.value ?? []).length === 0" class="px-2 py-3 text-xs text-muted-foreground">
                    还没有标签，点右上角「管理」新建。
                  </p>
                  <label
                    v-for="tag in tagsQuery.data.value ?? []"
                    :key="tag.id"
                    class="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-muted"
                  >
                    <input
                      type="checkbox"
                      class="accent-[var(--primary)]"
                      :checked="selectedTagIds.includes(tag.id)"
                      @change="toggleTagFilter(tag.id, ($event.target as HTMLInputElement).checked)"
                    />
                    <span
                      class="size-2 shrink-0 rounded-full"
                      :style="tagBadgeStyle(tag.color) ?? { backgroundColor: 'var(--muted-foreground)' }"
                    />
                    {{ tag.name }}
                  </label>
                  <template v-if="selectedTagIds.length">
                    <DropdownMenuSeparator />
                    <DropdownMenuItem @click="selectedTagIds = []">清除筛选</DropdownMenuItem>
                  </template>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>

            <!-- 批量操作栏：选中行 > 0 时出现 -->
            <div
              v-if="selectedIds.length > 0"
              class="flex flex-wrap items-center gap-2 rounded-lg border bg-muted/40 px-3 py-2"
            >
              <span class="text-sm text-muted-foreground">已选 {{ selectedIds.length }} 项</span>
              <div class="flex-1" />
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="outline" size="sm">
                    <FolderPlusIcon />
                    移入文件夹
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" class="max-h-72 overflow-y-auto">
                  <DropdownMenuItem @click="handleMoveTo('')">（根目录）</DropdownMenuItem>
                  <DropdownMenuItem
                    v-for="path in folderPaths"
                    :key="path"
                    :title="path"
                    @click="handleMoveTo(path)"
                  >
                    {{ path }}
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem @click="moveNewOpen = true">
                    <FolderPlusIcon />
                    新建文件夹并移入…
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <Button variant="outline" size="sm" @click="handleBatchReparse">
                <RotateCcwIcon />
                重新解析
              </Button>
              <Button variant="outline" size="sm" @click="batchTagOpen = true">
                <TagIcon />
                设置标签
              </Button>
              <Button variant="destructive" size="sm" @click="batchDeleteOpen = true">
                <Trash2Icon />
                删除
              </Button>
              <Button variant="ghost" size="sm" @click="clearSelection">取消选择</Button>
            </div>

            <DataTable
              :columns="columns"
              :data="items ?? []"
              :loading="listQuery.isLoading.value"
              :table-options="tableOptions"
              :empty-text="
                keyword || statusFilter !== 'all' || selectedFolder !== undefined || selectedTagIds.length
                  ? '没有符合条件（含正在解析）的文档'
                  : '知识库还没有文档，点击右上角上传'
              "
              :pagination="{
                page,
                pageSize: PAGE_SIZE,
                total,
                onPageChange: (p: number) => (page = p),
                onPageSizeChange: () => {},
              }"
            />
          </div>
        </div>
      </TabsContent>

      <TabsContent v-if="isWiki" value="wiki" class="min-h-0 flex-1">
        <WikiBrowserPanel :kb-id="kbId ?? ''" />
      </TabsContent>

      <TabsContent value="search" class="min-h-0 flex-1">
        <SearchPanel :kb-id="kbId" />
      </TabsContent>
    </Tabs>

    <UploadDialog
      v-model:open="uploadOpen"
      :kb-id="kbId"
      :folder-path="selectedFolder ?? ''"
      @uploaded="
        () => {
          page = 1;
          listQuery.refetch();
        }
      "
    />

    <DocumentDetailSheet
      :open="Boolean(detailTarget)"
      :knowledge="detailTarget"
      @update:open="open => !open && (detailTarget = null)"
    />

    <KbEditorDialog :open="editorOpen" :kb="kbQuery.data.value" @update:open="open => (editorOpen = open)" />

    <TagManageDialog :open="manageTagsOpen" :kb-id="kbId ?? ''" :tags="tagsQuery.data.value ?? []" @update:open="open => (manageTagsOpen = open)" />

    <BatchTagDialog
      :open="batchTagOpen"
      :kb-id="kbId ?? ''"
      :tags="tagsQuery.data.value ?? []"
      :count="selectedIds.length"
      :loading="batchUpdateTags.isPending.value"
      @update:open="open => (batchTagOpen = open)"
      @apply="handleBatchApplyTags"
    />

    <ConfirmDialog
      :open="Boolean(deleteTarget)"
      destructive
      title="删除文档"
      :description="`确定删除「${deleteTarget?.file_name || deleteTarget?.title || '该文档'}」？其分块与索引会一并删除。`"
      confirm-text="删除"
      @update:open="open => !open && (deleteTarget = null)"
      @confirm="handleDelete"
    />

    <ConfirmDialog
      :open="batchDeleteOpen"
      destructive
      title="批量删除文档"
      :description="`确定删除选中的 ${selectedIds.length} 个文档？其分块与索引会一并删除。`"
      confirm-text="全部删除"
      @update:open="open => !open && (batchDeleteOpen = false)"
      @confirm="handleBatchDelete"
    />

    <Dialog :open="moveNewOpen" @update:open="moveNewOpen = $event">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>新建文件夹并移入</DialogTitle>
          <DialogDescription>
            将选中的 {{ selectedIds.length }} 个文档移入新文件夹。文件夹路径由文档路径派生，随第一个文档一并创建。
          </DialogDescription>
        </DialogHeader>
        <div class="space-y-2">
          <Label for="move-new-path">文件夹路径</Label>
          <Input
            id="move-new-path"
            v-model="moveNewPath"
            placeholder="例如：产品文档/2026"
            @keydown.enter="handleMoveToNew"
          />
        </div>
        <DialogFooter>
          <Button variant="outline" @click="moveNewOpen = false">取消</Button>
          <Button :disabled="moveToFolder.isPending.value" @click="handleMoveToNew">
            {{ moveToFolder.isPending.value ? "移动中..." : "移入" }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog :open="renameOpen" @update:open="renameOpen = $event">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>重命名 / 移动文件夹</DialogTitle>
          <DialogDescription>
            修改文件夹路径（含其所有子目录）。目标路径已存在时两个文件夹会合并。
          </DialogDescription>
        </DialogHeader>
        <div class="space-y-2">
          <Label for="rename-folder-to">新路径</Label>
          <Input
            id="rename-folder-to"
            v-model="renameTo"
            placeholder="例如：产品文档/2025"
            @keydown.enter="handleFolderRename"
          />
          <p class="text-xs text-muted-foreground">原路径：{{ renameFrom }}</p>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="renameOpen = false">取消</Button>
          <Button
            :disabled="renameFolder.isPending.value || !renameTo.trim() || renameTo.trim() === renameFrom"
            @click="handleFolderRename"
          >
            {{ renameFolder.isPending.value ? "保存中..." : "保存" }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </SagPage>
</template>
