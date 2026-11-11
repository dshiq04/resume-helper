import { useCallback, useEffect, useMemo, useState } from "react";
import { Dropdown, Tooltip } from "antd";
import type { MenuProps } from "antd";
import {
  BorderOutlined,
  CloseOutlined,
  CompressOutlined,
  FullscreenExitOutlined,
  FullscreenOutlined,
  GlobalOutlined,
  MinusOutlined,
  MoonOutlined,
  SunOutlined,
} from "@ant-design/icons";
import { Events } from "@wailsio/runtime";
import { localeOptions, useI18n, type Locale } from "../i18n";
import {
  closeWindow,
  getDesktopWindowState,
  minimiseWindow,
  toggleFullscreenWindow,
  toggleMaximiseWindow,
} from "../utils/desktop";
import { Logo } from "./Logo";

interface TitleBarProps {
  themeMode: "light" | "dark";
  onToggleTheme: () => void;
}

export function TitleBar({ themeMode, onToggleTheme }: TitleBarProps) {
  const { t, locale, setLocale } = useI18n();
  const [maximised, setMaximised] = useState(false);
  const [fullscreen, setFullscreen] = useState(false);

  const refreshWindowState = useCallback(() => {
    void getDesktopWindowState().then((state) => {
      setMaximised(state.maximised);
      setFullscreen(state.fullscreen);
    });
  }, []);

  useEffect(() => {
    refreshWindowState();

    const unsubscribers = [
      Events.On("common:WindowMaximise", () => setMaximised(true)),
      Events.On("common:WindowUnMaximise", () => setMaximised(false)),
      Events.On("common:WindowFullscreen", () => setFullscreen(true)),
      Events.On("common:WindowUnFullscreen", () => setFullscreen(false)),
      Events.On("common:WindowRestore", () => {
        setMaximised(false);
        setFullscreen(false);
      }),
    ];
    window.addEventListener("resize", refreshWindowState);
    document.addEventListener("fullscreenchange", refreshWindowState);
    return () => {
      for (const unsubscribe of unsubscribers) unsubscribe();
      window.removeEventListener("resize", refreshWindowState);
      document.removeEventListener("fullscreenchange", refreshWindowState);
    };
  }, [refreshWindowState]);

  const languageItems = useMemo<MenuProps["items"]>(
    () =>
      localeOptions.map((option) => ({
        key: option.value,
        label: (
          <div className={`language-option ${option.value === locale ? "is-active" : ""}`}>
            <span className="language-option-short">{option.short}</span>
            <span>{option.label}</span>
          </div>
        ),
      })),
    [locale],
  );

  const currentLocale =
    localeOptions.find((option) => option.value === locale) ?? localeOptions[0];

  const handleToggleMaximise = async () => {
    await toggleMaximiseWindow();
    refreshWindowState();
  };

  const handleToggleFullscreen = async () => {
    await toggleFullscreenWindow();
    refreshWindowState();
  };

  return (
    <header className="titlebar">
      <div className="titlebar-brand">
        <Logo size={18} />
        <div className="titlebar-brand-copy">
          <span className="titlebar-brand-name">{t("app.brand")}</span>
          <span className="titlebar-brand-tagline">{t("app.tagline")}</span>
        </div>
      </div>

      <div className="titlebar-drag-space" aria-hidden="true" />

      <div className="titlebar-actions">
        <Dropdown
          menu={{
            items: languageItems,
            selectedKeys: [locale],
            onClick: ({ key }) => setLocale(key as Locale),
          }}
          placement="bottomRight"
          trigger={["click"]}
        >
          <button className="titlebar-tool" type="button" aria-label={t("settings.language")}>
            <GlobalOutlined />
            <span className="titlebar-tool-label">{currentLocale.short}</span>
          </button>
        </Dropdown>

        <Tooltip title={themeMode === "light" ? t("settings.dark") : t("settings.light")}>
          <button
            className="titlebar-tool"
            type="button"
            onClick={onToggleTheme}
            aria-label={t("settings.theme")}
          >
            {themeMode === "light" ? <MoonOutlined /> : <SunOutlined />}
          </button>
        </Tooltip>

        <span className="titlebar-divider" aria-hidden="true" />

        <button
          className="window-button"
          type="button"
          title={t("window.minimize")}
          aria-label={t("window.minimize")}
          onClick={() => void minimiseWindow()}
        >
          <MinusOutlined />
        </button>
        <button
          className="window-button"
          type="button"
          title={maximised ? t("window.restore") : t("window.maximize")}
          aria-label={maximised ? t("window.restore") : t("window.maximize")}
          onClick={() => void handleToggleMaximise()}
        >
          {maximised ? <CompressOutlined /> : <BorderOutlined />}
        </button>
        <button
          className="window-button"
          type="button"
          title={fullscreen ? t("window.exitFullscreen") : t("window.fullscreen")}
          aria-label={fullscreen ? t("window.exitFullscreen") : t("window.fullscreen")}
          onClick={() => void handleToggleFullscreen()}
        >
          {fullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
        </button>
        <button
          className="window-button window-button-close"
          type="button"
          title={t("window.close")}
          aria-label={t("window.close")}
          onClick={() => void closeWindow()}
        >
          <CloseOutlined />
        </button>
      </div>
    </header>
  );
}
