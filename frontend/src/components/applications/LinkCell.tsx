import { Input, Tooltip } from "antd";
import { ExportOutlined, LinkOutlined } from "@ant-design/icons";
import type { Application } from "../../api/types";
import type { InlineEditController } from "../../hooks/useInlineEdit";
import { useI18n } from "../../i18n";
import { openExternal } from "../../utils/desktop";
import { isWebLink, prettyLink, withProtocol } from "../../utils/display";

interface LinkCellProps {
  application: Application;
  inline: InlineEditController;
  mobile?: boolean;
}

export function LinkCell({ application, inline, mobile = false }: LinkCellProps) {
  const { t } = useI18n();
  const isEditing =
    inline.editing?.id === application.id &&
    inline.editing.field === "link";

  if (isEditing) {
    return (
      <Input
        autoFocus
        size="small"
        className="inline-edit-input"
        value={inline.value}
        maxLength={2048}
        disabled={inline.saving}
        placeholder={t("form.linkPlaceholder")}
        onChange={(event) => inline.setValue(event.target.value)}
        onBlur={() => void inline.commit()}
        onPressEnter={(event) => event.currentTarget.blur()}
        onKeyDown={(event) => {
          if (event.key === "Escape") inline.cancel();
        }}
      />
    );
  }

  if (!application.link) {
    return (
      <button
        type="button"
        className="link-empty"
        onClick={() => inline.start(application, "link")}
      >
        <LinkOutlined />
        <span>{t("table.noLink")}</span>
      </button>
    );
  }

  return (
    <div className={`link-cell ${mobile ? "is-mobile" : ""}`}>
      <button
        type="button"
        className="link-text"
        title={application.link}
        onClick={() => inline.start(application, "link")}
      >
        {prettyLink(application.link)}
      </button>
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
