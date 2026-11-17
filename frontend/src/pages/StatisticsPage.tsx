import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  App as AntdApp,
  Button,
  DatePicker,
  Empty,
  Segmented,
  Select,
  Spin,
  Tooltip,
} from "antd";
import {
  BarChartOutlined,
  DownloadOutlined,
  FieldTimeOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import dayjs, { type Dayjs } from "dayjs";
import html2canvas from "html2canvas";
import { useOutletContext } from "react-router-dom";
import { api } from "../api/client";
import {
  CATEGORIES,
  type Category,
  type DailyStat,
  type FunnelStats,
  type Stats,
} from "../api/types";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n";
import type { WorkspaceOutletContext } from "../layouts/WorkspaceLayout";
import { apiErrorMessage } from "../utils/errors";
import { TrendChart } from "../components/stats/TrendChart";
import { StatusDonut } from "../components/stats/StatusDonut";
import { CalendarHeatmap } from "../components/stats/CalendarHeatmap";
import { FunnelChart } from "../components/stats/FunnelChart";
import { CompanyComparisonChart } from "../components/stats/CompanyComparisonChart";

const CATEGORY_LABEL_KEYS: Record<Category, MessageKey> = {
  spring: "category.spring",
  autumn: "category.autumn",
  internship: "category.internship",
  social: "category.social",
};

type RangeValue = [Dayjs, Dayjs] | null;

function currentSeasonCategory(): Category {
  const month = new Date().getMonth() + 1;
  if (month >= 8 && month <= 11) return "autumn";
  if (month >= 2 && month <= 5) return "spring";
  return "internship";
}

function quarterStart(): Dayjs {
  const now = dayjs();
  return now.month(Math.floor(now.month() / 3) * 3).startOf("month");
}

export function StatisticsPage() {
  const { t, locale } = useI18n();
  const { message } = AntdApp.useApp();
  const { stats: workspaceStats, themeMode } = useOutletContext<WorkspaceOutletContext>();
  const [category, setCategory] = useState<Category>(currentSeasonCategory);
  const [range, setRange] = useState<RangeValue>(null);
  const [daily, setDaily] = useState<DailyStat[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [funnel, setFunnel] = useState<FunnelStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [year, setYear] = useState<number | null>(null);
  const [exporting, setExporting] = useState(false);
  const pageRef = useRef<HTMLDivElement>(null);

  const rangeQuery = useMemo(() => {
    if (!range) return undefined;
    return {
      from: range[0].startOf("day").toISOString(),
      to: range[1].endOf("day").toISOString(),
    };
  }, [range]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [dailyResult, statsResult, funnelResult] = await Promise.all([
        api.dailyStats(rangeQuery),
        api.stats(rangeQuery),
        api.funnel(rangeQuery),
      ]);
      setDaily(dailyResult.daily ?? []);
      setStats(statsResult);
      setFunnel(funnelResult);
    } catch (error) {
      message.error(apiErrorMessage(error, t));
    } finally {
      setLoading(false);
    }
  }, [message, rangeQuery, t]);

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
      : range && years.includes(range[0].year())
        ? range[0].year()
        : years[0] ?? new Date().getFullYear();

  const categoryCount = stats?.byCategory[category] ?? 0;
  const total = stats?.total ?? 0;
  const stuck = stats?.stuckScreening;

  const categoryOptions = CATEGORIES.map((value) => ({
    value,
    label: t(CATEGORY_LABEL_KEYS[value]),
  }));

  const handleCategoryChange = (value: string | number) => {
    setCategory(value as Category);
    setYear(null);
  };

  const presets = [
    {
      label: t("stats.thisMonth"),
      value: [dayjs().startOf("month"), dayjs().endOf("month")] as [Dayjs, Dayjs],
    },
    {
      label: t("stats.thisQuarter"),
      value: [quarterStart(), dayjs().endOf("day")] as [Dayjs, Dayjs],
    },
    {
      label: t("stats.thisYear"),
      value: [dayjs().startOf("year"), dayjs().endOf("year")] as [Dayjs, Dayjs],
    },
  ];

  const exportImage = async () => {
    if (!pageRef.current || exporting) return;
    const target = pageRef.current;
    const originalWidth = target.style.width;
    setExporting(true);
    try {
      const background =
        getComputedStyle(document.documentElement).getPropertyValue("--bg").trim() ||
        (themeMode === "dark" ? "#090e1a" : "#f4f5fb");

      // Export at the full desktop layout width even when the app window is
      // narrow. The calendar heatmap needs enough horizontal room for all
      // twelve months; otherwise the right-hand months were clipped.
      target.style.width = `${Math.max(target.clientWidth, 1180)}px`;
      window.dispatchEvent(new Event("resize"));
      await new Promise<void>((resolve) => {
        window.requestAnimationFrame(() =>
          window.requestAnimationFrame(() => resolve()),
        );
      });

      let captureWidth = Math.ceil(target.scrollWidth);
      let captureHeight = Math.ceil(target.scrollHeight);
      target.querySelectorAll<HTMLElement>("*").forEach((element) => {
        captureWidth = Math.max(captureWidth, Math.ceil(element.scrollWidth));
        captureHeight = Math.max(captureHeight, Math.ceil(element.scrollHeight));
      });

      const canvas = await html2canvas(target, {
        backgroundColor: background,
        scale: Math.max(2, window.devicePixelRatio || 1),
        useCORS: true,
        logging: false,
        width: captureWidth,
        height: captureHeight,
        windowWidth: Math.max(captureWidth, window.innerWidth),
        windowHeight: Math.max(captureHeight, window.innerHeight),
        scrollX: 0,
        scrollY: 0,
      });
      const link = document.createElement("a");
      const stamp = dayjs().format("YYYYMMDD-HHmm");
      link.download = `resumetrail-stats-${stamp}.png`;
      link.href = canvas.toDataURL("image/png");
      document.body.appendChild(link);
      link.click();
      link.remove();
      message.success(t("stats.exportedImage"));
    } catch {
      message.error(t("stats.exportFailed"));
    } finally {
      target.style.width = originalWidth;
      window.dispatchEvent(new Event("resize"));
      setExporting(false);
    }
  };

  return (
    <>
    <div className="stats-page" ref={pageRef}>
      <header className="stats-page-header stats-page-header-with-action">
        <div>
          <span className="season-pill">
            <BarChartOutlined /> {t("nav.library")} · {t("nav.statistics")}
          </span>
          <h1>{t("stats.title")}</h1>
          <p>{t("stats.subtitle")}</p>
        </div>
        <Tooltip title={t("stats.rangeHint")}>
          <Button
            className="stats-export-button"
            icon={<DownloadOutlined />}
            loading={exporting}
            onClick={() => void exportImage()}
          >
            {exporting ? t("stats.exportingImage") : t("stats.exportImage")}
          </Button>
        </Tooltip>
      </header>

      <div className="stats-toolbar">
        <Segmented
          value={category}
          options={categoryOptions}
          onChange={handleCategoryChange}
        />
        <div className="stats-toolbar-spacer" />
        <DatePicker.RangePicker
          className="stats-range-picker"
          value={range}
          allowClear
          presets={presets}
          format="YYYY-MM-DD"
          placeholder={[t("stats.range"), t("stats.range")]}
          onChange={(values) => {
            if (!values || !values[0] || !values[1]) {
              setRange(null);
              setYear(null);
              return;
            }
            setRange([values[0], values[1]]);
            setYear(null);
          }}
        />
        <span className="stats-toolbar-total">
          {range
            ? t("stats.filteredTotal", { count: total })
            : `${t("stats.applicationCount")} · ${categoryCount}`}
        </span>
      </div>

      {loading ? (
        <div className="stats-loading">
          <Spin size="large" />
        </div>
      ) : (
        <>
          <section className="stats-kpi-row">
            <article className="stats-kpi-card is-screening">
              <span className="stats-kpi-icon">
                <FieldTimeOutlined />
              </span>
              <div>
                <span>{t("stats.stuckScreening")}</span>
                <strong>{stuck?.count ?? 0}</strong>
                <p>{t("stats.stuckScreeningHint")}</p>
              </div>
            </article>
            <article className="stats-kpi-card is-interview">
              <span className="stats-kpi-icon">
                <TeamOutlined />
              </span>
              <div>
                <span>{t("stats.avgToInterview")}</span>
                <strong>
                  {funnel?.averageDaysToInterview ?? 0}
                  <small>{t("stats.daysUnit")}</small>
                </strong>
                <p>
                  {t("stats.stuckAverage")} {stuck?.averageDays ?? 0} ·{" "}
                  {t("stats.stuckMax")} {stuck?.maxDays ?? 0}
                </p>
              </div>
            </article>
            <article className="stats-kpi-card is-offer">
              <span className="stats-kpi-icon">
                <BarChartOutlined />
              </span>
              <div>
                <span>{t("stats.avgInterviewToOffer")}</span>
                <strong>
                  {funnel?.averageInterviewToOffer ?? 0}
                  <small>{t("stats.daysUnit")}</small>
                </strong>
                <p>
                  {t("stats.interviewToOfferRate")} ·{" "}
                  {funnel && funnel.interviewCount > 0
                    ? Math.round((funnel.offerCount / funnel.interviewCount) * 100)
                    : 0}
                  %
                </p>
              </div>
            </article>
          </section>

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

          <section className="stats-chart-grid">
            <article className="stats-chart-card stats-chart-card-wide">
              <header className="stats-chart-head">
                <div>
                  <h3>{t("stats.funnel")}</h3>
                  <p>{t("stats.funnelHint")}</p>
                </div>
              </header>
              {funnel && funnel.total > 0 ? (
                <FunnelChart data={funnel} themeMode={themeMode} t={t} />
              ) : (
                <div className="stats-chart-empty">
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={t("stats.noData")}
                  />
                </div>
              )}
            </article>

            <article className="stats-chart-card">
              <header className="stats-chart-head">
                <div>
                  <h3>{t("stats.companyComparison")}</h3>
                  <p>{t("stats.companyHint")}</p>
                </div>
              </header>
              {funnel && funnel.companies.length > 0 ? (
                <CompanyComparisonChart
                  companies={funnel.companies}
                  themeMode={themeMode}
                  t={t}
                />
              ) : (
                <div className="stats-chart-empty">
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={t("stats.noData")}
                  />
                </div>
              )}
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

          {!workspaceStats && total === 0 && (
            <p className="stats-footnote">{t("stats.noData")}</p>
          )}
        </>
      )}
    </div>
    {exporting && (
      <div className="stats-export-overlay" role="status" aria-live="polite">
        <div className="stats-export-overlay-card">
          <Spin size="large" />
          <strong>{t("stats.exportingImage")}</strong>
        </div>
      </div>
    )}
    </>
  );
}
