import { useCallback, useEffect, useState } from "react";
import {
  Alert,
  App as AntdApp,
  Button,
  Form,
  Input,
  Select,
  Spin,
} from "antd";
import {
  BankOutlined,
  CloseOutlined,
  EditOutlined,
  LinkOutlined,
  MinusOutlined,
} from "@ant-design/icons";
import { Events } from "@wailsio/runtime";
import { api, ApiError } from "../api/client";
import type { Category, Status } from "../api/types";
import { useCategoryOptions, useStatusOptions, StatusBadge } from "../components/badges";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";
import { closeWindow, isDesktopRuntime, minimiseWindow } from "../utils/desktop";

interface EditFormValues {
  category: Category;
  companyName: string;
  link: string;
  status: Status;
}

function readRecordId(): number | null {
  const raw = new URLSearchParams(window.location.search).get("id");
  const value = Number(raw);
  return Number.isInteger(value) && value > 0 ? value : null;
}

export function ApplicationEditWindowPage() {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [form] = Form.useForm<EditFormValues>();
  const [recordId] = useState(readRecordId);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const categoryOptions = useCategoryOptions();
  const statusOptions = useStatusOptions();

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") void closeWindow();
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  useEffect(() => {
    if (recordId === null) {
      setLoading(false);
      setLoadError(t("error.generic"));
      return;
    }

    let active = true;
    void api
      .getApplication(recordId)
      .then((record) => {
        if (!active) return;
        form.setFieldsValue({
          category: record.category,
          companyName: record.companyName,
          link: record.link,
          status: record.status,
        });
      })
      .catch((error: unknown) => {
        if (!active) return;
        const text = apiErrorMessage(error, t);
        setLoadError(text);
        message.error(text);
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [form, message, recordId, t]);

  const notifyDataChanged = useCallback(() => {
    window.dispatchEvent(new Event("resumetrail:data-changed"));
    if (isDesktopRuntime()) {
      void Events.Emit("resumetrail:data-changed").catch(() => {
        // Ignore: the main window may already be closing.
      });
    }
  }, []);

  const handleSubmit = async (values: EditFormValues) => {
    if (recordId === null) return;
    setSaving(true);
    try {
      await api.updateApplication(recordId, {
        category: values.category,
        companyName: values.companyName.trim(),
        link: values.link?.trim() ?? "",
        status: values.status,
      });
      message.success(t("toast.updated"));
      notifyDataChanged();
      await closeWindow();
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        setLoadError(apiErrorMessage(error, t));
      }
      message.error(apiErrorMessage(error, t));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="edit-window-shell">
      <header className="quick-window-bar">
        <span className="quick-window-title">
          <EditOutlined />
          {t("edit.title")}
        </span>
        <div className="quick-window-actions">
          <button
            className="quick-window-min"
            type="button"
            aria-label={t("window.minimize")}
            title={t("window.minimize")}
            onClick={() => void minimiseWindow()}
          >
            <MinusOutlined />
          </button>
          <button
            className="quick-window-close"
            type="button"
            aria-label={t("window.close")}
            title={t("window.close")}
            onClick={() => void closeWindow()}
          >
            <CloseOutlined />
          </button>
        </div>
      </header>

      <div className="edit-window-body">
        {loading ? (
          <div className="edit-window-loading">
            <Spin />
          </div>
        ) : (
          <>
            {loadError && (
              <Alert
                type="error"
                showIcon
                message={loadError}
                className="edit-window-alert"
              />
            )}

            <Form
              form={form}
              layout="vertical"
              requiredMark={false}
              disabled={recordId === null}
              onFinish={(values) => void handleSubmit(values)}
            >
              <Form.Item
                name="companyName"
                label={t("form.company")}
                rules={[
                  {
                    required: true,
                    whitespace: true,
                    message: t("form.companyRequired"),
                  },
                ]}
              >
                <Input
                  maxLength={255}
                  prefix={<BankOutlined className="field-icon" />}
                  placeholder={t("form.companyPlaceholder")}
                />
              </Form.Item>

              <Form.Item
                name="category"
                label={t("edit.category")}
                rules={[{ required: true }]}
              >
                <Select options={categoryOptions} />
              </Form.Item>

              <Form.Item name="link" label={t("form.link")}>
                <Input
                  maxLength={2048}
                  prefix={<LinkOutlined className="field-icon" />}
                  placeholder={t("form.linkPlaceholder")}
                />
              </Form.Item>

              <Form.Item
                name="status"
                label={t("form.status")}
                rules={[{ required: true }]}
              >
                <Select
                  options={statusOptions}
                  labelRender={({ value }) => (
                    <StatusBadge status={value as Status} />
                  )}
                  optionRender={(option) => (
                    <StatusBadge status={option.value as Status} />
                  )}
                />
              </Form.Item>
            </Form>
          </>
        )}
      </div>

      <footer className="edit-window-footer">
        <Button onClick={() => void closeWindow()}>{t("action.close")}</Button>
        <Button
          type="primary"
          className="edit-save-button"
          loading={saving}
          disabled={loading || recordId === null || Boolean(loadError)}
          onClick={() => form.submit()}
        >
          {t("action.modify")}
        </Button>
      </footer>
    </div>
  );
}
