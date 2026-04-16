import {
  IconBolt,
  IconDashboard,
  IconDatabase,
  IconMessageChatbot,
  IconSearch,
  IconSettings,
} from "@tabler/icons-react";
import * as React from "react";

import avatarImage1 from "@/assets/user-avatar/avatar1.jpg";
import avatarImage2 from "@/assets/user-avatar/avatar2.jpg";
import avatarImage3 from "@/assets/user-avatar/avatar3.jpg";
import avatarImage4 from "@/assets/user-avatar/avatar4.jpg";
import avatarImage5 from "@/assets/user-avatar/avatar5.jpg";
import avatarImage6 from "@/assets/user-avatar/avatar6.jpg";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
} from "@/components/ui/sidebar";
import { useAuthStore } from "@/stores/auth";

import { Logo } from "./_components/logo";
import { NavDocuments } from "./_components/nav-documents";
import { NavMain } from "./_components/nav-main";
import { NavSecondary } from "./_components/nav-secondary";
import { NavSystem } from "./_components/nav-system";
import { User } from "./_components/user";

const avatarImages = [
  avatarImage1,
  avatarImage2,
  avatarImage3,
  avatarImage4,
  avatarImage5,
  avatarImage6,
];

function getRandomAvatar(): string {
  const randomIndex = Math.floor(Math.random() * avatarImages.length);
  return avatarImages[randomIndex];
}

const data = {
  navMain: [
    {
      title: "仪表盘",
      url: "/",
      icon: IconDashboard,
    },
    {
      title: "RAG 对话",
      url: "/rag-chat",
      icon: IconMessageChatbot,
    },
  ],
  navSecondary: [
    {
      title: "应用设置",
      url: "/app-setting",
      icon: IconSettings,
    },
    {
      title: "AI模型配置",
      url: "/model-config",
      icon: IconBolt,
    },
    {
      title: "向量数据库",
      url: "/collection",
      icon: IconDatabase,
    },
    {
      title: "搜索",
      url: "#",
      icon: IconSearch,
    },
  ],
};

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const user = useAuthStore(state => state.user);

  const defaultUser = {
    name: "未登录",
    email: "",
    avatar: "",
  };

  const sidebarUser = user
    ? {
        name: user.full_name || user.username,
        email: user.email,
        avatar: user.avatar || getRandomAvatar(),
      }
    : { ...defaultUser, avatar: getRandomAvatar() };

  return (
    <Sidebar collapsible="icon" {...props}>
      <Logo />

      <NavMain items={data.navMain} />
      <NavSystem />

      <SidebarContent>
        <NavDocuments />
      </SidebarContent>

      <NavSecondary items={data.navSecondary} className="mt-auto" />

      <SidebarFooter>
        <User user={sidebarUser} />
      </SidebarFooter>
    </Sidebar>
  );
}
