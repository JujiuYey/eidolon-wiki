<script setup lang="ts">
import { SagPage } from "@/components/sag-ui";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import CreateUserForm from "./_components/CreateUserForm.vue";
import ResetPasswordForm from "./_components/ResetPasswordForm.vue";
</script>

<template>
  <SagPage>
    <template #title>用户管理</template>
    <template #description>
      平台级账号操作，仅系统管理员可用。
    </template>

    <Tabs default-value="create" class="flex h-full min-h-0 flex-col gap-4">
      <TabsList class="w-fit">
        <TabsTrigger value="create">创建用户</TabsTrigger>
        <TabsTrigger value="reset">重置密码</TabsTrigger>
      </TabsList>

      <TabsContent value="create" class="min-h-0 flex-1 overflow-y-auto pb-6">
        <CreateUserForm />
      </TabsContent>

      <TabsContent value="reset" class="min-h-0 flex-1 overflow-y-auto pb-6">
        <ResetPasswordForm />
      </TabsContent>
    </Tabs>
  </SagPage>
</template>
