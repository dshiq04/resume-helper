import {
  Component,
  useCallback,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { Button, Result, Spin, Tooltip } from "antd";
import {
  CloseOutlined,
  CompressOutlined,
  FileTextOutlined,
  FullscreenOutlined,
  MinusOutlined,
  PushpinFilled,
  PushpinOutlined,
} from "@ant-design/icons";
import { Events, Window } from "@wailsio/runtime";
import {
  FilePreviewEmbed,
  configurePdfjs,
} from "@eternalheart/react-file-preview";
import "@eternalheart/react-file-preview/style.css";
import { ApiError, api } from "../api/client";
import type { Resume } from "../api/types";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";
import { ResizeHandles } from "../components/ResizeHandles";
import {
  closeWindow,
  getDesktopWindowState,
  isDesktopRuntime,
  minimiseWindow,
  toggleFullscreenWindow,
} from "../utils/desktop";

// Bundle PDF.js assets with the frontend so the desktop app can preview PDFs
// offline. These files are copied into dist/pdfjs by vite-plugin-static-copy.
configurePdfjs({
  workerSrc: "/pdfjs/pdf.worker.min.mjs",
  cMapUrl: "/pdfjs/cmaps/",
  cMapPacked: true,
  standardFontDataUrl: "/pdfjs/standard_fonts/",
  wasmUrl: "/pdfjs/wasm/",
});

const MIME_TYPES: Record<string, string> = {
  docx: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  pdf: "application/pdf",
  pptx: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  png: "image/png",
  md: "text/markdown",
  markdown: "text/markdown",
};

function readResumeID(): number | null {
  const raw = new URLSearchParams(window.location.search).get("id");
  if (!raw) return null;
  const value = Number(raw);
  return Number.isInteger(value) && value > 0 ? value : null;
}

interface PreviewBoundaryProps {
  resetKey: string;
  fallback: (error: Error) => ReactNode;
  children: ReactNode;
}

interface PreviewBoundaryState {
  error: Error | null;
}

class PreviewErrorBoundary extends Component<
  PreviewBoundaryProps,
  PreviewBoundaryState
> {
  state: PreviewBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): PreviewBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error) {
    console.error("resume preview failed", error);
  }

  componentDidUpdate(previous: PreviewBoundaryProps) {
    if (previous.resetKey !== this.props.resetKey && this.state.error) {
      this.setState({ error: null });
    }
  }

  render() {
    if (this.state.error) {
      return this.props.fallback(this.state.error);
    }
    return this.props.children;
  }
}

export function ResumePreviewWindowPage() {
  const { t, locale } = useI18n();
  const [resume, setResume] = useState<Resume | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [windowPinned, setWindowPinned] = useState(false);
  const [fullscreen, setFullscreen] = useState(false);
  const [theme] = useState<"light" | "dark">(() =>
    localStorage.getItem("resumetrail.theme") === "dark" ? "dark" : "light",
  );

  useEffect(() => {
    let active = true;
    const id = readResumeID();
    setResume(null);
    setFile(null);
    setError(null);

    if (!id) {
      setError(t("resumes.previewError"));
      return () => {
        active = false;
      };
    }

    void (async () => {
      try {
        const record = await api.getResume(id);
        if (!active) return;
        setResume(record);
        const blob = await api.fetchResumeFile(id);
        if (!active) return;
        const type = MIME_TYPES[record.fileType] || blob.type;
        const preview = new File([blob], record.fileName, { type });
        setFile(preview);
        document.title = `${record.name} · ${t("resumes.previewTitle")}`;
      } catch (caught) {
        if (!active) return;
        setError(
          caught instanceof ApiError && caught.status === 401
            ? t("resumes.previewUnauthorized")
            : apiErrorMessage(caught, t),
        );
      }
    })();

    return () => {
      active = false;
    };
  }, [attempt, t]);

  const refreshWindowState = useCallback(() => {
    void getDesktopWindowState().then((state) => {
      setFullscreen(state.fullscreen);
    });
  }, []);

  useEffect(() => {
    refreshWindowState();
    const unsubscribers = [
      Events.On("common:WindowFullscreen", () => setFullscreen(true)),
      Events.On("common:WindowUnFullscreen", () => setFullscreen(false)),
    ];
    window.addEventListener("resize", refreshWindowState);
    document.addEventListener("fullscreenchange", refreshWindowState);
    return () => {
      for (const unsubscribe of unsubscribers) unsubscribe();
      window.removeEventListener("resize", refreshWindowState);
      document.removeEventListener("fullscreenchange", refreshWindowState);
    };
  }, [refreshWindowState]);

  const togglePin = async () => {
    const next = !windowPinned;
    setWindowPinned(next);
    if (!isDesktopRuntime()) return;
    try {
      await Window.SetAlwaysOnTop(next);
    } catch {
      setWindowPinned(!next);
    }
  };

  const toggleFullscreen = async () => {
    await toggleFullscreenWindow();
    refreshWindowState();
  };

  const handleClose = () => {
    if (isDesktopRuntime()) {
      void closeWindow();
      return;
    }
    // Browser preview mode: preview was opened in the same tab, so going back
    // is the closest equivalent to closing a native preview window.
    if (window.opener) {
      window.close();
      return;
    }
    if (document.referrer.startsWith(window.location.origin)) {
      window.history.back();
      return;
    }
    window.location.assign("/resumes");
  };

  const libraryLocale = locale.startsWith("zh") ? "zh-CN" : "en-US";
  const title = resume?.name || t("resumes.previewTitle");

  return (
    <div className="resume-preview-window">
      <header className="titlebar preview-titlebar">
        <span className="preview-titlebar-title" title={title}>
          <FileTextOutlined />
          <span>{title}</span>
        </span>
        <div className="titlebar-drag-space" aria-hidden="true" />
        <div className="titlebar-actions">
          <Tooltip
            title={
              windowPinned ? t("quick.windowUnpin") : t("quick.windowPin")
            }
          >
            <button
              className={`titlebar-tool preview-pin-button${
                windowPinned ? " is-on" : ""
              }`}
              type="button"
              aria-label={
                windowPinned ? t("quick.windowUnpin") : t("quick.windowPin")
              }
              onClick={() => void togglePin()}
            >
              {windowPinned ? <PushpinFilled /> : <PushpinOutlined />}
            </button>
          </Tooltip>
          <span className="titlebar-divider" aria-hidden="true" />
          <button
            className="window-button"
            type="button"
            title={t("window.minimize")}
            aria-label={t("window.minimize")}
            onClick={() => void minimiseWindow()}
          >
            <MinusOutlined />
          </button>
          <button
            className="window-button"
            type="button"
            title={
              fullscreen ? t("window.exitFullscreen") : t("window.fullscreen")
            }
            aria-label={
              fullscreen ? t("window.exitFullscreen") : t("window.fullscreen")
            }
            onClick={() => void toggleFullscreen()}
          >
            {fullscreen ? <CompressOutlined /> : <FullscreenOutlined />}
          </button>
          <button
            className="window-button window-button-close"
            type="button"
            title={t("window.close")}
            aria-label={t("window.close")}
            onClick={handleClose}
          >
            <CloseOutlined />
          </button>
        </div>
      </header>

      <div className="resume-preview-content">
        {error ? (
          <Result
            status="error"
            title={t("resumes.previewError")}
            subTitle={error}
            extra={
              <Button
                type="primary"
                onClick={() => setAttempt((value) => value + 1)}
              >
                {t("common.retry")}
              </Button>
            }
          />
        ) : file ? (
          <PreviewErrorBoundary
            resetKey={`${resume?.id ?? "resume"}-${attempt}`}
            fallback={(caught) => (
              <Result
                status="error"
                title={t("resumes.previewError")}
                subTitle={caught.message || t("resumes.previewError")}
                extra={
                  <Button
                    type="primary"
                    onClick={() => setAttempt((value) => value + 1)}
                  >
                    {t("common.retry")}
                  </Button>
                }
              />
            )}
          >
            <FilePreviewEmbed
              key={resume?.id}
              files={[file]}
              currentIndex={0}
              theme={theme}
              locale={libraryLocale}
            />
          </PreviewErrorBoundary>
        ) : (
          <div className="resume-preview-loading">
            <Spin size="large" />
            <p>{t("resumes.previewLoading")}</p>
          </div>
        )}
      </div>

      <ResizeHandles />
    </div>
  );
}
