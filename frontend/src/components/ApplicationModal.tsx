import { useEffect, useState } from "react";
import { App as AntdApp, Button, Form, Input, Modal, Segmented, Select } from "antd";
import {
  BankOutlined,
  LinkOutlined,
  PlusOutlined,
  TagOutlined,
} from "@ant-design/icons";
import { api } from "../api/client";
import type {
  Application,
  Category,
  CategoryFilter,
  Status,
} from "../api/types";
import { useCategoryOptions, useStatusOptions, StatusBadge } from "./badges";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";

interface ApplicationModalProps {
  open: boolean;
  defaultCategory: CategoryFilter;
  onClose: () => void;
  onCreated: (application: Application) => void;
}

interface FormValues {
  category: Category;
  companyName: string;
  link: string;
  status: Status;
}

export function ApplicationModal({
  open,
  defaultCategory,
  onClose,
  onCreated,
}: ApplicationModalProps) {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [form] = Form.useForm<FormValues>();
  const [submitting, setSubmitting] = useState(false);
  const categoryOptions = useCategoryOptions();
  const statusOptions = useStatusOptions();

  useEffect(() => {
    if (!open) return;
    form.setFieldsValue({
      category: defaultCategory === "all" ? "spring" : defaultCategory,
      companyName: "",
      link: "",
      status: "screening",
    });
  }, [defaultCategory, form, open]);

  const handleFinish = async (values: FormValues) => {
    setSubmitting(true);
    try {
      const created = await api.createApplication({
        category: values.category,
        companyName: values.companyName.trim(),
        link: values.link?.trim() ?? "",
        status: values.status,
      });
      message.success(t("toast.created"));
      form.resetFields();
      onCreated(created);
      onClose();
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      open={open}
      onCancel={onClose}
      footer={null}
      width={540}
      className="application-modal"
      centered
      title={null}
    >
      <div className="dialog-heading">
        <span className="dialog-eyebrow">
          <PlusOutlined /> {t("modal.addTitle")}
        </span>
        <h2>{t("modal.addTitle")}</h2>
        <p>{t("modal.addSubtitle")}</p>
      </div>

      <Form
        form={form}
        layout="vertical"
        requiredMark={false}
        onFinish={handleFinish}
        className="application-form"
      >
        <Form.Item
          name="category"
          label={t("form.category")}
          rules={[{ required: true }]}
        >
          <Segmented block options={categoryOptions} />
        </Form.Item>

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
            size="large"
            autoFocus
            maxLength={255}
            prefix={<BankOutlined className="field-icon" />}
            placeholder={t("form.companyPlaceholder")}
          />
        </Form.Item>

        <Form.Item name="link" label={t("form.link")}>
          <Input
            size="large"
            maxLength={2048}
            prefix={<LinkOutlined className="field-icon" />}
            placeholder={t("form.linkPlaceholder")}
          />
        </Form.Item>
        <div className="form-hint">{t("form.linkHint")}</div>

        <Form.Item name="status" label={t("form.status")} className="status-field">
          <Select
            size="large"
            options={statusOptions}
            labelRender={({ value }) => <StatusBadge status={value as Status} />}
            optionRender={(option) => <StatusBadge status={option.value as Status} />}
            suffixIcon={<TagOutlined className="field-icon" />}
          />
        </Form.Item>

        <div className="dialog-actions">
          <Button onClick={onClose}>{t("action.cancel")}</Button>
          <Button
            type="primary"
            loading={submitting}
            onClick={() => form.submit()}
            icon={<PlusOutlined />}
          >
            {t("form.add")}
          </Button>
        </div>
      </Form>
    </Modal>
  );
}
