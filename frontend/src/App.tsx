import { useCallback, useEffect, useMemo, useState } from 'react';

import * as api from './api.ts';
import ManagerView from './components/ManagerView';
import QuickOverlay from './components/QuickOverlay';
import { RetryIcon } from './components/Icons';
import { Button } from './components/ui';
import type { AppActions, BootstrapPayload, WindowMode } from './types.ts';

export default function App() {
  const [data, setData] = useState<BootstrapPayload | null>(null);
  const [mode, setMode] = useState<WindowMode>('overlay');
  const [creatorRequested, setCreatorRequested] = useState(false);
  const [loading, setLoading] = useState(true);
  const [bootstrapError, setBootstrapError] = useState<string | null>(null);
  const [session, setSession] = useState(0);

  const handleCreatorHandled = useCallback(() => setCreatorRequested(false), []);

  const refresh = useCallback(async () => {
    const payload = await api.getBootstrapData();
    setData(payload);
  }, []);

  const loadBootstrap = useCallback(async () => {
    setLoading(true);
    setBootstrapError(null);
    try {
      const payload = await api.getBootstrapData();
      setData(payload);
      setMode(payload.windowState.mode);
      setCreatorRequested(payload.windowState.openCreator);
    } catch (err) {
      setBootstrapError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadBootstrap();
  }, [loadBootstrap]);

  useEffect(
    () =>
      api.onWindowMode((state) => {
        setMode(state.mode);
        // 每次唤出都是新的检索会话，重置悬浮窗的筛选与焦点。
        setSession((value) => value + 1);
        setCreatorRequested(state.openCreator);
        void refresh().catch((err: unknown) => console.error('刷新数据失败', err));
      }),
    [refresh],
  );

  const actions = useMemo<AppActions>(
    () => ({
      async saveSnippet(input) {
        await api.saveSnippet(input);
        await refresh();
      },
      async deleteSnippets(ids) {
        await api.deleteSnippets(ids);
        await refresh();
      },
      async saveGroup(id, name) {
        const group = await api.saveGroup(id, name);
        await refresh();
        return group;
      },
      async deleteGroup(id) {
        await api.deleteGroup(id);
        await refresh();
      },
      async saveTag(id, name) {
        const tag = await api.saveTag(id, name);
        await refresh();
        return tag;
      },
      async deleteTag(id) {
        await api.deleteTag(id);
        await refresh();
      },
      async batchMoveGroup(ids, groupId) {
        await api.batchMoveGroup(ids, groupId);
        await refresh();
      },
      async batchUpdateTags(ids, addTagIds, removeTagIds) {
        await api.batchUpdateTags(ids, addTagIds, removeTagIds);
        await refresh();
      },
      triggerPaste: (id) => api.triggerPaste(id),
      hideWindow: () => api.hideWindow(),
      minimizeWindow: () => api.minimizeWindow(),
      switchToManager: (open) => api.switchToManager(open),
      switchToOverlay: () => api.switchToOverlay(),
    }),
    [refresh],
  );

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center gap-2 text-[12px] text-faint">
        <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-line border-t-brand" />
        加载中
      </div>
    );
  }

  if (!data) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 px-8 text-center">
        <p className="text-[13px] font-medium text-ink">数据加载失败</p>
        <p className="max-w-[420px] text-[11.5px] leading-5 text-muted">{bootstrapError}</p>
        <Button variant="primary" onClick={() => void loadBootstrap()}>
          <RetryIcon className="h-3.5 w-3.5" />
          重试
        </Button>
      </div>
    );
  }

  if (mode === 'manager') {
    return (
      <ManagerView
        groups={data.groups}
        tags={data.tags}
        snippets={data.snippets}
        creatorRequested={creatorRequested}
        onCreatorHandled={handleCreatorHandled}
        actions={actions}
      />
    );
  }

  return (
    <QuickOverlay key={session} snippets={data.snippets} groups={data.groups} tags={data.tags} actions={actions} />
  );
}
