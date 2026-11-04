export interface Group {
  id: number;
  name: string;
  createdAt: string;
}

export interface Tag {
  id: number;
  name: string;
  createdAt: string;
}

export interface Snippet {
  id: number;
  title: string;
  content: string;
  groupId: number | null;
  groupName: string;
  tags: Tag[];
  updatedAt: string;
}

export interface SnippetInput {
  id: number;
  title: string;
  content: string;
  groupId: number | null;
  tagIds: number[];
}

export type GroupScope = 'all' | 'ungrouped' | 'group';

export interface FilterState {
  keyword: string;
  groupScope: GroupScope;
  groupId: number | null;
  tagIds: number[];
}

export type WindowMode = 'overlay' | 'manager';

export interface WindowStatePayload {
  mode: WindowMode;
  openCreator: boolean;
}

export interface SettingsPayload {
  hotkey: string;
}

export interface BootstrapPayload {
  groups: Group[];
  tags: Tag[];
  snippets: Snippet[];
  windowState: WindowStatePayload;
  settings: SettingsPayload;
}

export interface AppActions {
  saveSnippet(input: SnippetInput): Promise<void>;
  deleteSnippets(ids: number[]): Promise<void>;
  saveGroup(id: number, name: string): Promise<Group>;
  deleteGroup(id: number): Promise<void>;
  saveTag(id: number, name: string): Promise<Tag>;
  deleteTag(id: number): Promise<void>;
  batchMoveGroup(ids: number[], groupId: number | null): Promise<void>;
  batchUpdateTags(ids: number[], addTagIds: number[], removeTagIds: number[]): Promise<void>;
  triggerPaste(id: number): Promise<void>;
  setHotkey(spec: string): Promise<void>;
  hideWindow(): Promise<void>;
  minimizeWindow(): Promise<void>;
  switchToManager(openCreator: boolean): Promise<void>;
  switchToOverlay(): Promise<void>;
}
