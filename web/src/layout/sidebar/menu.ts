import {
  ChatIcon,
  FolderIcon,
  ModelConfigIcon,
  QueueIcon,
  SystemSettingsIcon,
  UserManageIcon,
  type SidebarNavIcon,
} from "./nav-icons";

export type NavItem = {
  title: string;
  url: string;
  icon: SidebarNavIcon;
  /** UI 显隐门槛（真实鉴权在服务端）：tenant-admin = 空间 Admin+，system-admin = 平台超管 */
  access?: "tenant-admin" | "system-admin";
};

/** 日常页面，全员可见 */
export const workNav: NavItem[] = [
  { title: "知识问答", url: "/rag-chat", icon: ChatIcon },
  { title: "知识库", url: "/wiki/knowledge-bases", icon: FolderIcon },
];

/** 管理向页面，整组仅 Admin+ 可见，组内按超管再细分 */
export const adminNav: NavItem[] = [
  { title: "模型配置", url: "/model-config", icon: ModelConfigIcon },
  { title: "用户管理", url: "/system/users", icon: UserManageIcon, access: "system-admin" },
  { title: "系统设置", url: "/system/settings", icon: SystemSettingsIcon, access: "system-admin" },
  { title: "运行队列", url: "/system/queues", icon: QueueIcon, access: "system-admin" },
];
