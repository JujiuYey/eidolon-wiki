import { Settings2Icon, UserCogIcon } from "@lucide/vue";
import { h, type Component, type SVGAttributes } from "vue";

import { cn } from "@/lib/utils";

export type SidebarNavIcon = Component<SVGAttributes>;

const BASE = "sidebar-nav-icon !size-5 group-data-[collapsible=icon]:!size-6";

// 用 lucide 线性图标包装，保持与自绘图标相同的侧边栏尺寸类
function lucideIcon(Icon: Component) {
  return (({ class: className, ...rest }: SVGAttributes) =>
    h(Icon, { ...rest, class: cn(BASE, className) })) as SidebarNavIcon;
}

function sidebarIcon(
  { class: className, ...rest }: SVGAttributes,
  // SVG 子元素节点数组（h() 调用的第三个参数）
  children: any[],
) {
  return h(
    "svg",
    {
      "aria-hidden": "true",
      class: cn(BASE, className),
      fill: "none",
      focusable: "false",
      viewBox: "0 0 24 24",
      ...rest,
    },
    children,
  );
}

export const UserManageIcon: SidebarNavIcon = lucideIcon(UserCogIcon);

export const SystemSettingsIcon: SidebarNavIcon = lucideIcon(Settings2Icon);

export const ChatIcon: SidebarNavIcon = ({ class: className, ...rest }) =>
  sidebarIcon({ class: className, ...rest }, [
    h("path", {
      "data-tone": "blue",
      d: "M6.5 4h11A2.5 2.5 0 0 1 20 6.5v7A2.5 2.5 0 0 1 17.5 16H12l-4.25 3.25c-.65.5-1.55.04-1.55-.78V16H6.5A2.5 2.5 0 0 1 4 13.5v-7A2.5 2.5 0 0 1 6.5 4Z",
      fill: "currentColor",
    }),
    h("circle", { "data-tone": "white", cx: "9", cy: "10", r: "1.2", fill: "currentColor" }),
    h("circle", { "data-tone": "white", cx: "12", cy: "10", r: "1.2", fill: "currentColor" }),
    h("circle", { "data-tone": "cyan", cx: "15", cy: "10", r: "1.2", fill: "currentColor" }),
  ]);

export const ModelConfigIcon: SidebarNavIcon = ({ class: className, ...rest }) =>
  sidebarIcon({ class: className, ...rest }, [
    h("path", {
      "data-tone": "blue",
      d: "M13.1 2.8 4.6 13.2c-.75.92-.1 2.3 1.1 2.3h5.1l-1 5.1c-.28 1.43 1.51 2.31 2.47 1.2l8.93-10.35c.8-.92.14-2.36-1.08-2.36H14.6l1.02-4.93c.3-1.43-1.58-2.45-2.52-1.36Z",
      fill: "currentColor",
    }),
    h("path", {
      "data-tone": "amber",
      d: "m11.2 9.3-2.1 3.1h4.2l-.55 3.1 3.55-4.45h-4.45l.6-1.75h-1.25Z",
      fill: "currentColor",
    }),
  ]);

export const FolderIcon: SidebarNavIcon = ({ class: className, ...rest }) =>
  sidebarIcon({ class: className, ...rest }, [
    h("path", {
      "data-tone": "amber",
      d: "M3.5 7.5A2.5 2.5 0 0 1 6 5h3.2c.58 0 1.13.2 1.56.56l1.05.88c.43.36.98.56 1.54.56H18A2.5 2.5 0 0 1 20.5 9.5V17A3 3 0 0 1 17.5 20h-11A3 3 0 0 1 3.5 17V7.5Z",
      fill: "currentColor",
    }),
    h("path", {
      "data-tone": "blue",
      d: "M3.5 10.5h17V17a3 3 0 0 1-3 3h-11a3 3 0 0 1-3-3v-6.5Z",
      fill: "currentColor",
    }),
    h("path", { "data-tone": "white", d: "M7.5 14h5", stroke: "currentColor", "stroke-linecap": "round", "stroke-width": "1.8" }),
  ]);

export const DocumentIcon: SidebarNavIcon = ({ class: className, ...rest }) =>
  sidebarIcon({ class: className, ...rest }, [
    h("path", {
      "data-tone": "blue",
      d: "M6 3h8.4L19 7.6V19a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z",
      fill: "currentColor",
    }),
    h("path", { "data-tone": "cyan", d: "M14 3v4a1 1 0 0 0 1 1h4", fill: "currentColor" }),
    h("path", {
      "data-tone": "white",
      d: "M8 11h7M8 14.5h7M8 18h4.5",
      stroke: "currentColor",
      "stroke-linecap": "round",
      "stroke-width": "1.8",
    }),
  ]);

export const ComposeIcon: SidebarNavIcon = ({ class: className, ...rest }) =>
  sidebarIcon({ class: className, ...rest }, [
    h("path", {
      "data-tone": "blue",
      d: "M6.5 3h9A2.5 2.5 0 0 1 18 5.5V19a2 2 0 0 1-2 2H6.5A2.5 2.5 0 0 1 4 18.5v-13A2.5 2.5 0 0 1 6.5 3Z",
      fill: "currentColor",
    }),
    h("path", { "data-tone": "cyan", d: "M8 3v18", stroke: "currentColor", "stroke-width": "2" }),
    h("path", {
      "data-tone": "white",
      d: "M11 8h3.8M11 11.5h3",
      stroke: "currentColor",
      "stroke-linecap": "round",
      "stroke-width": "1.7",
    }),
    h("path", {
      "data-tone": "amber",
      d: "m13.2 17.6 4.85-4.85 2.2 2.2-4.85 4.85-2.75.55.55-2.75Z",
      fill: "currentColor",
    }),
  ]);

export const QueueIcon: SidebarNavIcon = ({ class: className, ...rest }) =>
  sidebarIcon({ class: className, ...rest }, [
    h("path", {
      "data-tone": "blue",
      d: "M6 3h8.4L19 7.6V19a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z",
      fill: "currentColor",
    }),
    h("path", { "data-tone": "cyan", d: "M14 3v4a1 1 0 0 0 1 1h4", fill: "currentColor" }),
    h("path", { "data-tone": "green", d: "M8 12h8v2H8zM8 16h5v2H8z", fill: "currentColor" }),
    h("circle", { "data-tone": "amber", cx: "16.8", cy: "16.9", r: "2.2", fill: "currentColor" }),
  ]);
