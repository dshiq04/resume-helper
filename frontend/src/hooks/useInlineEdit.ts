import { useCallback, useRef, useState } from "react";
import { App as AntdApp } from "antd";
import type { Application, ApplicationUpdate } from "../api/types";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";

export type InlineEditField = "companyName" | "link";

interface InlineEditState {
  id: number;
  field: InlineEditField;
}

export interface InlineEditController {
  editing: InlineEditState | null;
  value: string;
  saving: boolean;
  setValue: (value: string) => void;
  start: (record: Application, field: InlineEditField) => void;
  commit: () => Promise<void>;
  cancel: () => void;
}

interface UseInlineEditOptions {
  items: Application[];
  onSave: (id: number, patch: ApplicationUpdate) => Promise<Application>;
}

export function useInlineEdit({
  items,
  onSave,
}: UseInlineEditOptions): InlineEditController {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [editing, setEditing] = useState<InlineEditState | null>(null);
  const [value, setValueState] = useState("");
  const valueRef = useRef("");
  const [saving, setSaving] = useState(false);
  const cancelRef = useRef(false);
  const savingRef = useRef(false);

  const setValue = useCallback((next: string) => {
    valueRef.current = next;
    setValueState(next);
  }, []);

  const cancel = useCallback(() => {
    cancelRef.current = true;
    setEditing(null);
  }, []);

  const start = useCallback((record: Application, field: InlineEditField) => {
    cancelRef.current = false;
    valueRef.current = record[field];
    setEditing({ id: record.id, field });
    setValueState(record[field]);
  }, []);

  const commit = useCallback(async () => {
    if (!editing || savingRef.current) return;
    if (cancelRef.current) {
      cancelRef.current = false;
      return;
    }

    const record = items.find((item) => item.id === editing.id);
    if (!record) {
      setEditing(null);
      return;
    }

    const nextValue = valueRef.current;
    const currentValue = record[editing.field];
    if (editing.field === "companyName" && !nextValue.trim()) {
      message.warning(t("error.companyRequired"));
      setEditing(null);
      return;
    }
    if (nextValue.trim() === currentValue) {
      setEditing(null);
      return;
    }

    savingRef.current = true;
    setSaving(true);
    try {
      await onSave(record.id, { [editing.field]: nextValue });
      message.success(t("toast.updated"));
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      savingRef.current = false;
      setSaving(false);
      setEditing(null);
    }
  }, [editing, items, message, onSave, t]);

  return {
    editing,
    value,
    saving,
    setValue,
    start,
    commit,
    cancel,
  };
}
