#!/usr/bin/env node
// 校验 SFC 模板中使用的组件都已在 <script> 中导入。
// 背景：vue-tsc 默认不检查模板里未导入的组件（漏 import 只在运行时炸，
// 见 2026-10-01 用户管理页事故）；strictTemplates 又与 reka-ui 的透传
// 属性类型不兼容（全库 ~430 误报），故用这个零依赖静态检查补位。
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

// Vue 内置大写组件，无需导入
const VUE_BUILTINS = new Set([
  "Transition",
  "TransitionGroup",
  "KeepAlive",
  "Teleport",
  "Suspense",
  // vue-router 全局注册
  "RouterView",
  "RouterLink",
]);

const srcDir = fileURLToPath(new URL("../src", import.meta.url));
const problems = [];

function collectImportedNames(script) {
  const names = new Set();
  // 默认导入：import Foo from … / import Foo, { A } from …
  for (const m of script.matchAll(/import\s+(?:type\s+)?([A-Za-z_$][\w$]*)\s*(?:,|\bfrom\b)/g)) {
    names.add(m[1]);
  }
  // 具名导入：import { A, B as C } from …
  for (const m of script.matchAll(/import\s+(?:type\s+)?\{([^}]*)\}/g)) {
    for (const part of m[1].split(",")) {
      const name = part.split(/\s+as\s+/).pop()?.trim();
      if (name) names.add(name);
    }
  }
  return names;
}

function checkFile(file) {
  const source = readFileSync(file, "utf8");
  const template = source.match(/<template>([\s\S]*)<\/template>/)?.[1];
  if (!template) return;
  const script = source.match(/<script[^>]*>([\s\S]*?)<\/script>/)?.[1] ?? "";
  const imported = collectImportedNames(script);
  // SFC 可按文件名自引用（隐式递归），如 TreeRow.vue 内用 <TreeRow>
  const selfName = file.replace(/\.vue$/, "").split("/").pop() ?? "";
  for (const m of template.matchAll(/<([A-Z][\w$]*)/g)) {
    const tag = m[1];
    if (!imported.has(tag) && !VUE_BUILTINS.has(tag) && tag !== selfName) {
      problems.push(`${relative(fileURLToPath(new URL("../", import.meta.url)), file)}: <${tag}> 未导入`);
    }
  }
}

function walk(dir) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p);
    else if (name.endsWith(".vue")) checkFile(p);
  }
}

walk(srcDir);
if (problems.length > 0) {
  console.error("模板中存在未导入的组件：");
  for (const p of problems) console.error(`  ${p}`);
  process.exit(1);
}
console.log("模板组件导入检查通过");
