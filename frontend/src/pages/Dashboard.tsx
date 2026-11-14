import { useState } from "react";
import { App as AntdApp } from "antd";
import { useOutletContext } from "react-router-dom";
import { api } from "../api/client";
import type { CategoryFilter } from "../api/types";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";
import { useApplications } from "../hooks/useApplications";
import { useApplicationDraft } from "../hooks/useApplicationDraft";
import { useInlineEdit } from "../hooks/useInlineEdit";
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
  const inlineEdit = useInlineEdit({
    items: board.items,
    onSave: board.updateField,
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
    try {
      await api.openApplicationEdit(record.id);
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
    inline: inlineEdit,
    hasFilters: board.hasFilters,
    draft: draft.draft,
    draftSaving: draft.saving,
    canChooseDraftCategory: draft.canChooseCategory,
    onDraftChange: draft.update,
    onDraftSave: draft.save,
    onDraftCancel: draft.cancel,
    onStatusChange: board.updateStatus,
    onPin: board.togglePin,
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
            onBatchStatus={board.updateSelectedStatus}
            onBatchDelete={board.deleteSelected}
            onClear={board.clearSelection}
          />
        )}

        <DashboardToolbar
          searchInput={board.searchInput}
          statusFilter={board.statusFilter}
          statusOptions={statusOptions}
          onSearchChange={board.setSearchInput}
          onStatusChange={board.setStatusFilter}
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
