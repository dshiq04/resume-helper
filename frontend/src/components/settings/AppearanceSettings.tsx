import { Select, Segmented } from "antd";
import { BgColorsOutlined, MoonOutlined, SunOutlined } from "@ant-design/icons";
import { localeOptions, useI18n } from "../../i18n";
import type { ThemeMode } from "../../theme";
import { SettingsCard } from "./SettingsCard";

interface AppearanceSettingsProps {
  themeMode: ThemeMode;
  onThemeChange: (mode: ThemeMode) => void;
}

export function AppearanceSettings({
  themeMode,
  onThemeChange,
}: AppearanceSettingsProps) {
  const { t, locale, setLocale } = useI18n();

  return (
    <SettingsCard
      icon={<BgColorsOutlined />}
      title={t("settings.appearance")}
      description={t("settings.themeHint")}
    >
      <div className="settings-field">
        <label>{t("settings.theme")}</label>
        <Segmented
          block
          value={themeMode}
          onChange={(value) => onThemeChange(value as ThemeMode)}
          options={[
            {
              value: "light",
              label: (
                <span className="settings-segment-label">
                  <SunOutlined /> {t("settings.light")}
                </span>
              ),
            },
            {
              value: "dark",
              label: (
                <span className="settings-segment-label">
                  <MoonOutlined /> {t("settings.dark")}
                </span>
              ),
            },
          ]}
        />
      </div>

      <div className="settings-field">
        <label>{t("settings.language")}</label>
        <Select
          value={locale}
          onChange={setLocale}
          options={localeOptions}
          optionRender={(option) => (
            <span className="settings-language-option">
              <span className="settings-language-short">{option.data.short}</span>
              {option.data.label}
            </span>
          )}
        />
        <span className="settings-field-hint">{t("settings.languageHint")}</span>
      </div>
    </SettingsCard>
  );
}
