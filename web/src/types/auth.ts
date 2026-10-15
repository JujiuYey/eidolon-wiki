// 认证相关类型定义
// 字段形状对齐 WeKnora 后端（参考 frontend/src/api/auth/index.ts）

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  success: boolean;
  message?: string;
  token?: string;
  /** 旧字段兼容：部分网关返回 access_token */
  access_token?: string;
  refresh_token?: string;
  user?: UserInfo;
  tenant?: TenantInfo | null;
  active_tenant?: TenantInfo | null;
  memberships?: MembershipInfo[];
  capabilities?: AuthCapabilities;
}

export interface RefreshTokenRequest {
  refresh_token: string;
}

export interface RefreshTokenResponse {
  access_token: string;
}

// POST /api/v1/auth/change-password
export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

export interface UserPreferences {
  last_active_tenant_id?: number | null;
  oidc_only_login?: boolean;
}

// 用户信息（WeKnora UserInfo）
export interface UserInfo {
  id: string;
  username: string;
  email: string;
  avatar?: string;
  tenant_id: string;
  can_access_all_tenants?: boolean;
  is_system_admin?: boolean;
  preferences?: UserPreferences;
  created_at?: string;
  updated_at?: string;
}

// 空间（租户）信息
export interface TenantInfo {
  id: string;
  name: string;
  description?: string;
  status?: string;
  business?: string;
  owner_id?: string;
  storage_quota?: number;
  storage_used?: number;
  created_at?: string;
  updated_at?: string;
}

// 用户在某个空间的成员关系（role: owner/admin/contributor/viewer）
export interface MembershipInfo {
  tenant_id: number;
  tenant_name?: string;
  role: string;
}

export interface AuthCapabilities {
  can_create_tenant: boolean;
  auto_accept_invitation: boolean;
}

// GET /api/v1/auth/config 响应（免鉴权；登录页据此显隐注册入口，超管表单据此应用密码策略）
export interface AuthConfig {
  success?: boolean;
  registration_mode: "self_serve" | "invite_only";
  complex_password_enabled: boolean;
}

// GET /api/v1/auth/me 响应
export interface CurrentUserResponse {
  success: boolean;
  message?: string;
  data?: {
    user: UserInfo;
    tenant?: TenantInfo | null;
    memberships?: MembershipInfo[];
    tenant_required?: boolean;
    capabilities?: AuthCapabilities;
  };
}
