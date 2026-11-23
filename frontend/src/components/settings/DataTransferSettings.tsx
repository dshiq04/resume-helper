import { useState } from "react";
import { App as AntdApp, Button } from "antd";
import {
  CloudDownloadOutlined,
  CloudUploadOutlined,
  SwapOutlined,
} from "@ant-design/icons";
import { api } from "../../api/client";
import { useI18n } from "../../i18n";
import { apiErrorMessage } from "../../utils/errors";
import { SettingsCard } from "./SettingsCard";

export function DataTransferSettings() {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [opening, setOpening] = useState<"export" | "import" | null>(null);

  const open = async (mode: "export" | "import") => {
    setOpening(mode);
    try {
      await api.openDataTransfer(mode);
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setOpening(null);
    }
  };

  return (
    <SettingsCard
      icon={<SwapOutlined />}
      tone="data"
      title={t("settings.dataTransfer")}
      description={t("settings.dataTransferHint")}
    >
      <div className="data-transfer-actions">
        <Button
          type="primary"
          icon={<CloudDownloadOutlined />}
          loading={opening === "export"}
          onClick={() => void open("export")}
        >
          {t("settings.exportData")}
        </Button>
        <Button
          icon={<CloudUploadOutlined />}
          loading={opening === "import"}
          onClick={() => void open("import")}
        >
          {t("settings.importData")}
        </Button>
      </div>
      <p className="settings-field-hint">{t("transfer.deviceNote")}</p>
    </SettingsCard>
  );
}
