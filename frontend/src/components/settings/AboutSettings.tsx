import { useEffect, useState } from "react";
import { InfoCircleOutlined } from "@ant-design/icons";
import { Typography } from "antd";
import { api } from "../../api/client";
import type { DeviceInfo } from "../../api/types";
import { useI18n } from "../../i18n";
import { SettingsCard } from "./SettingsCard";

const APP_VERSION = "1.1";

export function AboutSettings() {
  const { t } = useI18n();
  const [device, setDevice] = useState<DeviceInfo | null>(null);

  useEffect(() => {
    let active = true;
    void api
      .deviceInfo()
      .then((info) => {
        if (active) setDevice(info);
      })
      .catch(() => {
        // The device identifier is informational; a failure must not break the
        // settings page.
      });
    return () => {
      active = false;
    };
  }, []);

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

      {device && (
        <div className="settings-device">
          <div className="settings-device-row">
            <span className="settings-device-label">
              {t("settings.deviceId")}
            </span>
            <Typography.Text
              className="settings-device-value"
              code
              copyable={{ text: device.deviceId }}
            >
              {device.deviceId}
            </Typography.Text>
          </div>
          <p>
            {t("settings.deviceEncryption", { algorithm: device.algorithm })}
          </p>
        </div>
      )}
    </SettingsCard>
  );
}
