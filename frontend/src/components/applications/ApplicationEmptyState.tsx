import { Button } from "antd";
import { InboxOutlined, PlusOutlined, SearchOutlined } from "@ant-design/icons";
import { useI18n } from "../../i18n";

interface ApplicationEmptyStateProps {
  hasFilters: boolean;
  onAdd: () => void;
}

export function ApplicationEmptyState({
  hasFilters,
  onAdd,
}: ApplicationEmptyStateProps) {
  const { t } = useI18n();

  return (
    <div className="table-empty">
      <span className="table-empty-icon">
        {hasFilters ? <SearchOutlined /> : <InboxOutlined />}
      </span>
      <h3>
        {hasFilters ? t("table.searchEmptyTitle") : t("table.emptyTitle")}
      </h3>
      <p>
        {hasFilters
          ? t("table.searchEmptyDescription")
          : t("table.emptyDescription")}
      </p>
      {!hasFilters && (
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={onAdd}
        >
          {t("table.addFirst")}
        </Button>
      )}
    </div>
  );
}
