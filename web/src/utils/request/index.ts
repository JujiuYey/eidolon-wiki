import axios, { type AxiosRequestConfig, type InternalAxiosRequestConfig } from "axios";

import { useAuthStore } from "@/stores/auth";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || import.meta.env.BASE_URL || "/";
const normalizedBaseUrl = API_BASE_URL.replace(/\/$/, "");

const instance = axios.create({
  baseURL: normalizedBaseUrl,
  timeout: 30000,
  headers: { "Content-Type": "application/json" },
});

type RetriableRequest = InternalAxiosRequestConfig & { _retry?: boolean };
type FailedRequest = {
  resolve: (token: string) => void;
  reject: (error: unknown) => void;
};

let isRefreshing = false;
let failedQueue: FailedRequest[] = [];

function processQueue(error: unknown, token?: string) {
  for (const request of failedQueue) {
    if (error) request.reject(error);
    else if (token) request.resolve(token);
  }
  failedQueue = [];
}

function getCurrentLanguage(): string {
  return localStorage.getItem("locale") || "zh-CN";
}

function isPublicAuthRequest(url?: string): boolean {
  if (!url) return false;
  return [
    "/auth/auto-setup",
    "/auth/login",
    "/auth/register",
    "/auth/oidc/",
    "/auth/invitations/lookup",
    "/api/v1/embed/",
  ].some((path) => url.includes(path));
}

function isEmbedPage(): boolean {
  return typeof window !== "undefined" && window.location.pathname.startsWith("/embed/");
}

function redirectToLogin() {
  if (typeof window === "undefined") return;
  if (window.location.pathname !== "/login" && !isEmbedPage()) {
    window.location.href = "/login";
  }
}

async function refreshAccessToken(): Promise<string | null> {
  const auth = useAuthStore();
  if (!auth.refreshToken) return null;

  try {
    const response = await axios.post<
      | { success?: boolean; access_token?: string; refresh_token?: string }
      | { code?: number; data?: { access_token?: string } }
    >(`${normalizedBaseUrl}/api/v1/auth/refresh`, { refreshToken: auth.refreshToken });
    const data = response.data;
    const token =
      ("success" in data && data.success ? data.access_token : undefined) ||
      ("code" in data && data.code === 0 ? data.data?.access_token : undefined);

    if (token) {
      if ("success" in data && data.refresh_token) {
        auth.setTokens(token, data.refresh_token);
      } else {
        auth.setAccessToken(token);
      }
      return token;
    }
  } catch {
    // The response interceptor clears auth and redirects below.
  }
  return null;
}

instance.interceptors.request.use((config) => {
  const existingAuth = config.headers?.Authorization ?? config.headers?.authorization;
  const isEmbedAuth = typeof existingAuth === "string" && existingAuth.startsWith("Embed ");
  const isEmbedPath = typeof config.url === "string" && config.url.includes("/api/v1/embed/");

  if (!isEmbedAuth) {
    const { accessToken } = useAuthStore();
    if (accessToken) config.headers.Authorization = `Bearer ${accessToken}`;
  }

  config.headers["Accept-Language"] = getCurrentLanguage();

  if (!isEmbedAuth && !isEmbedPath) {
    const tenantId = localStorage.getItem("weknora_selected_tenant_id");
    if (tenantId) config.headers["X-Tenant-ID"] = tenantId;
  }

  config.headers["X-Request-ID"] =
    typeof crypto !== "undefined" && "randomUUID" in crypto
      ? crypto.randomUUID()
      : Math.random().toString(36).slice(2);

  if (config.data instanceof FormData) delete config.headers["Content-Type"];
  return config;
});

instance.interceptors.response.use(
  (response) => response.data,
  async (error: any) => {
    const originalRequest = error.config as RetriableRequest | undefined;
    const status = error.response?.status;

    if (!error.response) return Promise.reject({ message: "网络请求失败" });

    if ((status === 401 || status === 403) && isPublicAuthRequest(originalRequest?.url)) {
      const data = error.response.data;
      const message =
        typeof data === "object" ? data?.error?.message || data?.error || data?.message : data;
      return Promise.reject({ status, message: message || "认证失败" });
    }

    if (status === 401 && isEmbedPage()) {
      return Promise.reject({ status, message: "认证失败" });
    }

    if (
      status === 401 &&
      originalRequest &&
      !originalRequest._retry &&
      !originalRequest.url?.includes("/auth/refresh")
    ) {
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({
            resolve: (token) => {
              originalRequest.headers.Authorization = `Bearer ${token}`;
              instance(originalRequest).then(resolve).catch(reject);
            },
            reject,
          });
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;
      try {
        const token = await refreshAccessToken();
        if (token) {
          processQueue(undefined, token);
          originalRequest.headers.Authorization = `Bearer ${token}`;
          return instance(originalRequest);
        }

        const expired = new Error("登录已过期");
        processQueue(expired);
        useAuthStore().clearAuth();
        redirectToLogin();
        return Promise.reject(expired);
      } finally {
        isRefreshing = false;
      }
    }

    const data = error.response.data;
    const message =
      typeof data === "object" ? data?.error?.message || data?.error || data?.message : data;
    return Promise.reject({
      status,
      message: message || error.message || "请求失败",
      ...(typeof data === "object" && data ? data : {}),
    });
  },
);

function get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> {
  return instance.get<T>(url, config) as unknown as Promise<T>;
}

function getDown(url: string, config?: AxiosRequestConfig): Promise<Blob> {
  return instance.get<Blob>(url, { ...config, responseType: "blob" }) as unknown as Promise<Blob>;
}

function post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  return instance.post<T>(url, data, config) as unknown as Promise<T>;
}

function postUpload<T = unknown>(
  url: string,
  data: unknown = {},
  onUploadProgress?: (progressEvent: unknown) => void,
  config: AxiosRequestConfig = {},
): Promise<T> {
  return instance.post<T>(url, data, {
    ...config,
    headers: { "Content-Type": "multipart/form-data", ...config.headers },
    onUploadProgress: onUploadProgress || config.onUploadProgress,
  }) as unknown as Promise<T>;
}

function postChat<T = unknown>(url: string, data: unknown = {}, config?: AxiosRequestConfig): Promise<T> {
  return post<T>(url, data, {
    ...config,
    headers: { "Content-Type": "text/event-stream;charset=utf-8", ...config?.headers },
  });
}

function put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  return instance.put<T>(url, data, config) as unknown as Promise<T>;
}

function patch<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  return instance.patch<T>(url, data, config) as unknown as Promise<T>;
}

function del<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  return instance.delete<T>(url, { ...config, data }) as unknown as Promise<T>;
}

const request = Object.assign(
  <T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> =>
    instance<T>(url, config) as unknown as Promise<T>,
  { get, post, put, patch, delete: del, del, postUpload, postChat, getDown },
);

export { del, get, getDown, patch, post, postChat, postUpload, put, request };
export type { AxiosRequestConfig };
