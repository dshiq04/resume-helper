import { Table } from "antd";
import type { TableColumnsType } from "antd";
import type { Application, Status, StatusOption } from "../../api/types";
import type { ApplicationDraft } from "../../hooks/useApplicationDraft";
import type { InlineEditController } from "../../hooks/useInlineEdit";
import { useI18n } from "../../i18n";
import { CompanyCell } from "./CompanyCell";
import { LinkCell } from "./LinkCell";
import { ApplicationStatusSelect } from "./ApplicationStatusSelect";
import { ApplicationActions } from "./ApplicationActions";
import { ApplicationEmptyState } from "./ApplicationEmptyState";
import {
  DraftActions,
  DraftCompanyCell,
  DraftLinkCell,
  DraftStatusSelect,
} from "./ApplicationDraftFields";

const DRAFT_ID = -1;

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
  draft: ApplicationDraft | null;
  draftSaving: boolean;
  canChooseDraftCategory: boolean;
  onDraftChange: (patch: Partial<ApplicationDraft>) => void;
  onDraftSave: () => Promise<void>;
  onDraftCancel: () => void;
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
  draft,
  draftSaving,
  canChooseDraftCategory,
  onDraftChange,
  onDraftSave,
  onDraftCancel,
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
      width: 260,
      render: (_value, record) =>
        record.id === DRAFT_ID && draft ? (
          <DraftCompanyCell
            draft={draft}
            saving={draftSaving}
            canChooseCategory={canChooseDraftCategory}
            onChange={onDraftChange}
            onSave={onDraftSave}
            onCancel={onDraftCancel}
          />
        ) : (
          <CompanyCell application={record} inline={inline} />
        ),
    },
    {
      title: t("table.link"),
      dataIndex: "link",
      key: "link",
      width: 210,
      render: (_value, record) =>
        record.id === DRAFT_ID && draft ? (
          <DraftLinkCell
            draft={draft}
            saving={draftSaving}
            canChooseCategory={canChooseDraftCategory}
            onChange={onDraftChange}
            onSave={onDraftSave}
            onCancel={onDraftCancel}
          />
        ) : (
          <LinkCell application={record} inline={inline} />
        ),
    },
    {
      title: t("table.status"),
      dataIndex: "status",
      key: "status",
      width: 145,
      render: (_value, record) =>
        record.id === DRAFT_ID && draft ? (
          <DraftStatusSelect
            draft={draft}
            saving={draftSaving}
            canChooseCategory={canChooseDraftCategory}
            onChange={onDraftChange}
            onSave={onDraftSave}
            onCancel={onDraftCancel}
          />
        ) : (
          <ApplicationStatusSelect
            application={record}
            options={statusOptions}
            saving={statusSavingId === record.id}
            onChange={onStatusChange}
          />
        ),
    },
    {
      title: t("table.appliedAt"),
      dataIndex: "appliedAt",
      key: "appliedAt",
      width: 125,
      render: (value: string, record) =>
        record.id === DRAFT_ID ? (
          <span className="draft-unsaved">{t("draft.unsaved")}</span>
        ) : (
          <span className="updated-at">{formatDate(value)}</span>
        ),
    },
    {
      title: t("table.actions"),
      key: "actions",
      width: 86,
      align: "right",
      render: (_value, record) =>
        record.id === DRAFT_ID ? (
          <DraftActions
            saving={draftSaving}
            onSave={onDraftSave}
            onCancel={onDraftCancel}
          />
        ) : (
          <ApplicationActions
            application={record}
            pinSaving={pinSavingId === record.id}
            onPin={onPin}
            onDelete={onDelete}
          />
        ),
    },
  ];

  const draftRecord: Application | null = draft
    ? {
        id: DRAFT_ID,
        category: draft.category,
        companyName: draft.companyName,
        link: draft.link,
        status: draft.status,
        pinned: false,
        appliedAt: "",
        createdAt: "",
        updatedAt: "",
      }
    : null;
  const dataSource = draftRecord ? [draftRecord, ...items] : items;

  return (
    <Table<Application>
      key={locale}
      rowKey="id"
      columns={columns}
      dataSource={dataSource}
      loading={loading}
      className="applications-table"
      locale={{
        emptyText: (
          <ApplicationEmptyState hasFilters={hasFilters} onAdd={onAdd} />
        ),
      }}
      rowClassName={(record) =>
        record.id === DRAFT_ID
          ? "is-draft-row"
          : record.pinned
            ? "is-pinned-row"
            : ""
      }
      rowSelection={{
        selectedRowKeys: selectedKeys,
        getCheckboxProps: (record) => ({ disabled: record.id === DRAFT_ID }),
        onChange: (keys) => onSelectionChange(keys.map(Number)),
      }}
      pagination={{
        current: page,
        pageSize,
        total,
        showSizeChanger: true,
        showQuickJumper: true,
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
