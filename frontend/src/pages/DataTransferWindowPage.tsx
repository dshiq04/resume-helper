import { useState } from "react";
import {
  Alert,
  Button,
  Descriptions,
  Result,
  Segmented,
  Spin,
  Tag,
} from "antd";
import {
  CheckCircleOutlined,
  CloudDownloadOutlined,
  CloudUploadOutlined,
  FileZipOutlined,
  WarningOutlined,
} from "@ant-design/icons";
import { Call, Dialogs, Events } from "@wailsio/runtime";
import type { BackupResult } from "../api/types";
import { useI18n } from "../i18n";
import { closeWindow, isDesktopRuntime } from "../utils/desktop";

type TransferMode = "export" | "import";

const EXPORT_BINDING = "main.BackupService.ExportBackup";
const INSPECT_BINDING = "main.BackupService.InspectBackup";
const IMPORT_BINDING = "main.BackupService.ImportBackup";

function readMode(): TransferMode {
  const value = new URLSearchParams(window.location.search).get("mode");
  return value === "import" ? "import" : "export";
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 KB";
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function errorText(error: unknown): string {
  if (error instanceof Error && error.message) return error.message;
  if (typeof error === "string" && error) return error;
  if (error && typeof error === "object") {
    const message = (error as { message?: unknown }).message;
    if (typeof message === "string" && message) return message;
  }
  return "";
}

function isCancelled(error: unknown): boolean {
  return errorText(error).toLowerCase().includes("cancel");
}

function todayStamp(): string {
  const now = new Date();
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}`;
}

export function DataTransferWindowPage() {
  const { t, formatFullDate } = useI18n();
  const [mode, setMode] = useState<TransferMode>(readMode);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<BackupResult | null>(null);
  const [preview, setPreview] = useState<BackupResult | null>(null);
  const [error, setError] = useState("");

  const reset = (nextMode?: TransferMode) => {
    setError("");
    setResult(null);
    setPreview(null);
    setBusy(false);
    if (nextMode) setMode(nextMode);
  };

  const exportData = async () => {
    if (!isDesktopRuntime()) {
      setError(t("transfer.desktopOnly"));
      return;
    }
    setError("");
    setResult(null);
    try {
      const path = await Dialogs.SaveFile({
        Title: t("transfer.exportTitle"),
        Filename: `resumetrail-backup-${todayStamp()}.zip`,
        Filters: [{ DisplayName: "ResumeTrail backup (*.zip)", Pattern: "*.zip" }],
        CanChooseFiles: true,
        CanChooseDirectories: false,
        CanCreateDirectories: true,
        AllowsOtherFiletypes: true,
      });
      if (!path) return;
      setBusy(true);
      const exported = (await Call.ByName(EXPORT_BINDING, path)) as BackupResult;
      setResult(exported);
      Events.Emit("resumetrail:data-changed").catch(() => {
        // The main window may not be listening; the operation still succeeded.
      });
    } catch (caught) {
      if (isCancelled(caught)) return;
      setError(errorText(caught) || t("transfer.failed"));
    } finally {
      setBusy(false);
    }
  };

  const chooseImport = async () => {
    if (!isDesktopRuntime()) {
      setError(t("transfer.desktopOnly"));
      return;
    }
    setError("");
    setResult(null);
    setPreview(null);
    try {
      const selected = await Dialogs.OpenFile({
        Title: t("transfer.importTitle"),
        Filters: [{ DisplayName: "ResumeTrail backup (*.zip)", Pattern: "*.zip" }],
        AllowsMultipleSelection: false,
        CanChooseFiles: true,
        CanChooseDirectories: false,
      });
      const path = Array.isArray(selected) ? selected[0] : selected;
      if (!path) return;
      setBusy(true);
      const inspected = (await Call.ByName(INSPECT_BINDING, path)) as BackupResult;
      setPreview(inspected);
    } catch (caught) {
      if (isCancelled(caught)) return;
      setError(errorText(caught) || t("transfer.failed"));
    } finally {
      setBusy(false);
    }
  };

  const importData = async () => {
    if (!preview) return;
    setBusy(true);
    setError("");
    try {
      const imported = (await Call.ByName(IMPORT_BINDING, preview.path)) as BackupResult;
      setResult(imported);
      setPreview(null);
      Events.Emit("resumetrail:data-changed").catch(() => {
        // The main window may not be listening; the operation still succeeded.
      });
    } catch (caught) {
      setError(errorText(caught) || t("transfer.failed"));
    } finally {
      setBusy(false);
    }
  };

  const countsPanel = (data: BackupResult) => (
    <div className="transfer-counts">
      <div className="transfer-count">
        <span>{t("transfer.applications")}</span>
        <strong>{data.counts.applications}</strong>
      </div>
      <div className="transfer-count">
        <span>{t("transfer.resumes")}</span>
        <strong>{data.counts.resumes}</strong>
      </div>
      <div className="transfer-count">
        <span>{t("transfer.changes")}</span>
        <strong>{data.counts.changes}</strong>
      </div>
      <div className="transfer-count">
        <span>{t("transfer.trash")}</span>
        <strong>{data.counts.trash}</strong>
      </div>
      <div className="transfer-count">
        <span>{t("transfer.settings")}</span>
        <strong>{data.counts.settings}</strong>
      </div>
      <div className="transfer-count">
        <span>{t("transfer.size")}</span>
        <strong>{formatBytes(data.counts.bytes)}</strong>
      </div>
    </div>
  );

  return (
    <div className="shortcut-prompt-shell transfer-shell">
      <header className="quick-window-bar">
        <span className="quick-window-title">
          <FileZipOutlined />
          {t("transfer.title")}
        </span>
        <div className="quick-window-actions">
          <button
            className="quick-window-close"
            type="button"
            aria-label={t("transfer.close")}
            title={t("transfer.close")}
            onClick={() => void closeWindow()}
          >
            ×
          </button>
        </div>
      </header>

      <div className="transfer-body">
        <p className="transfer-subtitle">{t("transfer.subtitle")}</p>

        <Segmented
          block
          value={mode}
          options={[
            {
              value: "export",
              label: (
                <span className="transfer-tab">
                  <CloudDownloadOutlined /> {t("transfer.exportTitle")}
                </span>
              ),
            },
            {
              value: "import",
              label: (
                <span className="transfer-tab">
                  <CloudUploadOutlined /> {t("transfer.importTitle")}
                </span>
              ),
            },
          ]}
          onChange={(value) => reset(value as TransferMode)}
        />

        {mode === "export" ? (
          <section className="transfer-panel">
            <div className="transfer-panel-head">
              <span className="transfer-panel-icon is-export">
                <CloudDownloadOutlined />
              </span>
              <div>
                <h3>{t("transfer.exportTitle")}</h3>
                <p>{t("transfer.exportDescription")}</p>
              </div>
            </div>
            <p className="transfer-note">{t("transfer.deviceNote")}</p>
            <Button
              type="primary"
              size="large"
              block
              icon={<CloudDownloadOutlined />}
              loading={busy}
              onClick={() => void exportData()}
            >
              {busy ? t("transfer.exporting") : t("transfer.chooseSave")}
            </Button>
            {result && (
              <div className="transfer-result is-success">
                <div className="transfer-result-title">
                  <CheckCircleOutlined />
                  {t("transfer.exportSuccess")}
                </div>
                <Descriptions
                  size="small"
                  column={1}
                  className="transfer-descriptions"
                  items={[
                    {
                      key: "path",
                      label: t("transfer.path"),
                      children: <code>{result.path}</code>,
                    },
                    {
                      key: "time",
                      label: t("transfer.exportedAt"),
                      children: formatFullDate(result.exportedAt),
                    },
                    {
                      key: "version",
                      label: t("transfer.appVersion"),
                      children: `v${result.appVersion}`,
                    },
                  ]}
                />
                {countsPanel(result)}
              </div>
            )}
          </section>
        ) : (
          <section className="transfer-panel">
            <div className="transfer-panel-head">
              <span className="transfer-panel-icon is-import">
                <CloudUploadOutlined />
              </span>
              <div>
                <h3>{t("transfer.importTitle")}</h3>
                <p>{t("transfer.importDescription")}</p>
              </div>
            </div>
            <p className="transfer-note is-warning">
              <WarningOutlined /> {t("transfer.passwordNote")}
            </p>
            {!preview ? (
              <Button
                type="primary"
                size="large"
                block
                icon={<CloudUploadOutlined />}
                loading={busy}
                  onClick={() => void chooseImport()}
              >
                {busy ? t("transfer.inspecting") : t("transfer.chooseFile")}
              </Button>
            ) : (
              <>
                <div className="transfer-preview">
                  <div className="transfer-preview-head">
                    <span className="transfer-preview-file">
                      <FileZipOutlined />
                      <span title={preview.path}>{preview.path}</span>
                    </span>
                    <Tag color="blue">v{preview.appVersion}</Tag>
                  </div>
                  <p className="transfer-preview-label">{t("transfer.contains")}</p>
                  {countsPanel(preview)}
                  <Alert
                    type="warning"
                    showIcon
                    className="transfer-alert"
                    message={t("transfer.overwriteTitle")}
                    description={t("transfer.overwriteMessage")}
                  />
                </div>
                <div className="transfer-actions">
                  <Button disabled={busy} onClick={() => setPreview(null)}>
                    {t("action.cancel")}
                  </Button>
                  <Button
                    danger
                    type="primary"
                    size="large"
                    loading={busy}
                    onClick={() => void importData()}
                  >
                    {busy ? t("transfer.importing") : t("transfer.confirmImport")}
                  </Button>
                </div>
              </>
            )}
            {result && (
              <Result
                status="success"
                icon={<CheckCircleOutlined />}
                title={t("transfer.importSuccess")}
                subTitle={`${t("transfer.applications")}: ${result.counts.applications} · ${t("transfer.resumes")}: ${result.counts.resumes}`}
              />
            )}
          </section>
        )}

        {error && (
          <Alert
            type="error"
            showIcon
            message={t("transfer.failed")}
            description={error}
            className="transfer-alert"
          />
        )}

        {busy && mode === "import" && !preview && (
          <div className="transfer-busy">
            <Spin size="small" />
            <span>{t("transfer.inspecting")}</span>
          </div>
        )}
      </div>
    </div>
  );
}
