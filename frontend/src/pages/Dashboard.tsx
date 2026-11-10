import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  App as AntdApp,
  Button,
  Dropdown,
  Input,
  Popconfirm,
  Select,
  Table,
  Tooltip,
} from "antd";
import type { MenuProps, TableColumnsType } from "antd";
import {
  DeleteOutlined,
  EditOutlined,
  ExportOutlined,
  FileExcelOutlined,
  InboxOutlined,
  LinkOutlined,
  PlusOutlined,
  SearchOutlined,
  SwapOutlined,
} from "@ant-design/icons";
import { api, ApiError } from "../api/client";
import type { Application, Status } from "../api/types";
import { useI18n, type MessageKey } from "../i18n";
import type { CategoryFilter } from "../components/Sidebar";
import { StatsCards } from "../components/StatsCards";
import { CategoryBadge, StatusBadge, useStatusOptions } from "../components/badges";
import { ApplicationModal } from "../components/ApplicationModal";
import { apiErrorMessage } from "../utils/errors";
import { openExternal } from "../utils/desktop";
import type { Stats } from "../api/types";

interface DashboardProps {
  category: CategoryFilter;
  stats: Stats | null;
  onDataChanged: () => void;
  onOpenImport?: () => void;
}

type EditingField = "companyName" | "link";

interface EditingState {
  id: number;
  field: EditingField;
}

export function Dashboard({
  category,
  stats,
  onDataChanged,
  onOpenImport,
}: DashboardProps) {
  const { t, formatDate, locale } = useI18n();
  const { message } = AntdApp.useApp();
  const statusOptions = useStatusOptions();

  const [items, setItems] = useState<Application[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<Status | "all">("all");
  const [loading, setLoading] = useState(true);
  const [selectedKeys, setSelectedKeys] = useState<number[]>([]);
  const [addOpen, setAddOpen] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);
  const [editing, setEditing] = useState<EditingState | null>(null);
  const [editValue, setEditValue] = useState("");
  const [inlineSaving, setInlineSaving] = useState(false);
  const [statusSavingId, setStatusSavingId] = useState<number | null>(null);

  const inlineCancelRef = useRef(false);
  const inlineSavingRef = useRef(false);

  const titleKey = `dashboard.title.${category}` as MessageKey;
  const subtitleKey = `dashboard.subtitle.${category}` as MessageKey;

  const refresh = useCallback(() => {
    setReloadToken((value) => value + 1);
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setSearch(searchInput.trim());
      setPage(1);
      setSelectedKeys([]);
    }, 320);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  useEffect(() => {
    setPage(1);
    setSelectedKeys([]);
  }, [category, statusFilter]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.listApplications({
        category,
        status: statusFilter,
        search,
        page,
        pageSize,
      });
      setItems(data.items);
      setTotal(data.total);
      if (data.items.length === 0 && data.total > 0 && page > 1) {
        setPage((current) => Math.max(1, current - 1));
      }
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        message.error(apiErrorMessage(error, t));
      }
    } finally {
      setLoading(false);
    }
  }, [category, message, page, pageSize, search, statusFilter, t]);

  useEffect(() => {
    void load();
  }, [load, reloadToken]);

  useEffect(() => {
    setSelectedKeys([]);
  }, [page, pageSize]);

  const startEdit = useCallback(
    (record: Application, field: EditingField) => {
      inlineCancelRef.current = false;
      setEditing({ id: record.id, field });
      setEditValue(record[field]);
    },
    [],
  );

  const commitInline = useCallback(async () => {
    if (!editing || inlineSavingRef.current) return;
    if (inlineCancelRef.current) {
      inlineCancelRef.current = false;
      return;
    }
    const record = items.find((item) => item.id === editing.id);
    if (!record) {
      setEditing(null);
      return;
    }

    const nextValue = editValue;
    const currentValue = record[editing.field];
    if (editing.field === "companyName" && !nextValue.trim()) {
      message.warning(t("error.companyRequired"));
      setEditing(null);
      return;
    }
    if (nextValue.trim() === currentValue) {
      setEditing(null);
      return;
    }

    inlineSavingRef.current = true;
    setInlineSaving(true);
    try {
      const updated = await api.updateApplication(record.id, {
        [editing.field]: nextValue,
      });
      setItems((current) =>
        current.map((item) => (item.id === updated.id ? updated : item)),
      );
      message.success(t("toast.updated"));
      onDataChanged();
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      inlineSavingRef.current = false;
      setInlineSaving(false);
      setEditing(null);
    }
  }, [editValue, editing, items, message, onDataChanged, t]);

  const handleStatusChange = useCallback(
    async (record: Application, status: Status) => {
      if (status === record.status) return;
      setStatusSavingId(record.id);
      try {
        const updated = await api.updateApplication(record.id, { status });
        setItems((current) =>
          current.map((item) => (item.id === updated.id ? updated : item)),
        );
        message.success(t("toast.updated"));
        onDataChanged();
      } catch (error) {
        message.error(apiErrorMessage(error, t));
      } finally {
        setStatusSavingId(null);
      }
    },
    [message, onDataChanged, t],
  );

  const handleDelete = useCallback(
    async (record: Application) => {
      try {
        await api.deleteApplication(record.id);
        message.success(t("toast.deleted"));
        setSelectedKeys((keys) => keys.filter((key) => key !== record.id));
        onDataChanged();
        if (items.length === 1 && page > 1) {
          setPage((current) => Math.max(1, current - 1));
        } else {
          refresh();
        }
      } catch (error) {
        message.error(apiErrorMessage(error, t));
      }
    },
    [items.length, message, onDataChanged, page, refresh, t],
  );

  const handleBatchDelete = useCallback(async () => {
    if (selectedKeys.length === 0) return;
    try {
      const result = await api.batchDelete(selectedKeys);
      message.success(
        t("toast.batchDeleted", { count: result.deleted }),
      );
      const deletingAll = selectedKeys.length === items.length;
      setSelectedKeys([]);
      onDataChanged();
      if (deletingAll && page > 1) {
        setPage((current) => Math.max(1, current - 1));
      } else {
        refresh();
      }
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    }
  }, [items.length, message, onDataChanged, page, refresh, selectedKeys, t]);

  const handleBatchStatus = useCallback(
    async (status: Status) => {
      if (selectedKeys.length === 0) return;
      try {
        const result = await api.batchStatus(selectedKeys, status);
        message.success(
          t("toast.statusUpdated", { count: result.updated }),
        );
        setSelectedKeys([]);
        onDataChanged();
        refresh();
      } catch (error) {
        message.error(apiErrorMessage(error, t));
      }
    },
    [message, onDataChanged, refresh, selectedKeys, t],
  );

  const handleCreated = useCallback(
    (created: Application) => {
      onDataChanged();
      if (category === "all" || category === created.category) {
        setPage(1);
        refresh();
      }
    },
    [category, onDataChanged, refresh],
  );

  const batchStatusItems = useMemo<MenuProps["items"]>(
    () =>
      statusOptions.map((option) => ({
        key: option.value,
        label: <StatusBadge status={option.value as Status} />,
      })),
    [statusOptions],
  );

  const editingId = editing?.id;
  const editingField = editing?.field;

  const columns: TableColumnsType<Application> = useMemo(
    () => [
      {
        title: t("table.company"),
        dataIndex: "companyName",
        key: "companyName",
        width: 300,
        render: (_value, record) => (
          <div className="company-cell">
            <span className={`company-avatar avatar-tone-${avatarTone(record.companyName)}`}>
              {companyInitial(record.companyName)}
            </span>
            <div className="company-copy">
              {editingId === record.id && editingField === "companyName" ? (
                <Input
                  autoFocus
                  size="small"
                  className="inline-edit-input"
                  value={editValue}
                  maxLength={255}
                  disabled={inlineSaving}
                  onChange={(event) => setEditValue(event.target.value)}
                  onBlur={() => void commitInline()}
                  onFocus={(event) => event.target.select()}
                  onPressEnter={(event) => event.currentTarget.blur()}
                  onKeyDown={(event) => {
                    if (event.key === "Escape") {
                      inlineCancelRef.current = true;
                      setEditing(null);
                    }
                  }}
                />
              ) : (
                <button
                  type="button"
                  className="company-name-button"
                  onClick={() => startEdit(record, "companyName")}
                >
                  {record.companyName}
                </button>
              )}
              <div className="company-meta">
                <CategoryBadge category={record.category} />
              </div>
            </div>
          </div>
        ),
      },
      {
        title: t("table.link"),
        dataIndex: "link",
        key: "link",
        width: 260,
        render: (_value, record) => {
          if (editingId === record.id && editingField === "link") {
            return (
              <Input
                autoFocus
                size="small"
                className="inline-edit-input"
                value={editValue}
                maxLength={2048}
                disabled={inlineSaving}
                placeholder={t("form.linkPlaceholder")}
                onChange={(event) => setEditValue(event.target.value)}
                onBlur={() => void commitInline()}
                onPressEnter={(event) => event.currentTarget.blur()}
                onKeyDown={(event) => {
                  if (event.key === "Escape") {
                    inlineCancelRef.current = true;
                    setEditing(null);
                  }
                }}
              />
            );
          }
          if (!record.link) {
            return (
              <button
                type="button"
                className="link-empty"
                onClick={() => startEdit(record, "link")}
              >
                <LinkOutlined />
                <span>{t("table.noLink")}</span>
              </button>
            );
          }
          return (
            <div className="link-cell">
              <button
                type="button"
                className="link-text"
                title={record.link}
                onClick={() => startEdit(record, "link")}
              >
                {prettyLink(record.link)}
              </button>
              <Tooltip title={t("action.openLink")}>
                <button
                  type="button"
                  className="link-open-button"
                  aria-label={t("action.openLink")}
                  onClick={(event) => {
                    event.stopPropagation();
                    void openExternal(record.link);
                  }}
                >
                  <ExportOutlined />
                </button>
              </Tooltip>
            </div>
          );
        },
      },
      {
        title: t("table.status"),
        dataIndex: "status",
        key: "status",
        width: 158,
        render: (_value, record) => (
          <Select
            value={record.status}
            className="status-select"
            variant="borderless"
            aria-label={t("table.status")}
            options={statusOptions}
            loading={statusSavingId === record.id}
            popupMatchSelectWidth={190}
            labelRender={({ value }) => <StatusBadge status={value as Status} />}
            optionRender={(option) => (
              <StatusBadge status={option.value as Status} />
            )}
            onChange={(value) => void handleStatusChange(record, value as Status)}
          />
        ),
      },
      {
        title: t("table.updatedAt"),
        dataIndex: "updatedAt",
        key: "updatedAt",
        width: 150,
        responsive: ["lg"],
        render: (value: string) => (
          <span className="updated-at">{formatDate(value)}</span>
        ),
      },
      {
        title: t("table.actions"),
        key: "actions",
        width: 64,
        align: "right",
        render: (_value, record) => (
          <Popconfirm
            title={t("delete.title")}
            description={t("delete.content", { company: record.companyName })}
            okText={t("action.delete")}
            cancelText={t("action.cancel")}
            okButtonProps={{ danger: true }}
            onConfirm={() => void handleDelete(record)}
          >
            <Tooltip title={t("action.delete")}>
              <Button
                type="text"
                danger
                size="small"
                className="row-delete-button"
                icon={<DeleteOutlined />}
              />
            </Tooltip>
          </Popconfirm>
        ),
      },
    ],
    [
      commitInline,
      editValue,
      editingField,
      editingId,
      formatDate,
      handleDelete,
      handleStatusChange,
      inlineSaving,
      statusOptions,
      statusSavingId,
      t,
    ],
  );

  const hasFilters = Boolean(search) || statusFilter !== "all";

  const emptyState = (
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
          onClick={() => setAddOpen(true)}
        >
          {t("table.addFirst")}
        </Button>
      )}
    </div>
  );

  return (
    <>
      <div className="dashboard-header">
        <div>
          <span className="season-pill">
            {t("dashboard.season", { year: new Date().getFullYear() })}
          </span>
          <h1>{t(titleKey)}</h1>
          <p>{t(subtitleKey)}</p>
        </div>
        <div className="dashboard-actions">
          {onOpenImport && (
            <Button
              className="soft-action-button"
              size="large"
              icon={<FileExcelOutlined />}
              onClick={onOpenImport}
            >
              {t("action.import")}
            </Button>
          )}
          <Button
            type="primary"
            size="large"
            className="primary-action-button"
            icon={<PlusOutlined />}
            onClick={() => setAddOpen(true)}
          >
            {t("action.add")}
          </Button>
        </div>
      </div>

      <StatsCards stats={stats} category={category} />

      <section className="table-card">
        {selectedKeys.length > 0 && (
          <div className="selection-bar">
            <div className="selection-info">
              <span className="selection-count">{selectedKeys.length}</span>
              <span>{t("table.selectedCount", { count: selectedKeys.length })}</span>
            </div>
            <div className="selection-actions">
              <Dropdown
                menu={{
                  items: batchStatusItems,
                  onClick: ({ key }) => void handleBatchStatus(key as Status),
                }}
                trigger={["click"]}
              >
                <Button icon={<SwapOutlined />}>{t("action.changeStatus")}</Button>
              </Dropdown>
              <Popconfirm
                title={t("delete.batchTitle", { count: selectedKeys.length })}
                description={t("delete.batchContent")}
                okText={t("action.delete")}
                cancelText={t("action.cancel")}
                okButtonProps={{ danger: true }}
                onConfirm={() => void handleBatchDelete()}
              >
                <Button danger icon={<DeleteOutlined />}>
                  {t("action.batchDelete")}
                </Button>
              </Popconfirm>
              <Button type="text" onClick={() => setSelectedKeys([])}>
                {t("action.clearSelection")}
              </Button>
            </div>
          </div>
        )}

        <div className="table-toolbar">
          <Input
            className="search-input"
            size="large"
            allowClear
            value={searchInput}
            onChange={(event) => setSearchInput(event.target.value)}
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
            onChange={(value) => setStatusFilter((value as Status) ?? "all")}
          />
          <div className="table-toolbar-spacer" />
          <span className="edit-hint">
            <EditOutlined />
            {t("table.editHint")}
          </span>
        </div>

        <Table<Application>
          key={locale}
          rowKey="id"
          columns={columns}
          dataSource={items}
          loading={loading}
          className="applications-table"
          locale={{ emptyText: emptyState }}
          scroll={{ x: 820 }}
          rowSelection={{
            selectedRowKeys: selectedKeys,
            onChange: (keys) => setSelectedKeys(keys.map(Number)),
          }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50],
            showTotal: (count) => t("table.totalItems", { count }),
            onChange: (nextPage, nextPageSize) => {
              setPage(nextPageSize !== pageSize ? 1 : nextPage);
              setPageSize(nextPageSize);
            },
          }}
        />
      </section>

      <ApplicationModal
        open={addOpen}
        defaultCategory={category}
        onClose={() => setAddOpen(false)}
        onCreated={handleCreated}
      />
    </>
  );
}

function companyInitial(name: string): string {
  return Array.from(name.trim())[0]?.toUpperCase() ?? "?";
}

function avatarTone(name: string): number {
  let hash = 0;
  for (const character of name) {
    hash = (hash * 31 + (character.codePointAt(0) ?? 0)) >>> 0;
  }
  return hash % 6;
}

function prettyLink(link: string): string {
  try {
    const url = new URL(link);
    const host = url.hostname.replace(/^www\./, "");
    const path = url.pathname === "/" ? "" : url.pathname;
    const display = `${host}${path}`;
    return display.length > 48 ? `${display.slice(0, 45)}...` : display;
  } catch {
    return link;
  }
}
