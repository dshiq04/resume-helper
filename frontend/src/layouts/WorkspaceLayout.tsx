import { Outlet } from "react-router-dom";
import type { Stats } from "../api/types";
import { Sidebar } from "../components/Sidebar";
import type { ThemeMode } from "../theme";

export interface WorkspaceOutletContext {
  stats: Stats | null;
  onDataChanged: () => void;
  themeMode: ThemeMode;
  onThemeChange: (mode: ThemeMode) => void;
  passwordEnabled: boolean;
  onPasswordEnabledChange: (enabled: boolean) => void;
  onLock: () => void;
}

export function WorkspaceLayout(props: WorkspaceOutletContext) {
  return (
    <div className="app-body">
      <Sidebar stats={props.stats} passwordEnabled={props.passwordEnabled} />
      <main className="main-content">
        <Outlet context={props} />
      </main>
    </div>
  );
}
