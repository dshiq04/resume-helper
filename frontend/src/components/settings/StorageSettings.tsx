import { useEffect, useState } from "react";
import { Alert, App as AntdApp, Button, Form, Input } from "antd";
import {
  ApiOutlined,
  DatabaseOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import { api } from "../../api/client";
import type {
  DatabaseInput,
  DatabaseSettings,
  DatabaseSettingsResponse,
} from "../../api/types";
import { useI18n } from "../../i18n";
import { apiErrorMessage } from "../../utils/errors";
import { SettingsCard } from "./SettingsCard";

interface DatabaseFormValues {
  sqlitePath?: string;
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
    sqlitePath: settings.sqlitePath,
  };
}

export function StorageSettings() {
  const { t } = useI18n();
  const { message } = AntdApp.useApp();
  const [form] = Form.useForm<DatabaseFormValues>();
  const [data, setData] = useState<DatabaseSettingsResponse | null>(null);
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testResult, setTestResult] = useState<{
    ok: boolean;
    text: string;
  } | null>(null);

  useEffect(() => {
    let active = true;
    api
      .databaseSettings()
      .then((response) => {
        if (!active) return;
        setData(response);
        form.setFieldsValue(toFormValues(response.settings));
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
      driver: "sqlite",
      sqlitePath: values.sqlitePath ?? "",
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
        initialValues={{ sqlitePath: "" }}
      >
        <Form.Item
          name="sqlitePath"
          label={t("settings.storageLocation")}
          extra={t("settings.dbSqliteHint")}
        >
          <Input placeholder="resume-helper.db" spellCheck={false} />
        </Form.Item>

        <div className="settings-db-actions">
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
          <strong>SQLite</strong>
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
