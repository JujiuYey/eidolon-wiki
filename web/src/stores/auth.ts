import { defineStore } from "pinia";

import type {
  AuthCapabilities,
  MembershipInfo,
  TenantInfo,
  UserInfo,
} from "@/types/auth";

// 空间内角色的权限等级（镜像 WeKnora rbac：viewer < contributor < admin < owner）
export const TENANT_ROLE_LEVEL = {
  viewer: 10,
  contributor: 20,
  admin: 30,
  owner: 40,
} as const;

export type TenantRoleLevel = keyof typeof TENANT_ROLE_LEVEL;

// 空间内角色字面量（原 @/api/system/member 随成员管理页一并裁撤，展示常量挪到这里）
export type TenantRole = keyof typeof TENANT_ROLE_LEVEL;

export const tenantRoleLabels: Record<TenantRole, string> = {
  owner: "所有者",
  admin: "管理员",
  contributor: "成员",
  viewer: "观察者",
};

// request 层会读取该 key 作为 X-Tenant-ID 头（与 WeKnora 官方前端一致）
export const SELECTED_TENANT_KEY = "weknora_selected_tenant_id";

export function persistSelectedTenant(tenantId: string | number | null | undefined) {
  const id = tenantId == null ? "" : String(tenantId);
  if (id && Number(id) > 0) {
    localStorage.setItem(SELECTED_TENANT_KEY, id);
  } else {
    localStorage.removeItem(SELECTED_TENANT_KEY);
  }
}

interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: UserInfo | null;
  tenant: TenantInfo | null;
  memberships: MembershipInfo[];
  capabilities: AuthCapabilities | null;
}

export const useAuthStore = defineStore("auth", {
  state: (): AuthState => ({
    accessToken: null,
    refreshToken: null,
    user: null,
    tenant: null,
    memberships: [],
    capabilities: null,
  }),
  actions: {
    setTokens(accessToken: string, refreshToken: string) {
      this.accessToken = accessToken;
      this.refreshToken = refreshToken;
    },
    setAccessToken(accessToken: string) {
      this.accessToken = accessToken;
    },
    setUser(user: UserInfo) {
      persistSelectedTenant(user.tenant_id);
      this.user = user;
    },
    setSession(payload: {
      user: UserInfo;
      tenant?: TenantInfo | null;
      memberships?: MembershipInfo[];
      capabilities?: AuthCapabilities | null;
    }) {
      persistSelectedTenant(payload.user.tenant_id || payload.tenant?.id);
      this.user = payload.user;
      this.tenant = payload.tenant ?? null;
      this.memberships = payload.memberships ?? [];
      this.capabilities = payload.capabilities ?? null;
    },
    clearAuth() {
      localStorage.removeItem(SELECTED_TENANT_KEY);
      this.$reset();
    },
  },
  persist: {
    key: "auth-storage",
    // accessToken/refreshToken/user 等会话信息全部持久化
    pick: ["accessToken", "refreshToken", "user", "tenant", "memberships", "capabilities"],
  },
});

// ---- 权限辅助（仅用于 UI 显隐，真实鉴权在服务端） ----

interface AuthSnapshot {
  user: UserInfo | null;
  memberships: MembershipInfo[];
  tenant: TenantInfo | null;
}

/** 当前生效空间的成员角色；跨空间超管按 admin 对待（与 WeKnora 官方前端一致，永不视为 owner） */
export function getTenantRole(state: AuthSnapshot): TenantRoleLevel | null {
  const { user, memberships } = state;
  const tenantId = Number(user?.tenant_id || 0);
  const matched = memberships.find((m) => m.tenant_id === tenantId)?.role as TenantRoleLevel | undefined;
  if (matched && matched in TENANT_ROLE_LEVEL) return matched;
  if (user?.can_access_all_tenants === true) return "admin";
  return null;
}

export function hasTenantRole(state: AuthSnapshot, min: TenantRoleLevel): boolean {
  const role = getTenantRole(state);
  if (!role) return false;
  return TENANT_ROLE_LEVEL[role] >= TENANT_ROLE_LEVEL[min];
}

export function isSystemAdmin(state: Pick<AuthSnapshot, "user">): boolean {
  return state.user?.is_system_admin === true;
}

/** 当前空间 id（数字）；未关联空间时为 0 */
export function getCurrentTenantId(state: AuthSnapshot): number {
  return Number(state.user?.tenant_id || state.tenant?.id || 0);
}
