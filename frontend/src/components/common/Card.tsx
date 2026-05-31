import type { ReactNode } from 'react';

type CardProps = {
  title?: string;
  subtitle?: string;
  action?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
  className?: string;
};

export function Card({ title, subtitle, action, footer, children, className = '' }: CardProps) {
  return (
    <section className={`surface-card p-4 md:p-5 ${className}`.trim()}>
      {(title || subtitle || action) && (
        <div className="mb-4 flex items-start justify-between gap-4">
          <div>
            {title && <h3 className="text-lg font-semibold text-text-primary">{title}</h3>}
            {subtitle && <p className="mt-1 text-sm text-text-muted">{subtitle}</p>}
          </div>
          {action}
        </div>
      )}
      {children}
      {footer && <div className="mt-4 border-t border-dark-elevated pt-4">{footer}</div>}
    </section>
  );
}