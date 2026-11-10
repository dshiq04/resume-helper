import { Input } from "antd";
import { PushpinFilled } from "@ant-design/icons";
import type { Application } from "../../api/types";
import type { InlineEditController } from "../../hooks/useInlineEdit";
import { useI18n } from "../../i18n";
import { avatarTone, companyInitial } from "../../utils/display";
import { CategoryBadge } from "../badges";

interface CompanyCellProps {
  application: Application;
  inline: InlineEditController;
}

export function CompanyCell({ application, inline }: CompanyCellProps) {
  const { t } = useI18n();
  const isEditing =
    inline.editing?.id === application.id &&
    inline.editing.field === "companyName";

  return (
    <div className="company-cell">
      <span
        className={`company-avatar avatar-tone-${avatarTone(application.companyName)}`}
      >
        {companyInitial(application.companyName)}
      </span>
      <div className="company-copy">
        <div className="company-name-row">
          {isEditing ? (
            <Input
              autoFocus
              size="small"
              className="inline-edit-input"
              value={inline.value}
              maxLength={255}
              disabled={inline.saving}
              onChange={(event) => inline.setValue(event.target.value)}
              onBlur={() => void inline.commit()}
              onFocus={(event) => event.target.select()}
              onPressEnter={(event) => event.currentTarget.blur()}
              onKeyDown={(event) => {
                if (event.key === "Escape") inline.cancel();
              }}
            />
          ) : (
            <button
              type="button"
              className="company-name-button"
              onClick={() => inline.start(application, "companyName")}
            >
              {application.companyName}
            </button>
          )}
          {application.pinned && (
            <span className="pinned-marker" title={t("action.pin")}>
              <PushpinFilled />
            </span>
          )}
        </div>
        <div className="company-meta">
          <CategoryBadge category={application.category} />
        </div>
      </div>
    </div>
  );
}
