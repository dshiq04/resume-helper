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
  "pending",
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
  appliedAt: string;
  createdAt: string;
  updatedAt: string;
  screeningDays?: number;
  statusDays?: number;
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
  stuckScreening?: {
    count: number;
    averageDays: number;
    maxDays: number;
  };
}

export interface ApplicationChange {
  id: number;
  applicationId: number;
  status: Status;
  changedAt: string;
  createdAt: string;
  daysSincePrevious?: number;
  daysInStatus?: number;
}

export interface ApplicationHistory {
  items: ApplicationChange[];
  missingStatuses: Status[];
  screeningDays: number;
  appliedAt: string;
}

export interface DeletedApplication {
  id: number;
  originalId: number;
  category: Category;
  companyName: string;
  link: string;
  status: Status;
  pinned: boolean;
  appliedAt: string;
  createdAt: string;
  updatedAt: string;
  history: ApplicationChange[];
  deletedAt: string;
}

export interface TrashList {
  items: DeletedApplication[];
  total: number;
}

export interface CompanySuggestions {
  items: string[];
}

export interface Preferences {
  closeAction: "ask" | "tray" | "quit";
  idleLockEnabled: boolean;
  idleLockMinutes: number;
  shortcutPromptEnabled: boolean;
}

export interface FunnelStage {
  key: "applied" | Status;
  count: number;
  conversion: number;
}

export interface CompanyFunnel {
  company: string;
  total: number;
  interviews: number;
  offers: number;
  interviewRate: number;
  interviewToOfferRate: number;
  offerRate: number;
}

export interface FunnelStats {
  total: number;
  stages: FunnelStage[];
  averageDaysToInterview: number;
  averageInterviewToOffer: number;
  companies: CompanyFunnel[];
  interviewCount: number;
  offerCount: number;
}

export interface BackupCounts {
  applications: number;
  resumes: number;
  settings: number;
  changes: number;
  trash: number;
  files: number;
  bytes: number;
}

export interface BackupResult {
  path: string;
  exportedAt: string;
  version: string;
  appVersion: string;
  importBlocked: boolean;
  counts: BackupCounts;
  warnings: string[];
}

export interface DeleteResult {
  deleted: number;
  trashId?: number;
  trashIds?: number[];
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

export interface AppInfoInput {
  version: string;
  name: string;
  title: string;
  defaultLanguage: string;
}

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

export interface PlatformInfo {
  kind: "computer" | "tablet" | "phone" | string;
  os: string;
  name: string;
  version: string;
}

export interface DeviceInfo {
  deviceId: string;
  algorithm: string;
  keyDerivation: string;
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
  from?: string;
  to?: string;
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
  encrypted: boolean;
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
