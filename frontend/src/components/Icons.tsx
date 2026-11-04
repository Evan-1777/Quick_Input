import type { ReactNode } from 'react';

interface IconProps {
  className?: string;
}

function Svg({ children, className = 'h-4 w-4' }: IconProps & { children: ReactNode }) {
  return (
    <svg
      viewBox="0 0 16 16"
      className={className}
      fill="none"
      stroke="currentColor"
      strokeWidth={1.3}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {children}
    </svg>
  );
}

export function SearchIcon(props: IconProps) {
  return (
    <Svg {...props}>
      <circle cx="6.8" cy="6.8" r="4.3" />
      <path d="M10.1 10.1 13.5 13.5" />
    </Svg>
  );
}

export function PlusIcon(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M8 3.2v9.6M3.2 8h9.6" />
    </Svg>
  );
}

export function CloseIcon(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M3.6 3.6l8.8 8.8M12.4 3.6l-8.8 8.8" />
    </Svg>
  );
}

export function MinimizeIcon(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M3.2 8h9.6" />
    </Svg>
  );
}

export function PencilIcon(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M9.9 2.9a1.6 1.6 0 0 1 2.3 0l.9.9a1.6 1.6 0 0 1 0 2.3l-6.7 6.7-2.9.6.6-2.9z" />
    </Svg>
  );
}

export function TrashIcon(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M2.8 4.4h10.4M6 4.4V2.9h4v1.5M4.2 4.4l.6 8.2a.9.9 0 0 0 .9.8h4.6a.9.9 0 0 0 .9-.8l.6-8.2" />
    </Svg>
  );
}

export function FolderIcon(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M2.2 4.3A1.3 1.3 0 0 1 3.5 3h2.6l1.4 1.6h5A1.3 1.3 0 0 1 13.8 6v5.7A1.3 1.3 0 0 1 12.5 13h-9a1.3 1.3 0 0 1-1.3-1.3z" />
    </Svg>
  );
}

export function TagIcon(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M2.4 2.4h4.8L14 9.2l-4.8 4.8L2.4 7.2z" />
      <circle cx="5.4" cy="5.4" r="1" />
    </Svg>
  );
}

export function WindowIcon(props: IconProps) {
  return (
    <Svg {...props}>
      <rect x="2.2" y="3.2" width="11.6" height="9.6" rx="1.3" />
      <path d="M2.2 6.4h11.6" />
    </Svg>
  );
}

export function ChevronDownIcon(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M4.2 6.2 8 10l3.8-3.8" />
    </Svg>
  );
}

export function CheckIcon(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M3 8.4l3.1 3.1L13 4.6" />
    </Svg>
  );
}

export function RetryIcon(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M13 8a5 5 0 1 1-1.6-3.7" />
      <path d="M13.2 2.8v3.4H9.8" />
    </Svg>
  );
}
