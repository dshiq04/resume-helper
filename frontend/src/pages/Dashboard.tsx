import { useState } from "react";
import { useOutletContext } from "react-router-dom";
import type { CategoryFilter } from "../api/types";
import { useApplications } from "../hooks/useApplications";
import { useInlineEdit } from "../hooks/useInlineEdit";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useStatusOptions } from "../components/badges";
import { StatsCards } from "../components/StatsCards";
import { DashboardHeader } from "../components/dashboard/DashboardHeader";
import { DashboardToolbar } from "../components/dashboard/DashboardToolbar";
import { SelectionBar } from "../components/dashboard/SelectionBar";
import { ApplicationTable } from "../components/applications/ApplicationTable";
import { ApplicationCards } from "../components/applications/ApplicationCards";
import { ApplicationModal } from "../components/ApplicationModal";
import { ImportModal } from "../components/ImportModal";
import type { WorkspaceOutletContext } from "../layouts/WorkspaceLayout";

interface DashboardProps {
  category: CategoryFilter;
}

export function Dashboard({ category }: DashboardProps) {
  const { stats, onDataChanged } =
    useOutletContext<WorkspaceOutletContext>();
  const board = useApplications({ category, onDataChanged });
  const inlineEdit = useInlineEdit({
    items: board.items,
    onSave: board.updateField,
  });
  const statusOptions = useStatusOptions();
  const isCompactList = useMediaQuery("(max-width: 1199px)");
  const [addOpen, setAddOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);

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
    onStatusChange: board.updateStatus,
    onPin: board.togglePin,
    onDelete: board.deleteRecord,
    onAdd: () => setAddOpen(true),
  };

  return (
    <>
      <DashboardHeader
        category={category}
        onAdd={() => setAddOpen(true)}
        onImport={() => setImportOpen(true)}
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

      <ApplicationModal
        open={addOpen}
        defaultCategory={category}
        onClose={() => setAddOpen(false)}
        onCreated={board.afterCreated}
      />

      <ImportModal
        open={importOpen}
        onClose={() => setImportOpen(false)}
        onImported={board.afterImported}
      />
    </>
  );
}
