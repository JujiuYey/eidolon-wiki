<script setup lang="ts">
import { FileTextIcon, PlusIcon, Trash2Icon } from "@lucide/vue";
import { computed, ref } from "vue";

import { useKnowledgeBaseList } from "@/api/wiki/knowledge-base";
import { useDeleteWikiPage, useWikiPages } from "@/api/wiki/wiki-page";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";

const props = defineProps<{
  kbId: string | undefined;
  slug: string | undefined;
  title: string;
  onCreatePage: (kbId: string, title: string) => Promise<void>;
  creating: boolean;
}>();

const emit = defineEmits<{
  kbChange: [kbId: string];
  pageSelect: [slug: string];
  titleChange: [title: string];
}>();

const kbList = useKnowledgeBaseList();
const pages = useWikiPages(() => props.kbId);
const deletePage = useDeleteWikiPage(() => props.kbId ?? "");

const pagesData = computed(() => pages.data.value ?? []);
const pagesLoading = computed(() => pages.isLoading.value);
const deleting = computed(() => deletePage.isPending.value);

const newTitle = ref("");
const deleteTarget = ref<{ slug: string; title: string } | null>(null);

const handleCreate = async () => {
  if (!props.kbId || !newTitle.value.trim()) return;
  await props.onCreatePage(props.kbId, newTitle.value.trim());
  newTitle.value = "";
};

const handleKbChange = (value: string) => {
  emit("kbChange", value);
};

const handleTitleInput = (value: string | number) => {
  emit("titleChange", String(value));
};

const handleDeleteDialogOpenChange = (open: boolean) => {
  if (!open) deleteTarget.value = null;
};

const confirmDelete = async () => {
  if (!deleteTarget.value) return;
  const targetSlug = deleteTarget.value.slug;
  await deletePage.mutateAsync(targetSlug);
  deleteTarget.value = null;
  if (props.slug === targetSlug) emit("pageSelect", "");
};
</script>

<template>
  <div class="flex h-full w-64 shrink-0 flex-col gap-3 border-r bg-background p-3">
    <div class="space-y-1">
      <p class="text-xs font-medium text-muted-foreground">知识库</p>
      <Select :model-value="kbId ?? ''" @update:model-value="handleKbChange($event as string)">
        <SelectTrigger class="w-full" size="sm">
          <SelectValue placeholder="选择知识库" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem v-for="kb in kbList.data.value ?? []" :key="kb.id" :value="kb.id">
            {{ kb.name }}
          </SelectItem>
        </SelectContent>
      </Select>
    </div>

    <template v-if="kbId">
      <div v-if="slug" class="space-y-1">
        <p class="text-xs font-medium text-muted-foreground">页面标题</p>
        <Input
          class="h-8 text-sm"
          :model-value="title"
          @update:model-value="handleTitleInput"
        />
      </div>

      <div class="space-y-1">
        <p class="text-xs font-medium text-muted-foreground">新建页面</p>
        <div class="flex gap-1">
          <Input
            v-model="newTitle"
            class="h-8 text-sm"
            placeholder="页面标题"
            @keydown.enter="handleCreate()"
          />
          <Button size="icon-sm" :disabled="creating || !newTitle.trim()" @click="handleCreate()">
            <PlusIcon />
          </Button>
        </div>
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto">
        <p class="mb-1 text-xs font-medium text-muted-foreground">
          页面列表（{{ pagesData.length }}）
        </p>
        <p v-if="pagesLoading" class="p-2 text-xs text-muted-foreground">加载中...</p>
        <p v-else-if="pagesData.length === 0" class="p-2 text-xs text-muted-foreground">
          还没有页面，先在上方新建
        </p>
        <template v-else>
          <div
            v-for="page in pagesData"
            :key="page.slug"
            :class="
              cn(
                'group flex items-center gap-1 rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-muted/50',
                slug === page.slug && 'bg-muted font-medium',
              )
            "
          >
            <button
              type="button"
              class="flex min-w-0 flex-1 items-center gap-2 text-left"
              @click="emit('pageSelect', page.slug)"
            >
              <FileTextIcon class="h-4 w-4 shrink-0 text-muted-foreground" />
              <span class="truncate">{{ page.title || page.slug }}</span>
            </button>
            <button
              type="button"
              :aria-label="`删除页面 ${page.title || page.slug}`"
              class="hidden shrink-0 group-hover:block"
              @click.stop="deleteTarget = { slug: page.slug, title: page.title || page.slug }"
            >
              <Trash2Icon class="h-3.5 w-3.5 text-muted-foreground hover:text-destructive" />
            </button>
          </div>
        </template>
      </div>
    </template>
    <p v-else class="text-xs text-muted-foreground">选择知识库后即可新建和打开 Wiki 页面。</p>

    <Dialog :open="Boolean(deleteTarget)" @update:open="handleDeleteDialogOpenChange">
      <DialogContent class="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>删除页面</DialogTitle>
          <DialogDescription>
            确定删除「{{ deleteTarget?.title }}」？历史版本会一并删除。
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="deleteTarget = null">
            取消
          </Button>
          <Button variant="destructive" :disabled="deleting" @click="confirmDelete()">
            删除
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
