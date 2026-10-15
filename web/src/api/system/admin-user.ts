import { useQuery, useMutation } from "@tanstack/vue-query";
import { toast } from "vue-sonner";

import { authApi } from "@/api/auth/auth";
import type { UserInfo } from "@/types/auth";
import { request } from "@/utils/request";

// ---- 类型（对齐 WeKnora internal/handler/system.go 与 frontend/src/api/system/index.ts）----

/** GET /system/admin/list 等端点返回的用户投影 */
export interface SystemAdminUser {
  id: string;
  username: string;
  email: string;
  avatar?: string;
  is_active: boolean;
  is_system_admin: boolean;
  created_at: string;
  updated_at: string;
}

// POST /system/admin/users/create：password 缺省/留空时由服务端生成，
// 明文只在响应 generated_password 出现一次（无邮件找回，前端需一次性展示）
export interface CreateSystemUserRequest {
  username: string;
  email: string;
  password?: string;
}

export interface CreateSystemUserResponse {
  user: UserInfo;
  /** 仅新建时存在；幂等命中已有账号（200）时为空 */
  generated_password?: string;
}

// POST /system/admin/users/reset-password：成功后目标用户全部会话立即失效；
// 不能重置自己（后端 400），邮箱不存在返回 404
export interface ResetUserPasswordRequest {
  email: string;
  new_password: string;
}

// ---- API（注意：/system/admin/* 端点均无 {data} 包裹）----

export const adminUserApi = {
  createUser: (data: CreateSystemUserRequest) =>
    request.post<CreateSystemUserResponse>("/api/v1/system/admin/users/create", data),

  resetPassword: (data: ResetUserPasswordRequest) =>
    request.post<{ message: string }>("/api/v1/system/admin/users/reset-password", data),
};

// ---- Query keys ----

export const adminUserKeys = {
  all: ["system-admin-user"] as const,
  authConfig: () => [...adminUserKeys.all, "auth-config"] as const,
};

// ---- Hooks ----

function errorMessage(error: unknown, fallback: string): string {
  const message = (error as { message?: string } | null)?.message;
  return message || fallback;
}

/** 认证配置（免鉴权）：这里只用 complex_password_enabled 驱动密码表单校验 */
export function useAuthConfig() {
  return useQuery({
    queryKey: adminUserKeys.authConfig(),
    queryFn: async () => (await authApi.getConfig()) ?? null,
  });
}

export function useCreateSystemUser() {
  return useMutation({
    mutationFn: (data: CreateSystemUserRequest) => adminUserApi.createUser(data),
    onSuccess: (response) => {
      if (response.generated_password) {
        // 一次性展示面板负责提示保存，这里不弹 toast 抢焦点
        return;
      }
      // 无 generated_password = 幂等命中完全一致的已有账号（200）
      toast.success("该用户已存在（邮箱与用户名一致），未重复创建");
    },
    onError: (error) => {
      toast.error(errorMessage(error, "创建用户失败"));
    },
  });
}

export function useResetUserPassword() {
  return useMutation({
    mutationFn: (data: ResetUserPasswordRequest) => adminUserApi.resetPassword(data),
    onSuccess: () => {
      toast.success("密码已重置，该用户的所有登录会话已下线");
    },
    onError: (error) => {
      toast.error(errorMessage(error, "重置密码失败"));
    },
  });
}
