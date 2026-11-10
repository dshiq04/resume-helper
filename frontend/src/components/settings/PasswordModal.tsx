import { useEffect } from "react";
import { Button, Form, Input, Modal } from "antd";
import { useI18n } from "../../i18n";

export type SecurityMode = "setup" | "change" | "disable";

export interface SecurityFormValues {
  currentPassword?: string;
  password?: string;
  newPassword?: string;
  confirmPassword?: string;
}

interface PasswordModalProps {
  mode: SecurityMode | null;
  submitting: boolean;
  onClose: () => void;
  onSubmit: (values: SecurityFormValues) => void;
}

export function PasswordModal({
  mode,
  submitting,
  onClose,
  onSubmit,
}: PasswordModalProps) {
  const { t } = useI18n();
  const [form] = Form.useForm<SecurityFormValues>();

  useEffect(() => {
    form.resetFields();
  }, [form, mode]);

  const title =
    mode === "setup"
      ? t("settings.setupTitle")
      : mode === "change"
        ? t("settings.changeTitle")
        : t("settings.disableTitle");

  return (
    <Modal
      open={mode !== null}
      title={title}
      onCancel={() => {
        if (!submitting) onClose();
      }}
      footer={null}
      width={430}
      centered
      className="settings-modal"
    >
      {mode === "disable" && (
        <p className="settings-modal-hint">{t("settings.disableHint")}</p>
      )}
      <Form
        form={form}
        layout="vertical"
        requiredMark={false}
        onFinish={onSubmit}
        className="settings-form"
      >
        {mode === "change" && (
          <Form.Item
            name="currentPassword"
            label={t("settings.currentPassword")}
            rules={[{ required: true, message: t("settings.passwordRequired") }]}
          >
            <Input.Password autoComplete="current-password" />
          </Form.Item>
        )}

        {mode === "disable" ? (
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
                    return Promise.reject(
                      new Error(t("settings.passwordMismatch")),
                    );
                  },
                }),
              ]}
            >
              <Input.Password autoComplete="new-password" />
            </Form.Item>
          </>
        )}

        <div className="dialog-actions">
          <Button
            onClick={onClose}
            disabled={submitting}
          >
            {t("action.cancel")}
          </Button>
          <Button
            type="primary"
            htmlType="submit"
            loading={submitting}
            danger={mode === "disable"}
          >
            {mode === "disable"
              ? t("settings.disablePassword")
              : t("action.save")}
          </Button>
        </div>
      </Form>
    </Modal>
  );
}
