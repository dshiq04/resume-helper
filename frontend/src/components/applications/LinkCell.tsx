import { Tooltip } from "antd";
import { ExportOutlined, LinkOutlined } from "@ant-design/icons";
import type { Application } from "../../api/types";
import { useI18n } from "../../i18n";
import { openExternal } from "../../utils/desktop";
import { isWebLink, prettyLink, withProtocol } from "../../utils/display";

interface LinkCellProps {
  application: Application;
  mobile?: boolean;
}

export function LinkCell({ application, mobile = false }: LinkCellProps) {
  const { t } = useI18n();

  if (!application.link) {
    return (
      <span className="link-empty">
        <LinkOutlined />
        <span>{t("table.noLink")}</span>
      </span>
    );
  }

  return (
    <div className={`link-cell ${mobile ? "is-mobile" : ""}`}>
      <span className="link-text" title={application.link}>
        {prettyLink(application.link)}
      </span>
      {isWebLink(application.link) && (
        <Tooltip title={t("action.openLink")}>
          <button
            type="button"
            className="link-open-button"
            aria-label={t("action.openLink")}
            onClick={(event) => {
              event.stopPropagation();
              void openExternal(withProtocol(application.link));
            }}
          >
            <ExportOutlined />
          </button>
        </Tooltip>
      )}
    </div>
  );
}
