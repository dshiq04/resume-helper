import { useMemo } from "react";
import type { EChartsCoreOption } from "echarts/core";
import type { Status } from "../../api/types";
import type { MessageKey } from "../../i18n";
import type { ThemeMode } from "../../theme";
import { EChart, echarts, getChartTheme, tooltipStyle } from "./echarts";

interface StatusDonutProps {
  counts: Record<Status, number> | undefined;
  themeMode: ThemeMode;
  t: (key: MessageKey, params?: Record<string, string | number>) => string;
}

interface StatusGroup {
  key: string;
  labelKey: MessageKey;
  color: string;
  value: number;
}

export function StatusDonut({ counts, themeMode, t }: StatusDonutProps) {
  const theme = useMemo(() => getChartTheme(themeMode), [themeMode]);

  const groups = useMemo<StatusGroup[]>(() => {
    const source = counts ?? {
      screening: 0,
      written: 0,
      interview: 0,
      pending: 0,
      offer: 0,
      rejected: 0,
    };
    return [
      {
        key: "active",
        labelKey: "stats.active",
        color: "#f59e0b",
        value: source.screening + source.written + source.pending,
      },
      {
        key: "interview",
        labelKey: "stats.interview",
        color: "#0ea5e9",
        value: source.interview,
      },
      {
        key: "offer",
        labelKey: "stats.offer",
        color: "#10b981",
        value: source.offer,
      },
      {
        key: "rejected",
        labelKey: "status.rejected",
        color: "#ef4444",
        value: source.rejected,
      },
    ];
  }, [counts]);

  const total = useMemo(
    () => groups.reduce((sum, group) => sum + group.value, 0),
    [groups],
  );

  const option = useMemo<EChartsCoreOption>(() => {
    const data = groups
      .filter((group) => group.value > 0)
      .map((group) => ({
        name: t(group.labelKey),
        value: group.value,
        itemStyle: { color: group.color },
      }));

    return {
      animationDuration: 700,
      animationEasing: "cubicOut",
      tooltip: {
        trigger: "item",
        ...tooltipStyle(theme),
        formatter: (params) => {
          return (
            `<div style="display:flex;align-items:center;gap:7px;font-weight:600;">` +
            `${params.marker}${params.name}</div>` +
            `<div style="margin-top:6px;color:${theme.textSoft};">${t("stats.applicationCount")}` +
            `<strong style="color:${theme.text};margin-left:8px;">${params.value}</strong>` +
            `<span style="margin-left:6px;">(${params.percent}%)</span></div>`
          );
        },
      },
      series: [
        {
          type: "pie",
          radius: ["61%", "82%"],
          center: ["50%", "50%"],
          startAngle: 90,
          minAngle: 3,
          avoidLabelOverlap: false,
          label: { show: false },
          labelLine: { show: false },
          itemStyle: {
            borderColor: theme.surface,
            borderWidth: 3,
            borderRadius: 6,
          },
          emphasis: { scale: true, scaleSize: 4 },
          data,
        },
      ],
    };
  }, [groups, t, theme]);

  return (
    <div className="stats-donut">
      <div className="stats-donut-chart">
        <EChart
          echarts={echarts}
          option={option}
          notMerge
          lazyUpdate
          style={{ width: "100%", height: 218 }}
        />
        <div className="stats-donut-center">
          <strong>{total}</strong>
          <span>{t("stats.total")}</span>
        </div>
      </div>

      <ul className="stats-donut-legend">
        {groups.map((group) => {
          const percent = total > 0 ? Math.round((group.value / total) * 100) : 0;
          return (
            <li key={group.key}>
              <span
                className="stats-donut-dot"
                style={{ background: group.color }}
              />
              <span className="stats-donut-label">{t(group.labelKey)}</span>
              <span className="stats-donut-value">{group.value}</span>
              <span className="stats-donut-percent">{percent}%</span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
