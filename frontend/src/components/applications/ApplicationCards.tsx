import { Checkbox, Pagination, Spin } from "antd";
import { PushpinFilled } from "@ant-design/icons";
import type { Application, Status, StatusOption } from "../../api/types";
import type { ApplicationDraft } from "../../hooks/useApplicationDraft";
import type { InlineEditController } from "../../hooks/useInlineEdit";
import { useI18n } from "../../i18n";
import { CompanyCell } from "./CompanyCell";
import { LinkCell } from "./LinkCell";
import { ApplicationStatusSelect } from "./ApplicationStatusSelect";
import { ApplicationActions } from "./ApplicationActions";
import { ApplicationEmptyState } from "./ApplicationEmptyState";
import { DraftCard } from "./ApplicationDraftFields";

interface ApplicationCardsProps {
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
  onToggleSelection: (id: number, checked: boolean) => void;
  onPageChange: (page: number) => void;
  onStatusChange: (record: Application, status: Status) => void;
  onPin: (record: Application) => void;
  onDelete: (record: Application) => void;
  onAdd: () => void;
}

export function ApplicationCards({
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
  onToggleSelection,
  onPageChange,
  onStatusChange,
  onPin,
  onDelete,
  onAdd,
}: ApplicationCardsProps) {
  const { t, formatDate } = useI18n();

  if (loading && items.length === 0 && !draft) {
    return (
      <div className="mobile-loading">
        <Spin />
      </div>
    );
  }

  if (items.length === 0 && !draft) {
    return (
      <ApplicationEmptyState hasFilters={hasFilters} onAdd={onAdd} />
    );
  }

  return (
    <div className="mobile-app-list">
      {draft && (
        <DraftCard
          draft={draft}
          saving={draftSaving}
          canChooseCategory={canChooseDraftCategory}
          onChange={onDraftChange}
          onSave={onDraftSave}
          onCancel={onDraftCancel}
        />
      )}
      {items.map((record) => (
        <article
          key={record.id}
          className={`mobile-app-card ${record.pinned ? "is-pinned" : ""} ${
            selectedKeys.includes(record.id) ? "is-selected" : ""
          }`}
        >
          <div className="mobile-card-top">
            <Checkbox
              className="mobile-select"
              checked={selectedKeys.includes(record.id)}
              aria-label={record.companyName}
              onChange={(event) =>
                onToggleSelection(record.id, event.target.checked)
              }
            />
            <CompanyCell application={record} inline={inline} />
            <div className="mobile-card-actions">
              <ApplicationActions
                application={record}
                pinSaving={pinSavingId === record.id}
                mobile
                onPin={onPin}
                onDelete={onDelete}
              />
            </div>
          </div>

          <div className="mobile-card-body">
            <div className="mobile-card-link-row">
              <LinkCell application={record} inline={inline} mobile />
            </div>
            <ApplicationStatusSelect
              application={record}
              options={statusOptions}
              saving={statusSavingId === record.id}
              mobile
              onChange={onStatusChange}
            />
          </div>

          <div className="mobile-card-footer">
            <span>
              {t("table.appliedAt")}: {formatDate(record.appliedAt)}
            </span>
            {record.pinned && (
              <span className="mobile-pinned-badge">
                <PushpinFilled /> {t("action.pin")}
              </span>
            )}
          </div>
        </article>
      ))}

      <div className="mobile-pagination">
        <Pagination
          simple
          current={page}
          pageSize={pageSize}
          total={total}
          showSizeChanger={false}
          onChange={onPageChange}
        />
      </div>
    </div>
  );
}
