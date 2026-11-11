import { SettingOutlined } from "@ant-design/icons";
import { useOutletContext } from "react-router-dom";
import type { WorkspaceOutletContext } from "../layouts/WorkspaceLayout";
import { useI18n } from "../i18n";
import { AppearanceSettings } from "../components/settings/AppearanceSettings";
import { SecuritySettings } from "../components/settings/SecuritySettings";
import { StorageSettings } from "../components/settings/StorageSettings";
import { ShortcutSettings } from "../components/settings/ShortcutSettings";
import { AboutSettings } from "../components/settings/AboutSettings";

export function SettingsPage() {
  const { t } = useI18n();
  const {
    themeMode,
    onThemeChange,
    passwordEnabled,
    onPasswordEnabledChange,
    onLock,
  } = useOutletContext<WorkspaceOutletContext>();

  return (
    <div className="settings-page">
      <div className="settings-page-header">
        <span className="season-pill">
          <SettingOutlined /> {t("nav.settings")}
        </span>
        <h1>{t("settings.title")}</h1>
        <p>{t("settings.subtitle")}</p>
      </div>

      <div className="settings-page-grid">
        <div className="settings-page-column">
          <AppearanceSettings
            themeMode={themeMode}
            onThemeChange={onThemeChange}
          />
          <SecuritySettings
            passwordEnabled={passwordEnabled}
            onPasswordEnabledChange={onPasswordEnabledChange}
            onLock={onLock}
          />
        </div>
        <div className="settings-page-column">
          <StorageSettings />
          <ShortcutSettings />
          <AboutSettings />
        </div>
      </div>
    </div>
  );
}
