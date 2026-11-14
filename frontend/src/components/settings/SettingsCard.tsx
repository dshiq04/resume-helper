import type { ReactNode } from "react";

interface SettingsCardProps {
  icon: ReactNode;
  tone?: "default" | "security" | "data";
  title: string;
  description?: string;
  extra?: ReactNode;
  children: ReactNode;
}

export function SettingsCard({
  icon,
  tone = "default",
  title,
  description,
  extra,
  children,
}: SettingsCardProps) {
  return (
    <section className={`settings-card settings-card-${tone}`}>
      <header className="settings-card-head">
        <span
          className={`settings-section-icon ${
            tone === "security"
              ? "is-security"
              : tone === "data"
                ? "is-data"
                : ""
          }`}
        >
          {icon}
        </span>
        <div>
          <h3>{title}</h3>
          {description && <p>{description}</p>}
        </div>
        {extra && <div className="settings-card-extra">{extra}</div>}
      </header>
      <div className="settings-card-body">{children}</div>
    </section>
  );
}
