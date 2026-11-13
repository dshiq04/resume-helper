import { Input, Select } from "antd";
import { EditOutlined, SearchOutlined } from "@ant-design/icons";
import type { Status, StatusOption } from "../../api/types";
import { useI18n } from "../../i18n";
import { StatusBadge } from "../badges";

interface DashboardToolbarProps {
  searchInput: string;
  statusFilter: Status | "all";
  statusOptions: StatusOption[];
  onSearchChange: (value: string) => void;
  onStatusChange: (value: Status | "all") => void;
}

export function DashboardToolbar({
  searchInput,
  statusFilter,
  statusOptions,
  onSearchChange,
  onStatusChange,
}: DashboardToolbarProps) {
  const { t } = useI18n();

  return (
    <div className="table-toolbar">
      <Input
        className="search-input"
        size="large"
        allowClear
        value={searchInput}
        onChange={(event) => onSearchChange(event.target.value)}
        prefix={<SearchOutlined className="toolbar-icon" />}
        placeholder={t("search.placeholder")}
        aria-label={t("search.placeholder")}
      />
      <Select
        className="status-filter"
        size="large"
        allowClear
        value={statusFilter === "all" ? undefined : statusFilter}
        placeholder={t("filter.allStatuses")}
        options={statusOptions}
        popupMatchSelectWidth={190}
        labelRender={({ value }) => <StatusBadge status={value as Status} />}
        optionRender={(option) => (
          <StatusBadge status={option.value as Status} />
        )}
        onChange={(value) => onStatusChange((value as Status) ?? "all")}
      />
      <div className="table-toolbar-spacer" />
      <span className="edit-hint">
        <EditOutlined />
        {t("table.editHint")}
      </span>
    </div>
  );
}
