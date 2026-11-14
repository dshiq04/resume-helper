import * as echarts from "echarts/core";
import { HeatmapChart, LineChart, PieChart } from "echarts/charts";
import {
  CalendarComponent,
  GridComponent,
  LegendComponent,
  TooltipComponent,
  VisualMapComponent,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";
import ReactEChartsCoreImport from "echarts-for-react/lib/core";
import type { EChartsReactProps } from "echarts-for-react/lib/types";
import type { ThemeMode } from "../../theme";
import { createElement } from "react";

// Only the pieces the statistics page needs are registered so the bundle does
// not pull in every ECharts chart type.
echarts.use([
  LineChart,
  PieChart,
  HeatmapChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  CalendarComponent,
  VisualMapComponent,
  CanvasRenderer,
]);

export { echarts };

// Vite/Rolldown can hand back the CommonJS module wrapper instead of its
// default export. Unwrap it so React receives a component, not an object.
const ReactEChartsCore =
  (
    ReactEChartsCoreImport as unknown as {
      default?: typeof ReactEChartsCoreImport;
    }
  ).default ?? ReactEChartsCoreImport;

export function EChart(props: EChartsReactProps) {
  return createElement(ReactEChartsCore, props);
}

export const CHART_PALETTE = [
  "#6366f1",
  "#0ea5e9",
  "#10b981",
  "#f59e0b",
  "#ec4899",
  "#8b5cf6",
  "#14b8a6",
  "#f97316",
];

export interface ChartTheme {
  isDark: boolean;
  text: string;
  textSoft: string;
  split: string;
  surface: string;
  tooltipBg: string;
  tooltipText: string;
}

export function getChartTheme(mode: ThemeMode): ChartTheme {
  const isDark = mode === "dark";
  return {
    isDark,
    text: isDark ? "#e8edf7" : "#0f172a",
    textSoft: isDark ? "#94a3b8" : "#64748b",
    split: isDark ? "#1f2a3d" : "#eceef6",
    surface: isDark ? "#111827" : "#ffffff",
    tooltipBg: isDark ? "#151f33" : "#ffffff",
    tooltipText: isDark ? "#e8edf7" : "#0f172a",
  };
}

export const tooltipStyle = (theme: ChartTheme) => ({
  backgroundColor: theme.tooltipBg,
  borderColor: theme.split,
  borderWidth: 1,
  padding: [10, 13] as [number, number],
  textStyle: {
    color: theme.tooltipText,
    fontSize: 12,
  },
  extraCssText:
    "border-radius: 12px; box-shadow: 0 14px 36px rgba(15, 23, 42, 0.16);",
});
