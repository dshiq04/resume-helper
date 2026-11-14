import { Button, Popconfirm, Tooltip } from "antd";
import {
  DeleteOutlined,
  EditOutlined,
  PushpinFilled,
  PushpinOutlined,
} from "@ant-design/icons";
import type { Application } from "../../api/types";
import { useI18n } from "../../i18n";

interface ApplicationActionsProps {
  application: Application;
  pinSaving: boolean;
  mobile?: boolean;
  onPin: (record: Application) => void;
  onEdit: (record: Application) => void;
  onDelete: (record: Application) => void;
}

export function ApplicationActions({
  application,
  pinSaving,
  mobile = false,
  onPin,
  onEdit,
  onDelete,
}: ApplicationActionsProps) {
  const { t } = useI18n();
  const pinLabel = t(application.pinned ? "action.unpin" : "action.pin");

  return (
    <div className="row-actions">
      <Tooltip title={pinLabel}>
        <Button
          type="text"
          size="small"
          className={`pin-button ${application.pinned ? "is-pinned" : ""} ${
            mobile ? "mobile-card-icon" : ""
          }`}
          aria-label={pinLabel}
          loading={pinSaving}
          icon={application.pinned ? <PushpinFilled /> : <PushpinOutlined />}
          onClick={() => onPin(application)}
        />
      </Tooltip>

      <Tooltip title={t("action.modify")}>
        <Button
          type="text"
          size="small"
          className={`row-edit-button ${mobile ? "mobile-card-icon" : ""}`}
          aria-label={t("action.modify")}
          icon={<EditOutlined />}
          onClick={() => onEdit(application)}
        />
      </Tooltip>

      <Popconfirm
        title={t("delete.title")}
        description={t("delete.content", { company: application.companyName })}
        okText={t("action.delete")}
        cancelText={t("action.cancel")}
        okButtonProps={{ danger: true }}
        onConfirm={() => onDelete(application)}
      >
        <Tooltip title={t("action.delete")}>
          <Button
            type="text"
            danger
            size="small"
            className={`row-delete-button ${mobile ? "mobile-card-icon" : ""}`}
            aria-label={t("action.delete")}
            icon={<DeleteOutlined />}
          />
        </Tooltip>
      </Popconfirm>
    </div>
  );
}
