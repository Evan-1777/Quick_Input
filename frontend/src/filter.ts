import type { FilterState, Snippet } from './types.ts';

/** 按关键字、单选分组与多选标签交集过滤预设文本。 */
export function filterSnippets(snippets: Snippet[], filter: FilterState): Snippet[] {
  const keyword = filter.keyword.trim().toLowerCase();

  return snippets.filter((snippet) => {
    if (keyword) {
      const haystack = `${snippet.title}\n${snippet.content}`.toLowerCase();
      if (!haystack.includes(keyword)) {
        return false;
      }
    }
    if (filter.groupScope === 'ungrouped' && snippet.groupId !== null) {
      return false;
    }
    if (filter.groupScope === 'group') {
      if (filter.groupId === null || snippet.groupId !== filter.groupId) {
        return false;
      }
    }
    if (filter.tagIds.length > 0) {
      const hasEveryTag = filter.tagIds.every((tagId) => snippet.tags.some((tag) => tag.id === tagId));
      if (!hasEveryTag) {
        return false;
      }
    }
    return true;
  });
}

/** 计算方向键循环导航后的激活索引。 */
export function nextActiveIndex(currentIndex: number, total: number, direction: 'up' | 'down'): number {
  if (total <= 0) {
    return 0;
  }
  if (currentIndex < 0 || currentIndex >= total) {
    return 0;
  }
  const step = direction === 'down' ? 1 : -1;
  return (currentIndex + step + total) % total;
}

/** 切换多选集合中的目标条目。 */
export function toggleSelection(selectedIds: number[], targetId: number): number[] {
  return selectedIds.includes(targetId)
    ? selectedIds.filter((id) => id !== targetId)
    : [...selectedIds, targetId];
}
