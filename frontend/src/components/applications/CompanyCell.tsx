import { PushpinFilled } from "@ant-design/icons";
import type { Application } from "../../api/types";
import { useI18n } from "../../i18n";
import { avatarTone, companyInitial } from "../../utils/display";
import { CategoryBadge } from "../badges";

interface CompanyCellProps {
  application: Application;
  onView?: (application: Application) => void;
}

export function CompanyCell({ application, onView }: CompanyCellProps) {
  const { t } = useI18n();

  return (
    <div className="company-cell">
      <button
        type="button"
        className={`company-avatar company-avatar-button avatar-tone-${avatarTone(application.companyName)}`}
        aria-label={application.companyName}
        title={application.companyName}
        onClick={(event) => {
          event.stopPropagation();
          onView?.(application);
        }}
      >
        {companyInitial(application.companyName)}
      </button>
      <div className="company-copy">
        <div className="company-name-row">
          <span className="company-name-button is-static">
            {application.companyName}
          </span>
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
