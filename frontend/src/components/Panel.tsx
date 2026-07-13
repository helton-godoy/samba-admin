import type { ReactNode } from 'react';

export function Panel({ title, subtitle, actions, children, tone }: { title: string; subtitle?: string; actions?: ReactNode; children: ReactNode; tone?: 'warning' | 'danger' | 'info' }) {
  return (
    <section className={`panel ${tone ? `panel-${tone}` : ''}`}>
      <div className="panel-header">
        <div>
          <h2>{title}</h2>
          {subtitle && <p>{subtitle}</p>}
        </div>
        {actions && <div>{actions}</div>}
      </div>
      <div className="panel-body">{children}</div>
    </section>
  );
}
