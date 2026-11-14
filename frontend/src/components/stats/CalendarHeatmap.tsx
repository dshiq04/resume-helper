import { useMemo } from "react";
import type { EChartsCoreOption } from "echarts/core";
import type { DailyStat } from "../../api/types";
import type { MessageKey } from "../../i18n";
import type { ThemeMode } from "../../theme";
import { EChart, echarts, getChartTheme, tooltipStyle } from "./echarts";

interface CalendarHeatmapProps {
  data: DailyStat[];
  year: number;
  locale: string;
  themeMode: ThemeMode;
  t: (key: MessageKey, params?: Record<string, string | number>) => string;
}

function pad(value: number): string {
  return String(value).padStart(2, "0");
}

export function CalendarHeatmap({
  data,
  year,
  locale,
  themeMode,
  t,
}: CalendarHeatmapProps) {
  const theme = useMemo(() => getChartTheme(themeMode), [themeMode]);

  const option = useMemo<EChartsCoreOption>(() => {
    const counts = new Map<string, number>();
    for (const item of data) {
      if (!item.date.startsWith(`${year}-`)) continue;
      counts.set(item.date, (counts.get(item.date) ?? 0) + item.count);
    }

    const cells: Array<[string, number]> = [];
    for (let month = 1; month <= 12; month += 1) {
      const daysInMonth = new Date(year, month, 0).getDate();
      for (let day = 1; day <= daysInMonth; day += 1) {
        const key = `${year}-${pad(month)}-${pad(day)}`;
        cells.push([key, counts.get(key) ?? 0]);
      }
    }

    const max = cells.reduce((peak, cell) => Math.max(peak, cell[1]), 0);

    const monthNames = Array.from({ length: 12 }, (_, index) =>
      new Intl.DateTimeFormat(locale, { month: "short" }).format(
        new Date(year, index, 1),
      ),
    );
    const chinese = locale.startsWith("zh");

    return {
      animationDuration: 600,
      tooltip: {
        trigger: "item",
        ...tooltipStyle(theme),
        formatter: (params) => {
          const [date, count] = params.value as [string, number];
          return (
            `<div style="font-weight:600;">${date}</div>` +
            `<div style="margin-top:5px;color:${theme.textSoft};">${t("stats.applicationCount")}` +
            `<strong style="color:${theme.text};margin-left:8px;">${count}</strong></div>`
          );
        },
      },
      visualMap: {
        show: max > 0,
        type: "continuous",
        min: 0,
        max: Math.max(max, 1),
        calculable: false,
        orient: "horizontal",
        left: "center",
        bottom: 2,
        itemWidth: 12,
        itemHeight: 110,
        precision: 0,
        text: [String(max), "0"],
        textGap: 8,
        textStyle: { color: theme.textSoft, fontSize: 10 },
        inRange: {
          color: theme.isDark
            ? ["#12251f", "#14532d", "#16a34a", "#4ade80"]
            : ["#eef7f1", "#bbf7d0", "#4ade80", "#15803d"],
        },
      },
      calendar: {
        top: 42,
        left: 46,
        right: 20,
        bottom: 56,
        range: String(year),
        cellSize: ["auto", "auto"],
        splitLine: { show: false },
        itemStyle: {
          color: theme.isDark ? "#0e1728" : "#f6f7fb",
          borderColor: theme.surface,
          borderWidth: 1.5,
          borderRadius: 3,
        },
        yearLabel: { show: false },
        dayLabel: {
          firstDay: 1,
          nameMap: chinese ? "ZH" : "EN",
          color: theme.textSoft,
          fontSize: 10,
        },
        monthLabel: {
          nameMap: monthNames,
          color: theme.textSoft,
          fontSize: 10,
        },
      },
      series: [
        {
          type: "heatmap",
          coordinateSystem: "calendar",
          data: cells,
          itemStyle: { borderRadius: 3 },
          emphasis: {
            itemStyle: {
              borderColor: theme.isDark ? "#6ee7b7" : "#059669",
              borderWidth: 1.5,
            },
          },
        },
      ],
    };
  }, [data, locale, t, theme, year]);

  return (
    <EChart
      echarts={echarts}
      option={option}
      notMerge
      lazyUpdate
      style={{ width: "100%", height: 268 }}
    />
  );
}
