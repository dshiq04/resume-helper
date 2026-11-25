import { useCallback, useEffect, useState } from "react";
import {
  Alert,
  App as AntdApp,
  Button,
  DatePicker,
  Descriptions,
  Form,
  Modal,
  Spin,
  Timeline,
  Tooltip,
} from "antd";
import {
  CloseOutlined,
  DeleteOutlined,
  FieldTimeOutlined,
  HistoryOutlined,
  LinkOutlined,
  MinusOutlined,
  EyeOutlined,
} from "@ant-design/icons";
import dayjs, { type Dayjs } from "dayjs";
import { Events } from "@wailsio/runtime";
import { api, ApiError } from "../api/client";
import type { Application, ApplicationHistory, Status } from "../api/types";
import { CategoryBadge, StatusBadge } from "../components/badges";
import { ResizeHandles } from "../components/ResizeHandles";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";
import { isWebLink, withProtocol } from "../utils/display";
import {
  closeWindow,
  isDesktopRuntime,
  minimiseWindow,
  openExternal,
} from "../utils/desktop";

const STATUS_TIMELINE_COLORS: Record<string, string> = {
  pending: "gray",
  screening: "blue",
  written: "geekblue",
  interview: "cyan",
  offer: "green",
  rejected: "red",
};

function readRecordId(): number | null {
  const raw = new URLSearchParams(window.location.search).get("id");
  const value = Number(raw);
  return Number.isInteger(value) && value > 0 ? value : null;
}

export function ApplicationViewWindowPage() {
  const { t, formatFullDate } = useI18n();
  const { message } = AntdApp.useApp();
  const [supplementForm] = Form.useForm<Record<string, Dayjs>>();
  const [recordId] = useState(readRecordId);
  const [record, setRecord] = useState<Application | null>(null);
  const [history, setHistory] = useState<ApplicationHistory | null>(null);
  const [loading, setLoading] = useState(true);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [supplementOpen, setSupplementOpen] = useState(false);
  const [supplementSaving, setSupplementSaving] = useState(false);
  const [supplementStatuses, setSupplementStatuses] = useState<Status[]>([]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") void closeWindow();
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  const notifyDataChanged = useCallback(() => {
    window.dispatchEvent(new Event("resumetrail:data-changed"));
    if (isDesktopRuntime()) {
      void Events.Emit("resumetrail:data-changed").catch(() => {
        // Ignore: the main window may already be closing.
      });
    }
  }, []);

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
    if (recordId === null) {
      setLoading(false);
      setError(t("error.generic"));
      return;
    }
    let active = true;
    setLoading(true);
    void Promise.all([api.getApplication(recordId), api.applicationHistory(recordId)])
      .then(([application, timeline]) => {
        if (!active) return;
        setRecord(application);
        setHistory(timeline);
        document.title = `${application.companyName} · ${t("history.viewTitle")}`;
      })
      .catch((caught: unknown) => {
        if (!active) return;
        setError(
          caught instanceof ApiError && caught.status === 401
            ? t("error.unauthorized")
            : apiErrorMessage(caught, t),
        );
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [recordId, t]);

  const openSupplement = () => {
    if (!history) return;
    const defaults: Record<string, Dayjs> = {};
    for (const status of history.missingStatuses) {
      defaults[status] = dayjs(history.appliedAt);
    }
    supplementForm.setFieldsValue(defaults);
    setSupplementStatuses([...history.missingStatuses]);
    setSupplementOpen(true);
  };

  const removeSupplementStatus = (status: Status) => {
    setSupplementStatuses((current) => current.filter((item) => item !== status));
    supplementForm.setFields([{ name: status, value: undefined, errors: [] }]);
  };

  const submitSupplement = async () => {
    if (!history || recordId === null) return;
    let values: Record<string, Dayjs>;
    try {
      values = await supplementForm.validateFields(supplementStatuses);
    } catch {
      return;
    }
    setSupplementSaving(true);
    try {
      for (const status of supplementStatuses) {
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
    } catch (caught) {
      message.error(apiErrorMessage(caught, t));
    } finally {
      setSupplementSaving(false);
    }
  };

  const currentStatus = record?.status ?? "screening";
  const currentDays =
    history?.items[history.items.length - 1]?.daysInStatus ?? record?.statusDays;
  const formatTimestamp = (value: string) => {
    const parsed = dayjs(value);
    if (!parsed.isValid()) return "—";
    return `${formatFullDate(value)} ${parsed.format("HH:mm")}`;
  };

  return (
    <div className="application-view-shell">
      <header className="quick-window-bar">
        <span className="quick-window-title">
          <EyeOutlined />
          {t("history.viewTitle")}
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

      <div className="application-view-body">
        {loading ? (
          <div className="edit-window-loading">
            <Spin size="large" />
          </div>
        ) : error ? (
          <Alert type="error" showIcon message={error} />
        ) : record ? (
          <>
            <section className="application-view-hero">
              <div className="application-view-identity">
                <span className="application-view-logo">
                  {record.companyName.trim().slice(0, 1).toUpperCase()}
                </span>
                <div>
                  <h1>{record.companyName}</h1>
                  <div className="application-view-tags">
                    <CategoryBadge category={record.category} />
                    <StatusBadge status={record.status} />
                  </div>
                </div>
              </div>

              <Descriptions
                className="application-view-meta"
                size="small"
                column={1}
                items={[
                  {
                    key: "appliedAt",
                    label: (
                      <span className="application-view-meta-label">
                        <FieldTimeOutlined /> {t("table.appliedAt")}
                      </span>
                    ),
                    children: (
                      <Tooltip title={formatFullDate(record.appliedAt)} placement="top">
                        <span>{formatTimestamp(record.appliedAt)}</span>
                      </Tooltip>
                    ),
                  },
                  {
                    key: "link",
                    label: (
                      <span className="application-view-meta-label">
                        <LinkOutlined /> {t("table.link")}
                      </span>
                    ),
                    children: record.link ? (
                      isWebLink(record.link) ? (
                        <button
                          type="button"
                          className="application-view-link"
                          onClick={() =>
                            void openExternal(withProtocol(record.link))
                          }
                        >
                          {record.link}
                        </button>
                      ) : (
                        <span>{record.link}</span>
                      )
                    ) : (
                      <span className="link-empty">{t("table.noLink")}</span>
                    ),
                  },
                  {
                    key: "status",
                    label: (
                      <span className="application-view-meta-label">
                        <HistoryOutlined /> {t("form.status")}
                      </span>
                    ),
                    children: (
                      <span className="application-view-status-line">
                        <StatusBadge status={currentStatus} />
                        {typeof currentDays === "number" && (
                          <span className="application-view-status-age">
                            {t("table.statusAge", {
                              status: t(`status.${currentStatus}`),
                              count: currentDays,
                            })}
                          </span>
                        )}
                      </span>
                    ),
                  },
                ]}
              />
            </section>

            <section className="application-view-timeline">
              <header className="application-view-section-head">
                <div>
                  <h2>
                    <HistoryOutlined /> {t("history.title")}
                  </h2>
                  <p>{t("history.viewHint")}</p>
                </div>
                {history && history.missingStatuses.length > 0 && (
                  <Button size="small" onClick={openSupplement}>
                    {t("history.missingButton")}
                  </Button>
                )}
              </header>

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
                  className="application-timeline application-view-timeline-list"
                  items={history.items.map((change, index) => ({
                    color: STATUS_TIMELINE_COLORS[change.status] ?? "blue",
                    children: (
                      <div className="timeline-entry application-view-timeline-entry">
                        <div className="timeline-entry-head">
                          <StatusBadge status={change.status} />
                          <Tooltip
                            title={formatFullDate(change.changedAt)}
                            placement="top"
                          >
                            <span className="timeline-date">
                              {formatTimestamp(change.changedAt)}
                            </span>
                          </Tooltip>
                        </div>
                        <span className="timeline-duration">
                          {t("history.daysInStatus", {
                            count: change.daysInStatus ?? 0,
                          })}
                        </span>
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
          </>
        ) : null}
      </div>

      <Modal
        open={supplementOpen}
        title={t("history.supplementTitle")}
        okText={t("action.save")}
        cancelText={t("action.cancel")}
        confirmLoading={supplementSaving}
        okButtonProps={{ disabled: supplementStatuses.length === 0 }}
        onOk={() => void submitSupplement()}
        onCancel={() => {
          if (!supplementSaving) setSupplementOpen(false);
        }}
        destroyOnHidden
      >
        <p className="edit-history-modal-hint">{t("history.missingHint")}</p>
        <Form form={supplementForm} layout="vertical" requiredMark={false}>
          {supplementStatuses.map((status) => (
            <div className="edit-history-field" key={status}>
              <Form.Item
                className="edit-history-form-item"
                name={status}
                label={<StatusBadge status={status} />}
                rules={[{ required: true, message: t("history.missingRequired") }]}
              >
                <DatePicker className="edit-history-date" showTime={false} />
              </Form.Item>
              <Button
                type="text"
                danger
                className="edit-history-delete"
                aria-label={t("action.delete")}
                title={t("action.delete")}
                icon={<DeleteOutlined />}
                onClick={() => removeSupplementStatus(status)}
              />
            </div>
          ))}
        </Form>
      </Modal>
      <ResizeHandles />
    </div>
  );
}
