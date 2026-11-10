import { useState } from "react";
import { App as AntdApp, Button, Tooltip } from "antd";
import { LockOutlined, SafetyCertificateOutlined } from "@ant-design/icons";
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
        description={t("settings.securityHint")}
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
              <span>{t("settings.securityHint")}</span>
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
