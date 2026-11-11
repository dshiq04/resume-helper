import { useEffect, useState } from "react";
import { Button, Result, Spin } from "antd";
import {
  FilePreviewEmbed,
  configurePdfjs,
} from "@eternalheart/react-file-preview";
import "@eternalheart/react-file-preview/style.css";
import { ApiError, api } from "../api/client";
import type { Resume } from "../api/types";
import { useI18n } from "../i18n";
import { apiErrorMessage } from "../utils/errors";

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

export function ResumePreviewWindowPage() {
  const { t, locale } = useI18n();
  const [resume, setResume] = useState<Resume | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
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

  const libraryLocale = locale.startsWith("zh") ? "zh-CN" : "en-US";

  return (
    <div className="resume-preview-shell">
      {error ? (
        <Result
          status="error"
          title={t("resumes.previewError")}
          subTitle={error}
          extra={
            <Button type="primary" onClick={() => setAttempt((value) => value + 1)}>
              {t("common.retry")}
            </Button>
          }
        />
      ) : file ? (
        <FilePreviewEmbed
          key={resume?.id}
          files={[file]}
          currentIndex={0}
          theme={theme}
          locale={libraryLocale}
        />
      ) : (
        <div className="resume-preview-loading">
          <Spin size="large" />
          <p>{t("resumes.previewLoading")}</p>
        </div>
      )}
    </div>
  );
}
