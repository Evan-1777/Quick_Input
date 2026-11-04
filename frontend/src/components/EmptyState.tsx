import { FolderIcon } from './Icons';

interface EmptyStateProps {
  title: string;
  actionLabel: string;
  onAction: () => void;
  secondaryLabel?: string;
  onSecondary?: () => void;
}

export default function EmptyState({ title, actionLabel, onAction, secondaryLabel, onSecondary }: EmptyStateProps) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
      <div className="flex h-10 w-10 items-center justify-center rounded-[8px] border border-line bg-surface text-faint">
        <FolderIcon className="h-[18px] w-[18px]" />
      </div>
      <p className="text-[12.5px] text-muted">{title}</p>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onAction}
          className="h-7 rounded-[4px] bg-brand px-3 text-[12px] font-medium text-white hover:bg-[#106ebe] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand"
        >
          {actionLabel}
        </button>
        {secondaryLabel && onSecondary ? (
          <button
            type="button"
            onClick={onSecondary}
            className="h-7 rounded-[4px] border border-line bg-surface px-3 text-[12px] text-ink hover:bg-black/[0.04]"
          >
            {secondaryLabel}
          </button>
        ) : null}
      </div>
    </div>
  );
}
