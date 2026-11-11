import { useEffect, useState } from "react";
import { App as AntdApp, Button, Switch } from "antd";
import { DesktopOutlined, PlusOutlined } from "@ant-design/icons";
import { api } from "../../api/client";
import type { ShortcutStatus } from "../../api/types";
import { useI18n } from "../../i18n";
import { apiErrorMessage } from "../../utils/errors";
import { SettingsCard } from "./SettingsCard";

export function ShortcutSettings() {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [status, setStatus] = useState<ShortcutStatus | null>(null);
  const [creating, setCreating] = useState(false);
  const [savingPrompt, setSavingPrompt] = useState(false);

  useEffect(() => {
    let active = true;
    api
      .shortcutStatus()
      .then((result) => {
        if (active) setStatus(result);
      })
      .catch((error) => {
        if (active) message.error(apiErrorMessage(error, t));
      });
    return () => {
      active = false;
    };
  }, [message, t]);

  const handleCreate = async () => {
    setCreating(true);
    try {
      const result = await api.createShortcut();
      setStatus(result);
      message.success(t("settings.shortcutCreated"));
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setCreating(false);
    }
  };

  const handlePromptChange = async (enabled: boolean) => {
    setSavingPrompt(true);
    try {
      await api.updateShortcutPromptSetting(enabled);
      setStatus((current) =>
        current ? { ...current, promptEnabled: enabled } : current,
      );
      message.success(t("settings.shortcutPromptSaved"));
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setSavingPrompt(false);
    }
  };

  return (
    <SettingsCard
      icon={<DesktopOutlined />}
      title={t("settings.shortcut")}
      description={t("settings.shortcutHint")}
    >
      {status && !status.supported ? (
        <p className="settings-field-hint shortcut-unsupported">
          {t("settings.shortcutUnsupported")}
        </p>
      ) : (
        <div className="shortcut-card">
          <div className="shortcut-status">
            <span
              className={`shortcut-status-dot ${status?.exists ? "is-on" : ""}`}
            />
            <div className="shortcut-status-copy">
              <strong>
                {status?.exists
                  ? t("settings.shortcutExists")
                  : t("settings.shortcutMissing")}
              </strong>
              {status?.path && <span title={status.path}>{status.path}</span>}
            </div>
            <Button
              size="small"
              type={status?.exists ? "default" : "primary"}
              icon={<PlusOutlined />}
              loading={creating}
              disabled={!status}
              onClick={() => void handleCreate()}
            >
              {status?.exists
                ? t("settings.shortcutRecreate")
                : t("settings.shortcutCreate")}
            </Button>
          </div>
        </div>
      )}

      {status?.supported && (
        <div className="shortcut-prompt-setting">
          <div className="shortcut-prompt-setting-copy">
            <strong>{t("settings.shortcutPrompt")}</strong>
            <span>{t("settings.shortcutPromptHint")}</span>
          </div>
          <Switch
            checked={status.promptEnabled}
            loading={savingPrompt}
            disabled={savingPrompt}
            onChange={(checked) => void handlePromptChange(checked)}
          />
        </div>
      )}
    </SettingsCard>
  );
}
