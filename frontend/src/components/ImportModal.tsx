import { useEffect, useState } from "react";
import { App as AntdApp, Button, Modal, Upload } from "antd";
import {
  CheckCircleFilled,
  CloseOutlined,
  CloudUploadOutlined,
  DownloadOutlined,
  FileExcelOutlined,
  InboxOutlined,
  WarningOutlined,
} from "@ant-design/icons";
import { Call, Dialogs, Events } from "@wailsio/runtime";
import { api } from "../api/client";
import type { ImportResult } from "../api/types";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";
import { isDesktopRuntime } from "../utils/desktop";

const FILES_DROPPED_EVENT = "resumetrail:file-dropped";
const EXCEL_DROP_TARGET = "excel-import";
const PROBE_EXCEL_BINDING = "main.DropImporter.ProbeExcelFile";
const IMPORT_EXCEL_BINDING = "main.DropImporter.ImportExcel";

interface ImportModalProps {
  open: boolean;
  onClose: () => void;
  onImported: (result: ImportResult) => void;
}

interface DroppedExcelFile {
  path: string;
  fileName: string;
  fileSize: number;
}

function bindingErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) return error.message;
  if (typeof error === "string" && error) return error;
  if (error && typeof error === "object") {
    const message = (error as { message?: unknown }).message;
    if (typeof message === "string" && message) return message;
    try {
      return JSON.stringify(error);
    } catch {
      // Fall through to the generic message.
    }
  }
  return "未知错误";
}

function isDialogCancelled(error: unknown): boolean {
  return bindingErrorMessage(error).toLowerCase().includes("cancel");
}

function isExcelPath(path: string): boolean {
  const lower = path.toLowerCase();
  return lower.endsWith(".xlsx") || lower.endsWith(".xlsm");
}

export function ImportModal({ open, onClose, onImported }: ImportModalProps) {
  const { t, locale } = useI18n();
  const { message } = AntdApp.useApp();
  const [file, setFile] = useState<File | null>(null);
  const [nativeFile, setNativeFile] = useState<DroppedExcelFile | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [importing, setImporting] = useState(false);
  const [downloading, setDownloading] = useState(false);
  const desktop = isDesktopRuntime();

  useEffect(() => {
    if (open) {
      setFile(null);
      setNativeFile(null);
      setResult(null);
      setImporting(false);
    }
  }, [open]);

  const acceptNativeExcel = async (path: string) => {
    try {
      const info = (await Call.ByName(PROBE_EXCEL_BINDING, path)) as {
        fileName: string;
        fileSize: number;
      };
      setNativeFile({
        path,
        fileName: info.fileName,
        fileSize: info.fileSize,
      });
      setFile(null);
      setResult(null);
    } catch (error) {
      message.error(bindingErrorMessage(error));
    }
  };

  useEffect(() => {
    if (!open || !desktop) return;

    return Events.On(FILES_DROPPED_EVENT, (event) => {
      const payload = event.data as
        | { target?: string; files?: string[] }
        | undefined;
      if (payload?.target !== EXCEL_DROP_TARGET) return;
      const paths = Array.isArray(payload.files) ? payload.files : [];
      const path = paths.find(isExcelPath) ?? paths[0];
      if (!path) return;
      void acceptNativeExcel(path);
    });
  }, [desktop, message, open]);

  const chooseNativeExcel = async () => {
    try {
      const selected = await Dialogs.OpenFile({
        Title: t("import.title"),
        Message: t("import.dragHint"),
        Filters: [
          {
            DisplayName: "Excel Workbook",
            Pattern: "*.xlsx;*.xlsm",
          },
        ],
        AllowsMultipleSelection: false,
        CanChooseFiles: true,
        CanChooseDirectories: false,
      });
      const path = Array.isArray(selected) ? selected[0] : selected;
      if (path) await acceptNativeExcel(path);
    } catch (error) {
      if (isDialogCancelled(error)) return;
      message.error(bindingErrorMessage(error));
    }
  };

  const acceptFile = (candidate: File) => {
    const lower = candidate.name.toLowerCase();
    if (!lower.endsWith(".xlsx") && !lower.endsWith(".xlsm")) {
      message.error(t("import.fileTypeError"));
      return Upload.LIST_IGNORE;
    }
    if (candidate.size > 10 * 1024 * 1024) {
      message.error(t("import.fileTooLarge"));
      return Upload.LIST_IGNORE;
    }
    if (candidate.size <= 0) {
      message.error(t("import.fileEmpty"));
      return Upload.LIST_IGNORE;
    }
    setFile(candidate);
    setResult(null);
    return false;
  };

  const startImport = async () => {
    if (desktop) {
      if (!nativeFile) return;
      setImporting(true);
      try {
        const imported = (await Call.ByName(
          IMPORT_EXCEL_BINDING,
          nativeFile.path,
        )) as ImportResult;
        setResult(imported);
        if (imported.imported > 0) {
          message.success(t("toast.imported", { count: imported.imported }));
          onImported(imported);
        }
      } catch (error) {
        message.error(bindingErrorMessage(error));
      } finally {
        setImporting(false);
      }
      return;
    }

    if (!file) return;
    setImporting(true);
    try {
      const imported = await api.importExcel(file);
      setResult(imported);
      if (imported.imported > 0) {
        message.success(t("toast.imported", { count: imported.imported }));
        onImported(imported);
      }
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setImporting(false);
    }
  };

  const downloadTemplate = async () => {
    setDownloading(true);
    try {
      await api.downloadTemplate();
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setDownloading(false);
    }
  };

  const close = () => {
    if (importing) return;
    onClose();
  };

  const canImport = desktop ? Boolean(nativeFile) : Boolean(file);

  return (
    <Modal
      open={open}
      onCancel={close}
      footer={null}
      width={620}
      centered
      title={null}
      className="import-modal"
    >
      <div className="dialog-heading import-heading">
        <span className="dialog-eyebrow">
          <FileExcelOutlined /> {t("import.title")}
        </span>
        <h2>{t("import.title")}</h2>
        <p>{t("import.subtitle")}</p>
      </div>

      <div className="import-body">
        <div className="import-template-card">
          <div className="import-template-icon">
            <FileExcelOutlined />
          </div>
          <div className="import-template-copy">
            <strong>{t("import.template")}</strong>
            <span>{t("import.templateHint")}</span>
          </div>
          <Button
            onClick={() => void downloadTemplate()}
            loading={downloading}
            icon={<DownloadOutlined />}
          >
            {t("action.downloadTemplate")}
          </Button>
        </div>

        {result ? (
          <div className="import-result">
            <div className={`import-result-banner ${result.imported > 0 ? "is-success" : "is-warning"}`}>
              {result.imported > 0 ? <CheckCircleFilled /> : <WarningOutlined />}
              <div>
                <strong>{t("import.resultTitle")}</strong>
                <span>
                  {result.imported > 0
                    ? t("toast.imported", { count: result.imported })
                    : t("table.emptyDescription")}
                </span>
              </div>
            </div>

            <div className="import-result-stats">
              <div className="import-result-stat is-imported">
                <span>{t("import.imported")}</span>
                <strong>{result.imported}</strong>
              </div>
              <div className="import-result-stat is-skipped">
                <span>{t("import.skipped")}</span>
                <strong>{result.skipped}</strong>
              </div>
              <div className="import-result-stat">
                <span>{t("import.totalRows")}</span>
                <strong>{result.total}</strong>
              </div>
            </div>

            {result.errors.length > 0 && (
              <div className="import-error-box">
                <div className="import-error-title">{t("import.errors")}</div>
                <div className="import-error-list">
                  {result.errors.map((error) => (
                    <div key={error}>{error}</div>
                  ))}
                </div>
              </div>
            )}

            <div className="dialog-actions">
              <Button type="primary" onClick={close}>
                {t("action.done")}
              </Button>
            </div>
          </div>
        ) : (
          <>
            {desktop ? (
              <div
                data-file-drop-target={EXCEL_DROP_TARGET}
                className="import-native-target"
              >
                {nativeFile ? (
                  <div className="import-native-file">
                    <div className="import-file-card">
                      <span className="import-file-icon">
                        <FileExcelOutlined />
                      </span>
                      <div className="import-file-copy">
                        <strong>{nativeFile.fileName}</strong>
                        <span>
                          {t("import.selected")} ·{" "}
                          {formatBytes(nativeFile.fileSize)}
                        </span>
                      </div>
                      <button
                        type="button"
                        className="import-file-remove"
                        aria-label={t("action.remove")}
                        onClick={(event) => {
                          event.stopPropagation();
                          setNativeFile(null);
                        }}
                      >
                        <CloseOutlined />
                      </button>
                    </div>
                  </div>
                ) : (
                  <div
                    className="import-native-drop"
                    role="button"
                    tabIndex={0}
                    onClick={() => void chooseNativeExcel()}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        void chooseNativeExcel();
                      }
                    }}
                  >
                    <span className="import-drop-icon">
                      <InboxOutlined />
                    </span>
                    <strong>{t("import.drag")}</strong>
                    <span>{t("import.dragHint")}</span>
                  </div>
                )}
              </div>
            ) : (
              <Upload.Dragger
                accept=".xlsx,.xlsm"
                maxCount={1}
                showUploadList={false}
                beforeUpload={acceptFile}
                className="import-dragger"
                key={locale}
              >
                {file ? (
                  <div className="import-file-card">
                    <span className="import-file-icon">
                      <FileExcelOutlined />
                    </span>
                    <div className="import-file-copy">
                      <strong>{file.name}</strong>
                      <span>
                        {t("import.selected")} · {formatBytes(file.size)}
                      </span>
                    </div>
                    <button
                      type="button"
                      className="import-file-remove"
                      aria-label={t("action.remove")}
                      onClick={(event) => {
                        event.stopPropagation();
                        setFile(null);
                      }}
                    >
                      <CloseOutlined />
                    </button>
                  </div>
                ) : (
                  <div className="import-drop-content">
                    <span className="import-drop-icon">
                      <InboxOutlined />
                    </span>
                    <strong>{t("import.drag")}</strong>
                    <span>{t("import.dragHint")}</span>
                  </div>
                )}
              </Upload.Dragger>
            )}

            <div className="dialog-actions">
              <Button onClick={close}>{t("action.cancel")}</Button>
              <Button
                type="primary"
                icon={<CloudUploadOutlined />}
                disabled={!canImport}
                loading={importing}
                onClick={() => void startImport()}
              >
                {importing ? t("import.importing") : t("action.startImport")}
              </Button>
            </div>
          </>
        )}
      </div>
    </Modal>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
