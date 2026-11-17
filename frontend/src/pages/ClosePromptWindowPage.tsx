import { useState } from "react";
import { App as AntdApp, Button, Checkbox } from "antd";
import { CloseCircleOutlined, PoweroffOutlined, WindowsOutlined } from "@ant-design/icons";
import { api } from "../api/client";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";
import { closeWindow } from "../utils/desktop";

export function ClosePromptWindowPage() {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [submitting, setSubmitting] = useState<"tray" | "quit" | null>(null);
  const [remember, setRemember] = useState(false);

  const resolve = async (action: "tray" | "quit") => {
    if (submitting) return;
    setSubmitting(action);
    try {
      await api.resolveClosePrompt(action, remember);
    } catch (error) {
      message.error(apiErrorMessage(error, t));
      setSubmitting(null);
    }
  };

  return (
    <div className="shortcut-prompt-shell close-prompt-shell">
      <header className="quick-window-bar">
        <span className="quick-window-title">
          <CloseCircleOutlined />
          {t("close.title")}
        </span>
      </header>

      <div className="shortcut-prompt-body close-prompt-body">
        <p className="close-prompt-message">{t("close.message")}</p>

        <div className="close-option-grid">
          <button
            type="button"
            className="close-option"
            disabled={submitting !== null}
            onClick={() => void resolve("tray")}
          >
            <span className="close-option-icon is-tray">
              {submitting === "tray" ? <span className="close-option-spinner" /> : <WindowsOutlined />}
            </span>
            <strong>{t("close.tray")}</strong>
            <span>{t("close.trayHint")}</span>
          </button>

          <button
            type="button"
            className="close-option is-danger"
            disabled={submitting !== null}
            onClick={() => void resolve("quit")}
          >
            <span className="close-option-icon is-quit">
              {submitting === "quit" ? <span className="close-option-spinner" /> : <PoweroffOutlined />}
            </span>
            <strong>{t("close.quit")}</strong>
            <span>{t("close.quitHint")}</span>
          </button>
        </div>

        <div className="shortcut-prompt-remember">
          <Checkbox
            checked={remember}
            disabled={submitting !== null}
            onChange={(event) => setRemember(event.target.checked)}
          >
            {t("close.remember")}
          </Checkbox>
          <span>{t("close.rememberHint")}</span>
        </div>

        <Button
          type="text"
          size="small"
          className="close-prompt-cancel"
          disabled={submitting !== null}
          onClick={() => void closeWindow()}
        >
          {t("action.cancel")}
        </Button>
      </div>
    </div>
  );
}
