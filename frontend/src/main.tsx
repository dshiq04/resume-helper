import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
import { api } from "./api/client";
import "./styles/global.css";

// Report the metadata from frontend/.env to the Go backend. The backend keeps
// defaults until this handshake completes, so a failure is not fatal.
void api
  .applyAppInfo({
    version: __APP_VERSION__,
    name: __APP_NAME__,
    title: __APP_TITLE__,
    defaultLanguage: __APP_DEFAULT_LANGUAGE__,
  })
  .catch(() => {
    // Ignore: the backend falls back to its built-in defaults.
  });

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
);
