<script setup lang="ts">
import {
  BookmarkIcon,
  BookOpenIcon,
  DatabaseIcon,
  FileTextIcon,
  MessageCircleIcon,
  SearchIcon,
  SparklesIcon,
  TagsIcon,
} from "@lucide/vue";
import type { Component, CSSProperties } from "vue";

/* 三级漂浮物：medium=带图标的知识卡，tile=图标磁贴，chunk=纯文字条碎片 */
type ChipKind = "medium" | "tile" | "chunk";

interface AmbientChip {
  id: string;
  kind: ChipKind;
  icon?: Component;
  /* 桌面端相对视口中心的定位，错落在登录卡四周的安全环外（卡半宽 256px + 间距） */
  position: CSSProperties;
  rotate: string;
  duration: string;
  delay: string;
  barWidth?: string;
  opacity: string;
  /* 远角元素只在更宽的视口出现，避免在 1024px 被裁切 */
  breakpoint?: "lg" | "xl";
}

const chips: AmbientChip[] = [
  {
    id: "document",
    kind: "medium",
    icon: FileTextIcon,
    position: { left: "calc(50% - 480px)", top: "21%" },
    rotate: "-5deg",
    duration: "10s",
    delay: "0s",
    barWidth: "w-19",
    opacity: "opacity-100",
  },
  {
    id: "knowledge",
    kind: "medium",
    icon: BookOpenIcon,
    position: { right: "calc(50% - 480px)", top: "17%" },
    rotate: "4deg",
    duration: "12.5s",
    delay: "-4s",
    barWidth: "w-14",
    opacity: "opacity-100",
  },
  {
    id: "chat",
    kind: "medium",
    icon: MessageCircleIcon,
    position: { right: "calc(50% - 468px)", bottom: "22%" },
    rotate: "-4deg",
    duration: "10.5s",
    delay: "-8s",
    barWidth: "w-17",
    opacity: "opacity-100",
  },
  {
    id: "search",
    kind: "medium",
    icon: SearchIcon,
    position: { left: "calc(50% - 468px)", bottom: "26%" },
    rotate: "3deg",
    duration: "13.5s",
    delay: "-2s",
    barWidth: "w-15",
    opacity: "opacity-100",
  },
  {
    id: "vector-store",
    kind: "tile",
    icon: DatabaseIcon,
    position: { right: "calc(50% - 348px)", top: "37%" },
    rotate: "6deg",
    duration: "9s",
    delay: "-6s",
    opacity: "opacity-90",
  },
  {
    id: "tags",
    kind: "tile",
    icon: TagsIcon,
    position: { left: "calc(50% - 348px)", bottom: "36%" },
    rotate: "-7deg",
    duration: "15.5s",
    delay: "-10s",
    opacity: "opacity-90",
  },
  {
    id: "chunk-left",
    kind: "chunk",
    position: { left: "calc(50% - 420px)", top: "46%" },
    rotate: "-3deg",
    duration: "9.5s",
    delay: "-3s",
    opacity: "opacity-75",
  },
  {
    id: "chunk-right",
    kind: "chunk",
    position: { right: "calc(50% - 424px)", bottom: "44%" },
    rotate: "5deg",
    duration: "11.5s",
    delay: "-7s",
    opacity: "opacity-75",
  },
  {
    id: "sparkles",
    kind: "tile",
    icon: SparklesIcon,
    position: { right: "calc(50% - 580px)", top: "10%" },
    rotate: "-3deg",
    duration: "14s",
    delay: "-5s",
    opacity: "opacity-70",
    breakpoint: "xl",
  },
  {
    id: "bookmark",
    kind: "tile",
    icon: BookmarkIcon,
    position: { left: "calc(50% - 576px)", bottom: "12%" },
    rotate: "5deg",
    duration: "16.5s",
    delay: "-12s",
    opacity: "opacity-70",
    breakpoint: "xl",
  },
];

/* 远端向量点：四角错落明灭的微小光点，填充碎片环之外的远角空旷区 */
const starDots = [
  {
    id: "star-1",
    position: "left-[7%] top-[14%]",
    tone: "bg-primary/35",
    size: "size-1.5",
    duration: "4.5s",
    delay: "-1s",
  },
  {
    id: "star-2",
    position: "left-[14%] top-[68%]",
    tone: "bg-foreground/25",
    size: "size-1",
    duration: "6s",
    delay: "-3s",
  },
  {
    id: "star-3",
    position: "left-[5%] top-[46%]",
    tone: "bg-foreground/25",
    size: "size-1",
    duration: "5.5s",
    delay: "-5s",
  },
  {
    id: "star-4",
    position: "right-[8%] top-[18%]",
    tone: "bg-primary/35",
    size: "size-1.5",
    duration: "7s",
    delay: "-2s",
  },
  {
    id: "star-5",
    position: "right-[5%] top-[56%]",
    tone: "bg-foreground/25",
    size: "size-1",
    duration: "5s",
    delay: "-4s",
  },
  {
    id: "star-6",
    position: "right-[16%] bottom-[12%]",
    tone: "bg-primary/35",
    size: "size-1.5",
    duration: "6s",
    delay: "-6s",
  },
  {
    id: "star-7",
    position: "left-[24%] top-[7%]",
    tone: "bg-foreground/25",
    size: "size-1",
    duration: "6.5s",
    delay: "-2.5s",
  },
  {
    id: "star-8",
    position: "right-[26%] bottom-[6%]",
    tone: "bg-primary/35",
    size: "size-1.5",
    duration: "4.5s",
    delay: "-0.5s",
  },
];
</script>

<template>
  <div
    aria-hidden="true"
    class="pointer-events-none fixed inset-0 z-0 overflow-hidden"
  >
    <div
      class="sag-login-glow sag-login-breathe absolute left-1/2 top-[38%] size-224 -translate-x-1/2 -translate-y-1/2"
      style="--breathe-duration: 11s"
    />
    <div
      class="sag-login-glow--soft sag-login-breathe absolute bottom-[-15%] left-[10%] size-160"
      style="--breathe-duration: 15s; --breathe-delay: -7s"
    />
    <div class="sag-login-dots absolute inset-0" />

    <div
      class="sag-anim-enter absolute left-1/2 top-[48%] hidden size-225 -translate-x-1/2 -translate-y-1/2 delay-200 lg:block"
    >
      <svg
        viewBox="0 0 100 100"
        class="sag-login-orbit size-full text-primary/20"
        style="--orbit-duration: 56s"
      >
        <circle
          cx="50"
          cy="50"
          r="49.5"
          fill="none"
          stroke="currentColor"
          stroke-width="1"
          vector-effect="non-scaling-stroke"
          pathLength="100"
          stroke-dasharray="1 3"
        />
        <circle class="text-primary/50" cx="50" cy="0.5" r="0.4" fill="currentColor" />
      </svg>
    </div>

    <div
      class="sag-anim-enter absolute left-[56%] top-[45%] hidden size-160 -translate-x-1/2 -translate-y-1/2 delay-300 lg:block"
    >
      <svg
        viewBox="0 0 100 100"
        class="sag-login-orbit sag-login-orbit--reverse size-full text-foreground/15"
        style="--orbit-duration: 38s"
      >
        <circle
          cx="50"
          cy="50"
          r="49.5"
          fill="none"
          stroke="currentColor"
          stroke-width="1"
          vector-effect="non-scaling-stroke"
          pathLength="100"
          stroke-dasharray="1 3"
        />
      </svg>
    </div>

    <span
      v-for="dot in starDots"
      :key="dot.id"
      class="sag-login-dot absolute rounded-full"
      :class="[dot.position, dot.tone, dot.size]"
      :style="{ '--dot-duration': dot.duration, '--dot-delay': dot.delay }"
    />

    <span class="sag-login-cross left-[13%] top-[33%] hidden lg:block" />
    <span class="sag-login-cross right-[15%] bottom-[24%] hidden lg:block" />
    <span class="sag-login-cross left-[28%] bottom-[10%] hidden xl:block" />

    <div
      v-for="chip in chips"
      :key="chip.id"
      class="sag-login-chip absolute hidden"
      :class="[chip.breakpoint === 'xl' ? 'xl:block' : 'lg:block', chip.opacity]"
      :style="{
        ...chip.position,
        '--chip-rotate': chip.rotate,
        '--chip-duration': chip.duration,
        '--chip-delay': chip.delay,
      }"
    >
      <!-- 图标磁贴 -->
      <div
        v-if="chip.kind === 'tile' && chip.icon"
        class="grid size-10 place-items-center rounded-xl border border-border bg-card text-muted-foreground shadow-md"
      >
        <component :is="chip.icon" class="size-4.5" aria-hidden="true" />
      </div>

      <!-- 纯文字条碎片 -->
      <div
        v-else-if="chip.kind === 'chunk'"
        class="flex w-32 flex-col gap-1.5 rounded-xl border border-border bg-card p-3 shadow-md"
      >
        <span class="h-1.5 w-full rounded-full bg-muted-foreground/40" />
        <span class="h-1.5 w-4/5 rounded-full bg-muted-foreground/25" />
        <span class="h-1.5 w-3/5 rounded-full bg-muted-foreground/25" />
      </div>

      <!-- 带图标的知识卡 -->
      <div
        v-else
        class="flex w-40 items-center gap-3 rounded-2xl border border-border bg-card p-3.5 shadow-md"
      >
        <span class="grid size-9 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary">
          <component :is="chip.icon" class="size-4.5" aria-hidden="true" />
        </span>
        <span class="flex flex-1 flex-col gap-1.5 pr-1">
          <span class="h-1.5 rounded-full bg-muted-foreground/45" :class="chip.barWidth" />
          <span class="h-1.5 w-3/5 rounded-full bg-muted-foreground/25" />
        </span>
      </div>
    </div>
  </div>
</template>
