import { useEffect, useState } from "react";
import {
  App as AntdApp,
  Form,
  Input,
  Modal,
  Select,
  Upload,
  type UploadFile,
  type UploadProps,
} from "antd";
import { InboxOutlined } from "@ant-design/icons";
import { api } from "../../api/client";
import type { Resume } from "../../api/types";
import { useI18n } from "../../i18n";
import { apiErrorMessage } from "../../utils/errors";

const MAX_FILE_SIZE = 50 * 1024 * 1024;
const ALLOWED_EXTENSIONS = [
  "docx",
  "pdf",
  "pptx",
  "png",
  "md",
  "markdown",
];

interface ResumeFormValues {
  name: string;
  tags?: string[];
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

export function ResumeFormModal({
  open,
  onClose,
  onCreated,
}: ResumeFormModalProps) {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [form] = Form.useForm<ResumeFormValues>();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [nameTouched, setNameTouched] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue({ tags: [] });
    setFileList([]);
    setNameTouched(false);
    setSaving(false);
  }, [form, open]);

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
    if (file.size <= 0) {
      message.error(t("resumes.fileEmpty"));
      return Upload.LIST_IGNORE;
    }
    return false;
  };

  const handleUploadChange: UploadProps["onChange"] = ({ fileList: next }) => {
    const limited = next.slice(-1);
    setFileList(limited);
    const raw = limited[0]?.originFileObj;
    if (!raw) return;
    const currentName = (form.getFieldValue("name") as string | undefined) ?? "";
    if (!nameTouched || !currentName.trim()) {
      form.setFieldValue("name", baseName(raw.name));
    }
  };

  const handleSubmit = async () => {
    const raw = fileList[0]?.originFileObj as File | undefined;
    if (!raw) {
      message.warning(t("resumes.fileRequired"));
      return;
    }
    if (raw.size <= 0) {
      message.error(t("resumes.fileEmpty"));
      return;
    }

    let values: ResumeFormValues;
    try {
      values = await form.validateFields();
    } catch {
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
        </Form.Item>
      </Form>
    </Modal>
  );
}
