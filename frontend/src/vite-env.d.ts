/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_SHOW_DEVELOPMENT_WATERMARK?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
