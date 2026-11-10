import {
  AppstoreOutlined,
  CloudOutlined,
  RocketOutlined,
  SettingOutlined,
  SunOutlined,
} from "@ant-design/icons";
import type { Stats } from "../api/types";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n";

export type CategoryFilter = "all" | "spring" | "autumn" | "internship";

interface SidebarProps {
  active: CategoryFilter;
  onChange: (value: CategoryFilter) => void;
  stats: Stats | null;
  onOpenSettings?: () => void;
  passwordEnabled: boolean;
}

const categories: Array<{
  key: CategoryFilter;
  labelKey: MessageKey;
  icon: typeof AppstoreOutlined;
  accent: string;
}> = [
  {
    key: "all",
    labelKey: "nav.all",
    icon: AppstoreOutlined,
    accent: "#818cf8",
  },
  { key: "spring", labelKey: "nav.spring", icon: SunOutlined, accent: "#34d399" },
  { key: "autumn", labelKey: "nav.autumn", icon: CloudOutlined, accent: "#fbbf24" },
  {
    key: "internship",
    labelKey: "nav.internship",
    icon: RocketOutlined,
    accent: "#38bdf8",
  },
];

export function Sidebar({
  active,
  onChange,
  stats,
  onOpenSettings,
  passwordEnabled,
}: SidebarProps) {
  const { t } = useI18n();

  const countFor = (key: CategoryFilter) => {
    if (!stats) return 0;
    return key === "all" ? stats.total : stats.byCategory[key] ?? 0;
  };

  const total = stats?.total ?? 0;
  const offers = stats?.byStatus.offer ?? 0;
  const rate = total > 0 ? Math.round((offers / total) * 100) : 0;

  return (
    <aside className="sidebar">
      <div className="sidebar-section-label">{t("nav.workspace")}</div>
      <nav className="sidebar-nav" aria-label={t("nav.categories")}>
        {categories.map((item) => {
          const Icon = item.icon;
          return (
            <button
              key={item.key}
              type="button"
              className={`sidebar-nav-item ${active === item.key ? "is-active" : ""}`}
              onClick={() => onChange(item.key)}
            >
              <span className="sidebar-nav-icon" style={{ color: item.accent }}>
                <Icon />
              </span>
              <span className="sidebar-nav-label">{t(item.labelKey)}</span>
              <span className="sidebar-nav-count">{countFor(item.key)}</span>
            </button>
          );
        })}
      </nav>

      <div className="sidebar-spacer" />

      <div className="sidebar-progress">
        <div className="sidebar-progress-head">
          <span>{t("stats.offerRate")}</span>
          <strong>{rate}%</strong>
        </div>
        <div className="sidebar-progress-track">
          <span style={{ width: `${Math.max(rate, total > 0 && offers > 0 ? 3 : 0)}%` }} />
        </div>
        <p>
          {offers} / {total} {t("stats.offer")}
        </p>
      </div>

      <div className="sidebar-footer">
        {onOpenSettings && (
          <button
            type="button"
            className="sidebar-settings"
            onClick={onOpenSettings}
          >
            <span className="sidebar-nav-icon">
              <SettingOutlined />
            </span>
            <span className="sidebar-nav-label">{t("nav.settings")}</span>
            <span
              className={`sidebar-security-dot ${passwordEnabled ? "is-on" : ""}`}
              title={passwordEnabled ? t("settings.passwordOn") : t("settings.passwordOff")}
            />
          </button>
        )}
        <p className="sidebar-note">{t("settings.aboutText")}</p>
      </div>
    </aside>
  );
}
