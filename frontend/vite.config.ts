import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { viteStaticCopy } from "vite-plugin-static-copy";

// During `npm run dev`, API calls are proxied to the local Gin server.
// In the packaged app the frontend is served by Gin itself, so /api is
// same-origin and no proxy is involved.
export default defineConfig({
  logLevel: "error",
  plugins: [
    // @eternalheart/react-file-preview uses PDF.js. Ship its worker, CMaps,
    // standard fonts and WASM files with the app so PDF previews work offline.
    viteStaticCopy({
      targets: [
        {
          src: "node_modules/pdfjs-dist/legacy/build/pdf.worker.min.mjs",
          dest: "pdfjs",
          rename: { stripBase: true },
        },
        {
          src: "node_modules/pdfjs-dist/cmaps",
          dest: "pdfjs/cmaps",
          rename: { stripBase: true },
        },
        {
          src: "node_modules/pdfjs-dist/standard_fonts",
          dest: "pdfjs/standard_fonts",
          rename: { stripBase: true },
        },
        {
          src: "node_modules/pdfjs-dist/wasm",
          dest: "pdfjs/wasm",
          rename: { stripBase: true },
        },
      ],
    }),
  ],
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
    proxy: {
      "/api": {
        target: process.env.RESUME_HELPER_API || "http://127.0.0.1:19876",
        changeOrigin: true,
      },
    },
  },
  worker: {
    format: "es",
  },
  build: {
    chunkSizeWarningLimit: 2400,
  },
});
