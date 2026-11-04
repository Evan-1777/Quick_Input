import { useEffect, useMemo, useRef, useState } from 'react';

import { filterSnippets, nextActiveIndex } from '../filter.ts';
import type { AppActions, FilterState, Group, Snippet, Tag } from '../types.ts';
import EmptyState from './EmptyState';
import { SearchIcon, WindowIcon } from './Icons';

interface QuickOverlayProps {
  snippets: Snippet[];
  groups: Group[];
  tags: Tag[];
  actions: AppActions;
}

const initialFilter: FilterState = { keyword: '', groupScope: 'all', groupId: null, tagIds: [] };

export default function QuickOverlay({ snippets, groups, tags, actions }: QuickOverlayProps) {
  const [filter, setFilter] = useState<FilterState>(initialFilter);
  const [activeIndex, setActiveIndex] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => filterSnippets(snippets, filter), [snippets, filter]);
  const hasFilter = filter.keyword !== '' || filter.groupScope !== 'all' || filter.tagIds.length > 0;

  useEffect(() => {
    setActiveIndex(0);
  }, [filter]);

  useEffect(() => {
    const node = listRef.current?.children[activeIndex];
    if (node instanceof HTMLElement) {
      node.scrollIntoView({ block: 'nearest' });
    }
  }, [activeIndex]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.isComposing) {
        return;
      }
      if (event.key === 'ArrowDown') {
        event.preventDefault();
        setActiveIndex((current) => nextActiveIndex(current, filtered.length, 'down'));
      } else if (event.key === 'ArrowUp') {
        event.preventDefault();
        setActiveIndex((current) => nextActiveIndex(current, filtered.length, 'up'));
      } else if (event.key === 'Enter') {
        const target = filtered[activeIndex];
        if (target) {
          event.preventDefault();
          void actions.triggerPaste(target.id);
        }
      } else if (event.key === 'Escape') {
        event.preventDefault();
        void actions.hideWindow();
      }
    };
    const onBlur = () => {
      void actions.hideWindow();
    };

    window.addEventListener('keydown', onKeyDown);
    window.addEventListener('blur', onBlur);
    return () => {
      window.removeEventListener('keydown', onKeyDown);
      window.removeEventListener('blur', onBlur);
    };
  }, [filtered, activeIndex, actions]);

  const selectGroup = (scope: FilterState['groupScope'], groupId: number | null) => {
    setFilter((prev) => ({ ...prev, groupScope: scope, groupId }));
    searchRef.current?.focus();
  };

  const toggleTag = (tagId: number) => {
    setFilter((prev) => ({
      ...prev,
      tagIds: prev.tagIds.includes(tagId) ? prev.tagIds.filter((id) => id !== tagId) : [...prev.tagIds, tagId],
    }));
    searchRef.current?.focus();
  };

  const clearFilter = () => {
    setFilter(initialFilter);
    searchRef.current?.focus();
  };

  return (
    <div className="animate-overlay-in flex h-full w-full flex-col overflow-hidden border border-[#d8d8d8] bg-[linear-gradient(160deg,#fcfcfc_0%,#f2f2f2_100%)]">
      <div className="flex min-h-0 flex-1">
        <section className="flex w-[62%] min-w-0 flex-col">
          <div className="flex h-12 shrink-0 items-center gap-2 border-b border-line px-3.5">
            <SearchIcon className="h-4 w-4 shrink-0 text-faint" />
            <input
              ref={searchRef}
              autoFocus
              value={filter.keyword}
              onChange={(event) => setFilter((prev) => ({ ...prev, keyword: event.target.value }))}
              placeholder="搜索文本或内容"
              spellCheck={false}
              className="h-full min-w-0 flex-1 bg-transparent text-[13.5px] text-ink outline-none"
            />
            <kbd className="shrink-0 rounded-[3px] border border-line bg-surface px-1 text-[10px] leading-[16px] text-faint">
              Esc
            </kbd>
          </div>

          <div ref={listRef} className="min-h-0 flex-1 overflow-y-auto p-1.5">
            {filtered.length === 0 ? (
              <EmptyState
                title={hasFilter ? '没有匹配的预设文本' : '暂无预设文本'}
                actionLabel={hasFilter ? '清除筛选' : '新建预设文本'}
                onAction={hasFilter ? clearFilter : () => void actions.switchToManager(true)}
              />
            ) : (
              filtered.map((snippet, index) => {
                const active = index === activeIndex;
                return (
                  <button
                    type="button"
                    key={snippet.id}
                    onMouseEnter={() => setActiveIndex(index)}
                    onClick={() => void actions.triggerPaste(snippet.id)}
                    className={`mb-0.5 flex w-full flex-col rounded-[5px] px-2.5 py-2 text-left transition-colors ${
                      active ? 'bg-brand-soft' : 'hover:bg-black/[0.035]'
                    }`}
                  >
                    <div className="flex w-full items-baseline justify-between gap-3">
                      <span className={`truncate text-[13px] ${active ? 'font-semibold text-brand' : 'font-medium text-ink'}`}>
                        {snippet.title}
                      </span>
                      {snippet.groupName ? (
                        <span className="shrink-0 text-[10.5px] text-faint">{snippet.groupName}</span>
                      ) : null}
                    </div>
                    <p className="mt-0.5 w-full truncate text-[12px] leading-[17px] text-muted">{snippet.content}</p>
                    {snippet.tags.length > 0 ? (
                      <div className="mt-1 flex flex-wrap gap-1">
                        {snippet.tags.map((tag) => (
                          <span
                            key={tag.id}
                            className="rounded-[3px] border border-line bg-surface px-1.5 py-px text-[10px] text-muted"
                          >
                            {tag.name}
                          </span>
                        ))}
                      </div>
                    ) : null}
                  </button>
                );
              })
            )}
          </div>

          <div className="flex h-8 shrink-0 items-center gap-3 border-t border-line px-3.5 text-[10.5px] text-faint">
            <span className="flex items-center gap-1">
              <kbd className="rounded-[3px] border border-line bg-surface px-1 leading-[15px]">↑</kbd>
              <kbd className="rounded-[3px] border border-line bg-surface px-1 leading-[15px]">↓</kbd>
              选择
            </span>
            <span className="flex items-center gap-1">
              <kbd className="rounded-[3px] border border-line bg-surface px-1 leading-[15px]">Enter</kbd>
              键入
            </span>
          </div>
        </section>

        <aside className="flex w-[38%] min-w-0 flex-col border-l border-line bg-[#f8f8f8]">
          <div className="min-h-0 flex-1 overflow-y-auto px-3 py-3">
            <p className="text-[10.5px] font-semibold tracking-wide text-faint">分组</p>
            <ul className="mt-1 space-y-px">
              <FilterRow label="全部" active={filter.groupScope === 'all'} onClick={() => selectGroup('all', null)} />
              <FilterRow
                label="未分组"
                active={filter.groupScope === 'ungrouped'}
                onClick={() => selectGroup('ungrouped', null)}
              />
              {groups.map((group) => (
                <FilterRow
                  key={group.id}
                  label={group.name}
                  active={filter.groupScope === 'group' && filter.groupId === group.id}
                  onClick={() => selectGroup('group', group.id)}
                />
              ))}
            </ul>

            <p className="mt-4 text-[10.5px] font-semibold tracking-wide text-faint">标签</p>
            <div className="mt-1.5 flex flex-wrap gap-1.5">
              {tags.length === 0 ? <span className="text-[11px] text-faint">暂无标签</span> : null}
              {tags.map((tag) => {
                const selected = filter.tagIds.includes(tag.id);
                return (
                  <button
                    key={tag.id}
                    type="button"
                    onClick={() => toggleTag(tag.id)}
                    aria-pressed={selected}
                    className={`rounded-[4px] border px-1.5 py-0.5 text-[11px] transition-colors ${
                      selected
                        ? 'border-brand bg-brand text-white'
                        : 'border-line bg-surface text-muted hover:border-[#c9c9c9] hover:text-ink'
                    }`}
                  >
                    {tag.name}
                  </button>
                );
              })}
            </div>
            {filter.tagIds.length > 0 ? (
              <button type="button" onClick={clearFilter} className="mt-2 text-[11px] text-brand hover:underline">
                清除筛选
              </button>
            ) : null}
          </div>

          <div className="border-t border-line p-2">
            <button
              type="button"
              onClick={() => void actions.switchToManager(false)}
              className="flex w-full items-center justify-center gap-1.5 rounded-[4px] py-1.5 text-[11.5px] text-muted hover:bg-black/[0.04] hover:text-ink"
            >
              <WindowIcon className="h-3.5 w-3.5" />
              管理窗口
            </button>
          </div>
        </aside>
      </div>
    </div>
  );
}

function FilterRow({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <li>
      <button
        type="button"
        onClick={onClick}
        aria-current={active}
        className={`w-full truncate rounded-[4px] px-2 py-[5px] text-left text-[12px] transition-colors ${
          active ? 'bg-brand-soft font-medium text-brand' : 'text-muted hover:bg-black/[0.04] hover:text-ink'
        }`}
      >
        {label}
      </button>
    </li>
  );
}
