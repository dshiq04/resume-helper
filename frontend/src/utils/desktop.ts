import { Browser, Window } from "@wailsio/runtime";

interface WailsWindow {
  _wails?: {
    environment?: unknown;
    invoke?: (message: string) => void;
  };
}

export type ResizeEdge =
  | "n-resize"
  | "ne-resize"
  | "e-resize"
  | "se-resize"
  | "s-resize"
  | "sw-resize"
  | "w-resize"
  | "nw-resize";

export interface DesktopWindowState {
  maximised: boolean;
  fullscreen: boolean;
}

export function isDesktopRuntime(): boolean {
  return Boolean(
    (window as unknown as WailsWindow)._wails?.environment,
  );
}

export async function openExternal(url: string) {
  if (!url) return;
  if (isDesktopRuntime()) {
    try {
      await Browser.OpenURL(url);
      return;
    } catch {
      // Fall through to the browser fallback below.
    }
  }
  window.open(url, "_blank", "noopener,noreferrer");
}

export async function minimiseWindow() {
  if (!isDesktopRuntime()) return;
  try {
    await Window.Minimise();
  } catch {
    // Browser preview: no-op.
  }
}

export async function toggleMaximiseWindow() {
  if (isDesktopRuntime()) {
    try {
      await Window.ToggleMaximise();
      return;
    } catch {
      // Fall through.
    }
  }
  await toggleBrowserFullscreen();
}

export async function toggleFullscreenWindow() {
  if (isDesktopRuntime()) {
    try {
      await Window.ToggleFullscreen();
      return;
    } catch {
      // Fall through.
    }
  }
  await toggleBrowserFullscreen();
}

export async function getDesktopWindowState(): Promise<DesktopWindowState> {
  if (!isDesktopRuntime()) {
    return {
      maximised: false,
      fullscreen: Boolean(document.fullscreenElement),
    };
  }
  try {
    const [maximised, fullscreen] = await Promise.all([
      Window.IsMaximised(),
      Window.IsFullscreen(),
    ]);
    return { maximised, fullscreen };
  } catch {
    return { maximised: false, fullscreen: false };
  }
}

/**
 * Starts a native window resize from the given edge. Wails listens for
 * "wails:resize:<edge>" messages and hands the gesture to the operating
 * system, which keeps resizing smooth even for frameless windows.
 */
export function startWindowResize(edge: ResizeEdge) {
  if (!isDesktopRuntime()) return;
  const invoke = (window as unknown as WailsWindow)._wails?.invoke;
  if (typeof invoke !== "function") return;
  try {
    invoke(`wails:resize:${edge}`);
  } catch {
    // Ignore: browser preview or a runtime without resize support.
  }
}

export async function closeWindow() {
  if (!isDesktopRuntime()) {
    window.close();
    return;
  }
  try {
    await Window.Close();
  } catch {
    // Browser preview: no-op.
  }
}

async function toggleBrowserFullscreen() {
  try {
    if (document.fullscreenElement) {
      await document.exitFullscreen();
    } else {
      await document.documentElement.requestFullscreen();
    }
  } catch {
    // Some browsers block fullscreen without a direct user gesture.
  }
}
