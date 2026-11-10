import { useMemo } from "react";
import {
  ClockCircleOutlined,
  FileTextOutlined,
  TeamOutlined,
  TrophyOutlined,
} from "@ant-design/icons";
import type { Stats } from "../api/types";
import { useI18n } from "../i18n";
import type { CategoryFilter } from "../components/Sidebar";

interface StatsCardsProps {
  stats: Stats | null;
  category: CategoryFilter;
}

export function StatsCards({ stats, category }: StatsCardsProps) {
  const { t } = useI18n();

  const counts = useMemo(() => {
    if (!stats) {
      return { total: 0, active: 0, interview: 0, offer: 0 };
    }
    if (category === "all") {
      return {
        total: stats.total,
        active: stats.byStatus.screening + stats.byStatus.written,
        interview: stats.byStatus.interview,
        offer: stats.byStatus.offer,
      };
    }
    const byStatus = stats.byCategoryStatus[category];
    return {
      total: stats.byCategory[category] ?? 0,
      active: (byStatus?.screening ?? 0) + (byStatus?.written ?? 0),
      interview: byStatus?.interview ?? 0,
      offer: byStatus?.offer ?? 0,
    };
  }, [category, stats]);

  const cards = [
    {
      key: "total",
      label: t("stats.total"),
      hint: t("stats.totalHint"),
      value: counts.total,
      icon: <FileTextOutlined />,
      accent: "#6366f1",
    },
    {
      key: "active",
      label: t("stats.active"),
      hint: t("stats.activeHint"),
      value: counts.active,
      icon: <ClockCircleOutlined />,
      accent: "#f59e0b",
    },
    {
      key: "interview",
      label: t("stats.interview"),
      hint: t("stats.interviewHint"),
      value: counts.interview,
      icon: <TeamOutlined />,
      accent: "#0ea5e9",
    },
    {
      key: "offer",
      label: t("stats.offer"),
      hint: t("stats.offerHint"),
      value: counts.offer,
      icon: <TrophyOutlined />,
      accent: "#10b981",
    },
  ];

  return (
    <div className="stats-grid">
      {cards.map((card) => (
        <div
          key={card.key}
          className={`stat-card stat-card-${card.key}`}
          style={{ ["--card-accent" as string]: card.accent }}
        >
          <div className="stat-card-top">
            <span className="stat-card-icon">{card.icon}</span>
            <span className="stat-card-dot" />
          </div>
          <div className="stat-card-value">{card.value}</div>
          <div className="stat-card-label">{card.label}</div>
          <div className="stat-card-hint">{card.hint}</div>
        </div>
      ))}
    </div>
  );
}
