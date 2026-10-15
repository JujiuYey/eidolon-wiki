import path from "path";

/// <reference types="vitest/config" />
import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vitest/config";

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: [
      {
        find: "@",
        replacement: path.resolve(__dirname, "src"),
      },
    ],
  },
  server: {
    port: 5200,
  },
  test: {
    environment: "jsdom",
    setupFiles: "./src/test/setup.ts",
    // react-reference/ 是只读移植参考（sag-wiki 拷贝），其 React 测试不属于本仓库
    exclude: ["**/node_modules/**", "react-reference/**"],
    passWithNoTests: true,
  },
});
