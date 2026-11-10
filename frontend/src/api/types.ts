export type Category = "spring" | "autumn" | "internship";
export type Status =
  | "screening"
  | "written"
  | "interview"
  | "offer"
  | "rejected";

export const CATEGORIES: Category[] = ["spring", "autumn", "internship"];
export const STATUSES: Status[] = [
  "screening",
  "written",
  "interview",
  "offer",
  "rejected",
];

export type CategoryFilter = Category | "all";

export interface StatusOption {
  value: Status;
  label: string;
}

export interface Application {
  id: number;
  category: Category;
  companyName: string;
  link: string;
  status: Status;
  pinned: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ApplicationList {
  items: Application[];
  total: number;
  page: number;
  pageSize: number;
}

export interface ApplicationInput {
  category: Category;
  companyName: string;
  link: string;
  status: Status;
}

export interface ApplicationUpdate {
  category?: Category;
  companyName?: string;
  link?: string;
  status?: Status;
  pinned?: boolean;
}

export interface Stats {
  total: number;
  byCategory: Record<Category, number>;
  byStatus: Record<Status, number>;
  byCategoryStatus: Record<Category, Record<Status, number>>;
}

export interface AuthStatus {
  enabled: boolean;
  authenticated: boolean;
}

export interface AuthSession {
  token: string;
  expiresAt: string;
  enabled: boolean;
}

export interface SystemInfo {
  version: string;
  goVersion: string;
  driver: "sqlite" | "mysql" | "postgres";
  database: string;
  location: string;
  configPath: string;
  dataDir: string;
}

export interface ImportResult {
  imported: number;
  skipped: number;
  total: number;
  errors: string[];
}

export interface ListParams {
  search?: string;
  category?: Category | "all";
  status?: Status | "all";
  page?: number;
  pageSize?: number;
}
