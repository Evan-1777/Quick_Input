import type { ButtonHTMLAttributes, ReactNode } from 'react';

type Variant = 'primary' | 'secondary' | 'danger';

const variants: Record<Variant, string> = {
  primary: 'bg-brand text-white hover:bg-[#106ebe]',
  secondary: 'border border-line bg-surface text-ink hover:bg-black/[0.04]',
  danger: 'bg-danger text-white hover:bg-[#a4262c]',
};

export function Button({
  variant = 'secondary',
  className = '',
  children,
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }) {
  return (
    <button
      type="button"
      className={`inline-flex h-7 items-center justify-center gap-1 rounded-[4px] px-3 text-[12px] font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand disabled:cursor-not-allowed disabled:opacity-45 ${variants[variant]} ${className}`}
      {...rest}
    >
      {children}
    </button>
  );
}

export function IconButton({
  label,
  className = '',
  children,
  onClick,
  disabled,
  size = 'md',
}: {
  label: string;
  className?: string;
  children: ReactNode;
  onClick: () => void;
  disabled?: boolean;
  size?: 'sm' | 'md';
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      onClick={onClick}
      disabled={disabled}
      className={`inline-flex items-center justify-center rounded-[4px] text-muted transition-colors hover:bg-black/[0.06] hover:text-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand disabled:opacity-40 ${
        size === 'sm' ? 'h-5 w-5' : 'h-7 w-7'
      } ${className}`}
    >
      {children}
    </button>
  );
}
