import { useState } from "react";
import { App as AntdApp, Button } from "antd";
import { DesktopOutlined } from "@ant-design/icons";
import { api } from "../api/client";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";

export function ShortcutPromptWindowPage() {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [submitting, setSubmitting] = useState<"yes" | "no" | null>(null);

  const resolve = async (create: boolean) => {
    if (submitting) return;
    setSubmitting(create ? "yes" : "no");
    try {
      await api.resolveShortcutPrompt(create);
    } catch (error) {
      message.error(apiErrorMessage(error, t));
      setSubmitting(null);
    }
  };

  return (
    <div className="shortcut-prompt-shell">
      <header className="quick-window-bar">
        <span className="quick-window-title">
          <DesktopOutlined />
          {t("shortcut.promptTitle")}
        </span>
      </header>

      <div className="shortcut-prompt-body">
        <p>{t("shortcut.promptMessage")}</p>
        <div className="shortcut-prompt-actions">
          <Button
            disabled={submitting !== null}
            loading={submitting === "no"}
            onClick={() => void resolve(false)}
          >
            {t("shortcut.promptNo")}
          </Button>
          <Button
            type="primary"
            disabled={submitting !== null}
            loading={submitting === "yes"}
            onClick={() => void resolve(true)}
          >
            {t("shortcut.promptYes")}
          </Button>
        </div>
      </div>
    </div>
  );
}
