import { Browser, Window } from "@wailsio/runtime";

export function isDesktopRuntime(): boolean {
  return Boolean((window as unknown as { _wails?: { environment?: unknown } })._wails?.environment);
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
