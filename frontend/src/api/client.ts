import type {
  Application,
  ApplicationInput,
  ApplicationList,
  ApplicationUpdate,
  AuthSession,
  AuthStatus,
  ImportResult,
  ListParams,
  Stats,
  SystemInfo,
} from "./types";

const TOKEN_KEY = "resumetrail.token";

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

interface RequestOptions extends RequestInit {
  auth?: boolean;
  handleUnauthorized?: boolean;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { auth = true, handleUnauthorized = true, headers, ...rest } = options;
  const token = getToken();
  const nextHeaders = new Headers(headers);
  if (!(rest.body instanceof FormData) && rest.body !== undefined && !nextHeaders.has("Content-Type")) {
    nextHeaders.set("Content-Type", "application/json");
  }
  if (auth && token) {
    nextHeaders.set("Authorization", `Bearer ${token}`);
  }

  let response: Response;
  try {
    response = await fetch(`/api${path}`, {
      ...rest,
      headers: nextHeaders,
    });
  } catch {
    throw new ApiError("network_error", 0, "network_error");
  }

  if (response.status === 401 && handleUnauthorized) {
    clearToken();
    window.dispatchEvent(new CustomEvent("resumetrail:unauthorized"));
  }

  if (!response.ok) {
    let message = response.statusText || "Request failed";
    let code: string | undefined;
    try {
      const payload = await response.json();
      if (typeof payload?.error === "string") message = payload.error;
      if (typeof payload?.code === "string") code = payload.code;
    } catch {
      // Ignore non-JSON bodies.
    }
    throw new ApiError(message, response.status, code);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const contentType = response.headers.get("Content-Type") ?? "";
  if (!contentType.includes("application/json")) {
    return (await response.blob()) as T;
  }
  return (await response.json()) as T;
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
      body: JSON.stringify({ password }),
      auth: false,
      handleUnauthorized: false,
    });
  },
  setupPassword(password: string) {
    return request<AuthSession>("/auth/setup", {
      method: "POST",
      body: JSON.stringify({ password }),
      auth: false,
      handleUnauthorized: false,
    });
  },
  changePassword(currentPassword: string, newPassword: string) {
    return request<AuthSession>("/auth/change-password", {
      method: "POST",
      body: JSON.stringify({ currentPassword, newPassword }),
      handleUnauthorized: false,
    });
  },
  disablePassword(password: string) {
    return request<{ enabled: boolean }>("/auth/disable", {
      method: "POST",
      body: JSON.stringify({ password }),
    });
  },
  logout() {
    return request<{ loggedOut: boolean }>("/auth/logout", {
      method: "POST",
    });
  },
  listApplications(params: ListParams) {
    const search = new URLSearchParams();
    if (params.search) search.set("search", params.search);
    if (params.category && params.category !== "all") {
      search.set("category", params.category);
    }
    if (params.status && params.status !== "all") {
      search.set("status", params.status);
    }
    search.set("page", String(params.page ?? 1));
    search.set("pageSize", String(params.pageSize ?? 10));
    return request<ApplicationList>(`/applications?${search.toString()}`);
  },
  createApplication(input: ApplicationInput) {
    return request<Application>("/applications", {
      method: "POST",
      body: JSON.stringify(input),
    });
  },
  updateApplication(id: number, input: ApplicationUpdate) {
    return request<Application>(`/application/${id}`, {
      method: "PUT",
      body: JSON.stringify(input),
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
      body: JSON.stringify({ ids }),
    });
  },
  batchStatus(ids: number[], status: string) {
    return request<{ updated: number }>("/applications/batch-status", {
      method: "PATCH",
      body: JSON.stringify({ ids, status }),
    });
  },
  importExcel(file: File) {
    const data = new FormData();
    data.append("file", file);
    return request<ImportResult>("/applications/import", {
      method: "POST",
      body: data,
    });
  },
  async downloadTemplate() {
    const blob = await request<Blob>("/applications/template");
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
};
