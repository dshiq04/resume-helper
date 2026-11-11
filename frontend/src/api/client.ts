import axios, { type AxiosRequestConfig } from "axios";
import type {
  Application,
  ApplicationInput,
  ApplicationList,
  ApplicationUpdate,
  AuthSession,
  AuthStatus,
  DatabaseInput,
  DatabaseSettingsResponse,
  DatabaseTestResult,
  DatabaseUpdateResult,
  ImportResult,
  ListParams,
  Stats,
  SystemInfo,
} from "./types";

const TOKEN_KEY = "resumetrail.token";

const http = axios.create({
  baseURL: "/api",
  timeout: 30_000,
  headers: {
    Accept: "application/json",
  },
});

http.interceptors.request.use((config) => {
  const token = getToken();
  if (token) {
    config.headers.set("Authorization", `Bearer ${token}`);
  }
  return config;
});

export class ApiError extends Error {
  status: number;
  code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) ?? "";
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

interface RequestOptions extends AxiosRequestConfig {
  handleUnauthorized?: boolean;
}

interface ApiErrorPayload {
  error?: string;
  code?: string;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { handleUnauthorized = true, ...config } = options;
  try {
    const response = await http.request<T>({
      url: path,
      ...config,
    });
    return response.data;
  } catch (error) {
    if (!axios.isAxiosError(error)) {
      throw new ApiError("network_error", 0, "network_error");
    }

    const status = error.response?.status ?? 0;
    let payload = error.response?.data as ApiErrorPayload | Blob | undefined;

    if (payload instanceof Blob) {
      try {
        payload = JSON.parse(await payload.text()) as ApiErrorPayload;
      } catch {
        payload = undefined;
      }
    }

    const apiPayload = payload as ApiErrorPayload | undefined;
    const code = apiPayload?.code;
    const message = apiPayload?.error ?? error.message ?? "Request failed";

    if (status === 401 && handleUnauthorized && code !== "invalid_password") {
      clearToken();
      window.dispatchEvent(new CustomEvent("resumetrail:unauthorized"));
    }

    if (status === 0) {
      throw new ApiError("network_error", 0, "network_error");
    }

    throw new ApiError(message, status, code);
  }
}

export const api = {
  authStatus() {
    return request<AuthStatus>("/auth/status", {
      handleUnauthorized: false,
    });
  },
  login(password: string) {
    return request<AuthSession>("/auth/login", {
      method: "POST",
      data: { password },
      handleUnauthorized: false,
    });
  },
  setupPassword(password: string) {
    return request<AuthSession>("/auth/setup", {
      method: "POST",
      data: { password },
      handleUnauthorized: false,
    });
  },
  changePassword(currentPassword: string, newPassword: string) {
    return request<AuthSession>("/auth/change-password", {
      method: "POST",
      data: { currentPassword, newPassword },
    });
  },
  disablePassword(password: string) {
    return request<{ enabled: boolean }>("/auth/disable", {
      method: "POST",
      data: { password },
    });
  },
  logout() {
    return request<{ loggedOut: boolean }>("/auth/logout", {
      method: "POST",
    });
  },
  listApplications(params: ListParams) {
    const query: Record<string, string | number> = {
      page: params.page ?? 1,
      pageSize: params.pageSize ?? 10,
    };
    if (params.search) query.search = params.search;
    if (params.category && params.category !== "all") {
      query.category = params.category;
    }
    if (params.status && params.status !== "all") {
      query.status = params.status;
    }
    return request<ApplicationList>("/applications", { params: query });
  },
  createApplication(input: ApplicationInput) {
    return request<Application>("/applications", {
      method: "POST",
      data: input,
    });
  },
  updateApplication(id: number, input: ApplicationUpdate) {
    return request<Application>(`/application/${id}`, {
      method: "PUT",
      data: input,
    });
  },
  deleteApplication(id: number) {
    return request<{ deleted: number }>(`/application/${id}`, {
      method: "DELETE",
    });
  },
  batchDelete(ids: number[]) {
    return request<{ deleted: number }>("/applications/batch-delete", {
      method: "POST",
      data: { ids },
    });
  },
  batchStatus(ids: number[], status: string) {
    return request<{ updated: number }>("/applications/batch-status", {
      method: "PATCH",
      data: { ids, status },
    });
  },
  importExcel(file: File) {
    const data = new FormData();
    data.append("file", file);
    return request<ImportResult>("/applications/import", {
      method: "POST",
      data,
    });
  },
  async downloadTemplate() {
    const blob = await request<Blob>("/applications/template", {
      responseType: "blob",
    });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "resume-trail-template.xlsx";
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
  },
  stats() {
    return request<Stats>("/stats");
  },
  systemInfo() {
    return request<SystemInfo>("/system/info");
  },
  databaseSettings() {
    return request<DatabaseSettingsResponse>("/system/database");
  },
  testDatabase(input: DatabaseInput) {
    return request<DatabaseTestResult>("/system/database/test", {
      method: "POST",
      data: input,
      timeout: 15_000,
    });
  },
  updateDatabase(input: DatabaseInput) {
    return request<DatabaseUpdateResult>("/system/database", {
      method: "PUT",
      data: input,
      timeout: 60_000,
    });
  },
};
