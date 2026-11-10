import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// During `npm run dev`, API calls are proxied to the local Gin server.
// In the packaged app the frontend is served by Gin itself, so /api is
// same-origin and no proxy is involved.
export default defineConfig({
  logLevel: "error",
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
  build: {
    chunkSizeWarningLimit: 1400,
  },
});
