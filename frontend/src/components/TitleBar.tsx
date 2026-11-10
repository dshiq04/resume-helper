import { useMemo } from "react";
import { Dropdown, Tooltip } from "antd";
import type { MenuProps } from "antd";
import {
  CloseOutlined,
  FullscreenOutlined,
  GlobalOutlined,
  MinusOutlined,
  MoonOutlined,
  SunOutlined,
} from "@ant-design/icons";
import { localeOptions, useI18n, type Locale } from "../i18n";
import {
  closeWindow,
  minimiseWindow,
  toggleMaximiseWindow,
} from "../utils/desktop";
import { Logo } from "./Logo";

interface TitleBarProps {
  themeMode: "light" | "dark";
  onToggleTheme: () => void;
}

export function TitleBar({ themeMode, onToggleTheme }: TitleBarProps) {
  const { t, locale, setLocale } = useI18n();

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

  return (
    <header className="titlebar">
      <div className="titlebar-brand">
        <Logo size={28} />
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
          title={t("window.maximize")}
          aria-label={t("window.maximize")}
          onClick={() => void toggleMaximiseWindow()}
        >
          <FullscreenOutlined />
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
