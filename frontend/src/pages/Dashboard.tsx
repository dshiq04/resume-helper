import { useState } from "react";
import { App as AntdApp } from "antd";
import { useOutletContext } from "react-router-dom";
import { api } from "../api/client";
import type { CategoryFilter } from "../api/types";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";
import { useApplications } from "../hooks/useApplications";
import { useApplicationDraft } from "../hooks/useApplicationDraft";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useStatusOptions } from "../components/badges";
import { StatsCards } from "../components/StatsCards";
import { DashboardHeader } from "../components/dashboard/DashboardHeader";
import { DashboardToolbar } from "../components/dashboard/DashboardToolbar";
import { SelectionBar } from "../components/dashboard/SelectionBar";
import { ApplicationTable } from "../components/applications/ApplicationTable";
import { ApplicationCards } from "../components/applications/ApplicationCards";
import { ImportModal } from "../components/ImportModal";
import type { WorkspaceOutletContext } from "../layouts/WorkspaceLayout";
import { isDesktopRuntime } from "../utils/desktop";

interface DashboardProps {
  category: CategoryFilter;
}

export function Dashboard({ category }: DashboardProps) {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const { stats, onDataChanged } =
    useOutletContext<WorkspaceOutletContext>();
  const board = useApplications({ category, onDataChanged });
  const draft = useApplicationDraft({
    category,
    onCreated: board.afterCreated,
  });
  const statusOptions = useStatusOptions();
  const isCompactList = useMediaQuery("(max-width: 1199px)");
  const [importOpen, setImportOpen] = useState(false);

  const handleQuickAdd = async () => {
    try {
      await api.openQuickEntry(category);
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    }
  };

  const handleEdit = async (record: { id: number }) => {
    if (!isDesktopRuntime()) {
      // Headless/browser preview: the standalone page works in the same tab.
      window.location.assign(`/application-edit?id=${record.id}`);
      return;
    }
    try {
      await api.openApplicationEdit(record.id);
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    }
  };

  const handleView = async (record: { id: number }) => {
    if (!isDesktopRuntime()) {
      window.location.assign(`/application-view?id=${record.id}`);
      return;
    }
    try {
      await api.openApplicationView(record.id);
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    }
  };

  const sharedListProps = {
    items: board.items,
    total: board.total,
    page: board.page,
    pageSize: board.pageSize,
    loading: board.loading,
    selectedKeys: board.selectedKeys,
    statusSavingId: board.statusSavingId,
    pinSavingId: board.pinSavingId,
    statusOptions,
    hasFilters: board.hasFilters,
    draft: draft.draft,
    draftSaving: draft.saving,
    canChooseDraftCategory: draft.canChooseCategory,
    onDraftChange: draft.update,
    onDraftSave: draft.save,
    onDraftCancel: draft.cancel,
    onStatusChange: board.updateStatus,
    onPin: board.togglePin,
    onView: (record: { id: number }) => void handleView(record),
    onEdit: (record: { id: number }) => void handleEdit(record),
    onDelete: board.deleteRecord,
    onAdd: draft.start,
  };

  return (
    <>
      <DashboardHeader
        category={category}
        onAdd={draft.start}
        onImport={() => setImportOpen(true)}
        onQuickAdd={() => void handleQuickAdd()}
      />

      <StatsCards stats={stats} category={category} />

      <section className="table-card">
        {board.selectedKeys.length > 0 && (
          <SelectionBar
            count={board.selectedKeys.length}
            statusOptions={statusOptions}
            onBatchCategory={board.updateSelectedCategory}
            onBatchStatus={board.updateSelectedStatus}
            onBatchDelete={board.deleteSelected}
            onClear={board.clearSelection}
          />
        )}

        <DashboardToolbar
          searchInput={board.searchInput}
          statusFilter={board.statusFilter}
          statusOptions={statusOptions}
          dateRange={board.dateRange}
          onSearchChange={board.setSearchInput}
          onStatusChange={board.setStatusFilter}
          onDateRangeChange={board.setDateRange}
        />

        {isCompactList ? (
          <ApplicationCards
            {...sharedListProps}
            onToggleSelection={board.toggleSelection}
            onPageChange={board.setPage}
          />
        ) : (
          <ApplicationTable
            {...sharedListProps}
            onSelectionChange={board.setSelectedKeys}
            onPageChange={board.setPage}
            onPageSizeChange={board.setPageSize}
          />
        )}
      </section>

      <ImportModal
        open={importOpen}
        onClose={() => setImportOpen(false)}
        onImported={board.afterImported}
      />
    </>
  );
}
