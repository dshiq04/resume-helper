import { useEffect, useState } from "react";
import { App as AntdApp } from "antd";
import { DatabaseOutlined } from "@ant-design/icons";
import { api } from "../../api/client";
import type { SystemInfo } from "../../api/types";
import { useI18n } from "../../i18n";
import { apiErrorMessage } from "../../utils/errors";
import { SettingsCard } from "./SettingsCard";

export function StorageSettings() {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);

  useEffect(() => {
    let active = true;
    api
      .systemInfo()
      .then((info) => {
        if (active) setSystemInfo(info);
      })
      .catch((error) => {
        if (active) message.error(apiErrorMessage(error, t));
      });
    return () => {
      active = false;
    };
  }, [message, t]);

  return (
    <SettingsCard
      icon={<DatabaseOutlined />}
      tone="data"
      title={t("settings.data")}
      description={t("settings.database")}
    >
      <div className="settings-info-list">
        <div className="settings-info-row">
          <span>{t("settings.database")}</span>
          <strong>{systemInfo?.database ?? "—"}</strong>
        </div>
        {systemInfo?.location && (
          <div className="settings-info-row settings-info-path">
            <span>{t("settings.storageLocation")}</span>
            <code title={systemInfo.location}>{systemInfo.location}</code>
          </div>
        )}
        <div className="settings-info-row settings-info-path">
          <span>{t("settings.dataDir")}</span>
          <code title={systemInfo?.dataDir}>{systemInfo?.dataDir ?? "—"}</code>
        </div>
        <div className="settings-info-row settings-info-path">
          <span>{t("settings.configPath")}</span>
          <code title={systemInfo?.configPath}>
            {systemInfo?.configPath ?? "—"}
          </code>
        </div>
        <div className="settings-info-row">
          <span>{t("settings.version")}</span>
          <strong>{systemInfo?.version ?? "1.0.0"}</strong>
        </div>
      </div>
    </SettingsCard>
  );
}
