import type { ReactNode } from 'react';

export function MetricCard({ title, value, detail, icon }: { title: string; value: ReactNode; detail?: string; icon?: ReactNode }) {
  return (
    <article className="metric-card">
      <div className="metric-card__icon" aria-hidden="true">{icon}</div>
      <div>
        <p className="eyebrow">{title}</p>
        <div className="metric-value">{value}</div>
        {detail && <p className="muted">{detail}</p>}
      </div>
    </article>
  );
}
