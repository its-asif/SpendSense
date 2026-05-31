import type { InputHTMLAttributes, ReactNode } from 'react';

type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  label?: string;
  error?: string;
  icon?: ReactNode;
};

export function Input({ label, error, icon, className = '', id, ...props }: InputProps) {
  const inputId = id ?? label?.toLowerCase().replace(/\s+/g, '-') ?? undefined;

  return (
    <label className="block">
      {label && <span className="mb-1.5 block text-xs font-semibold text-text-secondary">{label}</span>}
      <div className="relative">
        {icon && <span className="pointer-events-none absolute inset-y-0 left-3 flex items-center text-text-muted">{icon}</span>}
        <input id={inputId} className={`input ${icon ? 'pl-10' : ''} ${className}`.trim()} {...props} />
      </div>
      {error && <span className="mt-1 block text-xs text-accent-red">{error}</span>}
    </label>
  );
}