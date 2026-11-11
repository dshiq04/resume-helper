import { useRef, useState } from "react";
import {
  App as AntdApp,
  Button,
  Input,
  Modal,
  Segmented,
} from "antd";
import type { InputRef } from "antd";
import {
  BankOutlined,
  CheckOutlined,
  ClearOutlined,
  LinkOutlined,
  PushpinFilled,
  PushpinOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
import { api } from "../api/client";
import type { Application, CategoryFilter } from "../api/types";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";

interface QuickEntryModalProps {
  open: boolean;
  category: CategoryFilter;
  onClose: () => void;
  onCreated: (application: Application) => void;
}

export function QuickEntryModal({
  open,
  category,
  onClose,
  onCreated,
}: QuickEntryModalProps) {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const companyRef = useRef<InputRef>(null);
  const [companyName, setCompanyName] = useState("");
  const [link, setLink] = useState("");
  const [pinned, setPinned] = useState(false);
  const [saving, setSaving] = useState(false);

  const clear = () => {
    setCompanyName("");
    setLink("");
    setPinned(false);
  };

  const handleAdd = async () => {
    const company = companyName.trim();
    if (!company) {
      message.warning(t("form.companyRequired"));
      companyRef.current?.focus();
      return;
    }
    if (saving) return;

    setSaving(true);
    try {
      const created = await api.createApplication({
        category: category === "all" ? "spring" : category,
        companyName: company,
        link: link.trim(),
        status: "screening",
        pinned,
      });
      message.success(t("quick.added"));
      clear();
      onCreated(created);
      companyRef.current?.focus();
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      open={open}
      onCancel={onClose}
      footer={null}
      width={400}
      centered
      title={
        <span className="quick-entry-title">
          <ThunderboltOutlined />
          {t("quick.title")}
        </span>
      }
      className="quick-entry-modal"
      afterOpenChange={(visible) => {
        if (visible) companyRef.current?.focus();
      }}
    >
      <div className="quick-entry-body">
        <Segmented
          block
          size="small"
          value={pinned ? "pinned" : "normal"}
          onChange={(value) => setPinned(value === "pinned")}
          options={[
            {
              value: "normal",
              label: (
                <span className="quick-entry-pin-option">
                  <PushpinOutlined /> {t("quick.unpinned")}
                </span>
              ),
            },
            {
              value: "pinned",
              label: (
                <span className="quick-entry-pin-option">
                  <PushpinFilled /> {t("quick.pin")}
                </span>
              ),
            },
          ]}
        />

        <Input
          ref={companyRef}
          size="small"
          maxLength={255}
          value={companyName}
          prefix={<BankOutlined className="field-icon" />}
          placeholder={t("form.companyPlaceholder")}
          onChange={(event) => setCompanyName(event.target.value)}
          onPressEnter={() => void handleAdd()}
        />

        <Input
          size="small"
          maxLength={2048}
          value={link}
          prefix={<LinkOutlined className="field-icon" />}
          placeholder={t("form.linkPlaceholder")}
          onChange={(event) => setLink(event.target.value)}
          onPressEnter={() => void handleAdd()}
        />

        <div className="quick-entry-actions">
          <Button
            size="small"
            icon={<ClearOutlined />}
            disabled={saving}
            onClick={clear}
          >
            {t("action.clear")}
          </Button>
          <Button
            type="primary"
            size="small"
            icon={<CheckOutlined />}
            loading={saving}
            onClick={() => void handleAdd()}
          >
            {t("action.join")}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
