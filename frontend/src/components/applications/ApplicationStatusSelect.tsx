import { Select } from "antd";
import type { Application, Status, StatusOption } from "../../api/types";
import { useI18n } from "../../i18n";
import { StatusBadge } from "../badges";

interface ApplicationStatusSelectProps {
  application: Application;
  options: StatusOption[];
  saving: boolean;
  mobile?: boolean;
  onChange: (record: Application, status: Status) => void;
}

export function ApplicationStatusSelect({
  application,
  options,
  saving,
  mobile = false,
  onChange,
}: ApplicationStatusSelectProps) {
  const { t } = useI18n();

  return (
    <Select
      value={application.status}
      className={`status-select ${mobile ? "mobile-status-select" : ""}`}
      variant="borderless"
      aria-label={t("table.status")}
      options={options}
      loading={saving}
      popupMatchSelectWidth={190}
      labelRender={({ value }) => <StatusBadge status={value as Status} />}
      optionRender={(option) => <StatusBadge status={option.value as Status} />}
      onChange={(value) => onChange(application, value as Status)}
    />
  );
}
