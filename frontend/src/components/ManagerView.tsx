import { useEffect, useMemo, useRef, useState } from 'react';
import type { CSSProperties, ReactNode } from 'react';

import { filterSnippets, toggleSelection } from '../filter.ts';
import type { AppActions, FilterState, Group, Snippet, Tag } from '../types.ts';
import EditorDrawer from './EditorDrawer';
import EmptyState from './EmptyState';
import {
  ChevronDownIcon,
  CloseIcon,
  MinimizeIcon,
  PencilIcon,
  PlusIcon,
  SearchIcon,
  TagIcon,
  TrashIcon,
  WindowIcon,
} from './Icons';
import { Button, IconButton } from './ui';

interface ManagerViewProps {
  groups: Group[];
  tags: Tag[];
  snippets: Snippet[];
  creatorRequested: boolean;
  onCreatorHandled: () => void;
  actions: AppActions;
}

const initialFilter: FilterState = { keyword: '', groupScope: 'all', groupId: null, tagIds: [] };

interface ConfirmState {
  title: string;
  description: string;
  actionLabel: string;
  run: () => Promise<void>;
}

const dragStyle = { '--wails-draggable': 'drag' } as CSSProperties;
const noDragStyle = { '--wails-draggable': 'no-drag' } as CSSProperties;

export default function ManagerView({
  groups,
  tags,
  snippets,
  creatorRequested,
  onCreatorHandled,
  actions,
}: ManagerViewProps) {
  const [filter, setFilter] = useState<FilterState>(initialFilter);
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [editor, setEditor] = useState<{ snippet: Snippet | null } | null>(null);
  const [confirm, setConfirm] = useState<ConfirmState | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [addingGroup, setAddingGroup] = useState(false);
  const [addingTag, setAddingTag] = useState(false);
  const [renaming, setRenaming] = useState<{ kind: 'group' | 'tag'; id: number; value: string } | null>(null);
  const selectAllRef = useRef<HTMLInputElement>(null);

  const filtered = useMemo(() => filterSnippets(snippets, filter), [snippets, filter]);
  const hasFilter = filter.keyword !== '' || filter.groupScope !== 'all' || filter.tagIds.length > 0;
  const allVisibleSelected = filtered.length > 0 && filtered.every((snippet) => selectedIds.includes(snippet.id));
  const someVisibleSelected = filtered.some((snippet) => selectedIds.includes(snippet.id)) && !allVisibleSelected;

  useEffect(() => {
    if (creatorRequested) {
      setEditor({ snippet: null });
      onCreatorHandled();
    }
  }, [creatorRequested, onCreatorHandled]);

  useEffect(() => {
    setSelectedIds((prev) => prev.filter((id) => snippets.some((snippet) => snippet.id === id)));
  }, [snippets]);

  useEffect(() => {
    // 分组或标签被删除后，过滤条件回落到仍然有效的范围。
    setFilter((prev) => {
      const tagIds = prev.tagIds.filter((id) => tags.some((tag) => tag.id === id));
      const groupMissing = prev.groupScope === 'group' && !groups.some((group) => group.id === prev.groupId);
      if (tagIds.length === prev.tagIds.length && !groupMissing) {
        return prev;
      }
      return groupMissing
        ? { ...prev, groupScope: 'all', groupId: null, tagIds }
        : { ...prev, tagIds };
    });
  }, [groups, tags]);

  useEffect(() => {
    if (selectAllRef.current) {
      selectAllRef.current.indeterminate = someVisibleSelected;
    }
  }, [someVisibleSelected]);

  const runAction = async (task: () => Promise<unknown>) => {
    setError(null);
    try {
      await task();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const selectGroup = (scope: FilterState['groupScope'], groupId: number | null) => {
    setFilter((prev) => ({ ...prev, groupScope: scope, groupId }));
  };

  const toggleTagFilter = (tagId: number) => {
    setFilter((prev) => ({
      ...prev,
      tagIds: prev.tagIds.includes(tagId) ? prev.tagIds.filter((id) => id !== tagId) : [...prev.tagIds, tagId],
    }));
  };

  const toggleSelectAll = () => {
    if (allVisibleSelected) {
      setSelectedIds((prev) => prev.filter((id) => !filtered.some((snippet) => snippet.id === id)));
    } else {
      setSelectedIds((prev) => [...new Set([...prev, ...filtered.map((snippet) => snippet.id)])]);
    }
  };

  const confirmDeleteSnippets = (ids: number[], label: string) => {
    setConfirm({
      title: '删除预设文本',
      description: `${label}将被永久删除，且不可撤销。`,
      actionLabel: '删除',
      run: async () => {
        await actions.deleteSnippets(ids);
        setSelectedIds((prev) => prev.filter((id) => !ids.includes(id)));
      },
    });
  };

  const renderSidebarRow = (
    label: string,
    active: boolean,
    onClick: () => void,
    actionsNode?: ReactNode,
    renameKey?: { kind: 'group' | 'tag'; id: number },
  ) => {
    if (renameKey && renaming && renaming.kind === renameKey.kind && renaming.id === renameKey.id) {
      return (
        <li key={`${renameKey.kind}-${renameKey.id}`} className="px-1 py-px">
          <InlineInput
            defaultValue={renaming.value}
            placeholder="名称"
            onSubmit={(value) => {
              setRenaming(null);
              if (value !== '') {
                void runAction(() =>
                  renameKey.kind === 'group' ? actions.saveGroup(renameKey.id, value) : actions.saveTag(renameKey.id, value),
                );
              }
            }}
            onCancel={() => setRenaming(null)}
          />
        </li>
      );
    }
    return (
      <li key={`${label}-${active}`} className="group flex items-center gap-0.5">
        <button
          type="button"
          onClick={onClick}
          aria-current={active}
          className={`min-w-0 flex-1 truncate rounded-[4px] px-2 py-[5px] text-left text-[12px] transition-colors ${
            active ? 'bg-brand-soft font-medium text-brand' : 'text-muted hover:bg-black/[0.04] hover:text-ink'
          }`}
        >
          {label}
        </button>
        {actionsNode ? (
          <div className="flex shrink-0 items-center opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
            {actionsNode}
          </div>
        ) : null}
      </li>
    );
  };

  return (
    <div className="flex h-full w-full flex-col overflow-hidden bg-canvas text-ink">
      <header
        style={dragStyle}
        className="flex h-10 shrink-0 items-center justify-between border-b border-line bg-surface pl-3.5 pr-1"
      >
        <div className="flex items-center gap-2">
          <span className="flex h-[18px] w-[18px] items-center justify-center rounded-[4px] bg-brand text-white">
            <TagIcon className="h-[11px] w-[11px]" />
          </span>
          <span className="text-[12.5px] font-semibold tracking-tight">Quick Input</span>
        </div>
        <div className="flex items-center gap-0.5" style={noDragStyle}>
          <IconButton label="悬浮窗预览" onClick={() => void actions.switchToOverlay()}>
            <WindowIcon className="h-3.5 w-3.5" />
          </IconButton>
          <IconButton label="最小化" onClick={() => void actions.minimizeWindow()}>
            <MinimizeIcon className="h-3.5 w-3.5" />
          </IconButton>
          <IconButton label="隐藏到托盘" onClick={() => void actions.hideWindow()}>
            <CloseIcon className="h-3.5 w-3.5" />
          </IconButton>
        </div>
      </header>

      <div className="flex min-h-0 flex-1">
        <aside className="flex w-[220px] shrink-0 flex-col border-r border-line bg-[#f7f7f7]">
          <div className="min-h-0 flex-1 overflow-y-auto px-2 py-3">
            <div className="mb-1 flex h-6 items-center justify-between px-1">
              <span className="text-[10.5px] font-semibold tracking-wide text-faint">分组</span>
              <IconButton size="sm" label="新建分组" onClick={() => setAddingGroup(true)}>
                <PlusIcon className="h-3 w-3" />
              </IconButton>
            </div>
            <ul className="space-y-px">
              {renderSidebarRow('全部', filter.groupScope === 'all', () => selectGroup('all', null))}
              {renderSidebarRow('未分组', filter.groupScope === 'ungrouped', () => selectGroup('ungrouped', null))}
              {groups.map((group) =>
                renderSidebarRow(
                  group.name,
                  filter.groupScope === 'group' && filter.groupId === group.id,
                  () => selectGroup('group', group.id),
                  <>
                    <IconButton
                      size="sm"
                      label={`重命名 ${group.name}`}
                      onClick={() => setRenaming({ kind: 'group', id: group.id, value: group.name })}
                    >
                      <PencilIcon className="h-3 w-3" />
                    </IconButton>
                    <IconButton
                      size="sm"
                      label={`删除 ${group.name}`}
                      onClick={() =>
                        setConfirm({
                          title: '删除分组',
                          description: `「${group.name}」下的预设文本将变为未分组，且不可撤销。`,
                          actionLabel: '删除',
                          run: () => actions.deleteGroup(group.id),
                        })
                      }
                    >
                      <TrashIcon className="h-3 w-3" />
                    </IconButton>
                  </>,
                  { kind: 'group', id: group.id },
                ),
              )}
              {addingGroup ? (
                <li className="px-1 py-px">
                  <InlineInput
                    placeholder="分组名称"
                    onSubmit={(value) => {
                      setAddingGroup(false);
                      if (value !== '') {
                        void runAction(() => actions.saveGroup(0, value));
                      }
                    }}
                    onCancel={() => setAddingGroup(false)}
                  />
                </li>
              ) : null}
            </ul>

            <div className="mb-1 mt-5 flex h-6 items-center justify-between px-1">
              <span className="text-[10.5px] font-semibold tracking-wide text-faint">标签</span>
              <IconButton size="sm" label="新建标签" onClick={() => setAddingTag(true)}>
                <PlusIcon className="h-3 w-3" />
              </IconButton>
            </div>
            <ul className="space-y-px">
              {tags.length === 0 && !addingTag ? <li className="px-2 py-1 text-[11px] text-faint">暂无标签</li> : null}
              {tags.map((tag) =>
                renderSidebarRow(
                  tag.name,
                  filter.tagIds.includes(tag.id),
                  () => toggleTagFilter(tag.id),
                  <>
                    <IconButton
                      size="sm"
                      label={`重命名 ${tag.name}`}
                      onClick={() => setRenaming({ kind: 'tag', id: tag.id, value: tag.name })}
                    >
                      <PencilIcon className="h-3 w-3" />
                    </IconButton>
                    <IconButton
                      size="sm"
                      label={`删除 ${tag.name}`}
                      onClick={() =>
                        setConfirm({
                          title: '删除标签',
                          description: `「${tag.name}」将从所有预设文本中解绑，且不可撤销。`,
                          actionLabel: '删除',
                          run: () => actions.deleteTag(tag.id),
                        })
                      }
                    >
                      <TrashIcon className="h-3 w-3" />
                    </IconButton>
                  </>,
                  { kind: 'tag', id: tag.id },
                ),
              )}
              {addingTag ? (
                <li className="px-1 py-px">
                  <InlineInput
                    placeholder="标签名称"
                    onSubmit={(value) => {
                      setAddingTag(false);
                      if (value !== '') {
                        void runAction(() => actions.saveTag(0, value));
                      }
                    }}
                    onCancel={() => setAddingTag(false)}
                  />
                </li>
              ) : null}
            </ul>
          </div>
        </aside>

        <main className="flex min-w-0 flex-1 flex-col bg-surface">
          <div className="flex h-12 shrink-0 items-center gap-2.5 border-b border-line px-4">
            <input
              ref={selectAllRef}
              type="checkbox"
              checked={allVisibleSelected}
              onChange={toggleSelectAll}
              aria-label="全选当前列表"
              className="h-3.5 w-3.5 shrink-0 accent-[#0078d4]"
            />
            <div className="flex h-8 min-w-0 flex-1 items-center gap-1.5 rounded-[4px] border border-line bg-canvas/60 px-2 focus-within:border-brand">
              <SearchIcon className="h-3.5 w-3.5 shrink-0 text-faint" />
              <input
                value={filter.keyword}
                onChange={(event) => setFilter((prev) => ({ ...prev, keyword: event.target.value }))}
                placeholder="搜索标题或内容"
                spellCheck={false}
                className="h-full min-w-0 flex-1 bg-transparent text-[12.5px] text-ink outline-none"
              />
              {filter.keyword !== '' ? (
                <IconButton size="sm" label="清除搜索" onClick={() => setFilter((prev) => ({ ...prev, keyword: '' }))}>
                  <CloseIcon className="h-3 w-3" />
                </IconButton>
              ) : null}
            </div>
            <Button variant="primary" onClick={() => setEditor({ snippet: null })}>
              <PlusIcon className="h-3.5 w-3.5" />
              新建
            </Button>
          </div>

          {selectedIds.length > 0 ? (
            <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-line bg-brand-soft/70 px-4 py-2">
              <span className="text-[11.5px] font-medium text-brand">已选 {selectedIds.length} 项</span>
              <select
                value=""
                onChange={(event) => {
                  const value = event.target.value;
                  if (value === '') {
                    return;
                  }
                  const groupId = value === 'ungrouped' ? null : Number(value);
                  void runAction(() => actions.batchMoveGroup(selectedIds, groupId));
                }}
                aria-label="批量移入分组"
                className="h-7 rounded-[4px] border border-line bg-surface px-1.5 text-[11.5px] text-ink outline-none focus:border-brand"
              >
                <option value="">移入分组…</option>
                <option value="ungrouped">未分组</option>
                {groups.map((group) => (
                  <option key={group.id} value={group.id}>
                    {group.name}
                  </option>
                ))}
              </select>
              <TagMenu
                label="添加标签"
                tags={tags}
                onPick={(tagId) => void runAction(() => actions.batchUpdateTags(selectedIds, [tagId], []))}
              />
              <TagMenu
                label="移除标签"
                tags={tags}
                onPick={(tagId) => void runAction(() => actions.batchUpdateTags(selectedIds, [], [tagId]))}
              />
              <Button
                variant="danger"
                className="h-7"
                onClick={() => confirmDeleteSnippets(selectedIds, `选中的 ${selectedIds.length} 条预设文本`)}
              >
                删除
              </Button>
              <button
                type="button"
                onClick={() => setSelectedIds([])}
                className="ml-auto text-[11.5px] text-muted hover:text-ink"
              >
                取消选择
              </button>
            </div>
          ) : null}

          {error ? (
            <div className="flex shrink-0 items-center gap-2 border-b border-[#f0c9c5] bg-[#fdf3f2] px-4 py-1.5 text-[11.5px] text-danger">
              <span className="min-w-0 flex-1 truncate">{error}</span>
              <IconButton size="sm" label="关闭提示" onClick={() => setError(null)}>
                <CloseIcon className="h-3 w-3" />
              </IconButton>
            </div>
          ) : null}

          <div className="min-h-0 flex-1 overflow-y-auto">
            {filtered.length === 0 ? (
              <EmptyState
                title={hasFilter ? '没有匹配的预设文本' : '暂无预设文本'}
                actionLabel={hasFilter ? '清除筛选' : '新建预设文本'}
                onAction={
                  hasFilter
                    ? () => setFilter(initialFilter)
                    : () => setEditor({ snippet: null })
                }
              />
            ) : (
              <ul>
                {filtered.map((snippet) => {
                  const selected = selectedIds.includes(snippet.id);
                  return (
                    <li
                      key={snippet.id}
                      className={`group flex items-center gap-3 border-b border-line/70 px-4 py-2.5 transition-colors ${
                        selected ? 'bg-brand-soft/50' : 'hover:bg-black/[0.025]'
                      }`}
                    >
                      <input
                        type="checkbox"
                        checked={selected}
                        onChange={() => setSelectedIds((prev) => toggleSelection(prev, snippet.id))}
                        aria-label={`选择 ${snippet.title}`}
                        className="h-3.5 w-3.5 shrink-0 accent-[#0078d4]"
                      />
                      <button
                        type="button"
                        onClick={() => setEditor({ snippet })}
                        className="min-w-0 flex-1 text-left"
                      >
                        <div className="flex items-center gap-2">
                          <span className="truncate text-[13px] font-medium text-ink">{snippet.title}</span>
                          {snippet.groupName ? (
                            <span className="shrink-0 rounded-[3px] bg-black/[0.05] px-1.5 py-px text-[10px] text-muted">
                              {snippet.groupName}
                            </span>
                          ) : null}
                        </div>
                        <p className="mt-0.5 truncate text-[12px] text-muted">{snippet.content}</p>
                      </button>
                      <div className="flex shrink-0 items-center gap-1">
                        <div className="hidden max-w-[180px] items-center gap-1 overflow-hidden sm:flex">
                          {snippet.tags.map((tag) => (
                            <span
                              key={tag.id}
                              className="shrink-0 rounded-[3px] border border-line px-1.5 py-px text-[10px] text-muted"
                            >
                              {tag.name}
                            </span>
                          ))}
                        </div>
                        <div className="flex items-center opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
                          <IconButton label={`编辑 ${snippet.title}`} onClick={() => setEditor({ snippet })}>
                            <PencilIcon className="h-3.5 w-3.5" />
                          </IconButton>
                          <IconButton
                            label={`删除 ${snippet.title}`}
                            onClick={() => confirmDeleteSnippets([snippet.id], `预设文本「${snippet.title}」`)}
                          >
                            <TrashIcon className="h-3.5 w-3.5" />
                          </IconButton>
                        </div>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        </main>
      </div>

      {editor ? (
        <EditorDrawer
          snippet={editor.snippet}
          groups={groups}
          tags={tags}
          onClose={() => setEditor(null)}
          onSave={async (input) => {
            await actions.saveSnippet(input);
          }}
          onCreateTag={(name) => actions.saveTag(0, name)}
        />
      ) : null}

      {confirm ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/25" onMouseDown={() => setConfirm(null)}>
          <div
            role="alertdialog"
            aria-modal="true"
            aria-label={confirm.title}
            className="w-[360px] rounded-[8px] border border-line bg-surface p-5 shadow-flyout"
            onMouseDown={(event) => event.stopPropagation()}
          >
            <h3 className="text-[14px] font-semibold text-ink">{confirm.title}</h3>
            <p className="mt-2 text-[12.5px] leading-5 text-muted">{confirm.description}</p>
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="secondary" onClick={() => setConfirm(null)}>
                取消
              </Button>
              <Button
                variant="danger"
                onClick={() => {
                  const task = confirm.run;
                  setConfirm(null);
                  void runAction(task);
                }}
              >
                {confirm.actionLabel}
              </Button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function InlineInput({
  defaultValue = '',
  placeholder,
  onSubmit,
  onCancel,
}: {
  defaultValue?: string;
  placeholder: string;
  onSubmit: (value: string) => void;
  onCancel: () => void;
}) {
  const [value, setValue] = useState(defaultValue);
  return (
    <input
      autoFocus
      value={value}
      placeholder={placeholder}
      onChange={(event) => setValue(event.target.value)}
      onKeyDown={(event) => {
        if (event.key === 'Enter') {
          event.preventDefault();
          onSubmit(value.trim());
        } else if (event.key === 'Escape') {
          event.preventDefault();
          onCancel();
        }
      }}
      onBlur={() => {
        if (value.trim() === '') {
          onCancel();
        }
      }}
      className="h-6 w-full rounded-[3px] border border-brand bg-surface px-1.5 text-[12px] text-ink outline-none"
    />
  );
}

function TagMenu({ label, tags, onPick }: { label: string; tags: Tag[]; onPick: (tagId: number) => void }) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    const onPointerDown = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', onPointerDown);
    return () => document.removeEventListener('mousedown', onPointerDown);
  }, [open]);

  return (
    <div ref={containerRef} className="relative">
      <Button variant="secondary" onClick={() => setOpen((prev) => !prev)} disabled={tags.length === 0}>
        {label}
        <ChevronDownIcon className="h-3 w-3" />
      </Button>
      {open ? (
        <div className="absolute left-0 top-8 z-20 max-h-56 w-44 overflow-y-auto rounded-[6px] border border-line bg-surface p-1 shadow-flyout">
          {tags.map((tag) => (
            <button
              key={tag.id}
              type="button"
              onClick={() => {
                setOpen(false);
                onPick(tag.id);
              }}
              className="block w-full truncate rounded-[4px] px-2 py-1.5 text-left text-[12px] text-ink hover:bg-black/[0.05]"
            >
              {tag.name}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
