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
import { api } from "../api/client";
import type { ImportResult } from "../api/types";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";

interface ImportModalProps {
  open: boolean;
  onClose: () => void;
  onImported: (result: ImportResult) => void;
}

export function ImportModal({ open, onClose, onImported }: ImportModalProps) {
  const { t, locale } = useI18n();
  const { message } = AntdApp.useApp();
  const [file, setFile] = useState<File | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [importing, setImporting] = useState(false);
  const [downloading, setDownloading] = useState(false);

  useEffect(() => {
    if (open) {
      setFile(null);
      setResult(null);
      setImporting(false);
    }
  }, [open]);

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
    setFile(candidate);
    setResult(null);
    return false;
  };

  const startImport = async () => {
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

            <div className="dialog-actions">
              <Button onClick={close}>{t("action.cancel")}</Button>
              <Button
                type="primary"
                icon={<CloudUploadOutlined />}
                disabled={!file}
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
