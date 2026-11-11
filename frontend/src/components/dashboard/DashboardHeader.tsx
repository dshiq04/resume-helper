import { Button } from "antd";
import {
  FileExcelOutlined,
  PlusOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
import type { CategoryFilter } from "../../api/types";
import { useI18n, type MessageKey } from "../../i18n";

interface DashboardHeaderProps {
  category: CategoryFilter;
  onAdd: () => void;
  onImport: () => void;
  onQuickAdd: () => void;
}

export function DashboardHeader({
  category,
  onAdd,
  onImport,
  onQuickAdd,
}: DashboardHeaderProps) {
  const { t } = useI18n();
  const titleKey = `dashboard.title.${category}` as MessageKey;
  const subtitleKey = `dashboard.subtitle.${category}` as MessageKey;

  return (
    <div className="dashboard-header">
      <div>
        <span className="season-pill">
          {t("dashboard.season", { year: new Date().getFullYear() })}
        </span>
        <h1>{t(titleKey)}</h1>
        <p>{t(subtitleKey)}</p>
      </div>
      <div className="dashboard-actions">
        <Button
          className="soft-action-button"
          size="large"
          icon={<ThunderboltOutlined />}
          onClick={onQuickAdd}
        >
          {t("action.quickAdd")}
        </Button>
        <Button
          className="soft-action-button"
          size="large"
          icon={<FileExcelOutlined />}
          onClick={onImport}
        >
          {t("action.import")}
        </Button>
        <Button
          type="primary"
          size="large"
          className="primary-action-button"
          icon={<PlusOutlined />}
          onClick={onAdd}
        >
          {t("action.add")}
        </Button>
      </div>
    </div>
  );
}
