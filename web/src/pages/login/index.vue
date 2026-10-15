<script setup lang="ts">
import logo from "@/assets/logo.svg";

import { Card } from "@/components/ui/card";

import LoginBackdrop from "./LoginBackdrop.vue";
import LoginForm from "./LoginForm.vue";
import "./login.css";
</script>

<template>
  <div class="sag-login min-h-svh bg-background text-foreground">
    <LoginBackdrop />

    <main
      class="relative z-1 flex min-h-svh flex-col items-center justify-center px-4 pb-12 pt-19 md:px-6 md:pb-14 md:pt-22"
    >
      <section aria-label="登录" class="w-full max-w-lg">
        <Card class="sag-anim-card relative w-full gap-0 p-0">
          <div
            class="flex flex-col items-center px-9 pb-8 pt-9 max-[620px]:px-6 max-[620px]:pb-7 max-[620px]:pt-8"
          >
            <span
              class="grid size-16 place-items-center rounded-2xl border border-border bg-muted max-[620px]:size-14"
            >
              <img :src="logo" alt="" aria-hidden="true" class="size-10 max-[620px]:size-9" />
            </span>

            <h1 class="mt-5 text-xl font-semibold tracking-tight">登录 SAG Wiki</h1>
            <LoginForm />
          </div>
        </Card>
      </section>

      <p class="sag-anim-enter mt-6 text-[13px] [animation-delay:240ms]">
        <a class="sag-login-link font-medium underline-offset-4 hover:underline" href="#">
          还没有账号？联系管理员开通
        </a>
      </p>
    </main>
  </div>
</template>
