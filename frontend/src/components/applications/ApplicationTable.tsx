import { Table } from "antd";
import type { TableColumnsType } from "antd";
import type { Application, Status, StatusOption } from "../../api/types";
import type { InlineEditController } from "../../hooks/useInlineEdit";
import { useI18n } from "../../i18n";
import { CompanyCell } from "./CompanyCell";
import { LinkCell } from "./LinkCell";
import { ApplicationStatusSelect } from "./ApplicationStatusSelect";
import { ApplicationActions } from "./ApplicationActions";
import { ApplicationEmptyState } from "./ApplicationEmptyState";

interface ApplicationTableProps {
  items: Application[];
  total: number;
  page: number;
  pageSize: number;
  loading: boolean;
  selectedKeys: number[];
  statusSavingId: number | null;
  pinSavingId: number | null;
  statusOptions: StatusOption[];
  inline: InlineEditController;
  hasFilters: boolean;
  onSelectionChange: (keys: number[]) => void;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  onStatusChange: (record: Application, status: Status) => void;
  onPin: (record: Application) => void;
  onDelete: (record: Application) => void;
  onAdd: () => void;
}

export function ApplicationTable({
  items,
  total,
  page,
  pageSize,
  loading,
  selectedKeys,
  statusSavingId,
  pinSavingId,
  statusOptions,
  inline,
  hasFilters,
  onSelectionChange,
  onPageChange,
  onPageSizeChange,
  onStatusChange,
  onPin,
  onDelete,
  onAdd,
}: ApplicationTableProps) {
  const { t, formatDate, locale } = useI18n();

  const columns: TableColumnsType<Application> = [
    {
      title: t("table.company"),
      dataIndex: "companyName",
      key: "companyName",
      width: 280,
      render: (_value, record) => (
        <CompanyCell application={record} inline={inline} />
      ),
    },
    {
      title: t("table.link"),
      dataIndex: "link",
      key: "link",
      width: 230,
      render: (_value, record) => (
        <LinkCell application={record} inline={inline} />
      ),
    },
    {
      title: t("table.status"),
      dataIndex: "status",
      key: "status",
      width: 155,
      render: (_value, record) => (
        <ApplicationStatusSelect
          application={record}
          options={statusOptions}
          saving={statusSavingId === record.id}
          onChange={onStatusChange}
        />
      ),
    },
    {
      title: t("table.updatedAt"),
      dataIndex: "updatedAt",
      key: "updatedAt",
      width: 140,
      render: (value: string) => (
        <span className="updated-at">{formatDate(value)}</span>
      ),
    },
    {
      title: t("table.actions"),
      key: "actions",
      width: 92,
      align: "right",
      render: (_value, record) => (
        <ApplicationActions
          application={record}
          pinSaving={pinSavingId === record.id}
          onPin={onPin}
          onDelete={onDelete}
        />
      ),
    },
  ];

  return (
    <Table<Application>
      key={locale}
      rowKey="id"
      columns={columns}
      dataSource={items}
      loading={loading}
      className="applications-table"
      locale={{
        emptyText: (
          <ApplicationEmptyState hasFilters={hasFilters} onAdd={onAdd} />
        ),
      }}
      scroll={{ x: 820 }}
      rowClassName={(record) => (record.pinned ? "is-pinned-row" : "")}
      rowSelection={{
        selectedRowKeys: selectedKeys,
        onChange: (keys) => onSelectionChange(keys.map(Number)),
      }}
      pagination={{
        current: page,
        pageSize,
        total,
        showSizeChanger: true,
        pageSizeOptions: [10, 20, 50],
        showTotal: (count) => t("table.totalItems", { count }),
        onChange: (nextPage, nextPageSize) => {
          if (nextPageSize !== pageSize) {
            onPageSizeChange(nextPageSize);
            onPageChange(1);
          } else {
            onPageChange(nextPage);
          }
        },
      }}
    />
  );
}
