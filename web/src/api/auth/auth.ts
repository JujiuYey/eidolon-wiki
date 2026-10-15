import type {
  AuthConfig,
  ChangePasswordRequest,
  CurrentUserResponse,
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  RefreshTokenResponse,
} from "@/types/auth";
import { request } from "@/utils/request";

const RESOURCE_PATH = "/api/v1/auth";

// 认证相关 API
export const authApi = {
  // 登录
  login: (data: LoginRequest) => request.post<LoginResponse>(`${RESOURCE_PATH}/login`, data),

  // 刷新 token
  refreshToken: (data: RefreshTokenRequest) =>
    request.post<RefreshTokenResponse>(`${RESOURCE_PATH}/refresh`, data),

  // 获取当前用户信息（含空间与成员关系）
  getCurrentUser: () => request.get<CurrentUserResponse>(`${RESOURCE_PATH}/me`),

  // 修改自己的密码
  changePassword: (data: ChangePasswordRequest) =>
    request.post(`${RESOURCE_PATH}/change-password`, data),

  // 认证配置（免鉴权）：注册模式与密码复杂度开关
  getConfig: () => request.get<AuthConfig>(`${RESOURCE_PATH}/config`),
};
