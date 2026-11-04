import type { BootstrapPayload, Group, Snippet, SnippetInput, Tag, WindowStatePayload } from './types.ts';

interface WailsBridge {
  GetBootstrapData(): Promise<BootstrapPayload>;
  SaveSnippet(input: SnippetInput): Promise<Snippet>;
  DeleteSnippets(ids: number[]): Promise<void>;
  SaveGroup(id: number, name: string): Promise<Group>;
  DeleteGroup(id: number): Promise<void>;
  SaveTag(id: number, name: string): Promise<Tag>;
  DeleteTag(id: number): Promise<void>;
  BatchMoveGroup(snippetIds: number[], groupId: number | null): Promise<void>;
  BatchUpdateTags(snippetIds: number[], addTagIds: number[], removeTagIds: number[]): Promise<void>;
  TriggerPaste(id: number): Promise<void>;
  HideWindow(): Promise<void>;
  MinimizeWindow(): Promise<void>;
  SwitchToManager(openCreator: boolean): Promise<void>;
  SwitchToOverlay(): Promise<void>;
}

interface WailsRuntime {
  EventsOn(eventName: string, callback: (data: unknown) => void): void;
  EventsOff(eventName: string): void;
}

declare global {
  interface Window {
    go?: { app?: { App?: WailsBridge } };
    runtime?: WailsRuntime;
  }
}

const bridge = (): WailsBridge | undefined => window.go?.app?.App;

const emptyPayload: BootstrapPayload = {
  groups: [],
  tags: [],
  snippets: [],
  windowState: { mode: 'overlay', openCreator: false },
};

export function getBootstrapData(): Promise<BootstrapPayload> {
  return bridge()?.GetBootstrapData() ?? Promise.resolve(emptyPayload);
}

export function saveSnippet(input: SnippetInput): Promise<Snippet> {
  const app = bridge();
  return app ? app.SaveSnippet(input) : Promise.reject(new Error('未连接应用后端'));
}

export function deleteSnippets(ids: number[]): Promise<void> {
  return bridge()?.DeleteSnippets(ids) ?? Promise.resolve();
}

export function saveGroup(id: number, name: string): Promise<Group> {
  const app = bridge();
  return app ? app.SaveGroup(id, name) : Promise.reject(new Error('未连接应用后端'));
}

export function deleteGroup(id: number): Promise<void> {
  return bridge()?.DeleteGroup(id) ?? Promise.resolve();
}

export function saveTag(id: number, name: string): Promise<Tag> {
  const app = bridge();
  return app ? app.SaveTag(id, name) : Promise.reject(new Error('未连接应用后端'));
}

export function deleteTag(id: number): Promise<void> {
  return bridge()?.DeleteTag(id) ?? Promise.resolve();
}

export function batchMoveGroup(snippetIds: number[], groupId: number | null): Promise<void> {
  return bridge()?.BatchMoveGroup(snippetIds, groupId) ?? Promise.resolve();
}

export function batchUpdateTags(snippetIds: number[], addTagIds: number[], removeTagIds: number[]): Promise<void> {
  return bridge()?.BatchUpdateTags(snippetIds, addTagIds, removeTagIds) ?? Promise.resolve();
}

export function triggerPaste(id: number): Promise<void> {
  return bridge()?.TriggerPaste(id) ?? Promise.resolve();
}

export function hideWindow(): Promise<void> {
  return bridge()?.HideWindow() ?? Promise.resolve();
}

export function minimizeWindow(): Promise<void> {
  return bridge()?.MinimizeWindow() ?? Promise.resolve();
}

export function switchToManager(openCreator: boolean): Promise<void> {
  return bridge()?.SwitchToManager(openCreator) ?? Promise.resolve();
}

export function switchToOverlay(): Promise<void> {
  return bridge()?.SwitchToOverlay() ?? Promise.resolve();
}

/** 订阅窗口模式事件，返回取消订阅函数。 */
export function onWindowMode(handler: (state: WindowStatePayload) => void): () => void {
  const runtime = window.runtime;
  if (!runtime) {
    return () => {};
  }
  runtime.EventsOn('window:mode', (data) => handler(data as WindowStatePayload));
  return () => runtime.EventsOff('window:mode');
}
