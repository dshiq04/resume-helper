import { useCallback, useRef, useState } from "react";
import { App as AntdApp } from "antd";
import { api } from "../api/client";
import type {
  Application,
  Category,
  CategoryFilter,
  Status,
} from "../api/types";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";

export interface ApplicationDraft {
  category: Category;
  companyName: string;
  link: string;
  status: Status;
}

export interface ApplicationDraftController {
  draft: ApplicationDraft | null;
  saving: boolean;
  canChooseCategory: boolean;
  start: () => void;
  update: (patch: Partial<ApplicationDraft>) => void;
  save: () => Promise<void>;
  cancel: () => void;
}

interface UseApplicationDraftOptions {
  category: CategoryFilter;
  onCreated: (application: Application) => void;
}

export function useApplicationDraft({
  category,
  onCreated,
}: UseApplicationDraftOptions): ApplicationDraftController {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [draft, setDraft] = useState<ApplicationDraft | null>(null);
  const [saving, setSaving] = useState(false);
  const savingRef = useRef(false);

  const start = useCallback(() => {
    if (draft) {
      message.info(t("draft.inProgress"));
      return;
    }
    setDraft({
      category: category === "all" ? "spring" : category,
      companyName: "",
      link: "",
      status: "screening",
    });
  }, [category, draft, message, t]);

  const update = useCallback((patch: Partial<ApplicationDraft>) => {
    setDraft((current) => (current ? { ...current, ...patch } : current));
  }, []);

  const cancel = useCallback(() => {
    if (savingRef.current) return;
    setDraft(null);
  }, []);

  const save = useCallback(async () => {
    if (!draft || savingRef.current) return;
    const companyName = draft.companyName.trim();
    if (!companyName) {
      message.warning(t("form.companyRequired"));
      return;
    }

    savingRef.current = true;
    setSaving(true);
    try {
      const created = await api.createApplication({
        category: draft.category,
        companyName,
        link: draft.link.trim(),
        status: draft.status,
      });
      message.success(t("toast.created"));
      setDraft(null);
      onCreated(created);
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      savingRef.current = false;
      setSaving(false);
    }
  }, [draft, message, onCreated, t]);

  return {
    draft,
    saving,
    canChooseCategory: category === "all",
    start,
    update,
    save,
    cancel,
  };
}
