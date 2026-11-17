import { useEffect, useState } from "react";
import { App as AntdApp, Radio, Spin } from "antd";
import { DesktopOutlined } from "@ant-design/icons";
import { api } from "../../api/client";
import type { Preferences } from "../../api/types";
import { useI18n } from "../../i18n";
import { apiErrorMessage } from "../../utils/errors";
import { SettingsCard } from "./SettingsCard";

type CloseAction = Preferences["closeAction"];

const CLOSE_OPTIONS: Array<{
  value: CloseAction;
  labelKey: "settings.closeAsk" | "settings.closeTray" | "settings.closeQuit";
  hintKey:
    | "settings.closeAskHint"
    | "settings.closeTrayHint"
    | "settings.closeQuitHint";
}> = [
  { value: "ask", labelKey: "settings.closeAsk", hintKey: "settings.closeAskHint" },
  {
    value: "tray",
    labelKey: "settings.closeTray",
    hintKey: "settings.closeTrayHint",
  },
  {
    value: "quit",
    labelKey: "settings.closeQuit",
    hintKey: "settings.closeQuitHint",
  },
];

export function DesktopSettings() {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [preferences, setPreferences] = useState<Preferences | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let active = true;
    void api
      .preferences()
      .then((data) => {
        if (active) setPreferences(data);
      })
      .catch((error) => {
        if (active) message.error(apiErrorMessage(error, t));
      });
    return () => {
      active = false;
    };
  }, [message, t]);

  const updateCloseAction = async (closeAction: CloseAction) => {
    setSaving(true);
    try {
      const updated = await api.updatePreferences({ closeAction });
      setPreferences(updated);
      window.dispatchEvent(new Event("resumetrail:preferences-changed"));
      message.success(t("settings.preferencesSaved"));
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setSaving(false);
    }
  };

  return (
    <SettingsCard
      icon={<DesktopOutlined />}
      tone="default"
      title={t("settings.desktop")}
      description={t("settings.trayNote")}
    >
      {!preferences ? (
        <div className="settings-inline-loading">
          <Spin size="small" />
        </div>
      ) : (
        <div className="desktop-preferences">
          <label className="settings-label">{t("settings.closeAction")}</label>
          <Radio.Group
            className="close-action-group"
            value={preferences.closeAction}
            disabled={saving}
            onChange={(event) => void updateCloseAction(event.target.value as CloseAction)}
          >
            {CLOSE_OPTIONS.map((option) => (
              <Radio key={option.value} value={option.value}>
                <span className="close-action-copy">
                  <strong>{t(option.labelKey)}</strong>
                  <span>{t(option.hintKey)}</span>
                </span>
              </Radio>
            ))}
          </Radio.Group>
          <p className="settings-field-hint">{t("settings.closeActionHint")}</p>
        </div>
      )}
    </SettingsCard>
  );
}
