import {
  useCallback,
  useEffect,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";
import { App as AntdApp, Button } from "antd";
import type { Dayjs } from "dayjs";
import { Events } from "@wailsio/runtime";
import { api, ApiError } from "../api/client";
import type {
  Application,
  ApplicationUpdate,
  Category,
  CategoryFilter,
  Status,
} from "../api/types";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";

interface UseApplicationsOptions {
  category: CategoryFilter;
  onDataChanged: () => void;
}

export interface ApplicationsBoard {
  items: Application[];
  total: number;
  page: number;
  pageSize: number;
  setPage: (page: number) => void;
  setPageSize: (pageSize: number) => void;
  searchInput: string;
  setSearchInput: (value: string) => void;
  statusFilter: Status | "all";
  setStatusFilter: (value: Status | "all") => void;
  dateRange: [Dayjs, Dayjs] | null;
  setDateRange: (value: [Dayjs, Dayjs] | null) => void;
  selectedKeys: number[];
  setSelectedKeys: Dispatch<SetStateAction<number[]>>;
  toggleSelection: (id: number, checked: boolean) => void;
  clearSelection: () => void;
  loading: boolean;
  statusSavingId: number | null;
  pinSavingId: number | null;
  hasFilters: boolean;
  refresh: () => void;
  updateField: (id: number, patch: ApplicationUpdate) => Promise<Application>;
  updateStatus: (record: Application, status: Status) => Promise<void>;
  togglePin: (record: Application) => Promise<void>;
  deleteRecord: (record: Application) => Promise<void>;
  deleteSelected: () => Promise<void>;
  updateSelectedStatus: (status: Status) => Promise<void>;
  updateSelectedCategory: (category: Category) => Promise<void>;
  afterCreated: (application: Application) => void;
  afterImported: () => void;
}

export function useApplications({
  category,
  onDataChanged,
}: UseApplicationsOptions): ApplicationsBoard {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();

  const [items, setItems] = useState<Application[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<Status | "all">("all");
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs] | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<number[]>([]);
  const [loading, setLoading] = useState(true);
  const [reloadToken, setReloadToken] = useState(0);
  const [statusSavingId, setStatusSavingId] = useState<number | null>(null);
  const [pinSavingId, setPinSavingId] = useState<number | null>(null);

  const refresh = useCallback(() => {
    setReloadToken((value) => value + 1);
  }, []);

  const clearSelection = useCallback(() => {
    setSelectedKeys([]);
  }, []);

  const toggleSelection = useCallback((id: number, checked: boolean) => {
    setSelectedKeys((keys) => {
      if (checked) {
        return keys.includes(id) ? keys : [...keys, id];
      }
      return keys.filter((key) => key !== id);
    });
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setSearch(searchInput.trim());
      setPage(1);
      setSelectedKeys([]);
    }, 320);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  useEffect(() => {
    setPage(1);
    setSelectedKeys([]);
  }, [category, statusFilter, dateRange]);

  useEffect(() => {
    setSelectedKeys([]);
  }, [page, pageSize]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.listApplications({
        category,
        status: statusFilter,
        search,
        from: dateRange?.[0].startOf("day").toISOString(),
        to: dateRange?.[1].endOf("day").toISOString(),
        page,
        pageSize,
      });
      setItems(data.items);
      setTotal(data.total);
      if (data.items.length === 0 && data.total > 0 && page > 1) {
        setPage((current) => Math.max(1, current - 1));
      }
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        message.error(apiErrorMessage(error, t));
      }
    } finally {
      setLoading(false);
    }
  }, [category, dateRange, message, page, pageSize, search, statusFilter, t]);

  useEffect(() => {
    void load();
  }, [load, reloadToken]);

  useEffect(() => {
    const refreshFromQuickEntry = () => setReloadToken((value) => value + 1);
    const unsubscribe = Events.On("resumetrail:data-changed", refreshFromQuickEntry);
    window.addEventListener("resumetrail:data-changed", refreshFromQuickEntry);
    return () => {
      unsubscribe();
      window.removeEventListener("resumetrail:data-changed", refreshFromQuickEntry);
    };
  }, []);

  const updateField = useCallback(
    async (id: number, patch: ApplicationUpdate) => {
      const updated = await api.updateApplication(id, patch);
      setItems((current) =>
        current.map((item) => (item.id === updated.id ? updated : item)),
      );
      return updated;
    },
    [],
  );

  const updateStatus = useCallback(
    async (record: Application, status: Status) => {
      if (status === record.status) return;
      setStatusSavingId(record.id);
      try {
        await updateField(record.id, { status });
        message.success(t("toast.updated"));
        onDataChanged();
      } catch (error) {
        message.error(apiErrorMessage(error, t));
      } finally {
        setStatusSavingId(null);
      }
    },
    [message, onDataChanged, t, updateField],
  );

  const togglePin = useCallback(
    async (record: Application) => {
      setPinSavingId(record.id);
      try {
        const updated = await updateField(record.id, {
          pinned: !record.pinned,
        });
        message.success(
          t(updated.pinned ? "toast.pinned" : "toast.unpinned"),
        );
        refresh();
      } catch (error) {
        message.error(apiErrorMessage(error, t));
      } finally {
        setPinSavingId(null);
      }
    },
    [message, refresh, t, updateField],
  );

  const undoDelete = useCallback(
    async (trashIds: number[]) => {
      if (trashIds.length === 0) return;
      try {
        await Promise.all(trashIds.map((id) => api.restoreTrash(id)));
        message.success(t("trash.undoSuccess"));
        onDataChanged();
        refresh();
      } catch (error) {
        message.error(apiErrorMessage(error, t));
      }
    },
    [message, onDataChanged, refresh, t],
  );

  const showDeletedToast = useCallback(
    (count: number, trashIds: number[]) => {
      const text =
        count === 1
          ? t("toast.deleted")
          : t("toast.batchDeleted", { count });
      message.success({
        duration: 6,
        content: (
          <span className="undo-toast">
            {text}
            {trashIds.length > 0 && (
              <Button
                type="link"
                size="small"
                onClick={() => void undoDelete(trashIds)}
              >
                {t("trash.undo")}
              </Button>
            )}
          </span>
        ),
      });
    },
    [message, t, undoDelete],
  );

  const deleteRecord = useCallback(
    async (record: Application) => {
      try {
        const result = await api.deleteApplication(record.id);
        showDeletedToast(1, result.trashId ? [result.trashId] : []);
        setSelectedKeys((keys) => keys.filter((key) => key !== record.id));
        onDataChanged();
        if (items.length === 1 && page > 1) {
          setPage((current) => Math.max(1, current - 1));
        } else {
          refresh();
        }
      } catch (error) {
        message.error(apiErrorMessage(error, t));
      }
    },
    [items.length, message, onDataChanged, page, refresh, showDeletedToast, t],
  );

  const deleteSelected = useCallback(async () => {
    if (selectedKeys.length === 0) return;
    try {
      const result = await api.batchDelete(selectedKeys);
      showDeletedToast(result.deleted, result.trashIds ?? []);
      const deletingAll = selectedKeys.length === items.length;
      setSelectedKeys([]);
      onDataChanged();
      if (deletingAll && page > 1) {
        setPage((current) => Math.max(1, current - 1));
      } else {
        refresh();
      }
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    }
  }, [
    items.length,
    message,
    onDataChanged,
    page,
    refresh,
    selectedKeys,
    showDeletedToast,
    t,
  ]);

  const updateSelectedStatus = useCallback(
    async (status: Status) => {
      if (selectedKeys.length === 0) return;
      try {
        const result = await api.batchStatus(selectedKeys, status);
        message.success(
          t("toast.statusUpdated", { count: result.updated }),
        );
        setSelectedKeys([]);
        onDataChanged();
        refresh();
      } catch (error) {
        message.error(apiErrorMessage(error, t));
      }
    },
    [message, onDataChanged, refresh, selectedKeys, t],
  );

  const updateSelectedCategory = useCallback(
    async (category: Category) => {
      if (selectedKeys.length === 0) return;
      try {
        const result = await api.batchCategory(selectedKeys, category);
        message.success(
          t("toast.categoryUpdated", { count: result.updated }),
        );
        setSelectedKeys([]);
        onDataChanged();
        refresh();
      } catch (error) {
        message.error(apiErrorMessage(error, t));
      }
    },
    [message, onDataChanged, refresh, selectedKeys, t],
  );

  const afterCreated = useCallback(
    (created: Application) => {
      onDataChanged();
      if (category === "all" || category === created.category) {
        setPage(1);
        refresh();
      }
    },
    [category, onDataChanged, refresh],
  );

  const afterImported = useCallback(() => {
    onDataChanged();
    setPage(1);
    refresh();
  }, [onDataChanged, refresh]);

  return {
    items,
    total,
    page,
    pageSize,
    setPage,
    setPageSize,
    searchInput,
    setSearchInput,
    statusFilter,
    setStatusFilter,
    dateRange,
    setDateRange,
    selectedKeys,
    setSelectedKeys,
    toggleSelection,
    clearSelection,
    loading,
    statusSavingId,
    pinSavingId,
    hasFilters: Boolean(search) || statusFilter !== "all" || dateRange !== null,
    refresh,
    updateField,
    updateStatus,
    togglePin,
    deleteRecord,
    deleteSelected,
    updateSelectedStatus,
    updateSelectedCategory,
    afterCreated,
    afterImported,
  };
}
