import { useMutation, useQuery, useQueryClient } from "@tanstack/vue-query";
import { toast } from "vue-sonner";

import { request } from "@/utils/request";

// ---- 类型（对齐 WeKnora frontend/src/api/system/index.ts SystemSettingItem）----

export interface SystemSettingItem {
  id: number;
  key: string;
  /** 原始 JSON 值，按 value_type 收窄后渲染 */
  value: unknown;
  value_type: "int" | "string" | "bool" | "string_list";
  category: string;
  description: string;
  is_secret: boolean;
  requires_restart: boolean;
  /** 非空 = DB 覆盖行；空 = 虚拟行（ENV/默认值） */
  last_modified_by: string;
  /** 服务端把 UUID 解析成的 username/email */
  last_modified_by_name?: string;
  created_at: string;
  updated_at: string;
  /** 非空时渲染下拉而非自由输入 */
  enum?: string[];
}

// 注意：该端点直接返回数组，无 {data} 包裹
export const settingsApi = {
  list: () => request.get<SystemSettingItem[]>("/api/v1/system/admin/settings"),

  update: (key: string, value: unknown) =>
    request.put<SystemSettingItem>(
      `/api/v1/system/admin/settings/${encodeURIComponent(key)}`,
      { value },
    ),

  reset: (key: string) =>
    request.delete(`/api/v1/system/admin/settings/${encodeURIComponent(key)}`),
};

// ---- Query keys ----

export const settingsKeys = {
  all: ["system-settings"] as const,
  list: () => [...settingsKeys.all, "list"] as const,
};

// ---- Hooks ----

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: string } | null)?.message;
  return message || fallback;
}

export function useSystemSettings() {
  return useQuery({
    queryKey: settingsKeys.list(),
    queryFn: async () => (await settingsApi.list()) ?? [],
  });
}

export function useUpdateSystemSetting() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ key, value }: { key: string; value: unknown }) => settingsApi.update(key, value),
    onSuccess: (updated) => {
      queryClient.setQueryData<SystemSettingItem[]>(settingsKeys.list(), (old) =>
        old ? old.map((item) => (item.key === updated.key ? updated : item)) : old,
      );
    },
    onError: (error) => {
      toast.error(errorMessage(error, "保存设置失败"));
    },
  });
}

export function useResetSystemSetting() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => settingsApi.reset(key),
    onSuccess: () => {
      toast.success("已重置为默认值");
      queryClient.invalidateQueries({ queryKey: settingsKeys.list() });
    },
    onError: (error) => {
      toast.error(errorMessage(error, "重置设置失败"));
    },
  });
}

// ---- 展示辅助 ----

/** 有 DB 覆盖行（显示"已覆盖"徽章与重置按钮） */
export function hasSettingOverride(item: SystemSettingItem): boolean {
  return Boolean(item.last_modified_by?.trim());
}

/** Go 零值时间（0001-…）视为未修改 */
export function settingModifiedAt(item: SystemSettingItem): string | null {
  if (!item.updated_at || item.updated_at.startsWith("0001-")) return null;
  return item.updated_at;
}

/** 高危 key：修改前需要二次确认（与官方前端 HIGH_RISK_KEYS 一致） */
export const HIGH_RISK_SETTING_KEYS = new Set(["auth.registration_mode", "sandbox.docker_enabled"]);

export const settingCategoryLabels: Record<string, string> = {
  auth: "访问与注册",
  tenant: "空间",
  model: "模型",
  asynq: "任务并发",
  ssrf: "安全防护",
  sandbox: "沙箱",
};

export function settingCategoryLabel(category: string): string {
  return settingCategoryLabels[category] ?? "其他";
}

/** 已知 key 的中文标签与说明（来源：官方前端 i18n zh-CN） */
export const settingKeyMeta: Record<string, { label: string; description?: string; highImpact?: boolean }> = {
  "auth.registration_mode": {
    label: "自助注册模式",
    description:
      "自助注册模式。self_serve = 任何人可注册账号；invite_only = 关闭公网注册，仅 Owner/Admin 可邀请。修改后立即生效，但谨慎对待 self_serve（公网会接受 spam）。",
    highImpact: true,
  },
  "auth.default_tenant_mode": {
    label: "注册默认空间策略",
    description:
      "公开注册后的空间初始化策略。create_personal 会自动创建个人空间并授予 Owner；tenantless 仅创建账户，用户需要接受邀请或主动创建空间。只影响之后注册的用户。",
  },
  "auth.complex_password_enabled": {
    label: "启用复杂密码",
    description: "是否启用复杂密码。开启后密码必须包含大小写字母、数字和特殊字符。修改后立即生效，只影响新注册用户或新密码修改/重置操作。",
  },
  "tenant.self_service_creation_enabled": {
    label: "允许用户自助创建空间",
    description:
      "是否允许非超管用户主动创建空间。关闭后，普通用户只能通过邀请加入已有空间；跨空间超管仍可创建。修改后立即生效。",
  },
  "tenant.max_owned_per_user": {
    label: "每用户最大空间数",
    description: "每个非超管用户通过自助创建可拥有的最大空间数。0 表示使用内置默认值 10；负数表示完全关闭限制。",
  },
  "tenant.default_storage_quota_gb": {
    label: "新空间默认存储配额 (GB)",
    description: "新建空间时默认分配的存储配额（GB）。仅在创建时读取，修改后只对之后新建的空间生效。",
  },
  "tenant.auto_create_api_key": {
    label: "创建空间时自动生成 API Key",
    description: "为新空间自动生成 full_access API Key。仅用于兼容旧行为的集成；默认关闭。",
  },
  "tenant.auto_accept_invitation": {
    label: "邀请已注册用户时自动加入",
    description: "开启后，空间管理员通过邮箱邀请已注册用户时，对方会立即成为成员，不再经过收件箱确认。",
  },
  "model.max_concurrency": {
    label: "模型默认并发上限",
    description: "后台任务对单个模型的默认并发上限。0 或负数表示关闭默认限制。仅影响后台任务，不影响交互式对话。",
  },
  "asynq.core_concurrency": { label: "核心解析保底并发数" },
  "asynq.postprocess_concurrency": { label: "后处理编排并发数" },
  "asynq.enrichment_concurrency": { label: "内容富化保底并发数" },
  "asynq.maintenance_concurrency": { label: "维护与同步并发数" },
  "asynq.shared_concurrency": { label: "共享弹性并发数" },
  "asynq.wiki_concurrency": { label: "Wiki Worker 并发数" },
  "ssrf.whitelist": {
    label: "SSRF 防护白名单",
    description: "可填入 example.com / *.foo.com / 10.0.0.0/8 / 2001:db8::1，每行一条。修改后立即生效。",
  },
  "sandbox.docker_enabled": {
    label: "启用 Docker 沙箱",
    description: "是否允许 Docker 沙箱后端。本机 docker.sock 等同宿主机 root，默认关闭。仅系统管理员可打开。",
    highImpact: true,
  },
};

export function settingLabel(item: SystemSettingItem): string {
  return settingKeyMeta[item.key]?.label ?? item.key;
}

export function settingDescription(item: SystemSettingItem): string {
  return settingKeyMeta[item.key]?.description || item.description || "";
}

/** 枚举选项的中文标签（已知 key 逐值映射，未知原样返回） */
export const settingEnumLabels: Record<string, Record<string, string>> = {
  "auth.registration_mode": {
    self_serve: "自助注册（任何人可注册）",
    invite_only: "仅邀请（关闭公网注册）",
  },
};

export function settingEnumLabel(item: SystemSettingItem, option: string): string {
  return settingEnumLabels[item.key]?.[option] ?? option;
}
