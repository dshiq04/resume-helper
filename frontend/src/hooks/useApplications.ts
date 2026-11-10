import {
  useCallback,
  useEffect,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";
import { App as AntdApp } from "antd";
import { api, ApiError } from "../api/client";
import type {
  Application,
  ApplicationUpdate,
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
  }, [category, statusFilter]);

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
  }, [category, message, page, pageSize, search, statusFilter, t]);

  useEffect(() => {
    void load();
  }, [load, reloadToken]);

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

  const deleteRecord = useCallback(
    async (record: Application) => {
      try {
        await api.deleteApplication(record.id);
        message.success(t("toast.deleted"));
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
    [items.length, message, onDataChanged, page, refresh, t],
  );

  const deleteSelected = useCallback(async () => {
    if (selectedKeys.length === 0) return;
    try {
      const result = await api.batchDelete(selectedKeys);
      message.success(t("toast.batchDeleted", { count: result.deleted }));
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
    selectedKeys,
    setSelectedKeys,
    toggleSelection,
    clearSelection,
    loading,
    statusSavingId,
    pinSavingId,
    hasFilters: Boolean(search) || statusFilter !== "all",
    refresh,
    updateField,
    updateStatus,
    togglePin,
    deleteRecord,
    deleteSelected,
    updateSelectedStatus,
    afterCreated,
    afterImported,
  };
}
