import "./index.css";

import { VueQueryPlugin, type VueQueryPluginOptions } from "@tanstack/vue-query";
import { createPinia } from "pinia";
import piniaPluginPersistedstate from "pinia-plugin-persistedstate";
import { createApp } from "vue";

import App from "./App.vue";
import router from "./router";
import { initializeTheme } from "./composables/use-theme";

const app = createApp(App);

const pinia = createPinia();
pinia.use(piniaPluginPersistedstate);
app.use(pinia);

const queryClientOptions: VueQueryPluginOptions = {
  queryClientConfig: {
    defaultOptions: {
      queries: {
        staleTime: 1000 * 60 * 5, // 5分钟内数据被认为新鲜
        retry: 1, // 失败重试一次
      },
    },
  },
};
app.use(VueQueryPlugin, queryClientOptions);

app.use(router);

initializeTheme();

app.mount("#app");
