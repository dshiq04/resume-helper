/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_SHOW_DEVELOPMENT_WATERMARK?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

// Injected from frontend/.env by vite.config.ts.
declare const __APP_VERSION__: string;
declare const __APP_NAME__: string;
declare const __APP_TITLE__: string;
declare const __APP_DEFAULT_LANGUAGE__: string;
