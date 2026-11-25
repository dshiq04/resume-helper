import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import { viteStaticCopy } from "vite-plugin-static-copy";
import { fileURLToPath } from "node:url";

// The .env lives next to the frontend sources. Vite injects the application
// metadata into the bundle, and the running frontend sends it to the Go backend
// so no .env file has to be shipped next to the executable.
const frontendRoot = fileURLToPath(new URL(".", import.meta.url));

// During `npm run dev`, API calls are proxied to the local Gin server.
// In the packaged app the frontend is served by Gin itself, so /api is
// same-origin and no proxy is involved.
export default defineConfig(({ mode }) => {
  const appEnv = loadEnv(mode, frontendRoot, "");
  const appVersion = process.env.APP_VERSION || appEnv.APP_VERSION || "1.2.1";
  const appName = process.env.APP_NAME || appEnv.APP_NAME || "ResumeTrail";
  const appTitle =
    process.env.APP_TITLE || appEnv.APP_TITLE || "履途 ResumeTrail";
  const defaultLanguage =
    process.env.APP_DEFAULT_LANGUAGE || appEnv.APP_DEFAULT_LANGUAGE || "zh-CN";

  return {
    logLevel: "error",
    define: {
      __APP_VERSION__: JSON.stringify(appVersion),
      __APP_NAME__: JSON.stringify(appName),
      __APP_TITLE__: JSON.stringify(appTitle),
      __APP_DEFAULT_LANGUAGE__: JSON.stringify(defaultLanguage),
    },
    plugins: [
      {
        // The Vite dev server serves its own index.html, so it never receives
        // the per-process API token that Gin injects. Fetch the token from the
        // running Gin server on demand and expose it to the dev frontend.
        name: "resume-trail-dev-token",
        configureServer(server) {
          const apiTarget =
            process.env.RESUME_HELPER_API || "http://127.0.0.1:19876";
          server.middlewares.use("/__resume_token", async (_req, res) => {
            try {
              const response = await fetch(`${apiTarget}/`, {
                headers: { Accept: "text/html" },
              });
              const html = await response.text();
              const match = html.match(
                /window\.__RESUME_TRAIL_TOKEN__="([0-9a-f]+)"/,
              );
              if (!match) {
                throw new Error("app token not found in index.html");
              }
              res.setHeader("Cache-Control", "no-store");
              res.setHeader("Content-Type", "application/json");
              res.end(JSON.stringify({ token: match[1] }));
            } catch (error) {
              res.statusCode = 502;
              res.setHeader("Content-Type", "application/json");
              res.end(
                JSON.stringify({
                  error: error instanceof Error ? error.message : String(error),
                }),
              );
            }
          });
        },
      },
      {
        // Vite only expands %VITE_*% placeholders in index.html, so inject the
        // shared .env values ourselves.
        name: "app-metadata-html",
        transformIndexHtml(html) {
          return html
            .replaceAll("%APP_TITLE%", appTitle)
            .replaceAll("%APP_DEFAULT_LANGUAGE%", defaultLanguage);
        },
      },
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
          // The dev server is same-origin for the browser, but Gin still
          // validates the forwarded Origin. Use the API origin explicitly so
          // the proxy is accepted in development.
          headers: {
            Origin: process.env.RESUME_HELPER_API || "http://127.0.0.1:19876",
          },
        },
      },
    },
    worker: {
      format: "es",
    },
    build: {
      chunkSizeWarningLimit: 2400,
    },
  };
});
