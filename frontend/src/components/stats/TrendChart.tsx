import { useMemo } from "react";
import type { EChartsCoreOption } from "echarts/core";
import type { DailyStat } from "../../api/types";
import type { MessageKey } from "../../i18n";
import type { ThemeMode } from "../../theme";
import {
  CHART_PALETTE,
  EChart,
  echarts,
  getChartTheme,
  tooltipStyle,
} from "./echarts";

interface TrendChartProps {
  data: DailyStat[];
  themeMode: ThemeMode;
  t: (key: MessageKey, params?: Record<string, string | number>) => string;
}

const DAYS = Array.from({ length: 31 }, (_, index) => index + 1);

export function TrendChart({ data, themeMode, t }: TrendChartProps) {
  const theme = useMemo(() => getChartTheme(themeMode), [themeMode]);

  const option = useMemo<EChartsCoreOption>(() => {
    const months = Array.from(
      new Set(data.map((item) => item.date.slice(0, 7))),
    ).sort();

    const series = months.map((month, index) => {
      const [year, monthNumber] = month.split("-").map(Number);
      const daysInMonth = new Date(year, monthNumber, 0).getDate();
      const counts = new Map<number, number>();
      for (const item of data) {
        if (!item.date.startsWith(`${month}-`)) continue;
        const day = Number(item.date.slice(8, 10));
        counts.set(day, (counts.get(day) ?? 0) + item.count);
      }
      return {
        name: month,
        type: "line",
        smooth: false,
        symbol: "circle",
        symbolSize: 6,
        showSymbol: false,
        connectNulls: false,
        lineStyle: { width: 2.2 },
        itemStyle: { color: CHART_PALETTE[index % CHART_PALETTE.length] },
        emphasis: { focus: "series", scale: 1.4 },
        data: DAYS.map((day) => (day <= daysInMonth ? counts.get(day) ?? 0 : null)),
      };
    });

    return {
      color: CHART_PALETTE,
      animationDuration: 700,
      animationEasing: "cubicOut",
      grid: { left: 6, right: 18, top: 46, bottom: 6, containLabel: true },
      tooltip: {
        trigger: "axis",
        ...tooltipStyle(theme),
        axisPointer: {
          type: "line",
          lineStyle: { color: theme.split, width: 1, type: "dashed" },
        },
        formatter: (params) => {
          const list = Array.isArray(params) ? params : [params];
          const rows = list
            .filter((item) => item.value !== null && item.value !== undefined)
            .map((item) => {
              const day = String(item.axisValue).padStart(2, "0");
              const date = `${item.seriesName}-${day}`;
              return (
                `<div style="display:flex;align-items:center;justify-content:space-between;gap:18px;margin-top:6px;">` +
                `<span style="display:inline-flex;align-items:center;gap:7px;">` +
                `<span style="width:8px;height:8px;border-radius:50%;background:${item.color};"></span>` +
                `<span>${date}</span></span>` +
                `<strong>${item.value}</strong></div>`
              );
            });
          if (rows.length === 0) return "";
          return `<div style="font-size:11px;color:${theme.textSoft};">${t("stats.date")}</div>${rows.join("")}`;
        },
      },
      legend: {
        type: "scroll",
        top: 0,
        icon: "roundRect",
        itemWidth: 10,
        itemHeight: 10,
        itemGap: 14,
        pageIconColor: theme.textSoft,
        pageIconInactiveColor: theme.split,
        pageTextStyle: { color: theme.textSoft, fontSize: 10 },
        textStyle: { color: theme.textSoft, fontSize: 11 },
      },
      xAxis: {
        type: "category",
        boundaryGap: false,
        data: DAYS,
        axisTick: { show: false },
        axisLine: { lineStyle: { color: theme.split } },
        axisLabel: {
          color: theme.textSoft,
          fontSize: 10,
          formatter: (value: number) => String(value),
        },
      },
      yAxis: {
        type: "value",
        minInterval: 1,
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { color: theme.textSoft, fontSize: 10 },
        splitLine: {
          lineStyle: { color: theme.split, type: "dashed" },
        },
      },
      series,
    };
  }, [data, t, theme]);

  return (
    <EChart
      echarts={echarts}
      option={option}
      notMerge
      lazyUpdate
      style={{ width: "100%", height: 320 }}
    />
  );
}
