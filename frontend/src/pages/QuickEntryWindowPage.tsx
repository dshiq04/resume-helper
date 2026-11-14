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
import { Events, Window } from "@wailsio/runtime";
import { api } from "../api/client";
import { CATEGORIES, type Category } from "../api/types";
import { useI18n, type MessageKey } from "../i18n";
import { apiErrorMessage } from "../utils/errors";
import { closeWindow, isDesktopRuntime } from "../utils/desktop";

function readCategory(): Category | null {
  const value = new URLSearchParams(window.location.search).get("category");
  return value && (CATEGORIES as string[]).includes(value)
    ? (value as Category)
    : null;
}

export function QuickEntryWindowPage() {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const companyRef = useRef<InputRef>(null);
  const savingRef = useRef(false);
  const [selectedCategory, setSelectedCategory] =
    useState<Category | null>(readCategory);
  const [defaultCategory, setDefaultCategory] = useState<Category>("spring");
  const [companyName, setCompanyName] = useState("");
  const [link, setLink] = useState("");
  const [saving, setSaving] = useState(false);
  const [windowPinned, setWindowPinned] = useState(true);

  useEffect(() => {
    companyRef.current?.focus();
    if (isDesktopRuntime()) {
      void Window.SetAlwaysOnTop(true).catch(() => {
        // Ignore: window pinning is a desktop-only convenience.
      });
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") void closeWindow();
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  // The default season comes from the server clock so every client agrees.
  useEffect(() => {
    let active = true;
    void api
      .quickEntryDefaults()
      .then((defaults) => {
        if (!active) return;
        setDefaultCategory(defaults.defaultCategory);
        setSelectedCategory((current) => current ?? defaults.defaultCategory);
      })
      .catch(() => {
        if (!active) return;
        setSelectedCategory((current) => current ?? "spring");
      });
    return () => {
      active = false;
    };
  }, []);

  const toggleWindowPin = async () => {
    const next = !windowPinned;
    setWindowPinned(next);
    if (!isDesktopRuntime()) return;
    try {
      await Window.SetAlwaysOnTop(next);
    } catch {
      setWindowPinned(!next);
    }
  };

  const clear = () => {
    setCompanyName("");
    setLink("");
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
    // Guard against synchronous double clicks / repeated Enter presses before
    // React has a chance to re-render the loading state.
    if (savingRef.current) return;
    savingRef.current = true;

    setSaving(true);
    try {
      await api.createApplication({
        category: selectedCategory ?? defaultCategory,
        companyName: company,
        link: link.trim(),
        status: "screening",
      });
      message.success(t("quick.added"));
      setCompanyName("");
      setLink("");
      notifyDataChanged();
      companyRef.current?.focus();
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      savingRef.current = false;
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
        <div className="quick-window-actions">
          <button
            className={`quick-window-pin ${windowPinned ? "is-on" : ""}`}
            type="button"
            aria-label={
              windowPinned ? t("quick.windowUnpin") : t("quick.windowPin")
            }
            title={windowPinned ? t("quick.windowUnpin") : t("quick.windowPin")}
            onClick={() => void toggleWindowPin()}
          >
            {windowPinned ? <PushpinFilled /> : <PushpinOutlined />}
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

      <div className="quick-window-body">
        <Input
          ref={companyRef}
          size="small"
          maxLength={255}
          value={companyName}
          allowClear
          prefix={<BankOutlined className="field-icon" />}
          placeholder={t("form.companyPlaceholder")}
          onChange={(event) => setCompanyName(event.target.value)}
          onPressEnter={() => void handleAdd()}
        />

        <Input
          size="small"
          maxLength={2048}
          value={link}
          allowClear
          prefix={<LinkOutlined className="field-icon" />}
          placeholder={t("form.linkPlaceholder")}
          onChange={(event) => setLink(event.target.value)}
          onPressEnter={() => void handleAdd()}
        />

        <div className="quick-entry-actions">
          <Segmented
            className="quick-entry-category"
            size="small"
            aria-label={t("quick.category")}
            title={t("quick.category")}
            value={selectedCategory ?? defaultCategory}
            options={CATEGORIES.map((value) => ({
              value,
              label: t(`category.${value}` as MessageKey),
            }))}
            onChange={(value) => setSelectedCategory(value as Category)}
          />
          <div className="quick-entry-buttons">
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
    </div>
  );
}
