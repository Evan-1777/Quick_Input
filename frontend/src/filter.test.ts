import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { filterSnippets, nextActiveIndex, toggleSelection } from './filter.ts';
import type { FilterState, Snippet } from './types.ts';

const snippet = (id: number, overrides: Partial<Snippet> = {}): Snippet => ({
  id,
  title: `文本 ${id}`,
  content: `内容 ${id}`,
  groupId: null,
  groupName: '',
  tags: [],
  updatedAt: '2026-11-04T00:00:00Z',
  ...overrides,
});

const baseFilter: FilterState = { keyword: '', groupScope: 'all', groupId: null, tagIds: [] };

const snippets: Snippet[] = [
  snippet(1, {
    title: '会议纪要',
    content: '周二产品评审结论',
    groupId: 10,
    groupName: '工作',
    tags: [{ id: 100, name: '常用', createdAt: '' }, { id: 101, name: '正式', createdAt: '' }],
  }),
  snippet(2, {
    title: '英文签名',
    content: 'Best regards, Lin',
    groupId: 20,
    groupName: '邮件',
    tags: [{ id: 100, name: '常用', createdAt: '' }],
  }),
  snippet(3, { title: '收货地址', content: '杭州市西湖区', tags: [{ id: 101, name: '正式', createdAt: '' }] }),
];

describe('filterSnippets', () => {
  it('按关键字大小写不敏感匹配标题与内容', () => {
    assert.deepEqual(filterSnippets(snippets, { ...baseFilter, keyword: '产品评审' }).map((s) => s.id), [1]);
    assert.deepEqual(filterSnippets(snippets, { ...baseFilter, keyword: 'BEST' }).map((s) => s.id), [2]);
    assert.deepEqual(filterSnippets(snippets, { ...baseFilter, keyword: '  ' }).map((s) => s.id), [1, 2, 3]);
  });

  it('按全部、未分组与指定分组切换', () => {
    assert.equal(filterSnippets(snippets, baseFilter).length, 3);
    assert.deepEqual(filterSnippets(snippets, { ...baseFilter, groupScope: 'ungrouped' }).map((s) => s.id), [3]);
    assert.deepEqual(
      filterSnippets(snippets, { ...baseFilter, groupScope: 'group', groupId: 20 }).map((s) => s.id),
      [2],
    );
    assert.deepEqual(filterSnippets(snippets, { ...baseFilter, groupScope: 'group', groupId: null }), []);
  });

  it('多选标签取交集', () => {
    assert.deepEqual(filterSnippets(snippets, { ...baseFilter, tagIds: [100] }).map((s) => s.id), [1, 2]);
    assert.deepEqual(filterSnippets(snippets, { ...baseFilter, tagIds: [100, 101] }).map((s) => s.id), [1]);
    assert.deepEqual(filterSnippets(snippets, { ...baseFilter, tagIds: [100, 101, 999] }), []);
  });

  it('组合过滤无结果时返回空数组', () => {
    const result = filterSnippets(snippets, {
      keyword: '地址',
      groupScope: 'group',
      groupId: 10,
      tagIds: [100],
    });
    assert.deepEqual(result, []);
  });
});

describe('nextActiveIndex', () => {
  it('在边界循环', () => {
    assert.equal(nextActiveIndex(0, 3, 'down'), 1);
    assert.equal(nextActiveIndex(2, 3, 'down'), 0);
    assert.equal(nextActiveIndex(0, 3, 'up'), 2);
    assert.equal(nextActiveIndex(1, 3, 'up'), 0);
  });

  it('空列表与越界索引归零', () => {
    assert.equal(nextActiveIndex(0, 0, 'down'), 0);
    assert.equal(nextActiveIndex(5, 2, 'up'), 0);
    assert.equal(nextActiveIndex(-1, 2, 'down'), 0);
  });
});

describe('toggleSelection', () => {
  it('切换加入与移除', () => {
    assert.deepEqual(toggleSelection([], 7), [7]);
    assert.deepEqual(toggleSelection([7], 7), []);
    assert.deepEqual(toggleSelection([1, 2], 3), [1, 2, 3]);
  });
});
