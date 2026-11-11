import { useEffect, useState } from "react";
import {
  Alert,
  App as AntdApp,
  Button,
  Form,
  Input,
  InputNumber,
  Select,
} from "antd";
import {
  ApiOutlined,
  DatabaseOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import { api } from "../../api/client";
import type {
  DatabaseDriver,
  DatabaseInput,
  DatabaseSettings,
  DatabaseSettingsResponse,
} from "../../api/types";
import { useI18n } from "../../i18n";
import { apiErrorMessage } from "../../utils/errors";
import { SettingsCard } from "./SettingsCard";

interface DatabaseFormValues {
  driver: DatabaseDriver;
  dsn?: string;
  sqlitePath?: string;
  host?: string;
  port?: number;
  user?: string;
  password?: string;
  database?: string;
  sslMode?: string;
}

const DEFAULT_PORTS: Record<DatabaseDriver, number> = {
  sqlite: 0,
  mysql: 3306,
  postgres: 5432,
};

function databaseLabel(driver: DatabaseDriver): string {
  switch (driver) {
    case "mysql":
      return "MySQL";
    case "postgres":
      return "PostgreSQL";
    default:
      return "SQLite";
  }
}

function isFormValidationError(error: unknown): boolean {
  return Boolean(
    error &&
      typeof error === "object" &&
      "errorFields" in (error as Record<string, unknown>),
  );
}

function toFormValues(settings: DatabaseSettings): DatabaseFormValues {
  return {
    driver: settings.driver,
    dsn: settings.dsn,
    sqlitePath: settings.sqlitePath,
    host: settings.host || "127.0.0.1",
    port: settings.port || DEFAULT_PORTS[settings.driver],
    user: settings.user,
    password: settings.password,
    database: settings.database,
    sslMode: settings.sslMode || "disable",
  };
}

export function StorageSettings() {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [form] = Form.useForm<DatabaseFormValues>();
  const [data, setData] = useState<DatabaseSettingsResponse | null>(null);
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [useDsn, setUseDsn] = useState(false);
  const [testResult, setTestResult] = useState<{
    ok: boolean;
    text: string;
  } | null>(null);
  const driver = Form.useWatch("driver", form) ?? "sqlite";

  useEffect(() => {
    let active = true;
    api
      .databaseSettings()
      .then((response) => {
        if (!active) return;
        setData(response);
        form.setFieldsValue(toFormValues(response.settings));
        setUseDsn(Boolean(response.settings.parseError));
      })
      .catch((error) => {
        if (active) message.error(apiErrorMessage(error, t));
      });
    return () => {
      active = false;
    };
  }, [form, message, t]);

  const buildInput = async (): Promise<DatabaseInput> => {
    const values = await form.validateFields();
    return {
      driver: values.driver,
      dsn: useDsn ? values.dsn ?? "" : "",
      sqlitePath: values.driver === "sqlite" ? values.sqlitePath ?? "" : "",
      host: values.host ?? "",
      port: Number(values.port ?? 0),
      user: values.user ?? "",
      password: values.password ?? "",
      database: values.database ?? "",
      sslMode: values.sslMode ?? "disable",
      useDsn,
    };
  };

  const handleTest = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      const input = await buildInput();
      await api.testDatabase(input);
      const text = t("settings.dbTestSuccess");
      setTestResult({ ok: true, text });
      message.success(text);
    } catch (error) {
      if (isFormValidationError(error)) return;
      const text = apiErrorMessage(error, t);
      setTestResult({ ok: false, text });
      message.error(text);
    } finally {
      setTesting(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setTestResult(null);
    try {
      const input = await buildInput();
      const response = await api.updateDatabase(input);
      setData(response);
      form.setFieldsValue(toFormValues(response.settings));
      setUseDsn(Boolean(response.settings.parseError));
      message.success(t("settings.dbSaved"));
      if (response.warning) {
        message.warning(response.warning);
      }
      window.dispatchEvent(new Event("resumetrail:database-changed"));
    } catch (error) {
      if (isFormValidationError(error)) return;
      message.error(apiErrorMessage(error, t));
    } finally {
      setSaving(false);
    }
  };

  const handleDriverChange = (value: DatabaseDriver) => {
    const currentHost = form.getFieldValue("host");
    form.setFieldsValue({
      driver: value,
      host: currentHost || "127.0.0.1",
      port: DEFAULT_PORTS[value],
      sslMode: form.getFieldValue("sslMode") || "disable",
    });
    setUseDsn(false);
    setTestResult(null);
  };

  return (
    <SettingsCard
      icon={<DatabaseOutlined />}
      tone="data"
      title={t("settings.database")}
      description={t("settings.databaseHint")}
    >
      {data?.envOverride && (
        <Alert
          type="warning"
          showIcon
          className="settings-db-alert"
          message={t("settings.dbEnvOverride")}
        />
      )}

      <Form
        form={form}
        layout="vertical"
        requiredMark={false}
        className="settings-db-form"
        initialValues={{
          driver: "sqlite" as DatabaseDriver,
          host: "127.0.0.1",
          port: 0,
          sslMode: "disable",
        }}
      >
        <Form.Item name="driver" label={t("settings.dbDriver")}>
          <Select
            onChange={handleDriverChange}
            options={[
              { value: "sqlite", label: "SQLite" },
              { value: "mysql", label: "MySQL" },
              { value: "postgres", label: "PostgreSQL" },
            ]}
          />
        </Form.Item>

        {useDsn ? (
          <>
            {data?.settings.parseError && (
              <Alert
                type="warning"
                showIcon
                className="settings-db-alert"
                message={t("settings.dbParseError")}
                description={data.settings.parseError}
              />
            )}
            <Form.Item
              name="dsn"
              label={t("settings.dbDsn")}
              rules={[{ required: true, message: t("settings.dbDsnRequired") }]}
            >
              <Input.TextArea rows={3} spellCheck={false} />
            </Form.Item>
            <Button
              type="link"
              size="small"
              className="settings-db-mode-toggle"
              onClick={() => setUseDsn(false)}
            >
              {t("settings.dbUseFields")}
            </Button>
          </>
        ) : (
          <>
            {driver === "sqlite" ? (
              <Form.Item
                name="sqlitePath"
                label={t("settings.storageLocation")}
                extra={t("settings.dbSqliteHint")}
              >
                <Input placeholder="resume-helper.db" spellCheck={false} />
              </Form.Item>
            ) : (
              <>
                <div className="settings-db-grid">
                  <Form.Item
                    name="host"
                    label={t("settings.dbHost")}
                    rules={[
                      { required: true, message: t("settings.dbHostRequired") },
                    ]}
                  >
                    <Input placeholder="127.0.0.1" spellCheck={false} />
                  </Form.Item>
                  <Form.Item
                    name="port"
                    label={t("settings.dbPort")}
                    rules={[
                      {
                        required: true,
                        type: "number",
                        min: 1,
                        max: 65535,
                        message: t("settings.dbPortInvalid"),
                      },
                    ]}
                  >
                    <InputNumber min={1} max={65535} style={{ width: "100%" }} />
                  </Form.Item>
                </div>

                <div className="settings-db-grid">
                  <Form.Item
                    name="user"
                    label={t("settings.dbUser")}
                    rules={[
                      { required: true, message: t("settings.dbUserRequired") },
                    ]}
                  >
                    <Input autoComplete="off" spellCheck={false} />
                  </Form.Item>
                  <Form.Item name="password" label={t("settings.dbPassword")}>
                    <Input.Password autoComplete="new-password" />
                  </Form.Item>
                </div>

                <Form.Item
                  name="database"
                  label={t("settings.dbName")}
                  rules={[
                    { required: true, message: t("settings.dbNameRequired") },
                  ]}
                >
                  <Input placeholder="resume_helper" spellCheck={false} />
                </Form.Item>

                {driver === "postgres" && (
                  <Form.Item name="sslMode" label={t("settings.dbSslMode")}>
                    <Select
                      options={[
                        { value: "disable", label: "disable" },
                        { value: "prefer", label: "prefer" },
                        { value: "require", label: "require" },
                        { value: "verify-ca", label: "verify-ca" },
                        { value: "verify-full", label: "verify-full" },
                      ]}
                    />
                  </Form.Item>
                )}
              </>
            )}
          </>
        )}

        <div className="settings-db-actions">
          {!useDsn && (
            <Button
              type="link"
              size="small"
              className="settings-db-mode-toggle"
              onClick={() => setUseDsn(true)}
            >
              {t("settings.dbUseDsn")}
            </Button>
          )}
          <span className="settings-db-actions-spacer" />
          <Button
            icon={<ApiOutlined />}
            loading={testing}
            disabled={saving}
            onClick={() => void handleTest()}
          >
            {t("settings.dbTest")}
          </Button>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            loading={saving}
            disabled={testing}
            onClick={() => void handleSave()}
          >
            {t("settings.dbSave")}
          </Button>
        </div>

        {testResult && (
          <Alert
            type={testResult.ok ? "success" : "error"}
            showIcon
            className="settings-db-alert"
            message={testResult.text}
          />
        )}
      </Form>

      <div className="settings-info-list settings-db-info">
        <div className="settings-info-row">
          <span>{t("settings.dbActive")}</span>
          <strong>{data ? databaseLabel(data.settings.driver) : "—"}</strong>
        </div>
        {data?.location && (
          <div className="settings-info-row settings-info-path">
            <span>{t("settings.storageLocation")}</span>
            <code title={data.location}>{data.location}</code>
          </div>
        )}
        <div className="settings-info-row settings-info-path">
          <span>{t("settings.configPath")}</span>
          <code title={data?.configPath}>{data?.configPath ?? "—"}</code>
        </div>
        <div className="settings-info-row settings-info-path">
          <span>{t("settings.dataDir")}</span>
          <code title={data?.dataDir}>{data?.dataDir ?? "—"}</code>
        </div>
      </div>
      <p className="settings-field-hint settings-db-warning">
        {t("settings.dbSwitchWarning")}
      </p>
    </SettingsCard>
  );
}
