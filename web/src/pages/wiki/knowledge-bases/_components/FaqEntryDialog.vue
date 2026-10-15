<script setup lang="ts">
import { PlusIcon, XIcon } from "@lucide/vue";
import { computed, reactive, ref, watch } from "vue";
import { toast } from "vue-sonner";

import { useKnowledgeTags } from "@/api/wiki/tag";
import {
  useCreateFAQEntry,
  useUpdateFAQEntry,
  type FAQEntry,
  type FAQEntryPayload,
} from "@/api/wiki/faq";
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
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";

// 字段上限与官方编辑器一致（相似问/反例 10、答案 5）
const MAX_SIMILAR = 10;
const MAX_NEGATIVE = 10;
const MAX_ANSWERS = 5;

const props = defineProps<{
  open: boolean;
  kbId: string;
  /** 传入则为编辑模式（全量替换语义） */
  entry?: FAQEntry | null;
}>();

const emit = defineEmits<{
  "update:open": [open: boolean];
}>();

const createEntry = useCreateFAQEntry(() => props.kbId);
const updateEntry = useUpdateFAQEntry(() => props.kbId);
const pending = computed(() => createEntry.isPending.value || updateEntry.isPending.value);
const isEdit = computed(() => Boolean(props.entry));

const tagsQuery = useKnowledgeTags(() => props.kbId);

const form = reactive({
  standardQuestion: "",
  similarQuestions: [] as string[],
  negativeQuestions: [] as string[],
  answers: [] as string[],
  answerStrategy: "all",
  tagId: "0",
});

watch(
  () => [props.open, props.entry] as const,
  ([open, entry]) => {
    if (!open) return;
    form.standardQuestion = entry?.standard_question ?? "";
    form.similarQuestions = [...(entry?.similar_questions ?? [])];
    form.negativeQuestions = [...(entry?.negative_questions ?? [])];
    form.answers = entry?.answers.length ? [...entry.answers] : [""];
    form.answerStrategy = entry?.answer_strategy ?? "all";
    form.tagId = String(entry?.tag_id ?? 0);
  },
  { immediate: true },
);

const addToList = (list: string[]) => {
  if (list === form.similarQuestions && list.length >= MAX_SIMILAR) return;
  if (list === form.negativeQuestions && list.length >= MAX_NEGATIVE) return;
  if (list === form.answers && list.length >= MAX_ANSWERS) return;
  list.push("");
};

const removeFromList = (list: string[], index: number) => {
  list.splice(index, 1);
};

const validationError = (): string | null => {
  if (!form.standardQuestion.trim()) return "请填写标准问题";
  const answers = form.answers.map((a) => a.trim()).filter(Boolean);
  if (answers.length === 0) return "至少填写一条答案";
  return null;
};

const handleSubmit = async () => {
  const error = validationError();
  if (error) {
    toast.error(error);
    return;
  }
  const payload: FAQEntryPayload = {
    standard_question: form.standardQuestion.trim(),
    similar_questions: form.similarQuestions.map((q) => q.trim()).filter(Boolean),
    negative_questions: form.negativeQuestions.map((q) => q.trim()).filter(Boolean),
    answers: form.answers.map((a) => a.trim()).filter(Boolean),
    answer_strategy: form.answerStrategy as "all" | "random",
    tag_id: Number(form.tagId) || 0,
  };
  if (props.entry) {
    await updateEntry.mutateAsync({ entryId: props.entry.id, data: payload });
  } else {
    await createEntry.mutateAsync(payload);
  }
  emit("update:open", false);
};

const listFields = [
  { key: "similarQuestions", label: "相似问题", hint: `用户换种问法也能命中（最多 ${MAX_SIMILAR} 条）` },
  { key: "negativeQuestions", label: "反例问题", hint: "命中这些问法时不会返回该条目" },
  { key: "answers", label: "答案", hint: `至少一条（最多 ${MAX_ANSWERS} 条）` },
] as const;
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
      <DialogHeader>
        <DialogTitle>{{ isEdit ? "编辑 FAQ 条目" : "新建 FAQ 条目" }}</DialogTitle>
        <DialogDescription>
          保存后会立即完成向量化索引（同步执行，稍等片刻）。标准问题必填，答案至少一条。
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <div class="space-y-2">
          <Label for="faq-standard-question">标准问题 <span class="text-destructive">*</span></Label>
          <Input
            id="faq-standard-question"
            v-model="form.standardQuestion"
            :maxlength="200"
            placeholder="用户最常问的原句"
          />
        </div>

        <div v-for="field in listFields" :key="field.key" class="space-y-2">
          <div class="flex items-center justify-between">
            <Label>{{ field.label }} <span v-if="field.key === 'answers'" class="text-destructive">*</span></Label>
            <Button
              type="button"
              variant="ghost"
              size="xs"
              @click="addToList(form[field.key] as string[])"
            >
              <PlusIcon />
              添加
            </Button>
          </div>
          <p class="text-xs text-muted-foreground">{{ field.hint }}</p>
          <div v-if="(form[field.key] as string[]).length === 0" class="rounded-md border border-dashed px-3 py-2 text-xs text-muted-foreground">
            暂无内容
          </div>
          <div
            v-for="(_, index) in form[field.key] as string[]"
            :key="`${field.key}-${index}`"
            class="flex items-start gap-2"
          >
            <Textarea
              v-if="field.key === 'answers'"
              v-model="(form[field.key] as string[])[index]"
              :rows="2"
              class="flex-1"
            />
            <Input
              v-else
              v-model="(form[field.key] as string[])[index]"
              class="flex-1"
              placeholder="问题"
            />
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              class="mt-1 shrink-0 text-muted-foreground hover:text-destructive"
              :aria-label="`删除第 ${index + 1} 条`"
              @click="removeFromList(form[field.key] as string[], index)"
            >
              <XIcon class="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>

        <div class="grid gap-4 sm:grid-cols-2">
          <div class="space-y-2">
            <Label>回答策略</Label>
            <Select v-model="form.answerStrategy">
              <SelectTrigger class="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部回复（所有答案一起返回）</SelectItem>
                <SelectItem value="random">随机回复（多条答案随机取一）</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label>分类标签</Label>
            <Select v-model="form.tagId">
              <SelectTrigger class="w-full">
                <SelectValue placeholder="未分类" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="0">未分类</SelectItem>
                <SelectItem
                  v-for="tag in tagsQuery.data.value ?? []"
                  :key="tag.id"
                  :value="String(tag.seq_id ?? 0)"
                >
                  {{ tag.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)">取消</Button>
        <Button :disabled="pending" @click="handleSubmit">
          {{ pending ? "保存中..." : isEdit ? "保存" : "创建" }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
