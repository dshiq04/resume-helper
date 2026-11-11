import { useEffect, useState } from "react";
import {
  App as AntdApp,
  Button,
  Form,
  Input,
  Modal,
  Select,
  Upload,
  type UploadFile,
  type UploadProps,
} from "antd";
import {
  DeleteOutlined,
  FileTextOutlined,
  InboxOutlined,
} from "@ant-design/icons";
import { Call, Events } from "@wailsio/runtime";
import { api } from "../../api/client";
import type { Resume } from "../../api/types";
import { useI18n } from "../../i18n";
import { apiErrorMessage } from "../../utils/errors";
import { isDesktopRuntime } from "../../utils/desktop";

const MAX_FILE_SIZE = 50 * 1024 * 1024;
const ALLOWED_EXTENSIONS = [
  "docx",
  "pdf",
  "pptx",
  "png",
  "md",
  "markdown",
];
const RESUME_FILES_DROPPED_EVENT = "resumetrail:resume-files-dropped";
const IMPORT_BINDING = "main.DropImporter.ImportResume";
const PROBE_BINDING = "main.DropImporter.ProbeResumeFile";

interface ResumeFormValues {
  name: string;
  tags?: string[];
}

interface DroppedResumeFile {
  path: string;
  fileName: string;
  fileType: string;
  fileSize: number;
}

interface ResumeFormModalProps {
  open: boolean;
  onClose: () => void;
  onCreated: (resume: Resume) => void;
}

function extensionOf(fileName: string): string {
  const index = fileName.lastIndexOf(".");
  if (index < 0) return "";
  return fileName.slice(index + 1).toLowerCase();
}

function baseName(fileName: string): string {
  const index = fileName.lastIndexOf(".");
  return (index > 0 ? fileName.slice(0, index) : fileName).trim();
}

function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 KB";
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function bindingErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) return error.message;
  if (typeof error === "string" && error) return error;
  return "未知错误";
}

export function ResumeFormModal({
  open,
  onClose,
  onCreated,
}: ResumeFormModalProps) {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [form] = Form.useForm<ResumeFormValues>();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [droppedFile, setDroppedFile] = useState<DroppedResumeFile | null>(null);
  const [nameTouched, setNameTouched] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue({ tags: [] });
    setFileList([]);
    setDroppedFile(null);
    setNameTouched(false);
    setSaving(false);
  }, [form, open]);

  // Wails delivers operating-system file drops as full paths, so the file is
  // imported through a Go binding instead of an HTML file input.
  useEffect(() => {
    if (!open || !isDesktopRuntime()) return;

    return Events.On(RESUME_FILES_DROPPED_EVENT, (event) => {
      const payload = event.data as string[] | undefined;
      const paths = Array.isArray(payload) ? payload : [];
      const path =
        paths.find((item) => ALLOWED_EXTENSIONS.includes(extensionOf(item))) ??
        paths[0];
      if (!path) return;

      void (async () => {
        try {
          const info = (await Call.ByName(PROBE_BINDING, path)) as {
            fileName: string;
            fileType: string;
            fileSize: number;
          };
          setDroppedFile({
            path,
            fileName: info.fileName,
            fileType: info.fileType,
            fileSize: info.fileSize,
          });
          setFileList([]);
          const currentName =
            (form.getFieldValue("name") as string | undefined) ?? "";
          if (!nameTouched || !currentName.trim()) {
            form.setFieldValue("name", baseName(info.fileName));
          }
        } catch (error) {
          message.error(bindingErrorMessage(error));
        }
      })();
    });
  }, [form, message, nameTouched, open]);

  const beforeUpload: UploadProps["beforeUpload"] = (file) => {
    const extension = extensionOf(file.name);
    if (!ALLOWED_EXTENSIONS.includes(extension)) {
      message.error(t("resumes.fileTypeError"));
      return Upload.LIST_IGNORE;
    }
    if (file.size > MAX_FILE_SIZE) {
      message.error(t("resumes.fileTooLarge"));
      return Upload.LIST_IGNORE;
    }
    // Wails desktop drops can hand the web view an empty File object even
    // though the real file exists on disk; the native binding above imports it.
    if (file.size <= 0) {
      if (!isDesktopRuntime()) {
        message.error(t("resumes.fileEmpty"));
      }
      return Upload.LIST_IGNORE;
    }
    return false;
  };

  const handleUploadChange: UploadProps["onChange"] = ({ fileList: next }) => {
    const limited = next.slice(-1);
    setFileList(limited);
    setDroppedFile(null);
    const raw = limited[0]?.originFileObj;
    if (!raw) return;
    const currentName = (form.getFieldValue("name") as string | undefined) ?? "";
    if (!nameTouched || !currentName.trim()) {
      form.setFieldValue("name", baseName(raw.name));
    }
  };

  const clearDroppedFile = () => {
    setDroppedFile(null);
  };

  const importDroppedFile = async (
    values: ResumeFormValues,
    dropped: DroppedResumeFile,
  ) => {
    const tags = (values.tags ?? [])
      .map((tag) => tag.trim())
      .filter((tag, index, list) => tag.length > 0 && list.indexOf(tag) === index);

    const created = (await Call.ByName(
      IMPORT_BINDING,
      dropped.path,
      values.name.trim(),
      tags,
    )) as Resume;
    message.success(t("resumes.created"));
    onCreated(created);
    onClose();
  };

  const handleSubmit = async () => {
    let values: ResumeFormValues;
    try {
      values = await form.validateFields();
    } catch {
      return;
    }

    if (droppedFile) {
      setSaving(true);
      try {
        await importDroppedFile(values, droppedFile);
      } catch (error) {
        message.error(bindingErrorMessage(error));
      } finally {
        setSaving(false);
      }
      return;
    }

    const raw = fileList[0]?.originFileObj as File | undefined;
    if (!raw) {
      message.warning(t("resumes.fileRequired"));
      return;
    }
    if (raw.size <= 0) {
      message.error(t("resumes.fileEmpty"));
      return;
    }

    const tags = (values.tags ?? [])
      .map((tag) => tag.trim())
      .filter((tag, index, list) => tag.length > 0 && list.indexOf(tag) === index);

    setSaving(true);
    try {
      const created = await api.createResume({
        name: values.name.trim(),
        tags,
        file: raw,
      });
      message.success(t("resumes.created"));
      onCreated(created);
      onClose();
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      open={open}
      title={
        <div className="resume-modal-title">
          <span>{t("resumes.addTitle")}</span>
          <small>{t("resumes.addSubtitle")}</small>
        </div>
      }
      className="resume-form-modal"
      width={580}
      okText={t("resumes.add")}
      cancelText={t("action.cancel")}
      confirmLoading={saving}
      onOk={() => void handleSubmit()}
      onCancel={() => {
        if (!saving) onClose();
      }}
      destroyOnHidden
      maskClosable={!saving}
    >
      <Form
        form={form}
        layout="vertical"
        requiredMark={false}
        className="resume-form"
      >
        <Form.Item
          name="name"
          label={t("resumes.name")}
          rules={[{ required: true, message: t("resumes.nameRequired") }]}
        >
          <Input
            maxLength={255}
            placeholder={t("resumes.namePlaceholder")}
            onChange={() => setNameTouched(true)}
          />
        </Form.Item>

        <Form.Item
          name="tags"
          label={t("resumes.tags")}
          extra={t("resumes.tagsHint")}
        >
          <Select
            mode="tags"
            tokenSeparators={[",", "，"]}
            placeholder={t("resumes.tagsPlaceholder")}
            suffixIcon={null}
            notFoundContent={null}
          />
        </Form.Item>

        <Form.Item label={t("resumes.file")} required>
          {droppedFile ? (
            <div className="resume-drop-file">
              <span className={`resume-drop-file-icon is-${droppedFile.fileType}`}>
                <FileTextOutlined />
              </span>
              <div className="resume-drop-file-info">
                <strong title={droppedFile.fileName}>{droppedFile.fileName}</strong>
                <span>
                  {droppedFile.fileType.toUpperCase()} ·{" "}
                  {formatFileSize(droppedFile.fileSize)}
                </span>
              </div>
              <Button
                type="text"
                danger
                size="small"
                icon={<DeleteOutlined />}
                aria-label={t("action.remove")}
                onClick={clearDroppedFile}
              >
                {t("action.remove")}
              </Button>
            </div>
          ) : (
            <div data-file-drop-target="resume-upload" className="resume-drop-target">
              <Upload.Dragger
                accept=".docx,.pdf,.pptx,.png,.md,.markdown"
                maxCount={1}
                multiple={false}
                fileList={fileList}
                beforeUpload={beforeUpload}
                onChange={handleUploadChange}
                onRemove={() => {
                  setFileList([]);
                  return true;
                }}
              >
                <p className="ant-upload-drag-icon">
                  <InboxOutlined />
                </p>
                <p className="ant-upload-text">{t("resumes.fileHint")}</p>
              </Upload.Dragger>
            </div>
          )}
        </Form.Item>
      </Form>
    </Modal>
  );
}
