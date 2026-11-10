import { useEffect, useState } from "react";
import {
  App as AntdApp,
  Button,
  Drawer,
  Form,
  Input,
  Modal,
  Segmented,
  Select,
  Tooltip,
} from "antd";
import {
  BgColorsOutlined,
  CloseOutlined,
  DatabaseOutlined,
  InfoCircleOutlined,
  LockOutlined,
  MoonOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  SunOutlined,
} from "@ant-design/icons";
import { api, clearToken, setToken } from "../api/client";
import type { SystemInfo } from "../api/types";
import { localeOptions, useI18n } from "../i18n";
import type { ThemeMode } from "../theme";
import { apiErrorMessage } from "../utils/errors";

type SecurityMode = "setup" | "change" | "disable" | null;

interface SecurityFormValues {
  currentPassword?: string;
  password?: string;
  newPassword?: string;
  confirmPassword?: string;
}

interface SettingsDrawerProps {
  open: boolean;
  onClose: () => void;
  themeMode: ThemeMode;
  onThemeChange: (mode: ThemeMode) => void;
  passwordEnabled: boolean;
  onPasswordEnabledChange: (enabled: boolean) => void;
  onLock: () => void;
}

export function SettingsDrawer({
  open,
  onClose,
  themeMode,
  onThemeChange,
  passwordEnabled,
  onPasswordEnabledChange,
  onLock,
}: SettingsDrawerProps) {
  const { t, locale, setLocale } = useI18n();
  const { message } = AntdApp.useApp();
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const [securityMode, setSecurityMode] = useState<SecurityMode>(null);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm<SecurityFormValues>();

  useEffect(() => {
    if (!open) return;
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
  }, [message, open, t]);

  const openSecurity = (mode: Exclude<SecurityMode, null>) => {
    form.resetFields();
    setSecurityMode(mode);
  };

  const closeSecurity = () => {
    if (submitting) return;
    setSecurityMode(null);
    form.resetFields();
  };

  const submitSecurity = async (values: SecurityFormValues) => {
    if (!securityMode) return;
    setSubmitting(true);
    try {
      if (securityMode === "setup") {
        const session = await api.setupPassword(values.newPassword ?? "");
        setToken(session.token);
        onPasswordEnabledChange(true);
        message.success(t("toast.passwordEnabled"));
      } else if (securityMode === "change") {
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
      setSecurityMode(null);
      form.resetFields();
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

  const securityTitle =
    securityMode === "setup"
      ? t("settings.setupTitle")
      : securityMode === "change"
        ? t("settings.changeTitle")
        : t("settings.disableTitle");

  return (
    <>
      <Drawer
        open={open}
        onClose={onClose}
        width={440}
        className="settings-drawer"
        closeIcon={<CloseOutlined />}
        title={
          <div className="settings-title">
            <span className="settings-title-eyebrow">
              <SettingOutlined /> {t("nav.settings")}
            </span>
            <h2>{t("settings.title")}</h2>
            <p>{t("settings.subtitle")}</p>
          </div>
        }
      >
        <section className="settings-section">
          <div className="settings-section-head">
            <span className="settings-section-icon">
              <BgColorsOutlined />
            </span>
            <div>
              <h3>{t("settings.appearance")}</h3>
              <p>{t("settings.themeHint")}</p>
            </div>
          </div>

          <div className="settings-field">
            <label>{t("settings.theme")}</label>
            <Segmented
              block
              value={themeMode}
              onChange={(value) => onThemeChange(value as ThemeMode)}
              options={[
                {
                  value: "light",
                  label: (
                    <span className="settings-segment-label">
                      <SunOutlined /> {t("settings.light")}
                    </span>
                  ),
                },
                {
                  value: "dark",
                  label: (
                    <span className="settings-segment-label">
                      <MoonOutlined /> {t("settings.dark")}
                    </span>
                  ),
                },
              ]}
            />
          </div>

          <div className="settings-field">
            <label>{t("settings.language")}</label>
            <Select
              value={locale}
              onChange={setLocale}
              options={localeOptions}
              optionRender={(option) => (
                <span className="settings-language-option">
                  <span className="settings-language-short">{option.data.short}</span>
                  {option.data.label}
                </span>
              )}
            />
            <span className="settings-field-hint">{t("settings.languageHint")}</span>
          </div>
        </section>

        <section className="settings-section">
          <div className="settings-section-head">
            <span className="settings-section-icon is-security">
              <SafetyCertificateOutlined />
            </span>
            <div>
              <h3>{t("settings.security")}</h3>
              <p>{t("settings.securityHint")}</p>
            </div>
          </div>

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
                  <Button onClick={() => openSecurity("change")}>
                    {t("settings.changePassword")}
                  </Button>
                  <Button icon={<LockOutlined />} onClick={() => void lockNow()}>
                    {t("settings.logout")}
                  </Button>
                  <Tooltip title={t("settings.disableHint")}>
                    <Button
                      danger
                      type="text"
                      onClick={() => openSecurity("disable")}
                    >
                      {t("settings.disablePassword")}
                    </Button>
                  </Tooltip>
                </>
              ) : (
                <Button
                  type="primary"
                  icon={<LockOutlined />}
                  onClick={() => openSecurity("setup")}
                >
                  {t("settings.enablePassword")}
                </Button>
              )}
            </div>
          </div>
        </section>

        <section className="settings-section">
          <div className="settings-section-head">
            <span className="settings-section-icon is-data">
              <DatabaseOutlined />
            </span>
            <div>
              <h3>{t("settings.data")}</h3>
              <p>{t("settings.database")}</p>
            </div>
          </div>

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
              <span>{t("settings.configPath")}</span>
              <code title={systemInfo?.configPath}>{systemInfo?.configPath ?? "—"}</code>
            </div>
            <div className="settings-info-row">
              <span>{t("settings.version")}</span>
              <strong>{systemInfo?.version ?? "1.0.0"}</strong>
            </div>
          </div>
        </section>

        <section className="settings-about">
          <span className="settings-about-icon">
            <InfoCircleOutlined />
          </span>
          <div>
            <strong>{t("settings.about")}</strong>
            <p>{t("settings.aboutText")}</p>
          </div>
        </section>
      </Drawer>

      <Modal
        open={securityMode !== null}
        title={securityTitle}
        onCancel={closeSecurity}
        footer={null}
        width={430}
        centered
        className="settings-modal"
      >
        {securityMode === "disable" && (
          <p className="settings-modal-hint">{t("settings.disableHint")}</p>
        )}
        <Form
          form={form}
          layout="vertical"
          requiredMark={false}
          onFinish={submitSecurity}
          className="settings-form"
        >
          {securityMode === "change" && (
            <Form.Item
              name="currentPassword"
              label={t("settings.currentPassword")}
              rules={[{ required: true, message: t("settings.passwordRequired") }]}
            >
              <Input.Password autoComplete="current-password" />
            </Form.Item>
          )}

          {securityMode === "disable" ? (
            <Form.Item
              name="password"
              label={t("settings.currentPassword")}
              rules={[{ required: true, message: t("settings.passwordRequired") }]}
            >
              <Input.Password autoComplete="current-password" autoFocus />
            </Form.Item>
          ) : (
            <>
              <Form.Item
                name="newPassword"
                label={t("settings.newPassword")}
                rules={[
                  { required: true, message: t("settings.passwordRequired") },
                  { min: 4, message: t("settings.passwordMin") },
                ]}
              >
                <Input.Password autoComplete="new-password" autoFocus />
              </Form.Item>
              <Form.Item
                name="confirmPassword"
                label={t("settings.confirmPassword")}
                dependencies={["newPassword"]}
                rules={[
                  { required: true, message: t("settings.passwordRequired") },
                  ({ getFieldValue }) => ({
                    validator(_rule, value) {
                      if (!value || getFieldValue("newPassword") === value) {
                        return Promise.resolve();
                      }
                      return Promise.reject(new Error(t("settings.passwordMismatch")));
                    },
                  }),
                ]}
              >
                <Input.Password autoComplete="new-password" />
              </Form.Item>
            </>
          )}

          <div className="dialog-actions">
            <Button onClick={closeSecurity}>{t("action.cancel")}</Button>
            <Button
              type="primary"
              htmlType="submit"
              loading={submitting}
              danger={securityMode === "disable"}
            >
              {securityMode === "disable"
                ? t("settings.disablePassword")
                : t("action.save")}
            </Button>
          </div>
        </Form>
      </Modal>
    </>
  );
}
