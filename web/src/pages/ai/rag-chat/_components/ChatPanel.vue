<script setup lang="ts">
import { ArrowUpIcon, CheckIcon, LibraryIcon, Loader2Icon, RefreshCwIcon, SearchIcon, SquareIcon, XIcon } from "@lucide/vue";
import { computed, nextTick, onUnmounted, ref, watch } from "vue";
import type { AcceptableValue } from "reka-ui";
import { toast } from "vue-sonner";

import {
  chatApi,
  startKnowledgeChat,
  useChatModels,
  useMessageHistory,
} from "@/api/ai/chat";
import type { ChatMessage, KnowledgeReference, MessageSuggestionItem, StreamResponse, TokenUsage } from "@/api/ai/chat";
import { BUILTIN_QUICK_ANSWER_ID, fetchSuggestedQuestions, type SuggestedQuestion } from "@/api/ai/agent";
import { useKnowledgeBaseList } from "@/api/wiki/knowledge-base";
import { StreamMarkdown } from "@/components/stream-markdown";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";

import CitationChunkDialog from "./CitationChunkDialog.vue";
import ReferenceList from "./ReferenceList.vue";

const CONTEXT_LIMIT = 256_000;
const CONTEXT_WARNING_THRESHOLD = 200_000;
const LAST_MODEL_KEY = "weknora_last_chat_model_id";
const KB_SCOPE_KEY = "weknora_chat_kb_scope";
const SUGGESTION_POLL_INTERVAL_MS = 1_000;
const SUGGESTION_MAX_POLLS = 60;

const props = defineProps<{
  sessionId?: string;
}>();

const emit = defineEmits<{
  sessionCreated: [sessionId: string];
  sessionTitle: [sessionId: string, title: string];
  turnComplete: [];
}>();

/** 渲染用的统一消息形状（服务端历史 + 本地流式） */
interface DisplayMessage {
  id: string;
  role: "user" | "assistant";
  content: string;
  /** 流式期间的原始累积文本（含 think 标签），完成后由 content 呈现 */
  rawContent?: string;
  thinkContent?: string;
  references?: KnowledgeReference[];
  usage?: TokenUsage;
  isFallback?: boolean;
  streaming?: boolean;
}

function formatTokenCount(tokens: number): string {
  if (tokens >= 1_000_000) return `${(tokens / 1_000_000).toFixed(1)}M`;
  if (tokens >= 1_000) return `${(tokens / 1_000).toFixed(1)}k`;
  return tokens.toString();
}

function loadKbScope(): string[] {
  try {
    const parsed = JSON.parse(localStorage.getItem(KB_SCOPE_KEY) ?? "[]");
    return Array.isArray(parsed) ? parsed.filter((id): id is string => typeof id === "string") : [];
  } catch {
    return [];
  }
}

/** 拆出 <think>…</think> 思考内容（未闭合时按已开未闭处理） */
function splitThinkContent(raw: string): { think: string; answer: string } {
  const match = raw.match(/<think>([\s\S]*?)(?:<\/think>|$)/);
  if (!match) return { think: "", answer: raw };
  return {
    think: match[1] ?? "",
    answer: raw.replace(/<think>[\s\S]*?(?:<\/think>|$)/, ""),
  };
}

function toDisplay(message: ChatMessage): DisplayMessage {
  const { think, answer } = splitThinkContent(message.content);
  return {
    id: message.id,
    role: message.role === "user" ? "user" : "assistant",
    // 引用标签保留，由 StreamMarkdown 渲染为行内引用药丸
    content: message.content,
    thinkContent: think || undefined,
    references: message.knowledge_references,
    usage: message.usage,
    isFallback: message.is_fallback,
  };
}

const { data: historyData, refetch: refetchHistory, fetchNextPage, hasNextPage, isFetchingNextPage } =
  useMessageHistory(() => props.sessionId);
const { data: modelsData, isLoading: modelsLoading } = useChatModels();
const { data: kbListData } = useKnowledgeBaseList();

const liveMessages = ref<DisplayMessage[]>([]);
const inputValue = ref("");
const streaming = ref(false);
const modelId = ref(localStorage.getItem(LAST_MODEL_KEY) ?? "");

// 知识库检索范围（空 = 全部知识），跨会话持久
const selectedKbIds = ref<string[]>(loadKbScope());
const kbPopoverOpen = ref(false);
const kbSearch = ref("");

const scrollRef = ref<HTMLDivElement | null>(null);
const textareaRef = ref<HTMLTextAreaElement | null>(null);

// 追问建议（回答完成后由后端异步生成）
const suggestionItems = ref<MessageSuggestionItem[]>([]);
const suggestionStatus = ref<"idle" | "loading" | "ready">("idle");
const suggestionSessionId = ref<string | null>(null);
let suggestionToken = 0;

// 空状态推荐问题（官方空状态 starters，走内置快速问答 agent）
const starterQuestions = ref<SuggestedQuestion[]>([]);
const startersLoading = ref(false);
let startersToken = 0;

// 引用药丸点击后的分块全文
const citationReference = ref<KnowledgeReference | null>(null);

// 流式控制柄（普通对象，非响应式）
let cancelStreamRef: (() => void) | null = null;
let streamSidRef: string | null = null;
let assistantServerIdRef: string | null = null;
let loadEarlierActive = false;

const historyMessages = computed<ChatMessage[]>(
  () => [...(historyData.value?.pages ?? [])].reverse().flatMap((page) => page),
);

// 会话切换：中断进行中的流并清空本地消息
watch(
  () => props.sessionId,
  () => {
    cancelStreamRef?.();
    cancelStreamRef = null;
    liveMessages.value = [];
    streaming.value = false;
    clearSuggestions();
  },
);

// 模型默认值：上次选择 > 第一个可用
watch(
  () => [modelsData.value, modelId.value] as const,
  ([data]) => {
    if (data?.length) {
      const current = data.find((model) => model.id === modelId.value);
      if (!current) {
        const fallback = data.find((model) => model.is_default) ?? data[0];
        if (fallback?.id) modelId.value = fallback.id;
      }
    }
  },
);

// 持久化知识库范围，并剔除已不存在的库
watch(selectedKbIds, (ids) => {
  localStorage.setItem(KB_SCOPE_KEY, JSON.stringify(ids));
});
watch(
  kbListData,
  (list) => {
    if (!list) return;
    const valid = new Set(list.map((kb) => kb.id));
    const filtered = selectedKbIds.value.filter((id) => valid.has(id));
    if (filtered.length !== selectedKbIds.value.length) selectedKbIds.value = filtered;
  },
  { immediate: true },
);

const filteredKbs = computed(() => {
  const keyword = kbSearch.value.trim().toLowerCase();
  const items = kbListData.value ?? [];
  if (!keyword) return items;
  return items.filter((kb) => kb.name.toLowerCase().includes(keyword));
});

const toggleKb = (id: string) => {
  selectedKbIds.value = selectedKbIds.value.includes(id)
    ? selectedKbIds.value.filter((kbId) => kbId !== id)
    : [...selectedKbIds.value, id];
};

const scrollToBottom = () => {
  requestAnimationFrame(() => {
    const el = scrollRef.value;
    if (el) el.scrollTop = el.scrollHeight;
  });
};

// 追加更早消息时保持滚动位置，其余场景（首屏/流式）贴底
watch(
  () => historyMessages.value.length,
  () => {
    if (!loadEarlierActive) scrollToBottom();
  },
);
watch(liveMessages, () => scrollToBottom());

const loadEarlier = async () => {
  if (loadEarlierActive || !hasNextPage.value) return;
  loadEarlierActive = true;
  const el = scrollRef.value;
  const prevHeight = el?.scrollHeight ?? 0;
  try {
    await fetchNextPage();
    await nextTick();
    const target = scrollRef.value;
    if (target) target.scrollTop += target.scrollHeight - prevHeight;
  } finally {
    loadEarlierActive = false;
  }
};

onUnmounted(() => {
  cancelStreamRef?.();
  cancelStreamRef = null;
  suggestionToken++;
  startersToken++;
});

const updateLive = (id: string, updater: (message: DisplayMessage) => DisplayMessage) => {
  liveMessages.value = liveMessages.value.map((message) => (message.id === id ? updater(message) : message));
};

const handleEvent = (sid: string, assistantId: string, event: StreamResponse) => {
  switch (event.response_type) {
    case "agent_query": {
      const serverId = event.assistant_message_id || event.data?.assistant_message_id;
      if (typeof serverId === "string" && serverId) assistantServerIdRef = serverId;
      break;
    }
    case "thinking": {
      const chunk = event.content ?? "";
      if (chunk) {
        updateLive(assistantId, (message) => ({
          ...message,
          thinkContent: (message.thinkContent ?? "") + chunk,
        }));
      }
      break;
    }
    case "references": {
      const refs = event.data?.references ?? event.knowledge_references;
      if (refs?.length) {
        updateLive(assistantId, (message) => ({ ...message, references: refs }));
      }
      break;
    }
    case "answer": {
      const chunk = event.content ?? "";
      if (chunk) {
        updateLive(assistantId, (message) => {
          const raw = (message.rawContent ?? "") + chunk;
          const { think, answer } = splitThinkContent(raw);
          return {
            ...message,
            rawContent: raw,
            thinkContent: think || message.thinkContent,
            content: answer,
          };
        });
      }
      break;
    }
    case "session_title": {
      const title = event.data?.title ?? event.content;
      if (typeof title === "string" && title) emit("sessionTitle", sid, title);
      break;
    }
    case "complete": {
      const usage = event.data?.usage ?? event.usage;
      if (usage) {
        updateLive(assistantId, (message) => ({ ...message, usage }));
      }
      updateLive(assistantId, (message) => ({ ...message, streaming: false }));
      break;
    }
    case "stop": {
      toast.info("已停止生成");
      updateLive(assistantId, (message) => ({ ...message, streaming: false }));
      break;
    }
    case "error": {
      const text = event.content || (event.data?.error as string | undefined) || "生成失败";
      toast.error(text);
      updateLive(assistantId, (message) => ({ ...message, streaming: false }));
      break;
    }
    default:
      break;
  }
  scrollToBottom();
};

const clearSuggestions = () => {
  suggestionToken++;
  suggestionItems.value = [];
  suggestionStatus.value = "idle";
  suggestionSessionId.value = null;
};

/** 回答完成后拉取追问建议：ensure 触发生成，1s 轮询至 ready（参考实现 chat/index.vue loadFollowUpSuggestions） */
const loadSuggestions = async (sid: string, messageId: string) => {
  const token = ++suggestionToken;
  suggestionSessionId.value = sid;
  suggestionStatus.value = "loading";
  suggestionItems.value = [];
  try {
    let set = (await chatApi.ensureSuggestions(sid, messageId)).data;
    for (let poll = 0; set?.status === "generating" && poll < SUGGESTION_MAX_POLLS; poll++) {
      await new Promise((resolve) => setTimeout(resolve, SUGGESTION_POLL_INTERVAL_MS));
      if (token !== suggestionToken) return;
      set = (await chatApi.getSuggestions(sid, messageId)).data;
    }
    if (token !== suggestionToken) return;
    if (set?.status === "ready" && set.questions?.length) {
      suggestionItems.value = set.questions;
      suggestionStatus.value = "ready";
    } else {
      clearSuggestions();
    }
  } catch {
    // 建议生成失败静默降级，不影响主流程
    if (token === suggestionToken) clearSuggestions();
  }
};

const handleSend = async (text?: string) => {
  const query = (text ?? inputValue.value).trim();
  if (!query || streaming.value) return;

  inputValue.value = "";
  if (textareaRef.value) textareaRef.value.style.height = "auto";
  clearSuggestions();

  let sid = props.sessionId;
  if (!sid) {
    try {
      const res = await chatApi.createSession({});
      sid = res.data?.id;
    } catch (error) {
      toast.error((error as { message?: string }).message || "创建会话失败");
      return;
    }
    if (!sid) {
      toast.error("创建会话失败");
      return;
    }
    emit("sessionCreated", sid);
  }

  const userLocalId = `local-user-${Date.now()}`;
  const assistantLocalId = `local-assistant-${Date.now()}`;
  streamSidRef = sid;
  assistantServerIdRef = null;

  liveMessages.value = [
    { id: userLocalId, role: "user", content: query },
    {
      id: assistantLocalId,
      role: "assistant",
      content: "",
      rawContent: "",
      streaming: true,
    },
  ];
  streaming.value = true;

  cancelStreamRef = startKnowledgeChat(
    sid,
    {
      query,
      // 选了知识库则限定检索范围；缺省 = 全部知识（官方语义）
      knowledge_base_ids: selectedKbIds.value.length ? selectedKbIds.value : undefined,
      summary_model_id: modelId.value || undefined,
    },
    (event) => handleEvent(sid!, assistantLocalId, event),
    async () => {
      // 完成（含用户主动停止的中断）
      streaming.value = false;
      cancelStreamRef = null;
      if (streamSidRef === sid) {
        await refetchHistory();
        liveMessages.value = [];
        const messageId = assistantServerIdRef;
        if (messageId) void loadSuggestions(sid, messageId);
      }
      emit("turnComplete");
    },
    (error) => {
      streaming.value = false;
      cancelStreamRef = null;
      toast.error(error.message || "请求失败");
      if (streamSidRef === sid) {
        liveMessages.value = liveMessages.value.filter((message) => message.id !== assistantLocalId);
      }
    },
  );
};

const handleStop = () => {
  cancelStreamRef?.();
  const sid = streamSidRef;
  const messageId = assistantServerIdRef;
  if (sid && messageId) {
    chatApi.stopGeneration(sid, messageId).catch(() => {
      // 停止接口失败不阻塞 UI，流已中断
    });
  }
};

const totalTokens = computed(() => {
  const historyTokens = historyMessages.value.reduce(
    (sum, message) => sum + (message.usage?.total_tokens ?? 0),
    0,
  );
  const liveTokens = liveMessages.value.reduce(
    (sum, message) => sum + (message.usage?.total_tokens ?? 0),
    0,
  );
  return historyTokens + liveTokens;
});

const displayMessages = computed<DisplayMessage[]>(
  () => [...historyMessages.value.map(toDisplay), ...liveMessages.value],
);

// 只在空会话拉推荐问题（对齐官方 fetchSuggestedQuestionsIfNeeded）
const loadStarterQuestions = async () => {
  if (displayMessages.value.length > 0) return;
  const token = ++startersToken;
  startersLoading.value = true;
  try {
    const questions = await fetchSuggestedQuestions(BUILTIN_QUICK_ANSWER_ID, {
      knowledgeBaseIds: selectedKbIds.value.length ? selectedKbIds.value : undefined,
      limit: 6,
    });
    if (token === startersToken) starterQuestions.value = questions;
  } catch {
    // 拉不到推荐问题静默降级为纯标题空状态
    if (token === startersToken) starterQuestions.value = [];
  } finally {
    if (token === startersToken) startersLoading.value = false;
  }
};

watch(
  () => displayMessages.value.length,
  (count) => {
    if (count === 0) void loadStarterQuestions();
    else startersToken++; // 会话已有消息，作废在途请求
  },
  { immediate: true },
);

// 空会话下调整知识库范围，推荐问题跟随刷新
watch(selectedKbIds, () => {
  if (displayMessages.value.length === 0) void loadStarterQuestions();
});

const showSuggestions = computed(
  () =>
    !streaming.value &&
    suggestionStatus.value !== "idle" &&
    suggestionSessionId.value !== null &&
    suggestionSessionId.value === props.sessionId &&
    displayMessages.value.length > 0,
);

const handleCitationClick = (message: DisplayMessage, chunkId: string) => {
  const refs = message.references ?? [];
  const hit =
    refs.find((ref) => ref.id === chunkId) ??
    refs.find((ref) => ref.knowledge_id === chunkId) ??
    refs.find((ref) => (ref.knowledge_title || ref.knowledge_filename) === chunkId);
  if (hit) {
    citationReference.value = hit;
  } else {
    toast.error("未找到对应的引用来源");
  }
};

const handleKeyDown = (event: KeyboardEvent) => {
  if (event.key === "Enter" && !event.shiftKey) {
    event.preventDefault();
    void handleSend();
  }
};

const handleTextareaInput = (event: Event) => {
  const el = event.target as HTMLTextAreaElement;
  inputValue.value = el.value;
  el.style.height = "auto";
  el.style.height = `${Math.min(el.scrollHeight, 200)}px`;
};

const handleModelChange = (value: AcceptableValue) => {
  modelId.value = String(value);
  localStorage.setItem(LAST_MODEL_KEY, String(value));
};
</script>

<template>
  <div class="flex h-full flex-col bg-background">
    <div ref="scrollRef" class="flex-1 space-y-6 overflow-y-auto p-6">
      <div
        v-if="hasNextPage"
        class="flex justify-center"
      >
        <Button variant="outline" size="sm" :disabled="isFetchingNextPage" @click="loadEarlier">
          <Loader2Icon v-if="isFetchingNextPage" class="mr-1 h-3.5 w-3.5 animate-spin" />
          {{ isFetchingNextPage ? "加载中…" : "加载更早消息" }}
        </Button>
      </div>
      <div
        v-if="displayMessages.length === 0"
        class="flex h-full flex-col items-center justify-center gap-3 text-center"
      >
        <h2 class="text-xl font-semibold">知识问答</h2>
        <p class="max-w-md text-sm text-muted-foreground">
          输入框左下角可选择知识库范围，回答会基于所选库内文档并附引用来源；不选范围则基于全部知识作答。
        </p>
        <!-- 空状态推荐问题（官方 starters，hybrid 模式：FAQ/配置/知识库生成） -->
        <div
          v-if="startersLoading && starterQuestions.length === 0"
          class="grid w-full max-w-2xl grid-cols-1 gap-2 pt-2 sm:grid-cols-2"
        >
          <Skeleton v-for="n in 4" :key="n" class="h-10 w-full" />
        </div>
        <div v-else-if="starterQuestions.length" class="w-full max-w-2xl pt-2">
          <div class="mb-2 flex items-center justify-center gap-1.5 text-xs text-muted-foreground">
            <span>试试这些问题</span>
            <button
              type="button"
              aria-label="换一批推荐问题"
              class="rounded p-0.5 transition-colors hover:bg-muted"
              :disabled="startersLoading"
              @click="loadStarterQuestions"
            >
              <RefreshCwIcon :class="startersLoading ? 'h-3.5 w-3.5 animate-spin' : 'h-3.5 w-3.5'" />
            </button>
          </div>
          <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <button
              v-for="item in starterQuestions"
              :key="item.question"
              type="button"
              class="flex items-center justify-between gap-2 rounded-lg border bg-background px-3 py-2 text-left text-sm transition-colors hover:bg-muted"
              @click="handleSend(item.question)"
            >
              <span class="min-w-0 flex-1 truncate">{{ item.question }}</span>
              <Badge v-if="item.source === 'faq'" variant="secondary" class="shrink-0 px-1 py-0 text-[10px]">
                FAQ
              </Badge>
            </button>
          </div>
        </div>
      </div>
      <template v-else>
        <div
          v-for="message in displayMessages"
          :key="message.id"
          class="flex"
          :class="message.role === 'user' ? 'justify-end' : 'justify-start'"
        >
          <div
            class="max-w-[85%] rounded-lg px-4 py-3 text-sm"
            :class="
              message.role === 'user'
                ? 'bg-primary text-primary-foreground'
                : 'bg-muted wrap-break-word'
            "
          >
            <template v-if="message.role === 'assistant'">
              <details v-if="message.thinkContent" class="mb-2 rounded-md border bg-background/60 px-3 py-2">
                <summary class="cursor-pointer text-xs text-muted-foreground">思考过程</summary>
                <p class="mt-2 whitespace-pre-wrap text-xs leading-relaxed text-muted-foreground">
                  {{ message.thinkContent }}
                </p>
              </details>
              <StreamMarkdown
                v-if="message.content"
                :content="message.content"
                :streaming="message.streaming"
                :citations="message.references"
                @citation-click="(chunkId) => handleCitationClick(message, chunkId)"
              />
              <span v-else-if="message.streaming" class="flex items-center gap-2 text-muted-foreground">
                <Loader2Icon class="h-4 w-4 animate-spin" />
                正在检索并生成回答…
              </span>
              <p v-if="message.isFallback" class="mt-2 text-xs text-amber-600">
                未命中知识库，以下为模型直接回答
              </p>
              <ReferenceList v-if="message.references" :references="message.references" />
              <p
                v-if="message.usage?.total_tokens"
                class="mt-2 border-t border-border/50 pt-2 text-xs text-muted-foreground"
              >
                Token：输入 {{ formatTokenCount(message.usage.prompt_tokens ?? 0) }} · 输出
                {{ formatTokenCount(message.usage.completion_tokens ?? 0) }} · 共
                {{ formatTokenCount(message.usage.total_tokens ?? 0) }}
              </p>
            </template>
            <p v-else class="whitespace-pre-wrap">{{ message.content }}</p>
          </div>
        </div>

        <!-- 追问建议（最新一轮回答完成后） -->
        <div v-if="showSuggestions" class="flex justify-start">
          <div v-if="suggestionStatus === 'loading'" class="flex items-center gap-2 text-xs text-muted-foreground">
            <Loader2Icon class="h-3.5 w-3.5 animate-spin" />
            正在生成追问建议…
          </div>
          <div v-else class="flex flex-wrap items-center gap-2">
            <span class="text-xs text-muted-foreground">继续问：</span>
            <button
              v-for="item in suggestionItems"
              :key="item.id"
              type="button"
              class="rounded-full border bg-background px-3 py-1 text-xs transition-colors hover:bg-muted"
              @click="handleSend(item.text)"
            >
              {{ item.text }}
            </button>
            <button
              type="button"
              aria-label="关闭追问建议"
              class="rounded-full p-1 text-muted-foreground transition-colors hover:bg-muted"
              @click="clearSuggestions"
            >
              <XIcon class="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      </template>
    </div>

    <div v-if="totalTokens >= CONTEXT_WARNING_THRESHOLD" class="border-t bg-muted/30 px-6 py-2">
      <p class="mx-auto max-w-4xl text-xs">
        <span class="text-muted-foreground">
          会话 Token 消耗：<strong class="text-foreground">{{ formatTokenCount(totalTokens) }}</strong> /
          {{ formatTokenCount(CONTEXT_LIMIT) }}
        </span>
        <span v-if="totalTokens >= CONTEXT_LIMIT" class="ml-3 font-medium text-destructive">
          已超出上下文限制，建议开启新一轮对话
        </span>
        <span v-else class="ml-3 font-medium text-orange-500">消耗较高，建议开启新一轮对话</span>
      </p>
    </div>

    <div class="border-t p-4">
      <div class="mx-auto max-w-4xl">
        <div class="rounded-lg border bg-background p-2">
          <textarea
            ref="textareaRef"
            :value="inputValue"
            placeholder="输入消息…（Enter 发送，Shift+Enter 换行）"
            class="max-h-50 min-h-9 w-full resize-none bg-transparent px-2 py-1.5 text-sm outline-none placeholder:text-muted-foreground"
            rows="1"
            @input="handleTextareaInput"
            @keydown="handleKeyDown"
          />
          <div class="flex items-center gap-2">
            <!-- 知识库检索范围 -->
            <Popover v-model:open="kbPopoverOpen">
              <PopoverTrigger as-child>
                <Button variant="outline" size="sm" class="h-8 shrink-0 gap-1.5 text-xs">
                  <LibraryIcon class="h-3.5 w-3.5" />
                  {{ selectedKbIds.length ? `知识库 · ${selectedKbIds.length}` : "全部知识" }}
                </Button>
              </PopoverTrigger>
              <PopoverContent align="start" class="w-72 p-0">
                <div class="border-b p-2">
                  <div class="relative">
                    <SearchIcon class="absolute left-2 top-2 h-4 w-4 text-muted-foreground" />
                    <Input v-model="kbSearch" placeholder="搜索知识库" class="h-8 pl-8 text-sm" />
                  </div>
                </div>
                <div class="max-h-64 overflow-y-auto p-1">
                  <button
                    v-for="kb in filteredKbs"
                    :key="kb.id"
                    type="button"
                    class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors hover:bg-muted"
                    @click="toggleKb(kb.id)"
                  >
                    <span
                      class="flex h-4 w-4 shrink-0 items-center justify-center rounded-full border"
                      :class="
                        selectedKbIds.includes(kb.id)
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-muted-foreground/40'
                      "
                    >
                      <CheckIcon v-if="selectedKbIds.includes(kb.id)" class="h-3 w-3" />
                    </span>
                    <span class="min-w-0 flex-1 truncate">{{ kb.name }}</span>
                    <Badge v-if="kb.type === 'faq'" variant="secondary" class="px-1 py-0 text-[10px]">
                      FAQ
                    </Badge>
                    <span class="shrink-0 text-xs text-muted-foreground">{{ kb.knowledge_count ?? 0 }}</span>
                  </button>
                  <p v-if="filteredKbs.length === 0" class="p-4 text-center text-xs text-muted-foreground">
                    没有匹配的知识库
                  </p>
                </div>
                <div class="flex items-center justify-between border-t px-3 py-2">
                  <span class="text-xs text-muted-foreground">不选 = 检索全部知识</span>
                  <Button
                    v-if="selectedKbIds.length"
                    variant="ghost"
                    size="sm"
                    class="h-7 text-xs"
                    @click="selectedKbIds = []"
                  >
                    清空
                  </Button>
                </div>
              </PopoverContent>
            </Popover>

            <div class="ml-auto flex items-center gap-2">
              <Select :model-value="modelId || undefined" @update:model-value="handleModelChange">
                <SelectTrigger size="sm" class="hidden w-44 shrink-0 md:flex">
                  <SelectValue :placeholder="modelsData?.length ? '选择模型' : '无可用模型'" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="model in modelsData ?? []"
                    :key="model.id ?? model.name"
                    :value="model.id ?? ''"
                  >
                    {{ model.display_name || model.name }}
                  </SelectItem>
                </SelectContent>
              </Select>

              <Button
                v-if="streaming"
                size="icon"
                variant="outline"
                aria-label="停止生成"
                @click="handleStop"
              >
                <SquareIcon class="h-4 w-4" />
              </Button>
              <Button
                v-else
                size="icon"
                aria-label="发送"
                :disabled="!inputValue.trim() || modelsLoading"
                @click="handleSend()"
              >
                <ArrowUpIcon class="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <CitationChunkDialog :reference="citationReference" @close="citationReference = null" />
  </div>
</template>
