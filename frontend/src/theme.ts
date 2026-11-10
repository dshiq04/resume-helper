import { theme as antdTheme, type ThemeConfig } from "antd";

export type ThemeMode = "light" | "dark";

const FONT_STACK =
  '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans SC", "Noto Sans JP", "Noto Sans KR", sans-serif';

export function createTheme(mode: ThemeMode): ThemeConfig {
  const dark = mode === "dark";
  return {
    algorithm: dark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
    token: {
      colorPrimary: "#6366f1",
      colorInfo: "#6366f1",
      colorSuccess: "#10b981",
      colorWarning: "#f59e0b",
      colorError: "#ef4444",
      colorTextBase: dark ? "#e5e7eb" : "#0f172a",
      colorBgBase: dark ? "#0b1020" : "#f5f6fb",
      colorBgContainer: dark ? "#111827" : "#ffffff",
      colorBgElevated: dark ? "#151f33" : "#ffffff",
      colorBorder: dark ? "#263247" : "#e7e9f2",
      colorBorderSecondary: dark ? "#1f2a3d" : "#eef0f7",
      borderRadius: 12,
      borderRadiusLG: 18,
      controlHeight: 40,
      fontFamily: FONT_STACK,
      fontSize: 14,
      boxShadowSecondary: dark
        ? "0 18px 50px rgba(0, 0, 0, 0.45)"
        : "0 18px 50px rgba(30, 41, 59, 0.12)",
    },
    components: {
      Button: {
        fontWeight: 600,
        primaryShadow: "0 10px 24px rgba(99, 102, 241, 0.28)",
        defaultBorderColor: dark ? "#2a3650" : "#e3e6f0",
      },
      Input: {
        activeShadow: "0 0 0 3px rgba(99, 102, 241, 0.14)",
        paddingInline: 14,
      },
      InputNumber: {
        activeShadow: "0 0 0 3px rgba(99, 102, 241, 0.14)",
      },
      Select: {
        optionSelectedBg: dark ? "rgba(99, 102, 241, 0.18)" : "#eef2ff",
        optionActiveBg: dark ? "rgba(148, 163, 184, 0.12)" : "#f6f7fb",
      },
      Table: {
        headerBg: "transparent",
        headerColor: dark ? "#94a3b8" : "#64748b",
        headerSplitColor: "transparent",
        borderColor: dark ? "#1f2a3d" : "#f0f1f7",
        rowHoverBg: dark ? "#16223a" : "#f8f9ff",
        headerBorderRadius: 0,
        cellPaddingBlock: 15,
        cellPaddingInline: 14,
      },
      Modal: {
        contentBg: dark ? "#111827" : "#ffffff",
        headerBg: "transparent",
        borderRadiusLG: 22,
      },
      Drawer: {
        colorBgElevated: dark ? "#0f1729" : "#ffffff",
      },
      Tooltip: {
        colorBgSpotlight: dark ? "#f8fafc" : "#0f172a",
        colorTextLightSolid: dark ? "#0f172a" : "#ffffff",
      },
      Segmented: {
        itemSelectedBg: dark ? "#1e293b" : "#ffffff",
        itemSelectedColor: dark ? "#c7d2fe" : "#4338ca",
        trackBg: dark ? "#0b1220" : "#eef0f7",
      },
      Popconfirm: {
        colorBgElevated: dark ? "#151f33" : "#ffffff",
      },
      Message: {
        contentBg: dark ? "#151f33" : "#ffffff",
      },
    },
  };
}
