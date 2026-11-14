import { useMemo } from "react";
import { Button, Dropdown, Popconfirm } from "antd";
import type { MenuProps } from "antd";
import {
  AppstoreOutlined,
  DeleteOutlined,
  SwapOutlined,
} from "@ant-design/icons";
import type { Category, Status, StatusOption } from "../../api/types";
import { useI18n } from "../../i18n";
import {
  CategoryBadge,
  StatusBadge,
  useCategoryOptions,
} from "../badges";

interface SelectionBarProps {
  count: number;
  statusOptions: StatusOption[];
  onBatchCategory: (category: Category) => void;
  onBatchStatus: (status: Status) => void;
  onBatchDelete: () => void;
  onClear: () => void;
}

export function SelectionBar({
  count,
  statusOptions,
  onBatchCategory,
  onBatchStatus,
  onBatchDelete,
  onClear,
}: SelectionBarProps) {
  const { t } = useI18n();
  const categoryOptions = useCategoryOptions();

  const categoryItems = useMemo<MenuProps["items"]>(
    () =>
      categoryOptions.map((option) => ({
        key: option.value,
        label: <CategoryBadge category={option.value} />,
      })),
    [categoryOptions],
  );

  const statusItems = useMemo<MenuProps["items"]>(
    () =>
      statusOptions.map((option) => ({
        key: option.value,
        label: <StatusBadge status={option.value} />,
      })),
    [statusOptions],
  );

  return (
    <div className="selection-bar">
      <div className="selection-info">
        <span className="selection-count">{count}</span>
        <span>{t("table.selectedCount", { count })}</span>
      </div>
      <div className="selection-actions">
        <Dropdown
          menu={{
            items: categoryItems,
            onClick: ({ key }) => onBatchCategory(key as Category),
          }}
          trigger={["click"]}
        >
          <Button icon={<AppstoreOutlined />}>
            {t("action.changeCategory")}
          </Button>
        </Dropdown>
        <Dropdown
          menu={{
            items: statusItems,
            onClick: ({ key }) => onBatchStatus(key as Status),
          }}
          trigger={["click"]}
        >
          <Button icon={<SwapOutlined />}>{t("action.changeStatus")}</Button>
        </Dropdown>
        <Popconfirm
          title={t("delete.batchTitle", { count })}
          description={t("delete.batchContent")}
          okText={t("action.delete")}
          cancelText={t("action.cancel")}
          okButtonProps={{ danger: true }}
          onConfirm={onBatchDelete}
        >
          <Button danger icon={<DeleteOutlined />}>
            {t("action.batchDelete")}
          </Button>
        </Popconfirm>
        <Button type="text" onClick={onClear}>
          {t("action.clearSelection")}
        </Button>
      </div>
    </div>
  );
}
