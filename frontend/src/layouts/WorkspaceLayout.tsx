import { useEffect, useState } from "react";
import { LeftOutlined, RightOutlined } from "@ant-design/icons";
import { Outlet } from "react-router-dom";
import type { Stats } from "../api/types";
import { Sidebar } from "../components/Sidebar";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useI18n } from "../i18n";
import type { ThemeMode } from "../theme";

export interface WorkspaceOutletContext {
  stats: Stats | null;
  onDataChanged: () => void;
  themeMode: ThemeMode;
  onThemeChange: (mode: ThemeMode) => void;
  passwordEnabled: boolean;
  onPasswordEnabledChange: (enabled: boolean) => void;
  onLock: () => void;
}

const SIDEBAR_COLLAPSED_KEY = "resumetrail.sidebarCollapsed";

export function WorkspaceLayout(props: WorkspaceOutletContext) {
  const { t } = useI18n();
  const isMobile = useMediaQuery("(max-width: 767px)");
  const isTablet = useMediaQuery("(min-width: 768px) and (max-width: 1199px)");
  const [collapsed, setCollapsed] = useState(
    () => localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === "1",
  );

  useEffect(() => {
    localStorage.setItem(SIDEBAR_COLLAPSED_KEY, collapsed ? "1" : "0");
  }, [collapsed]);

  // Tablets already use the compact rail, so only desktop windows need to
  // honour the persisted toggle.
  const rail = !isMobile && (collapsed || isTablet);

  return (
    <div className={`app-body${rail ? " is-sidebar-collapsed" : ""}`}>
      <Sidebar
        stats={props.stats}
        passwordEnabled={props.passwordEnabled}
        collapsed={rail}
      />
      {!isMobile && !isTablet && (
        <button
          type="button"
          className="sidebar-collapse-toggle"
          onClick={() => setCollapsed((value) => !value)}
          aria-label={
            collapsed ? t("sidebar.expand") : t("sidebar.collapse")
          }
          title={collapsed ? t("sidebar.expand") : t("sidebar.collapse")}
        >
          {collapsed ? <RightOutlined /> : <LeftOutlined />}
        </button>
      )}
      <main className="main-content">
        <Outlet context={props} />
      </main>
    </div>
  );
}
