import { useEffect, useRef, useState } from 'react';

import type { Group, Snippet, SnippetInput, Tag } from '../types.ts';
import { CloseIcon, PlusIcon } from './Icons';
import { Button, IconButton } from './ui';

interface EditorDrawerProps {
  snippet: Snippet | null;
  groups: Group[];
  tags: Tag[];
  onClose: () => void;
  onSave: (input: SnippetInput) => Promise<void>;
  onCreateTag: (name: string) => Promise<Tag>;
}

export default function EditorDrawer({ snippet, groups, tags, onClose, onSave, onCreateTag }: EditorDrawerProps) {
  const [title, setTitle] = useState(snippet?.title ?? '');
  const [content, setContent] = useState(snippet?.content ?? '');
  const [groupId, setGroupId] = useState<number | null>(snippet?.groupId ?? null);
  const [tagIds, setTagIds] = useState<number[]>(snippet?.tags.map((tag) => tag.id) ?? []);
  const [newTag, setNewTag] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const titleRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    titleRef.current?.focus();
  }, []);

  const invalid = title.trim() === '' || content.trim() === '';

  const toggleTag = (tagId: number) => {
    setTagIds((prev) => (prev.includes(tagId) ? prev.filter((id) => id !== tagId) : [...prev, tagId]));
  };

  const createTag = async () => {
    const name = newTag.trim();
    if (name === '') {
      return;
    }
    try {
      const tag = await onCreateTag(name);
      setTagIds((prev) => (prev.includes(tag.id) ? prev : [...prev, tag.id]));
      setNewTag('');
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const submit = async () => {
    if (invalid || saving) {
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await onSave({
        id: snippet?.id ?? 0,
        title: title.trim(),
        content,
        groupId,
        tagIds,
      });
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-30 flex justify-end bg-black/20" onMouseDown={onClose}>
      <div
        role="dialog"
        aria-modal="true"
        aria-label={snippet ? '编辑预设文本' : '新建预设文本'}
        className="animate-drawer-in flex h-full w-[420px] flex-col border-l border-line bg-surface shadow-flyout"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className="flex h-11 shrink-0 items-center justify-between border-b border-line px-4">
          <h2 className="text-[13px] font-semibold text-ink">{snippet ? '编辑预设文本' : '新建预设文本'}</h2>
          <IconButton label="关闭" onClick={onClose}>
            <CloseIcon className="h-3.5 w-3.5" />
          </IconButton>
        </header>

        <form
          className="flex min-h-0 flex-1 flex-col"
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-4">
            <label className="block">
              <span className="mb-1 block text-[11.5px] font-medium text-muted">标题</span>
              <input
                ref={titleRef}
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                maxLength={120}
                className="h-8 w-full rounded-[4px] border border-line bg-surface px-2 text-[13px] text-ink outline-none focus:border-brand"
              />
            </label>

            <label className="block">
              <span className="mb-1 block text-[11.5px] font-medium text-muted">内容</span>
              <textarea
                value={content}
                onChange={(event) => setContent(event.target.value)}
                rows={9}
                className="w-full resize-none rounded-[4px] border border-line bg-surface px-2 py-1.5 text-[13px] leading-5 text-ink outline-none focus:border-brand"
              />
            </label>

            <label className="block">
              <span className="mb-1 block text-[11.5px] font-medium text-muted">分组</span>
              <select
                value={groupId === null ? '' : String(groupId)}
                onChange={(event) => setGroupId(event.target.value === '' ? null : Number(event.target.value))}
                className="h-8 w-full rounded-[4px] border border-line bg-surface px-1.5 text-[12.5px] text-ink outline-none focus:border-brand"
              >
                <option value="">未分组</option>
                {groups.map((group) => (
                  <option key={group.id} value={group.id}>
                    {group.name}
                  </option>
                ))}
              </select>
            </label>

            <div>
              <span className="mb-1.5 block text-[11.5px] font-medium text-muted">标签</span>
              <div className="flex flex-wrap gap-1.5">
                {tags.length === 0 ? <span className="text-[11.5px] text-faint">暂无标签</span> : null}
                {tags.map((tag) => {
                  const selected = tagIds.includes(tag.id);
                  return (
                    <button
                      key={tag.id}
                      type="button"
                      onClick={() => toggleTag(tag.id)}
                      aria-pressed={selected}
                      className={`rounded-[4px] border px-2 py-0.5 text-[11.5px] transition-colors ${
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
              <div className="mt-2 flex gap-1.5">
                <input
                  value={newTag}
                  onChange={(event) => setNewTag(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') {
                      event.preventDefault();
                      void createTag();
                    }
                  }}
                  placeholder="新标签名称"
                  className="h-7 min-w-0 flex-1 rounded-[4px] border border-line bg-surface px-2 text-[12px] outline-none focus:border-brand"
                />
                <Button variant="secondary" onClick={() => void createTag()} disabled={newTag.trim() === ''}>
                  <PlusIcon className="h-3 w-3" />
                  添加
                </Button>
              </div>
            </div>
          </div>

          <footer className="flex shrink-0 items-center gap-2 border-t border-line px-4 py-3">
            {error ? <p className="mr-auto truncate text-[11.5px] text-danger">{error}</p> : <span className="mr-auto" />}
            <Button variant="secondary" onClick={onClose}>
              取消
            </Button>
            <Button type="submit" variant="primary" disabled={invalid || saving}>
              {saving ? '保存中' : '保存'}
            </Button>
          </footer>
        </form>
      </div>
    </div>
  );
}
