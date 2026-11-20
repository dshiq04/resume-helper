import { useEffect, useState } from "react";
import { InfoCircleOutlined } from "@ant-design/icons";
import { Typography } from "antd";
import { api } from "../../api/client";
import type { DeviceInfo, PlatformInfo } from "../../api/types";
import { useI18n, type MessageKey } from "../../i18n";
import { SettingsCard } from "./SettingsCard";

const APP_VERSION = __APP_VERSION__;

const PLATFORM_KIND_KEYS: Record<string, MessageKey> = {
  computer: "settings.platform.computer",
  tablet: "settings.platform.tablet",
  phone: "settings.platform.phone",
};

export function AboutSettings() {
  const { t } = useI18n();
  const [device, setDevice] = useState<DeviceInfo | null>(null);
  const [platform, setPlatform] = useState<PlatformInfo | null>(null);

  useEffect(() => {
    let active = true;
    void Promise.allSettled([api.deviceInfo(), api.platformInfo()]).then(
      ([deviceResult, platformResult]) => {
        if (!active) return;
        if (deviceResult.status === "fulfilled") {
          setDevice(deviceResult.value);
        }
        if (platformResult.status === "fulfilled") {
          setPlatform(platformResult.value);
        }
      },
    );
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
          {platform && (
            <div className="settings-device-platform">
              <div className="settings-device-row">
                <span className="settings-device-label">
                  {t("settings.platform")}
                </span>
                <span className="settings-device-value">
                  {t(
                    PLATFORM_KIND_KEYS[platform.kind] ??
                      "settings.platform.computer",
                  )}
                </span>
              </div>
              <div className="settings-device-row">
                <span className="settings-device-label">
                  {t("settings.operatingSystem")}
                </span>
                <span className="settings-device-value">
                  {[platform.name, platform.version]
                    .filter(Boolean)
                    .join(" ") || "—"}
                </span>
              </div>
            </div>
          )}
        </div>
      )}
    </SettingsCard>
  );
}
