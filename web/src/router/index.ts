import { createRouter, createWebHistory } from "vue-router";

import { useAuthStore, hasTenantRole } from "@/stores/auth";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/login",
      name: "login",
      component: () => import("@/pages/login/index.vue"),
      meta: { public: true },
    },
    {
      path: "/",
      component: () => import("@/layout/index.vue"),
      children: [
        // 仪表盘已移除：根路径直接落到知识问答
        { path: "", redirect: { name: "rag-chat" } },
        { path: "rag-chat", name: "rag-chat", component: () => import("@/pages/ai/rag-chat/index.vue") },
        {
          path: "model-config",
          name: "model-config",
          component: () => import("@/pages/ai/model-config/index.vue"),
          // 管理向页面：菜单不露出，路由层也拦一层
          meta: { adminOnly: true },
        },
        {
          path: "wiki/knowledge-bases",
          name: "wiki-knowledge-bases",
          component: () => import("@/pages/wiki/knowledge-bases/index.vue"),
        },
        {
          path: "wiki/knowledge-bases/detail",
          name: "wiki-knowledge-base-detail",
          component: () => import("@/pages/wiki/knowledge-bases/detail.vue"),
        },
        {
          path: "wiki/editor",
          name: "wiki-editor",
          component: () => import("@/pages/wiki/tiptap-editor/index.vue"),
        },
        { path: "account", name: "account", component: () => import("@/pages/common/account/index.vue") },
        {
          path: "system/users",
          name: "system-users",
          component: () => import("@/pages/system/users/index.vue"),
        },
        {
          path: "system/settings",
          name: "system-settings",
          component: () => import("@/pages/system/settings/index.vue"),
        },
        { path: "system/queues", name: "system-queues", component: () => import("@/pages/system/queues/index.vue") },
      ],
    },
    {
      path: "/:pathMatch(.*)*",
      name: "not-found",
      component: () => import("@/pages/errors/404.vue"),
    },
  ],
});

// 布局路由需要登录；登录页与 404 公开
router.beforeEach((to) => {
  if (to.meta.public) return true;
  const auth = useAuthStore();
  if (!auth.accessToken) {
    // 记录原始目标，登录成功后回跳
    return { path: "/login", query: { redirect: to.fullPath } };
  }
  // 管理向页面（adminOnly）：非 Admin+ 落到知识问答
  if (to.meta.adminOnly && !hasTenantRole(auth, "admin")) {
    return { path: "/rag-chat" };
  }
  return true;
});

export default router;
