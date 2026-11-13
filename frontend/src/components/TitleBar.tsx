import { useCallback, useEffect, useState } from "react";
import { Tooltip } from "antd";
import {
  BorderOutlined,
  CloseOutlined,
  CompressOutlined,
  MinusOutlined,
  MoonOutlined,
  SunOutlined,
} from "@ant-design/icons";
import { Events } from "@wailsio/runtime";
import { useI18n } from "../i18n";
import {
  closeWindow,
  getDesktopWindowState,
  minimiseWindow,
  toggleMaximiseWindow,
} from "../utils/desktop";

interface TitleBarProps {
  themeMode: "light" | "dark";
  onToggleTheme: () => void;
}

export function TitleBar({ themeMode, onToggleTheme }: TitleBarProps) {
  const { t } = useI18n();
  const [maximised, setMaximised] = useState(false);

  const refreshWindowState = useCallback(() => {
    void getDesktopWindowState().then((state) => {
      setMaximised(state.maximised);
    });
  }, []);

  useEffect(() => {
    refreshWindowState();

    const unsubscribers = [
      Events.On("common:WindowMaximise", () => setMaximised(true)),
      Events.On("common:WindowUnMaximise", () => setMaximised(false)),
      Events.On("common:WindowRestore", () => setMaximised(false)),
    ];
    window.addEventListener("resize", refreshWindowState);
    return () => {
      for (const unsubscribe of unsubscribers) unsubscribe();
      window.removeEventListener("resize", refreshWindowState);
    };
  }, [refreshWindowState]);

  const handleToggleMaximise = async () => {
    await toggleMaximiseWindow();
    refreshWindowState();
  };

  return (
    <header className="titlebar">
      <div className="titlebar-drag-space" aria-hidden="true" />

      <div className="titlebar-actions">
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
