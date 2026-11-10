import { useCallback, useEffect, useMemo, useState } from "react";
import { App as AntdApp, Button, ConfigProvider, Spin } from "antd";
import { CloudServerOutlined, ReloadOutlined } from "@ant-design/icons";
import { ApiError, api } from "./api/client";
import type { AuthSession, Stats } from "./api/types";
import { Sidebar, type CategoryFilter } from "./components/Sidebar";
import { LockScreen } from "./components/LockScreen";
import { Logo } from "./components/Logo";
import { SettingsDrawer } from "./components/SettingsDrawer";
import { TitleBar } from "./components/TitleBar";
import { Dashboard } from "./pages/Dashboard";
import { I18nProvider, useI18n } from "./i18n";
import { createTheme, type ThemeMode } from "./theme";

const THEME_KEY = "resumetrail.theme";

function getInitialTheme(): ThemeMode {
  const saved = localStorage.getItem(THEME_KEY);
  if (saved === "light" || saved === "dark") return saved;
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

type Stage = "checking" | "locked" | "ready" | "error";

function AppRoot() {
  const { t, antdLocale } = useI18n();
  const [themeMode, setThemeMode] = useState<ThemeMode>(getInitialTheme);
  const [stage, setStage] = useState<Stage>("checking");
  const [passwordEnabled, setPasswordEnabled] = useState(false);
  const [stats, setStats] = useState<Stats | null>(null);
  const [activeCategory, setActiveCategory] = useState<CategoryFilter>("all");
  const [settingsOpen, setSettingsOpen] = useState(false);

  const checkAuth = useCallback(async () => {
    setStage("checking");
    try {
      const status = await api.authStatus();
      setPasswordEnabled(status.enabled);
      setStage(status.enabled && !status.authenticated ? "locked" : "ready");
    } catch {
      setStage("error");
    }
  }, []);

  useEffect(() => {
    void checkAuth();
  }, [checkAuth]);

  useEffect(() => {
    document.documentElement.dataset.theme = themeMode;
    localStorage.setItem(THEME_KEY, themeMode);
  }, [themeMode]);

  useEffect(() => {
    const handler = () => {
      setStage("locked");
    };
    window.addEventListener("resumetrail:unauthorized", handler);
    return () => window.removeEventListener("resumetrail:unauthorized", handler);
  }, []);

  const loadStats = useCallback(async () => {
    try {
      setStats(await api.stats());
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        // Keep the shell usable even if the statistics request fails.
        setStats(null);
      }
    }
  }, []);

  useEffect(() => {
    if (stage === "ready") {
      void loadStats();
    }
  }, [loadStats, stage]);

  const handleAuthenticated = useCallback((_session: AuthSession) => {
    setStage("ready");
  }, []);

  const handleDataChanged = useCallback(() => {
    void loadStats();
  }, [loadStats]);

  const handleLock = useCallback(() => {
    setSettingsOpen(false);
    setStage("locked");
  }, []);

  const themeConfig = useMemo(() => createTheme(themeMode), [themeMode]);

  return (
    <ConfigProvider locale={antdLocale} theme={themeConfig}>
      <AntdApp>
        <div className="app-shell">
          <TitleBar
            themeMode={themeMode}
            onToggleTheme={() =>
              setThemeMode((current) => (current === "light" ? "dark" : "light"))
            }
          />

          {stage === "checking" && (
            <div className="boot-screen">
              <div className="boot-logo">
                <Logo size={58} />
              </div>
              <Spin size="large" />
              <p>{t("common.loading")}</p>
            </div>
          )}

          {stage === "error" && (
            <div className="boot-screen boot-error">
              <div className="boot-error-icon">
                <CloudServerOutlined />
              </div>
              <h2>{t("error.network")}</h2>
              <Button
                type="primary"
                icon={<ReloadOutlined />}
                onClick={() => void checkAuth()}
              >
                {t("common.retry")}
              </Button>
            </div>
          )}

          {stage === "locked" && (
            <LockScreen onAuthenticated={handleAuthenticated} />
          )}

          {stage === "ready" && (
            <>
              <div className="app-body">
                <Sidebar
                  active={activeCategory}
                  onChange={setActiveCategory}
                  stats={stats}
                  passwordEnabled={passwordEnabled}
                  onOpenSettings={() => setSettingsOpen(true)}
                />

                <main className="main-content">
                  <Dashboard
                    category={activeCategory}
                    stats={stats}
                    onDataChanged={handleDataChanged}
                  />
                </main>
              </div>

              <SettingsDrawer
                open={settingsOpen}
                onClose={() => setSettingsOpen(false)}
                themeMode={themeMode}
                onThemeChange={setThemeMode}
                passwordEnabled={passwordEnabled}
                onPasswordEnabledChange={setPasswordEnabled}
                onLock={handleLock}
              />
            </>
          )}
        </div>
      </AntdApp>
    </ConfigProvider>
  );
}

export default function App() {
  return (
    <I18nProvider>
      <AppRoot />
    </I18nProvider>
  );
}
