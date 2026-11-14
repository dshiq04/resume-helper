import { InfoCircleOutlined } from "@ant-design/icons";
import { useI18n } from "../../i18n";
import { SettingsCard } from "./SettingsCard";

const APP_VERSION = "1.1";

export function AboutSettings() {
  const { t } = useI18n();

  return (
    <SettingsCard
      icon={<InfoCircleOutlined />}
      title={t("settings.about")}
      extra={
        <span className="settings-version-badge">
          {t("settings.versionBadge", { version: APP_VERSION })}
        </span>
      }
    >
      <div className="settings-about">
        <div>
          <strong>{t("app.brand")}</strong>
          <p>{t("settings.aboutText")}</p>
        </div>
      </div>
    </SettingsCard>
  );
}
