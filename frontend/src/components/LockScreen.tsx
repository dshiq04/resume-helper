import { useCallback, useState } from "react";
import { Alert, Button, Input } from "antd";
import { ArrowRightOutlined, LockOutlined, SafetyCertificateOutlined } from "@ant-design/icons";
import { api, ApiError, setToken } from "../api/client";
import type { AuthSession } from "../api/types";
import { useI18n } from "../i18n";
import { Logo } from "./Logo";

interface LockScreenProps {
  onAuthenticated: (session: AuthSession) => void;
}

export function LockScreen({ onAuthenticated }: LockScreenProps) {
  const { t } = useI18n();
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const submit = useCallback(async () => {
    if (!password || loading) return;
    setLoading(true);
    setError("");
    try {
      const session = await api.login(password);
      setToken(session.token);
      setPassword("");
      onAuthenticated(session);
    } catch (err) {
      if (err instanceof ApiError && err.code === "invalid_password") {
        setError(t("auth.error"));
      } else if (err instanceof ApiError && err.code === "network_error") {
        setError(t("error.network"));
      } else {
        setError(t("error.generic"));
      }
      setLoading(false);
    }
  }, [loading, onAuthenticated, password, t]);

  return (
    <div className="auth-screen">
      <div className="auth-orb auth-orb-one" />
      <div className="auth-orb auth-orb-two" />
      <div className="auth-grid" aria-hidden="true" />

      <div className="auth-card">
        <div className="auth-logo-row">
          <Logo size={52} />
          <div>
            <div className="auth-brand">{t("app.brand")}</div>
            <div className="auth-tagline">{t("app.tagline")}</div>
          </div>
        </div>

        <div className="auth-heading">
          <span className="auth-eyebrow">
            <LockOutlined /> {t("settings.security")}
          </span>
          <h1>{t("auth.title")}</h1>
          <p>{t("auth.subtitle")}</p>
        </div>

        {error && (
          <Alert
            className="auth-alert"
            type="error"
            showIcon
            message={error}
          />
        )}

        <form
          className="auth-form"
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          <label className="auth-label" htmlFor="unlock-password">
            {t("auth.password")}
          </label>
          <Input.Password
            id="unlock-password"
            size="large"
            autoFocus
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="••••••••"
            prefix={<LockOutlined className="auth-input-icon" />}
            autoComplete="current-password"
          />
          <Button
            type="primary"
            size="large"
            block
            loading={loading}
            htmlType="submit"
            className="auth-submit"
          >
            {loading ? t("auth.loggingIn") : t("auth.login")}
            {!loading && <ArrowRightOutlined />}
          </Button>
        </form>

        <div className="auth-hint">
          <SafetyCertificateOutlined />
          <span>{t("auth.hint")}</span>
        </div>
      </div>
    </div>
  );
}
