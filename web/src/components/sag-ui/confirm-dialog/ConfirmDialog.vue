<script setup lang="ts">
import { ref } from "vue";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { buttonVariants } from "@/components/ui/button";

const props = withDefaults(
  defineProps<{
    open: boolean;
    title?: string;
    description: string;
    confirmText?: string;
    cancelText?: string;
    destructive?: boolean;
    /** 确认回调；支持 Promise，等待期间按钮显示"处理中..."。模板里用 @confirm="handler" 传入即可 */
    onConfirm?: () => void | Promise<void>;
  }>(),
  {
    title: "确认操作",
    confirmText: "确定",
    cancelText: "取消",
    destructive: false,
    onConfirm: undefined,
  },
);

const emit = defineEmits<{
  "update:open": [open: boolean];
}>();

const loading = ref(false);

const handleConfirm = async () => {
  loading.value = true;
  try {
    await props.onConfirm?.();
  } finally {
    loading.value = false;
    emit("update:open", false);
  }
};
</script>

<template>
  <AlertDialog :open="props.open" @update:open="emit('update:open', $event)">
    <AlertDialogContent class="sm:max-w-sm">
      <AlertDialogHeader>
        <AlertDialogTitle>{{ props.title }}</AlertDialogTitle>
        <AlertDialogDescription>{{ props.description }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel :disabled="loading">{{ props.cancelText }}</AlertDialogCancel>
        <AlertDialogAction
          :class="buttonVariants({ variant: props.destructive ? 'destructive' : 'default' })"
          :disabled="loading"
          @click.prevent="handleConfirm"
        >
          {{ loading ? "处理中..." : props.confirmText }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
