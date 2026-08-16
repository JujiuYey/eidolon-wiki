import { defineConfig } from "@rspress/core";

export default defineConfig({
  root: "docs",
  base: "/sag-wiki/",
  lang: "zh",
  title: "SAG Wiki",
  description: "面向团队知识沉淀、检索与问答的开源 RAG 工作台。",
  logo: "/logo.svg",
  icon: "/logo.svg",
  route: {
    cleanUrls: true,
  },
  markdown: {
    link: {
      checkDeadLinks: true,
    },
  },
});
