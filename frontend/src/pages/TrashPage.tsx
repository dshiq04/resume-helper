import { useCallback, useEffect, useState } from "react";
import {
  App as AntdApp,
  Button,
  Empty,
  Popconfirm,
  Spin,
  Table,
  Tag,
  Tooltip,
} from "antd";
import type { TableColumnsType } from "antd";
import { DeleteOutlined, UndoOutlined } from "@ant-design/icons";
import { useOutletContext } from "react-router-dom";
import { api } from "../api/client";
import type { DeletedApplication } from "../api/types";
import { useI18n } from "../i18n";
import type { WorkspaceOutletContext } from "../layouts/WorkspaceLayout";
import { apiErrorMessage } from "../utils/errors";
import { CategoryBadge, StatusBadge } from "../components/badges";

export function TrashPage() {
  const { t, formatDate, formatFullDate, locale } = useI18n();
  const { message } = AntdApp.useApp();
  const { onDataChanged } = useOutletContext<WorkspaceOutletContext>();
  const [items, setItems] = useState<DeletedApplication[]>([]);
  const [loading, setLoading] = useState(true);
  const [restoringId, setRestoringId] = useState<number | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [emptying, setEmptying] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const result = await api.listTrash();
      setItems(result.items);
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setLoading(false);
    }
  }, [message, t]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    const refresh = () => void load();
    window.addEventListener("resumetrail:data-changed", refresh);
    return () => window.removeEventListener("resumetrail:data-changed", refresh);
  }, [load]);

  const restore = async (item: DeletedApplication) => {
    setRestoringId(item.id);
    try {
      await api.restoreTrash(item.id);
      message.success(t("trash.restoreSuccess"));
      setItems((current) => current.filter((row) => row.id !== item.id));
      onDataChanged();
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setRestoringId(null);
    }
  };

  const remove = async (item: DeletedApplication) => {
    setDeletingId(item.id);
    try {
      await api.deleteTrash(item.id);
      message.success(t("trash.deleteSuccess"));
      setItems((current) => current.filter((row) => row.id !== item.id));
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setDeletingId(null);
    }
  };

  const empty = async () => {
    setEmptying(true);
    try {
      await api.emptyTrash();
      message.success(t("trash.emptySuccess"));
      setItems([]);
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setEmptying(false);
    }
  };

  const columns: TableColumnsType<DeletedApplication> = [
    {
      title: t("table.company"),
      dataIndex: "companyName",
      key: "companyName",
      width: 260,
      render: (_value, record) => (
        <div className="company-copy">
          <div className="company-name-row">
            <span className="company-name-button is-static">{record.companyName}</span>
            {record.pinned && <Tag>{t("action.pin")}</Tag>}
          </div>
          <div className="company-meta">
            <CategoryBadge category={record.category} />
          </div>
        </div>
      ),
    },
    {
      title: t("table.link"),
      dataIndex: "link",
      key: "link",
      render: (value: string) =>
        value ? (
          <span className="link-text" title={value}>
            {value}
          </span>
        ) : (
          <span className="link-empty">{t("table.noLink")}</span>
        ),
    },
    {
      title: t("table.status"),
      dataIndex: "status",
      key: "status",
      width: 150,
      render: (value) => <StatusBadge status={value} />,
    },
    {
      title: t("trash.deletedAt"),
      dataIndex: "deletedAt",
      key: "deletedAt",
      width: 170,
      render: (value: string) => (
        <Tooltip title={formatFullDate(value)} placement="top">
          <span className="updated-at">{formatDate(value)}</span>
        </Tooltip>
      ),
    },
    {
      title: t("table.actions"),
      key: "actions",
      width: 220,
      align: "right",
      render: (_value, record) => (
        <div className="trash-actions">
          <Button
            type="primary"
            ghost
            size="small"
            icon={<UndoOutlined />}
            loading={restoringId === record.id}
            onClick={() => void restore(record)}
          >
            {t("trash.restore")}
          </Button>
          <Popconfirm
            title={t("trash.deleteTitle")}
            description={t("trash.deleteContent", {
              company: record.companyName,
            })}
            okText={t("action.delete")}
            cancelText={t("action.cancel")}
            okButtonProps={{ danger: true }}
            onConfirm={() => void remove(record)}
          >
            <Button
              danger
              type="text"
              size="small"
              icon={<DeleteOutlined />}
              loading={deletingId === record.id}
            >
              {t("action.delete")}
            </Button>
          </Popconfirm>
        </div>
      ),
    },
  ];

  return (
    <div className="trash-page">
      <header className="settings-page-header">
        <span className="season-pill">
          <DeleteOutlined /> {t("nav.trash")}
        </span>
        <h1>{t("trash.title")}</h1>
        <p>{t("trash.subtitle")}</p>
      </header>

      <section className="table-card trash-card">
        <div className="table-toolbar">
          <span className="resume-total">
            {t("trash.count", { count: items.length })}
          </span>
          <div className="table-toolbar-spacer" />
          <Popconfirm
            title={t("trash.emptyTitleConfirm")}
            description={t("trash.emptyContent")}
            okText={t("trash.empty")}
            cancelText={t("action.cancel")}
            okButtonProps={{ danger: true }}
            onConfirm={() => void empty()}
            disabled={items.length === 0}
          >
            <Button
              danger
              icon={<DeleteOutlined />}
              loading={emptying}
              disabled={items.length === 0}
            >
              {t("trash.empty")}
            </Button>
          </Popconfirm>
        </div>

        {loading ? (
          <div className="resume-loading">
            <Spin size="large" />
          </div>
        ) : items.length === 0 ? (
          <div className="resume-empty">
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                <div className="resume-empty-copy">
                  <strong>{t("trash.emptyTitle")}</strong>
                  <span>{t("trash.emptyDescription")}</span>
                </div>
              }
            />
          </div>
        ) : (
          <Table<DeletedApplication>
            key={locale}
            rowKey="id"
            columns={columns}
            dataSource={items}
            pagination={{ pageSize: 10, hideOnSinglePage: true }}
            className="applications-table trash-table"
          />
        )}
      </section>
    </div>
  );
}
