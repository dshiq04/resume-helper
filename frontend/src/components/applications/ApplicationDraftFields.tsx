import { Button, Input, Select, Tooltip } from "antd";
import {
  CheckOutlined,
  CloseOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import type { Status } from "../../api/types";
import type { ApplicationDraft } from "../../hooks/useApplicationDraft";
import { useI18n } from "../../i18n";
import { CategoryBadge, StatusBadge, useCategoryOptions } from "../badges";

interface DraftFieldProps {
  draft: ApplicationDraft;
  saving: boolean;
  canChooseCategory: boolean;
  onChange: (patch: Partial<ApplicationDraft>) => void;
  onSave: () => Promise<void>;
  onCancel: () => void;
}

interface DraftCompanyCellProps extends DraftFieldProps {}

export function DraftCompanyCell({
  draft,
  saving,
  canChooseCategory,
  onChange,
  onSave,
  onCancel,
}: DraftCompanyCellProps) {
  const { t } = useI18n();
  const categoryOptions = useCategoryOptions();

  return (
    <div className="company-cell draft-company-cell">
      <span className="company-avatar avatar-tone-0 draft-avatar">
        <PlusOutlined />
      </span>
      <div className="company-copy">
        <Input
          autoFocus
          size="small"
          className="inline-edit-input draft-input"
          value={draft.companyName}
          maxLength={255}
          disabled={saving}
          placeholder={t("form.companyPlaceholder")}
          onChange={(event) => onChange({ companyName: event.target.value })}
          onPressEnter={() => void onSave()}
          onKeyDown={(event) => {
            if (event.key === "Escape") onCancel();
          }}
        />
        <div className="company-meta">
          {canChooseCategory ? (
            <Select
              size="small"
              variant="borderless"
              className="draft-category-select"
              value={draft.category}
              options={categoryOptions}
              popupMatchSelectWidth={false}
              disabled={saving}
              onChange={(category) => onChange({ category })}
            />
          ) : (
            <CategoryBadge category={draft.category} />
          )}
        </div>
      </div>
    </div>
  );
}

export function DraftLinkCell({
  draft,
  saving,
  onChange,
  onSave,
  onCancel,
}: DraftFieldProps) {
  const { t } = useI18n();

  return (
    <Input
      size="small"
      className="inline-edit-input draft-input"
      value={draft.link}
      maxLength={2048}
      disabled={saving}
      placeholder={t("form.linkPlaceholder")}
      onChange={(event) => onChange({ link: event.target.value })}
      onPressEnter={() => void onSave()}
      onKeyDown={(event) => {
        if (event.key === "Escape") onCancel();
      }}
    />
  );
}

export function DraftStatusSelect({
  draft,
  saving,
  onChange,
}: DraftFieldProps) {
  const { t } = useI18n();
  const statusOptions = [
    { value: "screening" as Status, label: t("status.screening") },
    { value: "written" as Status, label: t("status.written") },
    { value: "interview" as Status, label: t("status.interview") },
    { value: "offer" as Status, label: t("status.offer") },
    { value: "rejected" as Status, label: t("status.rejected") },
  ];

  return (
    <Select
      size="small"
      className="status-select draft-status-select"
      value={draft.status}
      options={statusOptions}
      disabled={saving}
      labelRender={({ value }) => <StatusBadge status={value as Status} />}
      optionRender={(option) => <StatusBadge status={option.value as Status} />}
      onChange={(status) => onChange({ status })}
    />
  );
}

export function DraftActions({
  saving,
  onSave,
  onCancel,
}: Pick<DraftFieldProps, "saving" | "onSave" | "onCancel">) {
  const { t } = useI18n();

  return (
    <div className="draft-actions">
      <Tooltip title={t("action.cancel")}>
        <Button
          type="text"
          size="small"
          shape="circle"
          aria-label={t("action.cancel")}
          disabled={saving}
          icon={<CloseOutlined />}
          onClick={onCancel}
        />
      </Tooltip>
      <Tooltip title={t("action.save")}>
        <Button
          type="primary"
          size="small"
          shape="circle"
          className="draft-save-button"
          aria-label={t("action.save")}
          loading={saving}
          icon={<CheckOutlined />}
          onClick={() => void onSave()}
        />
      </Tooltip>
    </div>
  );
}

export function DraftCard({
  draft,
  saving,
  canChooseCategory,
  onChange,
  onSave,
  onCancel,
}: DraftFieldProps) {
  const { t } = useI18n();
  const categoryOptions = useCategoryOptions();

  return (
    <article className="mobile-app-card draft-card">
      <div className="mobile-card-top">
        <span className="draft-card-label">
          <span className="draft-card-dot" />
          {t("draft.new")}
        </span>
        <div className="mobile-card-actions">
          <DraftActions saving={saving} onSave={onSave} onCancel={onCancel} />
        </div>
      </div>

      <div className="draft-card-fields">
        <label>
          <span>{t("form.company")}</span>
          <Input
            autoFocus
            value={draft.companyName}
            maxLength={255}
            disabled={saving}
            placeholder={t("form.companyPlaceholder")}
            onChange={(event) => onChange({ companyName: event.target.value })}
            onPressEnter={() => void onSave()}
            onKeyDown={(event) => {
              if (event.key === "Escape") onCancel();
            }}
          />
        </label>
        <label>
          <span>{t("form.link")}</span>
          <Input
            value={draft.link}
            maxLength={2048}
            disabled={saving}
            placeholder={t("form.linkPlaceholder")}
            onChange={(event) => onChange({ link: event.target.value })}
            onPressEnter={() => void onSave()}
            onKeyDown={(event) => {
              if (event.key === "Escape") onCancel();
            }}
          />
        </label>
        <div className="draft-card-selects">
          {canChooseCategory ? (
            <Select
              value={draft.category}
              options={categoryOptions}
              disabled={saving}
              onChange={(category) => onChange({ category })}
            />
          ) : (
            <CategoryBadge category={draft.category} />
          )}
          <DraftStatusSelect
            draft={draft}
            saving={saving}
            canChooseCategory={canChooseCategory}
            onChange={onChange}
            onSave={onSave}
            onCancel={onCancel}
          />
        </div>
      </div>
    </article>
  );
}
