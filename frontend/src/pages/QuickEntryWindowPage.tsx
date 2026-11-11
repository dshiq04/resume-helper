import { useEffect, useRef, useState } from "react";
import { App as AntdApp, Button, Input, Segmented } from "antd";
import type { InputRef } from "antd";
import {
  BankOutlined,
  CheckOutlined,
  ClearOutlined,
  CloseOutlined,
  LinkOutlined,
  PushpinFilled,
  PushpinOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
import { Events } from "@wailsio/runtime";
import { api } from "../api/client";
import type { Category } from "../api/types";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";
import { closeWindow, isDesktopRuntime } from "../utils/desktop";

function readCategory(): Category {
  const value = new URLSearchParams(window.location.search).get("category");
  return value === "autumn" || value === "internship" ? value : "spring";
}

export function QuickEntryWindowPage() {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const companyRef = useRef<InputRef>(null);
  const [category] = useState<Category>(readCategory);
  const [companyName, setCompanyName] = useState("");
  const [link, setLink] = useState("");
  const [pinned, setPinned] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    companyRef.current?.focus();
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") void closeWindow();
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  const clear = () => {
    setCompanyName("");
    setLink("");
    setPinned(false);
    companyRef.current?.focus();
  };

  const notifyDataChanged = () => {
    window.dispatchEvent(new Event("resumetrail:data-changed"));
    if (isDesktopRuntime()) {
      void Events.Emit("resumetrail:data-changed").catch(() => {
        // Ignore: the main window may already be closing.
      });
    }
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
      await api.createApplication({
        category,
        companyName: company,
        link: link.trim(),
        status: "screening",
        pinned,
      });
      message.success(t("quick.added"));
      setCompanyName("");
      setLink("");
      setPinned(false);
      notifyDataChanged();
      companyRef.current?.focus();
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="quick-window-shell">
      <header className="quick-window-bar">
        <span className="quick-window-title">
          <ThunderboltOutlined />
          {t("quick.title")}
        </span>
        <button
          className="quick-window-close"
          type="button"
          aria-label={t("window.close")}
          title={t("window.close")}
          onClick={() => void closeWindow()}
        >
          <CloseOutlined />
        </button>
      </header>

      <div className="quick-window-body">
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
    </div>
  );
}
