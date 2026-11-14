import { useCallback, useEffect, useMemo, useState } from "react";
import { App as AntdApp, Empty, Segmented, Select, Spin } from "antd";
import { BarChartOutlined } from "@ant-design/icons";
import { useOutletContext } from "react-router-dom";
import { api } from "../api/client";
import { CATEGORIES, type Category, type DailyStat } from "../api/types";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n";
import type { WorkspaceOutletContext } from "../layouts/WorkspaceLayout";
import { apiErrorMessage } from "../utils/errors";
import { TrendChart } from "../components/stats/TrendChart";
import { StatusDonut } from "../components/stats/StatusDonut";
import { CalendarHeatmap } from "../components/stats/CalendarHeatmap";

const CATEGORY_LABEL_KEYS: Record<Category, MessageKey> = {
  spring: "category.spring",
  autumn: "category.autumn",
  internship: "category.internship",
};

function currentSeasonCategory(): Category {
  const month = new Date().getMonth() + 1;
  if (month >= 8 && month <= 11) return "autumn";
  if (month >= 2 && month <= 5) return "spring";
  return "internship";
}

export function StatisticsPage() {
  const { t, locale } = useI18n();
  const { message } = AntdApp.useApp();
  const { stats, themeMode } = useOutletContext<WorkspaceOutletContext>();
  const [category, setCategory] = useState<Category>(currentSeasonCategory);
  const [daily, setDaily] = useState<DailyStat[]>([]);
  const [loading, setLoading] = useState(true);
  const [year, setYear] = useState<number | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const result = await api.dailyStats();
      setDaily(result.daily ?? []);
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setLoading(false);
    }
  }, [message, t]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    const refresh = () => void load();
    window.addEventListener("resumetrail:data-changed", refresh);
    return () => window.removeEventListener("resumetrail:data-changed", refresh);
  }, [load]);

  const categoryDaily = useMemo(
    () => daily.filter((item) => item.category === category),
    [category, daily],
  );

  const years = useMemo(() => {
    const set = new Set<number>();
    for (const item of categoryDaily) {
      set.add(Number(item.date.slice(0, 4)));
    }
    return Array.from(set)
      .filter((value) => Number.isFinite(value) && value > 0)
      .sort((left, right) => right - left);
  }, [categoryDaily]);

  const activeYear =
    year !== null && years.includes(year)
      ? year
      : years[0] ?? new Date().getFullYear();

  const categoryCount = stats?.byCategory[category] ?? 0;

  const categoryOptions = CATEGORIES.map((value) => ({
    value,
    label: t(CATEGORY_LABEL_KEYS[value]),
  }));

  const handleCategoryChange = (value: string | number) => {
    setCategory(value as Category);
    setYear(null);
  };

  return (
    <div className="stats-page">
      <header className="stats-page-header">
        <span className="season-pill">
          <BarChartOutlined /> {t("nav.library")} · {t("nav.statistics")}
        </span>
        <h1>{t("stats.title")}</h1>
        <p>{t("stats.subtitle")}</p>
      </header>

      <div className="stats-toolbar">
        <Segmented
          value={category}
          options={categoryOptions}
          onChange={handleCategoryChange}
        />
        <span className="stats-toolbar-total">
          {t("stats.applicationCount")} · {categoryCount}
        </span>
      </div>

      {loading ? (
        <div className="stats-loading">
          <Spin size="large" />
        </div>
      ) : (
        <>
          <section className="stats-chart-grid">
            <article className="stats-chart-card stats-chart-card-wide">
              <header className="stats-chart-head">
                <div>
                  <h3>{t("stats.trend")}</h3>
                  <p>{t("stats.trendHint")}</p>
                </div>
              </header>
              {categoryDaily.length > 0 ? (
                <TrendChart
                  data={categoryDaily}
                  themeMode={themeMode}
                  t={t}
                />
              ) : (
                <div className="stats-chart-empty">
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={t("stats.empty")}
                  />
                </div>
              )}
            </article>

            <article className="stats-chart-card">
              <header className="stats-chart-head">
                <div>
                  <h3>{t("stats.statusBreakdown")}</h3>
                  <p>{t("stats.statusHint")}</p>
                </div>
              </header>
              <StatusDonut
                counts={stats?.byCategoryStatus[category]}
                themeMode={themeMode}
                t={t}
              />
            </article>
          </section>

          <article className="stats-chart-card stats-chart-card-full">
            <header className="stats-chart-head">
              <div>
                <h3>{t("stats.calendar")}</h3>
                <p>{t("stats.calendarHint")}</p>
              </div>
              {years.length > 1 && (
                <Select
                  size="small"
                  value={activeYear}
                  options={years.map((value) => ({
                    value,
                    label: String(value),
                  }))}
                  onChange={(value) => setYear(value)}
                  className="stats-year-select"
                  aria-label={t("stats.year")}
                />
              )}
            </header>
            <CalendarHeatmap
              data={categoryDaily}
              year={activeYear}
              locale={locale}
              themeMode={themeMode}
              t={t}
            />
          </article>
        </>
      )}
    </div>
  );
}
