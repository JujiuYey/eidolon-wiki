<script setup lang="ts">
import { CustomError } from "@/components/sag-ui";
</script>

<template>
  <div class="flex min-h-svh items-center justify-center">
    <CustomError :code="404" subtitle="页面不存在" error="你访问的页面可能已被移除或地址有误。" />
  </div>
</template>
