import { InfoCircleOutlined } from "@ant-design/icons";
import { useI18n } from "../../i18n";
import { SettingsCard } from "./SettingsCard";

export function AboutSettings() {
  const { t } = useI18n();

  return (
    <SettingsCard
      icon={<InfoCircleOutlined />}
      title={t("settings.about")}
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
