import { useMemo } from "react";
import type { EChartsCoreOption } from "echarts/core";
import type { FunnelStats } from "../../api/types";
import type { MessageKey } from "../../i18n";
import type { ThemeMode } from "../../theme";
import {
  CHART_PALETTE,
  EChart,
  echarts,
  getChartTheme,
  tooltipStyle,
} from "./echarts";

interface FunnelChartProps {
  data: FunnelStats;
  themeMode: ThemeMode;
  t: (key: MessageKey, params?: Record<string, string | number>) => string;
}

export function FunnelChart({ data, themeMode, t }: FunnelChartProps) {
  const theme = useMemo(() => getChartTheme(themeMode), [themeMode]);

  const option = useMemo<EChartsCoreOption>(() => {
    const items = data.stages.map((stage, index) => ({
      name:
        stage.key === "applied"
          ? t("funnel.applied")
          : t(`status.${stage.key}` as MessageKey),
      value: stage.count,
      conversion: stage.conversion,
      itemStyle: { color: CHART_PALETTE[index % CHART_PALETTE.length] },
    }));

    return {
      animationDuration: 700,
      animationEasing: "cubicOut",
      tooltip: {
        trigger: "item",
        ...tooltipStyle(theme),
        formatter: (params: {
          name: string;
          value: number;
          marker: string;
          data: { conversion: number };
        }) =>
          `<div style="display:flex;align-items:center;gap:7px;font-weight:600;">` +
          `${params.marker}${params.name}</div>` +
          `<div style="margin-top:6px;color:${theme.textSoft};">${t("stats.totalApplications")}` +
          `<strong style="color:${theme.text};margin-left:8px;">${params.value}</strong></div>` +
          `<div style="margin-top:4px;color:${theme.textSoft};">${t("stats.interviewRate")}` +
          `<strong style="color:${theme.text};margin-left:8px;">${params.data.conversion}%</strong></div>`,
      },
      series: [
        {
          type: "funnel",
          top: 14,
          bottom: 14,
          left: "8%",
          right: "8%",
          minSize: "26%",
          maxSize: "100%",
          sort: "none",
          gap: 3,
          label: {
            show: true,
            position: "inside",
            color: "#ffffff",
            fontSize: 12,
            fontWeight: 600,
            formatter: (params: { name: string; value: number }) =>
              `${params.name}  ${params.value}`,
          },
          labelLine: { show: false },
          itemStyle: {
            borderColor: theme.surface,
            borderWidth: 2,
            borderRadius: 6,
          },
          emphasis: { label: { fontSize: 13 } },
          data: items,
        },
      ],
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
