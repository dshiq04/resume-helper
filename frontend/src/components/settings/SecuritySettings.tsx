import { useEffect, useState } from "react";
import { App as AntdApp, Button, InputNumber, Switch, Tooltip } from "antd";
import {
  ClockCircleOutlined,
  LockOutlined,
  SafetyCertificateOutlined,
} from "@ant-design/icons";
import { api, clearToken, setToken } from "../../api/client";
import { useI18n } from "../../i18n";
import { apiErrorMessage } from "../../utils/errors";
import { SettingsCard } from "./SettingsCard";
import { PasswordModal, type SecurityFormValues, type SecurityMode } from "./PasswordModal";

interface SecuritySettingsProps {
  passwordEnabled: boolean;
  onPasswordEnabledChange: (enabled: boolean) => void;
  onLock: () => void;
}

export function SecuritySettings({
  passwordEnabled,
  onPasswordEnabledChange,
  onLock,
}: SecuritySettingsProps) {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [mode, setMode] = useState<SecurityMode | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [idleEnabled, setIdleEnabled] = useState(false);
  const [idleMinutes, setIdleMinutes] = useState(15);
  const [idleSaving, setIdleSaving] = useState(false);

  useEffect(() => {
    let active = true;
    void api
      .preferences()
      .then((data) => {
        if (!active) return;
        setIdleEnabled(data.idleLockEnabled);
        setIdleMinutes(data.idleLockMinutes);
      })
      .catch(() => {
        // Preferences are optional; the security card still works.
      });
    return () => {
      active = false;
    };
  }, []);

  const saveIdleLock = async (enabled: boolean, minutes: number) => {
    setIdleSaving(true);
    try {
      const updated = await api.updatePreferences({
        idleLockEnabled: enabled,
        idleLockMinutes: minutes,
      });
      setIdleEnabled(updated.idleLockEnabled);
      setIdleMinutes(updated.idleLockMinutes);
      window.dispatchEvent(new Event("resumetrail:preferences-changed"));
      message.success(t("settings.idleSaved"));
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setIdleSaving(false);
    }
  };

  const submit = async (values: SecurityFormValues) => {
    if (!mode) return;
    setSubmitting(true);
    try {
      if (mode === "setup") {
        const session = await api.setupPassword(values.newPassword ?? "");
        setToken(session.token);
        onPasswordEnabledChange(true);
        message.success(t("toast.passwordEnabled"));
      } else if (mode === "change") {
        const session = await api.changePassword(
          values.currentPassword ?? "",
          values.newPassword ?? "",
        );
        setToken(session.token);
        message.success(t("toast.passwordChanged"));
      } else {
        await api.disablePassword(values.password ?? "");
        clearToken();
        onPasswordEnabledChange(false);
        message.success(t("toast.passwordDisabled"));
      }
      setMode(null);
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setSubmitting(false);
    }
  };

  const lockNow = async () => {
    try {
      await api.logout();
    } catch {
      // Locking locally is still the safe outcome.
    }
    clearToken();
    onLock();
  };

  return (
    <>
      <SettingsCard
        icon={<SafetyCertificateOutlined />}
        tone="security"
        title={t("settings.security")}
      >
        <div className={`security-card ${passwordEnabled ? "is-on" : ""}`}>
          <div className="security-card-head">
            <span className={`security-shield ${passwordEnabled ? "is-on" : ""}`}>
              <LockOutlined />
            </span>
            <div>
              <strong>
                {passwordEnabled
                  ? t("settings.passwordOn")
                  : t("settings.passwordOff")}
              </strong>
            </div>
            <span className={`security-status ${passwordEnabled ? "is-on" : ""}`}>
              {passwordEnabled ? t("common.on") : t("common.off")}
            </span>
          </div>

          <div className="security-actions">
            {passwordEnabled ? (
              <>
                <Button onClick={() => setMode("change")}>
                  {t("settings.changePassword")}
                </Button>
                <Button icon={<LockOutlined />} onClick={() => void lockNow()}>
                  {t("settings.logout")}
                </Button>
                <Tooltip title={t("settings.disableHint")}>
                  <Button
                    danger
                    type="text"
                    onClick={() => setMode("disable")}
                  >
                    {t("settings.disablePassword")}
                  </Button>
                </Tooltip>
              </>
            ) : (
              <Button
                type="primary"
                icon={<LockOutlined />}
                onClick={() => setMode("setup")}
              >
                {t("settings.enablePassword")}
              </Button>
            )}
          </div>
        </div>

        <div className={`idle-lock-block${idleEnabled ? " is-on" : ""}`}>
          <div className="idle-lock-head">
            <span className="security-shield idle-lock-icon">
              <ClockCircleOutlined />
            </span>
            <div className="idle-lock-copy">
              <strong>{t("settings.idleLock")}</strong>
              <span>{t("settings.idleLockHint")}</span>
            </div>
            <Switch
              checked={idleEnabled}
              disabled={!passwordEnabled || idleSaving}
              onChange={(checked) => void saveIdleLock(checked, idleMinutes)}
            />
          </div>
          <div className="idle-lock-controls">
            <span>{t("settings.idleMinutes")}</span>
            <InputNumber
              min={1}
              max={240}
              value={idleMinutes}
              disabled={!passwordEnabled || idleSaving}
              onChange={(value) => {
                if (typeof value === "number") setIdleMinutes(value);
              }}
            />
            <Button
              size="small"
              disabled={!passwordEnabled || idleSaving}
              onClick={() => void saveIdleLock(idleEnabled, idleMinutes)}
            >
              {t("action.save")}
            </Button>
          </div>
          <p className="settings-field-hint">{t("settings.idleRange")}</p>
        </div>
      </SettingsCard>

      <PasswordModal
        mode={mode}
        submitting={submitting}
        onClose={() => {
          if (!submitting) setMode(null);
        }}
        onSubmit={(values) => void submit(values)}
      />
    </>
  );
}
