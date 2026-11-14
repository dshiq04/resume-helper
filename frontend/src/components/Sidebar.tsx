import {
  AppstoreOutlined,
  BarChartOutlined,
  CloudOutlined,
  FileTextOutlined,
  IdcardOutlined,
  RocketOutlined,
  SettingOutlined,
  SunOutlined,
} from "@ant-design/icons";
import { NavLink } from "react-router-dom";
import type { Stats } from "../api/types";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n";
import { Logo } from "./Logo";

interface SidebarProps {
  stats: Stats | null;
  passwordEnabled: boolean;
  collapsed: boolean;
}

const categories: Array<{
  path: string;
  labelKey: MessageKey;
  icon: typeof AppstoreOutlined;
  accent: string;
}> = [
  {
    path: "/",
    labelKey: "nav.all",
    icon: AppstoreOutlined,
    accent: "#818cf8",
  },
  { path: "/spring", labelKey: "nav.spring", icon: SunOutlined, accent: "#34d399" },
  { path: "/autumn", labelKey: "nav.autumn", icon: CloudOutlined, accent: "#fbbf24" },
  {
    path: "/internship",
    labelKey: "nav.internship",
    icon: RocketOutlined,
    accent: "#38bdf8",
  },
  {
    path: "/social",
    labelKey: "nav.social",
    icon: IdcardOutlined,
    accent: "#fb7185",
  },
];

export function Sidebar({ stats, passwordEnabled, collapsed }: SidebarProps) {
  const { t } = useI18n();

  const countFor = (path: string) => {
    if (!stats) return 0;
    if (path === "/") return stats.total;
    if (path === "/resumes") return stats.resumes ?? 0;
    const key = path.slice(1);
    return stats.byCategory[key as keyof typeof stats.byCategory] ?? 0;
  };

  const total = stats?.total ?? 0;
  const offers = stats?.byStatus.offer ?? 0;
  const rate = total > 0 ? Math.round((offers / total) * 100) : 0;

  return (
    <aside className={`sidebar${collapsed ? " is-collapsed" : ""}`}>
      <NavLink
        to="/"
        end
        className="sidebar-brand"
        aria-label={t("app.brand")}
        title={t("app.brand")}
      >
        <Logo size={30} />
        <span className="sidebar-brand-name">{t("app.brand")}</span>
      </NavLink>

      <div className="sidebar-section-label">{t("nav.workspace")}</div>
      <nav className="sidebar-nav" aria-label={t("nav.categories")}>
        {categories.map((item) => {
          const Icon = item.icon;
          return (
            <NavLink
              key={item.path}
              to={item.path}
              end={item.path === "/"}
              className={({ isActive }) =>
                `sidebar-nav-item ${isActive ? "is-active" : ""}`
              }
              title={t(item.labelKey)}
            >
              <span className="sidebar-nav-icon" style={{ color: item.accent }}>
                <Icon />
              </span>
              <span className="sidebar-nav-label">{t(item.labelKey)}</span>
              <span className="sidebar-nav-count">{countFor(item.path)}</span>
            </NavLink>
          );
        })}
      </nav>

      <div className="sidebar-section-label sidebar-library-label">
        {t("nav.library")}
      </div>
      <nav className="sidebar-nav" aria-label={t("nav.resumes")}>
        <NavLink
          to="/resumes"
          className={({ isActive }) =>
            `sidebar-nav-item ${isActive ? "is-active" : ""}`
          }
          title={t("nav.resumes")}
        >
          <span className="sidebar-nav-icon" style={{ color: "#f472b6" }}>
            <FileTextOutlined />
          </span>
          <span className="sidebar-nav-label">{t("nav.resumes")}</span>
          <span className="sidebar-nav-count">{countFor("/resumes")}</span>
        </NavLink>
        <NavLink
          to="/statistics"
          className={({ isActive }) =>
            `sidebar-nav-item ${isActive ? "is-active" : ""}`
          }
          title={t("nav.statistics")}
        >
          <span className="sidebar-nav-icon" style={{ color: "#a78bfa" }}>
            <BarChartOutlined />
          </span>
          <span className="sidebar-nav-label">{t("nav.statistics")}</span>
        </NavLink>
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
        <NavLink
          to="/settings"
          className={({ isActive }) =>
            `sidebar-settings ${isActive ? "is-active" : ""}`
          }
          title={t("nav.settings")}
        >
          <span className="sidebar-nav-icon">
            <SettingOutlined />
          </span>
          <span className="sidebar-nav-label">{t("nav.settings")}</span>
          <span
            className={`sidebar-security-dot ${passwordEnabled ? "is-on" : ""}`}
            title={passwordEnabled ? t("settings.passwordOn") : t("settings.passwordOff")}
          />
        </NavLink>
        <p className="sidebar-note">{t("settings.aboutText")}</p>
      </div>
    </aside>
  );
}
