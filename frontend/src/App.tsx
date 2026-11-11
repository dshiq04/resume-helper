import { Suspense, lazy, useCallback, useEffect, useMemo, useState } from "react";
import { App as AntdApp, Button, ConfigProvider, Spin, Watermark } from "antd";
import { CloudServerOutlined, ReloadOutlined } from "@ant-design/icons";
import { Navigate, Route, Routes } from "react-router-dom";
import { Events } from "@wailsio/runtime";
import { ApiError, api } from "./api/client";
import type { AuthSession, Stats } from "./api/types";
import { LockScreen } from "./components/LockScreen";
import { Logo } from "./components/Logo";
import { ResizeHandles } from "./components/ResizeHandles";
import { TitleBar } from "./components/TitleBar";
import { I18nProvider, useI18n } from "./i18n";
import { WorkspaceLayout } from "./layouts/WorkspaceLayout";
import { Dashboard } from "./pages/Dashboard";
import { QuickEntryWindowPage } from "./pages/QuickEntryWindowPage";
import { ResumeLibraryPage } from "./pages/ResumeLibraryPage";
import { SettingsPage } from "./pages/SettingsPage";
import { ShortcutPromptWindowPage } from "./pages/ShortcutPromptWindowPage";
import { createTheme, type ThemeMode } from "./theme";

const ResumePreviewWindowPage = lazy(() =>
  import("./pages/ResumePreviewWindowPage").then((module) => ({
    default: module.ResumePreviewWindowPage,
  })),
);

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
    const handler = () => setStage("locked");
    window.addEventListener("resumetrail:unauthorized", handler);
    return () => window.removeEventListener("resumetrail:unauthorized", handler);
  }, []);

  const loadStats = useCallback(async () => {
    try {
      setStats(await api.stats());
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        setStats(null);
      }
    }
  }, []);

  useEffect(() => {
    if (stage === "ready") {
      void loadStats();
    }
  }, [loadStats, stage]);

  useEffect(() => {
    const handleDatabaseChanged = () => {
      void checkAuth();
      void loadStats();
    };
    const refreshStats = () => void loadStats();
    const unsubscribe = Events.On("resumetrail:data-changed", refreshStats);
    window.addEventListener("resumetrail:database-changed", handleDatabaseChanged);
    window.addEventListener("resumetrail:data-changed", refreshStats);
    return () => {
      unsubscribe();
      window.removeEventListener("resumetrail:database-changed", handleDatabaseChanged);
      window.removeEventListener("resumetrail:data-changed", refreshStats);
    };
  }, [checkAuth, loadStats]);

  const handleAuthenticated = useCallback((_session: AuthSession) => {
    setStage("ready");
  }, []);

  const handleDataChanged = useCallback(() => {
    void loadStats();
  }, [loadStats]);

  const handleLock = useCallback(() => {
    setStage("locked");
  }, []);

  const themeConfig = useMemo(() => createTheme(themeMode), [themeMode]);

  if (window.location.pathname === "/quick-entry") {
    return (
      <ConfigProvider locale={antdLocale} theme={themeConfig}>
        <AntdApp>
          <QuickEntryWindowPage />
        </AntdApp>
      </ConfigProvider>
    );
  }

  if (window.location.pathname === "/shortcut-prompt") {
    return (
      <ConfigProvider locale={antdLocale} theme={themeConfig}>
        <AntdApp>
          <ShortcutPromptWindowPage />
        </AntdApp>
      </ConfigProvider>
    );
  }

  if (window.location.pathname === "/resume-preview") {
    return (
      <ConfigProvider locale={antdLocale} theme={themeConfig}>
        <AntdApp>
          <Suspense
            fallback={
              <div className="resume-preview-shell">
                <div className="resume-preview-loading">
                  <Spin size="large" />
                </div>
              </div>
            }
          >
            <ResumePreviewWindowPage />
          </Suspense>
        </AntdApp>
      </ConfigProvider>
    );
  }

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
          <ResizeHandles />

          <Watermark
            className="app-watermark"
            content="开发中品质，请谅解"
            gap={[140, 140]}
            offset={[24, 24]}
            rotate={-24}
            zIndex={4}
            font={{
              color:
                themeMode === "dark"
                  ? "rgba(226, 232, 240, 0.09)"
                  : "rgba(71, 85, 105, 0.11)",
              fontSize: 14,
              fontWeight: 600,
            }}
          >
            <div className="app-content">
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
            <Routes>
              <Route
                element={
                  <WorkspaceLayout
                    stats={stats}
                    onDataChanged={handleDataChanged}
                    themeMode={themeMode}
                    onThemeChange={setThemeMode}
                    passwordEnabled={passwordEnabled}
                    onPasswordEnabledChange={setPasswordEnabled}
                    onLock={handleLock}
                  />
                }
              >
                <Route index element={<Dashboard category="all" />} />
                <Route path="spring" element={<Dashboard category="spring" />} />
                <Route path="autumn" element={<Dashboard category="autumn" />} />
                <Route
                  path="internship"
                  element={<Dashboard category="internship" />}
                />
                <Route path="resumes" element={<ResumeLibraryPage />} />
                <Route path="settings" element={<SettingsPage />} />
                <Route path="*" element={<Navigate to="/" replace />} />
              </Route>
            </Routes>
          )}
            </div>
          </Watermark>
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
