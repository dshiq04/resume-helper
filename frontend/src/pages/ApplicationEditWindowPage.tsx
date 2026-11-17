import { useCallback, useEffect, useState } from "react";
import {
  Alert,
  App as AntdApp,
  Button,
  DatePicker,
  Form,
  Input,
  Modal,
  Select,
  Spin,
  Timeline,
  Tooltip,
} from "antd";
import {
  BankOutlined,
  CloseOutlined,
  EditOutlined,
  HistoryOutlined,
  LinkOutlined,
  MinusOutlined,
} from "@ant-design/icons";
import dayjs, { type Dayjs } from "dayjs";
import { Events } from "@wailsio/runtime";
import { api, ApiError } from "../api/client";
import type {
  ApplicationHistory,
  Category,
  Status,
} from "../api/types";
import { useCategoryOptions, useStatusOptions, StatusBadge } from "../components/badges";
import { CompanyAutoComplete } from "../components/CompanyAutoComplete";
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
  const { t, formatFullDate } = useI18n();
  const { message } = AntdApp.useApp();
  const [form] = Form.useForm<EditFormValues>();
  const [supplementForm] = Form.useForm<Record<string, Dayjs>>();
  const [recordId] = useState(readRecordId);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [history, setHistory] = useState<ApplicationHistory | null>(null);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [supplementOpen, setSupplementOpen] = useState(false);
  const [supplementSaving, setSupplementSaving] = useState(false);
  const categoryOptions = useCategoryOptions();
  const statusOptions = useStatusOptions();
  const currentStatus = Form.useWatch("status", form);

  const loadHistory = useCallback(async () => {
    if (recordId === null) return;
    setHistoryLoading(true);
    try {
      setHistory(await api.applicationHistory(recordId));
    } catch {
      setHistory(null);
    } finally {
      setHistoryLoading(false);
    }
  }, [recordId]);

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

    void loadHistory();

    return () => {
      active = false;
    };
  }, [form, loadHistory, message, recordId, t]);

  const notifyDataChanged = useCallback(() => {
    window.dispatchEvent(new Event("resumetrail:data-changed"));
    if (isDesktopRuntime()) {
      void Events.Emit("resumetrail:data-changed").catch(() => {
        // Ignore: the main window may already be closing.
      });
    }
  }, []);

  const openSupplement = () => {
    if (!history) return;
    const defaults: Record<string, Dayjs> = {};
    for (const status of history.missingStatuses) {
      defaults[status] = dayjs(history.appliedAt);
    }
    supplementForm.setFieldsValue(defaults);
    setSupplementOpen(true);
  };

  const submitSupplement = async () => {
    if (!history || recordId === null) return;
    let values: Record<string, Dayjs>;
    try {
      values = await supplementForm.validateFields();
    } catch {
      return;
    }
    setSupplementSaving(true);
    try {
      for (const status of history.missingStatuses) {
        const value = values[status];
        if (!value) continue;
        await api.addApplicationHistory(
          recordId,
          status,
          value.endOf("day").toISOString(),
        );
      }
      message.success(t("history.saved"));
      setSupplementOpen(false);
      notifyDataChanged();
      await loadHistory();
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setSupplementSaving(false);
    }
  };

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
                <CompanyAutoComplete
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

            <section className="edit-history">
              <header className="edit-history-head">
                <div>
                  <h3>
                    <HistoryOutlined /> {t("history.title")}
                  </h3>
                  <p>{t("history.hint")}</p>
                </div>
                {history && history.missingStatuses.length > 0 && (
                  <Button size="small" onClick={openSupplement}>
                    {t("history.missingButton")}
                  </Button>
                )}
              </header>

              {history && currentStatus === "screening" && (
                <div className="history-screening-age">
                  {t("history.screeningAge", { count: history.screeningDays })}
                </div>
              )}

              {history && history.missingStatuses.length > 0 && (
                <Alert
                  type="warning"
                  showIcon
                  className="edit-history-alert"
                  message={t("history.missingTitle")}
                  description={t("history.missingHint")}
                />
              )}

              {historyLoading ? (
                <div className="edit-history-loading">
                  <Spin size="small" />
                </div>
              ) : history && history.items.length > 0 ? (
                <Timeline
                  className="application-timeline"
                  items={history.items.map((change, index) => ({
                    color: change.status,
                    children: (
                      <div className="timeline-entry">
                        <div className="timeline-entry-head">
                          <StatusBadge status={change.status} />
                          <Tooltip
                            title={formatFullDate(change.changedAt)}
                            placement="top"
                          >
                            <span className="timeline-date">
                              {formatFullDate(change.changedAt)}
                            </span>
                          </Tooltip>
                        </div>
                        {typeof change.daysSincePrevious === "number" && (
                          <span className="timeline-gap">
                            {t("history.interval", {
                              count: change.daysSincePrevious,
                            })}
                          </span>
                        )}
                        {index === history.items.length - 1 && (
                          <span className="timeline-current">
                            {t("history.current")}
                          </span>
                        )}
                      </div>
                    ),
                  }))}
                />
              ) : (
                <p className="edit-history-empty">{t("history.empty")}</p>
              )}
            </section>

            <Modal
              open={supplementOpen}
              title={t("history.supplementTitle")}
              okText={t("action.save")}
              cancelText={t("action.cancel")}
              confirmLoading={supplementSaving}
              onOk={() => void submitSupplement()}
              onCancel={() => {
                if (!supplementSaving) setSupplementOpen(false);
              }}
              destroyOnHidden
            >
              <p className="edit-history-modal-hint">
                {t("history.missingHint")}
              </p>
              <Form form={supplementForm} layout="vertical" requiredMark={false}>
                {history?.missingStatuses.map((status) => (
                  <Form.Item
                    key={status}
                    name={status}
                    label={<StatusBadge status={status} />}
                    rules={[{ required: true, message: t("history.missingRequired") }]}
                  >
                    <DatePicker className="edit-history-date" showTime={false} />
                  </Form.Item>
                ))}
              </Form>
            </Modal>
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
