import { useState } from 'react';

import { CheckIcon, ChevronLeftIcon } from './Icons';
import { IconButton } from './ui';

const hotkeyOptions = ['Alt+Space', 'Alt+Q', 'Ctrl+Alt+Space', 'Ctrl+Alt+Q', 'Ctrl+Shift+Space', 'Ctrl+Shift+Q'];

interface SettingsPageProps {
  hotkey: string;
  onBack: () => void;
  onSelect: (spec: string) => Promise<void>;
}

export default function SettingsPage({ hotkey, onBack, onSelect }: SettingsPageProps) {
  const [pending, setPending] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const active = pending ?? hotkey;

  const select = async (spec: string) => {
    if (busy || spec === hotkey) {
      return;
    }
    setPending(spec);
    setError(null);
    setBusy(true);
    try {
      await onSelect(spec);
      setPending(null);
    } catch (err) {
      setPending(null);
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex h-12 shrink-0 items-center gap-1.5 border-b border-line bg-surface px-2.5">
        <IconButton label="返回预设文本" onClick={onBack}>
          <ChevronLeftIcon className="h-3.5 w-3.5" />
        </IconButton>
        <span className="text-[13px] font-semibold text-ink">设置</span>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto bg-canvas px-6 py-5">
        <div className="mx-auto w-full max-w-[560px]">
          <h2 className="mb-2 px-0.5 text-[11.5px] font-semibold text-muted">快捷键</h2>
          <section className="overflow-hidden rounded-[8px] border border-line bg-surface">
            <div className="px-4 py-3.5">
              <p className="text-[13px] font-medium text-ink">唤出快捷键</p>
              <p className="mt-0.5 text-[11.5px] text-muted">在任意应用中唤出快捷输入面板。</p>
            </div>
            <div role="radiogroup" aria-label="唤出快捷键">
              {hotkeyOptions.map((spec) => {
                const selected = active === spec;
                return (
                  <label
                    key={spec}
                    className={`flex items-center gap-3 border-t border-line/70 px-4 py-2.5 transition-colors focus-within:outline focus-within:outline-2 focus-within:-outline-offset-2 focus-within:outline-brand ${
                      selected ? 'bg-brand-soft/60 text-brand' : 'text-ink hover:bg-black/[0.025]'
                    } ${busy ? 'cursor-not-allowed' : 'cursor-pointer'}`}
                  >
                    <input
                      type="radio"
                      name="hotkey"
                      className="sr-only"
                      checked={selected}
                      disabled={busy}
                      onChange={() => void select(spec)}
                    />
                    <span className="flex min-w-0 flex-1 flex-wrap items-center gap-1">
                      {spec.split('+').map((part) => (
                        <kbd
                          key={part}
                          className="rounded-[3px] border border-line bg-surface px-1 text-[10.5px] leading-[17px] text-faint"
                        >
                          {part}
                        </kbd>
                      ))}
                    </span>
                    {selected ? (
                      <span className="flex shrink-0 items-center gap-1 text-[11.5px] font-medium text-brand">
                        <CheckIcon className="h-3 w-3" />
                        使用中
                      </span>
                    ) : null}
                  </label>
                );
              })}
            </div>
            {error ? (
              <p role="alert" className="border-t border-line/70 px-4 py-2.5 text-[11.5px] text-danger">
                {error}
              </p>
            ) : null}
          </section>
        </div>
      </div>
    </div>
  );
}
