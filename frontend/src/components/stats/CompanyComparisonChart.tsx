import { useMemo } from "react";
import type { EChartsCoreOption } from "echarts/core";
import type { CompanyFunnel } from "../../api/types";
import type { MessageKey } from "../../i18n";
import type { ThemeMode } from "../../theme";
import { EChart, echarts, getChartTheme, tooltipStyle } from "./echarts";

interface CompanyComparisonChartProps {
  companies: CompanyFunnel[];
  themeMode: ThemeMode;
  t: (key: MessageKey, params?: Record<string, string | number>) => string;
}

export function CompanyComparisonChart({
  companies,
  themeMode,
  t,
}: CompanyComparisonChartProps) {
  const theme = useMemo(() => getChartTheme(themeMode), [themeMode]);

  const option = useMemo<EChartsCoreOption>(() => {
    const rows = [...companies].reverse();

    return {
      animationDuration: 650,
      animationEasing: "cubicOut",
      grid: { left: 8, right: 46, top: 28, bottom: 8, containLabel: true },
      tooltip: {
        trigger: "item",
        ...tooltipStyle(theme),
        formatter: (params: { dataIndex: number }) => {
          const row = rows[params.dataIndex];
          if (!row) return "";
          return (
            `<div style="font-weight:600;">${row.company}</div>` +
            `<div style="margin-top:6px;color:${theme.textSoft};">${t("transfer.applications")}<strong style="color:${theme.text};margin-left:8px;">${row.total}</strong></div>` +
            `<div style="margin-top:4px;color:${theme.textSoft};">${t("status.interview")}<strong style="color:${theme.text};margin-left:8px;">${row.interviews}</strong></div>` +
            `<div style="margin-top:4px;color:${theme.textSoft};">${t("status.offer")}<strong style="color:${theme.text};margin-left:8px;">${row.offers}</strong></div>` +
            `<div style="margin-top:4px;color:${theme.textSoft};">${t("stats.interviewToOfferRate")}<strong style="color:${theme.text};margin-left:8px;">${row.interviewToOfferRate}%</strong></div>`
          );
        },
      },
      xAxis: {
        type: "value",
        max: 100,
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: theme.textSoft,
          fontSize: 10,
          formatter: "{value}%",
        },
        splitLine: { lineStyle: { color: theme.split, type: "dashed" } },
      },
      yAxis: {
        type: "category",
        data: rows.map((row) => row.company),
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: theme.textSoft,
          fontSize: 11,
          width: 120,
          overflow: "truncate",
        },
      },
      series: [
        {
          type: "bar",
          name: t("stats.interviewToOfferRate"),
          data: rows.map((row) => row.interviewToOfferRate),
          barWidth: 14,
          showBackground: true,
          backgroundStyle: {
            color: theme.isDark ? "rgba(148,163,184,0.12)" : "#f1f3fa",
            borderRadius: 8,
          },
          itemStyle: {
            borderRadius: 8,
            color: {
              type: "linear",
              x: 0,
              y: 0,
              x2: 1,
              y2: 0,
              colorStops: [
                { offset: 0, color: "#6366f1" },
                { offset: 1, color: "#22d3ee" },
              ],
            },
          },
          label: {
            show: true,
            position: "right",
            color: theme.textSoft,
            fontSize: 11,
            formatter: "{c}%",
          },
        },
      ],
    };
  }, [companies, t, theme]);

  return (
    <EChart
      echarts={echarts}
      option={option}
      notMerge
      lazyUpdate
      style={{ width: "100%", height: Math.max(240, companies.length * 34 + 50) }}
    />
  );
}
