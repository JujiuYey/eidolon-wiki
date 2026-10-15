<script setup lang="ts">
import "./editor.css";
import type { Editor } from "@tiptap/core";
import { computed, onBeforeUnmount, ref, shallowRef, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { toast } from "vue-sonner";

import { wikiPageApi } from "@/api/wiki/wiki-page";
import { ScrollArea } from "@/components/ui/scroll-area";

import EditorCatalog from "./_components/EditorCatalog.vue";
import EditorFooter from "./_components/EditorFooter.vue";
import EditorToolbar from "./_components/editor-toolbar/index.vue";
import TiptapEditor from "./_components/TiptapEditor.vue";
import WikiSidebar from "./_components/WikiSidebar.vue";
import { useEditorPageSetup } from "./_hooks/useEditorPageSetup";

type SaveStatus = "saved" | "saving" | "unsaved";

const route = useRoute();
const router = useRouter();

const kbId = computed(() => (route.query.kbId as string | undefined) ?? undefined);
const slug = computed(() => (route.query.slug as string | undefined) ?? undefined);

const content = ref("");
const pageTitle = ref("");
const version = ref<number | undefined>(undefined);
const catalogOpen = ref(true);
const saveStatus = ref<SaveStatus>("saved");
const editor = shallowRef<Editor | null | undefined>(null);
const creating = ref(false);
let saveTimeout: ReturnType<typeof setTimeout> | undefined;
const pageSetup = useEditorPageSetup();

const updateSearch = (next: { kbId?: string; slug?: string }) => {
  void router.replace({
    path: "/wiki/editor",
    query: { kbId: next.kbId, slug: next.slug },
  });
};

// 加载页面内容
let loadSeq = 0;
watch(
  [kbId, slug],
  async ([nextKbId, nextSlug]) => {
    const seq = ++loadSeq;
    const cancelled = () => seq !== loadSeq;

    if (!nextKbId || !nextSlug) {
      content.value = "";
      pageTitle.value = "";
      version.value = undefined;
      saveStatus.value = "saved";
      return;
    }

    try {
      const response = await wikiPageApi.get(nextKbId, nextSlug);
      if (!cancelled() && response.data) {
        content.value = response.data.content ?? "";
        pageTitle.value = response.data.title ?? "";
        version.value = response.data.version;
        saveStatus.value = "saved";
      }
    } catch (error) {
      if (!cancelled()) {
        toast.error((error as { message?: string }).message || "页面加载失败");
        updateSearch({ kbId: nextKbId });
      }
    }
  },
  { immediate: true },
);

// 处理内容变化
const handleContentChange = (nextContent: string) => {
  content.value = nextContent;
  saveStatus.value = "unsaved";
};

const handleTitleChange = (nextTitle: string) => {
  pageTitle.value = nextTitle;
  saveStatus.value = "unsaved";
};

const handleEditorReady = (nextEditor: Editor | null | undefined) => {
  editor.value = nextEditor;
};

// 保存：PUT 带 version 乐观锁，409 时提示加载最新
const handleSave = async () => {
  const currentKbId = kbId.value;
  const currentSlug = slug.value;
  if (!currentKbId || !currentSlug) {
    return;
  }

  saveStatus.value = "saving";
  try {
    const response = await wikiPageApi.update(currentKbId, currentSlug, {
      title: pageTitle.value,
      content: content.value,
      version: version.value,
    });
    if (response.data) {
      version.value = response.data.version;
    }
    saveStatus.value = "saved";
  } catch (error) {
    saveStatus.value = "unsaved";
    const status = (error as { status?: number }).status;
    if (status === 409) {
      toast.error("页面已被其他人修改", {
        description: "点击「加载最新」会丢弃当前未保存的修改。",
        action: {
          label: "加载最新",
          onClick: () => updateSearch({ kbId: currentKbId, slug: currentSlug }),
        },
      });
    } else {
      toast.error((error as { message?: string }).message || "保存失败");
    }
  }
};

// 自动保存逻辑（2 秒防抖）
watch(
  [saveStatus, kbId, slug, content, pageTitle, version],
  () => {
    if (saveTimeout) {
      clearTimeout(saveTimeout);
    }

    if (saveStatus.value !== "unsaved" || !kbId.value || !slug.value) {
      return;
    }

    saveTimeout = setTimeout(() => {
      void handleSave();
    }, 2000);
  },
);

// 组件卸载时清除定时器
onBeforeUnmount(() => {
  if (saveTimeout) {
    clearTimeout(saveTimeout);
  }
});

// 新建页面：创建成功后跳转到新页面的 slug
const handleCreatePage = async (nextKbId: string, title: string) => {
  creating.value = true;
  try {
    const response = await wikiPageApi.create(nextKbId, { title, content: "" });
    if (response.data?.slug) {
      updateSearch({ kbId: nextKbId, slug: response.data.slug });
    }
  } finally {
    creating.value = false;
  }
};

// 文档字数
const wordCount = computed(() => content.value.split(/\s+/).filter(Boolean).length);

const handleKbChange = (nextKbId: string) => {
  updateSearch({ kbId: nextKbId });
};

const handlePageSelect = (nextSlug: string) => {
  updateSearch({ kbId: kbId.value, slug: nextSlug || undefined });
};
</script>

<template>
  <div class="flex h-screen flex-col bg-background">
    <EditorToolbar
      :editor="editor"
      :doc-title="pageTitle || '未选择页面'"
      :on-save="handleSave"
    />

    <div class="flex min-h-0 flex-1 overflow-hidden">
      <WikiSidebar
        :kb-id="kbId"
        :slug="slug"
        :title="pageTitle"
        :on-create-page="handleCreatePage"
        :creating="creating"
        @kb-change="handleKbChange"
        @page-select="handlePageSelect"
        @title-change="handleTitleChange"
      />

      <EditorCatalog
        :open="catalogOpen"
        :editor="editor"
        @toggle="catalogOpen = !catalogOpen"
      />

      <ScrollArea class="flex-1 bg-[#f5f5f5] p-6">
        <div
          class="mx-auto bg-white shadow-sm"
          :style="{
            width: `${pageSetup.paperSize.width}px`,
            zoom: pageSetup.zoom,
            paddingTop: `${pageSetup.margins.top}px`,
            paddingBottom: `${pageSetup.margins.bottom}px`,
            paddingLeft: `${pageSetup.margins.left}px`,
            paddingRight: `${pageSetup.margins.right}px`,
          }"
        >
          <TiptapEditor
            v-if="kbId && slug"
            :content="content"
            @change="handleContentChange"
            @editor-ready="handleEditorReady"
          />
          <div
            v-else
            class="flex min-h-[400px] items-center justify-center text-sm text-muted-foreground"
          >
            在左侧选择知识库并打开或新建一个页面开始编辑
          </div>
        </div>
      </ScrollArea>
    </div>

    <EditorFooter
      :save-status="saveStatus"
      :word-count="wordCount"
      :on-save="handleSave"
      :page-setup="pageSetup"
    />
  </div>
</template>
