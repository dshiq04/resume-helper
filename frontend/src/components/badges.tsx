import { useMemo } from "react";
import { CATEGORIES, STATUSES } from "../api/types";
import type { Category, Status, StatusOption } from "../api/types";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n";

export function StatusBadge({ status }: { status: Status }) {
  const { t } = useI18n();
  return (
    <span className={`status-badge status-badge-${status}`}>
      <span className="status-badge-dot" />
      <span>{t(`status.${status}` as MessageKey)}</span>
    </span>
  );
}

export function CategoryBadge({ category }: { category: Category }) {
  const { t } = useI18n();
  return (
    <span className={`category-badge category-badge-${category}`}>
      {t(`category.${category}` as MessageKey)}
    </span>
  );
}

export function useStatusOptions(): StatusOption[] {
  const { t, locale } = useI18n();
  return useMemo(
    () =>
      STATUSES.map((status) => ({
        value: status,
        label: t(`status.${status}` as MessageKey),
      })),
    [locale, t],
  );
}

export function useCategoryOptions() {
  const { t, locale } = useI18n();
  return useMemo(
    () =>
      CATEGORIES.map((category) => ({
        value: category,
        label: t(`category.${category}` as MessageKey),
      })),
    [locale, t],
  );
}
