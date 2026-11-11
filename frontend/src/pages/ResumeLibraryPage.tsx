import { useCallback, useEffect, useMemo, useState } from "react";
import {
  App as AntdApp,
  Button,
  Card,
  Empty,
  Input,
  Popconfirm,
  Spin,
  Tag,
  Tooltip,
} from "antd";
import {
  DeleteOutlined,
  EyeOutlined,
  FileImageOutlined,
  FileMarkdownOutlined,
  FileOutlined,
  FilePdfOutlined,
  FilePptOutlined,
  FileTextOutlined,
  FileWordOutlined,
  PlusOutlined,
  PushpinFilled,
  PushpinOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import { useOutletContext } from "react-router-dom";
import { ApiError, api } from "../api/client";
import type { Resume } from "../api/types";
import { useI18n } from "../i18n";
import type { WorkspaceOutletContext } from "../layouts/WorkspaceLayout";
import { apiErrorMessage } from "../utils/errors";
import { ResumeFormModal } from "../components/resumes/ResumeFormModal";

const FILE_META: Record<
  string,
  { icon: typeof FileOutlined; tone: string; label: string }
> = {
  pdf: { icon: FilePdfOutlined, tone: "pdf", label: "PDF" },
  docx: { icon: FileWordOutlined, tone: "docx", label: "DOCX" },
  pptx: { icon: FilePptOutlined, tone: "pptx", label: "PPTX" },
  png: { icon: FileImageOutlined, tone: "png", label: "PNG" },
  md: { icon: FileMarkdownOutlined, tone: "md", label: "MD" },
  markdown: { icon: FileMarkdownOutlined, tone: "md", label: "MD" },
};

function sortResumes(items: Resume[]): Resume[] {
  return [...items].sort((left, right) => {
    if (left.pinned !== right.pinned) return left.pinned ? -1 : 1;
    return (
      new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime()
    );
  });
}

function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 KB";
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function ResumeLibraryPage() {
  const { t, formatDate } = useI18n();
  const { message } = AntdApp.useApp();
  const { onDataChanged } = useOutletContext<WorkspaceOutletContext>();
  const [items, setItems] = useState<Resume[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const [pinSavingId, setPinSavingId] = useState<number | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [previewingId, setPreviewingId] = useState<number | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.listResumes();
      setItems(sortResumes(data.items));
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setLoading(false);
    }
  }, [message, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const filtered = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    if (!keyword) return items;
    return items.filter((item) => {
      const haystack = [item.name, item.fileName, ...item.tags]
        .join(" ")
        .toLowerCase();
      return haystack.includes(keyword);
    });
  }, [items, search]);

  const handleCreated = (created: Resume) => {
    setItems((current) => sortResumes([created, ...current]));
    onDataChanged();
  };

  const handleTogglePin = async (resume: Resume) => {
    setPinSavingId(resume.id);
    try {
      const updated = await api.updateResume(resume.id, {
        pinned: !resume.pinned,
      });
      setItems((current) =>
        sortResumes(current.map((item) => (item.id === updated.id ? updated : item))),
      );
      message.success(
        updated.pinned ? t("resumes.pinnedToast") : t("resumes.unpinnedToast"),
      );
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setPinSavingId(null);
    }
  };

  const handleDelete = async (resume: Resume) => {
    setDeletingId(resume.id);
    try {
      await api.deleteResume(resume.id);
      setItems((current) => current.filter((item) => item.id !== resume.id));
      message.success(t("resumes.deleted"));
      onDataChanged();
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setDeletingId(null);
    }
  };

  const handleView = async (resume: Resume) => {
    setPreviewingId(resume.id);
    try {
      await api.openResumePreview(resume.id);
    } catch (error) {
      if (
        error instanceof ApiError &&
        (error.code === "desktop_only" || error.status === 503)
      ) {
        window.open(
          `/resume-preview?id=${resume.id}`,
          "_blank",
          "noopener,noreferrer",
        );
      } else {
        message.error(apiErrorMessage(error, t));
      }
    } finally {
      setPreviewingId(null);
    }
  };

  const renderCard = (resume: Resume) => {
    const meta = FILE_META[resume.fileType] ?? {
      icon: FileOutlined,
      tone: "file",
      label: resume.fileType.toUpperCase(),
    };
    const Icon = meta.icon;
    return (
      <Card
        key={resume.id}
        className={`resume-card${resume.pinned ? " is-pinned" : ""}`}
        hoverable
        onClick={() => void handleView(resume)}
      >
        <div className="resume-card-top">
          <span className={`resume-file-icon is-${meta.tone}`}>
            <Icon />
          </span>
          <div className="resume-card-heading">
            <h3 title={resume.name}>{resume.name}</h3>
            <span className="resume-card-file" title={resume.fileName}>
              {resume.fileName}
            </span>
          </div>
          <Tooltip
            title={
              resume.pinned
                ? t("action.unpin")
                : t("action.pin")
            }
          >
            <Button
              type="text"
              className={`resume-pin-button${resume.pinned ? " is-on" : ""}`}
              icon={
                resume.pinned ? <PushpinFilled /> : <PushpinOutlined />
              }
              loading={pinSavingId === resume.id}
              aria-label={
                resume.pinned ? t("action.unpin") : t("action.pin")
              }
              onClick={(event) => {
                event.stopPropagation();
                void handleTogglePin(resume);
              }}
            />
          </Tooltip>
        </div>

        <div className="resume-card-tags">
          {resume.pinned && (
            <Tag
              className="resume-pinned-tag"
              icon={<PushpinFilled />}
              bordered={false}
            >
              {t("resumes.pinned")}
            </Tag>
          )}
          {resume.tags.length > 0 ? (
            resume.tags.map((tag) => (
              <Tag key={tag} bordered={false}>
                {tag}
              </Tag>
            ))
          ) : (
            <span className="resume-card-no-tags">
              {t("resumes.tags")} · {t("common.optional")}
            </span>
          )}
        </div>

        <div className="resume-card-meta">
          <span className="resume-type-pill">{meta.label}</span>
          <span>{formatFileSize(resume.fileSize)}</span>
          <span>{formatDate(resume.createdAt)}</span>
        </div>

        <div
          className="resume-card-footer"
          onClick={(event) => event.stopPropagation()}
        >
          <Button
            type="primary"
            ghost
            size="small"
            icon={<EyeOutlined />}
            loading={previewingId === resume.id}
            onClick={() => void handleView(resume)}
          >
            {t("resumes.view")}
          </Button>
          <Popconfirm
            title={t("resumes.deleteTitle")}
            description={t("resumes.deleteContent", { name: resume.name })}
            okText={t("action.delete")}
            cancelText={t("action.cancel")}
            okButtonProps={{ danger: true, loading: deletingId === resume.id }}
            onConfirm={() => handleDelete(resume)}
          >
            <Button
              danger
              type="text"
              size="small"
              icon={<DeleteOutlined />}
            >
              {t("action.delete")}
            </Button>
          </Popconfirm>
        </div>
      </Card>
    );
  };

  return (
    <>
      <header className="dashboard-header">
        <div>
          <span className="season-pill">
            <FileTextOutlined /> {t("resumes.count", { count: items.length })}
          </span>
          <h1>{t("resumes.title")}</h1>
          <p>{t("resumes.subtitle")}</p>
        </div>
        <div className="dashboard-actions">
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setFormOpen(true)}
          >
            {t("resumes.add")}
          </Button>
        </div>
      </header>

      <section className="resume-board">
        <div className="resume-toolbar">
          <Input
            className="search-input"
            allowClear
            prefix={<SearchOutlined />}
            placeholder={t("resumes.searchPlaceholder")}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
          <span className="resume-total">
            {t("resumes.count", { count: filtered.length })}
          </span>
        </div>

        {loading ? (
          <div className="resume-loading">
            <Spin size="large" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="resume-empty">
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={
                <div className="resume-empty-copy">
                  <strong>
                    {search.trim()
                      ? t("resumes.searchEmptyTitle")
                      : t("resumes.emptyTitle")}
                  </strong>
                  <span>
                    {search.trim()
                      ? t("resumes.searchEmptyDescription")
                      : t("resumes.emptyDescription")}
                  </span>
                </div>
              }
            >
              {!search.trim() && (
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setFormOpen(true)}
                >
                  {t("resumes.addFirst")}
                </Button>
              )}
            </Empty>
          </div>
        ) : (
          <div className="resume-grid">{filtered.map(renderCard)}</div>
        )}
      </section>

      <ResumeFormModal
        open={formOpen}
        onClose={() => setFormOpen(false)}
        onCreated={handleCreated}
      />
    </>
  );
}
