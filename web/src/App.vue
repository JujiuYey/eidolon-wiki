<script setup lang="ts">
import { Toaster } from "@/components/ui/sonner";
import { useThemeSync } from "@/composables/use-theme";

useThemeSync();
</script>

<template>
  <RouterView />
  <Toaster position="bottom-right" />
</template>
