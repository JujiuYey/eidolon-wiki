<script setup lang="ts">
import { PlusIcon, SearchIcon } from "@lucide/vue";
import type { ColumnDef } from "@tanstack/vue-table";
import type { RowSelectionState } from "@tanstack/vue-table";
import { computed, h, reactive, ref, watch } from "vue";
import { toast } from "vue-sonner";

import { useKnowledgeTags, tagBadgeStyle, type KnowledgeTag } from "@/api/wiki/tag";
import {
  useBatchDeleteFAQEntries,
  useBatchUpdateFAQFields,
  useFAQEntries,
  type FAQEntry,
} from "@/api/wiki/faq";
import { DataTable } from "@/components/sag-ui";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import FaqEntryDialog from "./FaqEntryDialog.vue";
import FaqSearchPanel from "./FaqSearchPanel.vue";
import { ConfirmDialog } from "@/components/sag-ui";

const PAGE_SIZE = 20;

const props = defineProps<{
  kbId: string;
}>();

const tab = ref<"entries" | "search">("entries");
const page = ref(1);
const keywordInput = ref("");
const keyword = ref("");
const tagFilter = ref<number>(0); // 0 = 全部

const entriesQuery = useFAQEntries(() => props.kbId, () => ({
  page: page.value,
  page_size: PAGE_SIZE,
  keyword: keyword.value || undefined,
  tag_id: tagFilter.value || undefined,
}));
const entries = computed(() => entriesQuery.data.value?.data ?? []);
const total = computed(() => entriesQuery.data.value?.total ?? 0);
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)));

const tagsQuery = useKnowledgeTags(() => props.kbId);
const tagNameById = computed(() => {
  const map = new Map<number, KnowledgeTag>();
  for (const tag of tagsQuery.data.value ?? []) map.set(tag.seq_id ?? 0, tag);
  return map;
});

const batchUpdateFields = useBatchUpdateFAQFields(() => props.kbId);
const batchDelete = useBatchDeleteFAQEntries(() => props.kbId);

// ---- 行选择（与文档表格同一套自绘模式）----

const rowSelection = reactive<RowSelectionState>({});
const selectedIds = computed(() =>
  Object.keys(rowSelection)
    .filter((key) => rowSelection[key])
    .map(Number),
);

const clearSelection = () => {
  for (const key of Object.keys(rowSelection)) delete rowSelection[key];
};

watch([page, keyword, tagFilter], clearSelection);

const tableOptions = {
  enableRowSelection: true,
  getRowId: (row: FAQEntry) => String(row.id),
  state: { rowSelection },
  onRowSelectionChange: (updater: RowSelectionState | ((old: RowSelectionState) => RowSelectionState)) => {
    const next = typeof updater === "function" ? updater(rowSelection) : updater;
    clearSelection();
    Object.assign(rowSelection, next);
  },
};

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
    checked ? [h("span", { class: "h-2 w-2 rounded-sm bg-current" })] : [],
  );

// ---- 编辑 / 批量 ----

const editorOpen = ref(false);
const editingEntry = ref<FAQEntry | null>(null);
const batchDeleteOpen = ref(false);

const openCreate = () => {
  editingEntry.value = null;
  editorOpen.value = true;
};

const openEdit = (entry: FAQEntry) => {
  editingEntry.value = entry;
  editorOpen.value = true;
};

const handleBatchFields = async (patch: { is_enabled?: boolean; is_recommended?: boolean }) => {
  if (selectedIds.value.length === 0) return;
  const byId: Record<string, { is_enabled?: boolean; is_recommended?: boolean }> = {};
  for (const id of selectedIds.value) byId[String(id)] = patch;
  await batchUpdateFields.mutateAsync(byId);
  clearSelection();
};

const handleBatchDelete = async () => {
  if (selectedIds.value.length === 0) return;
  await batchDelete.mutateAsync(selectedIds.value);
  batchDeleteOpen.value = false;
  clearSelection();
};

const handleToggle = async (entry: FAQEntry, patch: { is_enabled?: boolean; is_recommended?: boolean }) => {
  await batchUpdateFields.mutateAsync({ [String(entry.id)]: patch });
};

const formatDateTime = (value: string | undefined): string => {
  if (!value) return "-";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString("zh-CN", { hour12: false });
};

const columns: ColumnDef<FAQEntry, any>[] = [
  {
    id: "select",
    header: ({ table }) =>
      renderCheckbox(table.getIsAllPageRowsSelected(), () => table.toggleAllPageRowsSelected(), "全选本页"),
    cell: ({ row }) => renderCheckbox(row.getIsSelected(), () => row.toggleSelected(), "选择条目"),
  },
  {
    accessorKey: "standard_question",
    header: "标准问题",
    cell: ({ row }) => {
      const entry = row.original;
      const similarCount = entry.similar_questions?.length ?? 0;
      const negativeCount = entry.negative_questions?.length ?? 0;
      return h("div", { class: "flex flex-col" }, [
        h("span", { class: "font-medium" }, entry.standard_question),
        h(
          "span",
          { class: "text-xs text-muted-foreground" },
          [similarCount ? `${similarCount} 条相似问` : "", negativeCount ? `${negativeCount} 条反例` : ""]
            .filter(Boolean)
            .join(" · ") || " ",
        ),
      ]);
    },
  },
  {
    id: "answers",
    header: "答案",
    cell: ({ row }) => {
      const entry = row.original;
      const first = entry.answers[0] ?? "-";
      return h("div", { class: "flex flex-col" }, [
        h("p", { class: "line-clamp-2 text-sm text-muted-foreground" }, first),
        entry.answers.length > 1
          ? h("span", { class: "text-xs text-muted-foreground" }, `共 ${entry.answers.length} 条答案`)
          : null,
      ]);
    },
  },
  {
    id: "tag",
    header: "分类",
    cell: ({ row }) => {
      const entry = row.original;
      const tag = tagNameById.value.get(entry.tag_id);
      if (!tag) return h("span", { class: "text-sm text-muted-foreground" }, "未分类");
      return h(
        Badge,
        { variant: "secondary", style: tagBadgeStyle(tag.color) },
        () => tag.name,
      );
    },
  },
  {
    id: "is_enabled",
    header: "启用",
    cell: ({ row }) =>
      h("div", { class: "flex justify-start" }, [
        h(Switch, {
          checked: row.original.is_enabled,
          "onUpdate:checked": (checked: boolean) => handleToggle(row.original, { is_enabled: checked }),
          disabled: batchUpdateFields.isPending.value,
          "aria-label": `启用条目 ${row.original.standard_question}`,
        }),
      ]),
  },
  {
    id: "is_recommended",
    header: "推荐",
    cell: ({ row }) =>
      h("div", { class: "flex justify-start" }, [
        h(Switch, {
          checked: row.original.is_recommended,
          "onUpdate:checked": (checked: boolean) => handleToggle(row.original, { is_recommended: checked }),
          disabled: batchUpdateFields.isPending.value,
          "aria-label": `推荐条目 ${row.original.standard_question}`,
        }),
      ]),
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
    cell: ({ row }) =>
      h("div", { class: "flex justify-end" }, [
        h(
          Button,
          { variant: "ghost", size: "sm", onClick: () => openEdit(row.original) },
          () => "编辑",
        ),
      ]),
  },
];
</script>

<template>
  <div class="flex h-full min-h-0 flex-col gap-4">
    <Tabs v-model="tab" class="flex h-full min-h-0 flex-col gap-4">
      <TabsList class="w-fit">
        <TabsTrigger value="entries">FAQ 条目（{{ total }}）</TabsTrigger>
        <TabsTrigger value="search">检索测试</TabsTrigger>
      </TabsList>

      <TabsContent value="entries" class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto pb-6">
        <div class="flex flex-wrap items-center gap-2">
          <div class="relative w-64">
            <SearchIcon class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              v-model="keywordInput"
              class="pl-8"
              placeholder="搜索问题或答案"
              @keydown.enter="
                () => {
                  page = 1;
                  keyword = keywordInput.trim();
                }
              "
            />
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button variant="outline" size="sm">
                分类：
                {{ tagFilter ? (tagNameById.get(tagFilter)?.name ?? "全部") : "全部" }}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" class="max-h-72 w-48 overflow-y-auto">
              <DropdownMenuItem @click="tagFilter = 0">全部</DropdownMenuItem>
              <DropdownMenuItem
                v-for="tag in tagsQuery.data.value ?? []"
                :key="tag.id"
                @click="tagFilter = tag.seq_id ?? 0"
              >
                {{ tag.name }}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <div class="flex-1" />
          <Button @click="openCreate">
            <PlusIcon />
            新建条目
          </Button>
        </div>

        <div v-if="selectedIds.length > 0" class="flex flex-wrap items-center gap-2 rounded-lg border bg-muted/40 px-3 py-2">
          <span class="text-sm text-muted-foreground">已选 {{ selectedIds.length }} 项</span>
          <div class="flex-1" />
          <Button variant="outline" size="sm" @click="handleBatchFields({ is_enabled: true })">批量启用</Button>
          <Button variant="outline" size="sm" @click="handleBatchFields({ is_enabled: false })">批量停用</Button>
          <Button variant="outline" size="sm" @click="handleBatchFields({ is_recommended: true })">设为推荐</Button>
          <Button variant="outline" size="sm" @click="handleBatchFields({ is_recommended: false })">取消推荐</Button>
          <Button variant="destructive" size="sm" @click="batchDeleteOpen = true">删除</Button>
          <Button variant="ghost" size="sm" @click="clearSelection">取消选择</Button>
        </div>

        <DataTable
          :columns="columns"
          :data="entries"
          :loading="entriesQuery.isLoading.value"
          :table-options="tableOptions"
          empty-text="还没有 FAQ 条目，点右上角新建"
          :pagination="{
            page,
            pageSize: PAGE_SIZE,
            total,
            onPageChange: (p: number) => (page = p),
            onPageSizeChange: () => {},
          }"
        />
      </TabsContent>

      <TabsContent value="search" class="min-h-0 flex-1">
        <FaqSearchPanel :kb-id="kbId" />
      </TabsContent>
    </Tabs>

    <FaqEntryDialog
      v-model:open="editorOpen"
      :kb-id="kbId"
      :entry="editingEntry"
    />

    <ConfirmDialog
      :open="batchDeleteOpen"
      destructive
      title="批量删除 FAQ 条目"
      :description="`确定删除选中的 ${selectedIds.length} 个条目？删除后不可恢复。`"
      confirm-text="全部删除"
      @update:open="open => !open && (batchDeleteOpen = false)"
      @confirm="handleBatchDelete"
    />
  </div>
</template>
