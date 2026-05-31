import type { ButtonHTMLAttributes, ReactNode } from 'react';

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'secondary';
  icon?: ReactNode;
};

export function Button({ variant = 'primary', icon, className = '', children, ...props }: ButtonProps) {
  const variantClass = variant === 'primary' ? 'btn-primary' : 'btn-secondary';

  return (
    <button className={`btn ${variantClass} ${className}`.trim()} {...props}>
      {icon}
      {children}
    </button>
  );
}