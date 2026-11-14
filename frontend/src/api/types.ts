export type Category = "spring" | "autumn" | "internship" | "social";
export type Status =
  | "screening"
  | "written"
  | "interview"
  | "pending"
  | "offer"
  | "rejected";

export const CATEGORIES: Category[] = [
  "spring",
  "autumn",
  "internship",
  "social",
];
export const STATUSES: Status[] = [
  "screening",
  "written",
  "interview",
  "pending",
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
  appliedAt: string;
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
  pinned?: boolean;
}

export interface ApplicationUpdate {
  category?: Category;
  companyName?: string;
  link?: string;
  status?: Status;
  pinned?: boolean;
}

export interface QuickEntryDefaults {
  defaultCategory: Category;
  serverTime: string;
}

export interface Stats {
  total: number;
  resumes: number;
  byCategory: Record<Category, number>;
  byStatus: Record<Status, number>;
  byCategoryStatus: Record<Category, Record<Status, number>>;
}

export interface DailyStat {
  date: string;
  category: Category;
  count: number;
}

export interface DailyStats {
  daily: DailyStat[];
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

export type DatabaseDriver = "sqlite" | "mysql" | "postgres";

export interface SystemInfo {
  version: string;
  goVersion: string;
  driver: DatabaseDriver;
  database: string;
  location: string;
  configPath: string;
  dataDir: string;
}

export interface DatabaseSettings {
  driver: DatabaseDriver;
  dsn: string;
  sqlitePath: string;
  host: string;
  port: number;
  user: string;
  password: string;
  database: string;
  sslMode: string;
  parseError?: string;
}

export interface DatabaseSettingsResponse {
  settings: DatabaseSettings;
  configPath: string;
  dataDir: string;
  location: string;
  envOverride: boolean;
}

export interface DatabaseInput extends DatabaseSettings {
  useDsn: boolean;
}

export interface DatabaseUpdateResult extends DatabaseSettingsResponse {
  ok: boolean;
  warning?: string;
}

export interface DatabaseTestResult {
  ok: boolean;
  message: string;
}

export interface ShortcutStatus {
  supported: boolean;
  exists: boolean;
  path: string;
  promptEnabled: boolean;
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

export type ResumeFileType =
  | "docx"
  | "pdf"
  | "pptx"
  | "png"
  | "md"
  | "markdown";

export interface Resume {
  id: number;
  name: string;
  tags: string[];
  fileName: string;
  fileType: ResumeFileType;
  fileSize: number;
  pinned: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ResumeList {
  items: Resume[];
  total: number;
}

export interface ResumeUpdate {
  name?: string;
  tags?: string[];
  pinned?: boolean;
}

export interface ResumeCreateInput {
  name: string;
  tags: string[];
  file: File;
}
