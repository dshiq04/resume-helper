import type { Resume } from "../api/types";

// Stored resume files always use the encrypted ".reb" suffix. The browser
// receives the decrypted bytes without a useful file name, so the original
// name and MIME type from the database are restored here before previewing.
const RESUME_MIME_TYPES: Record<string, string> = {
  docx: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  pdf: "application/pdf",
  pptx: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  png: "image/png",
  md: "text/markdown",
  markdown: "text/markdown",
};

export function restoreResumeFile(record: Resume, blob: Blob): File {
  const type =
    RESUME_MIME_TYPES[record.fileType] ||
    blob.type ||
    "application/octet-stream";
  return new File([blob], record.fileName, { type });
}
