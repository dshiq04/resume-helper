import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { viteStaticCopy } from "vite-plugin-static-copy";

// During `npm run dev`, API calls are proxied to the local Gin server.
// In the packaged app the frontend is served by Gin itself, so /api is
// same-origin and no proxy is involved.
export default defineConfig({
  logLevel: "error",
  plugins: [
    // Anyview bundles its own PDF.js worker. Ship its CMaps, standard fonts,
    // WASM and ICC assets with the app so PDF previews work offline.
    viteStaticCopy({
      targets: [
        {
          src: "node_modules/anyview/dist/pdfjs/cmaps",
          dest: "pdfjs/cmaps",
          rename: { stripBase: true },
        },
        {
          src: "node_modules/anyview/dist/pdfjs/standard_fonts",
          dest: "pdfjs/standard_fonts",
          rename: { stripBase: true },
        },
        {
          src: "node_modules/anyview/dist/pdfjs/wasm",
          dest: "pdfjs/wasm",
          rename: { stripBase: true },
        },
        {
          src: "node_modules/anyview/dist/pdfjs/iccs",
          dest: "pdfjs/iccs",
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
