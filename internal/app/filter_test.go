package app

import (
	"reflect"
	"testing"

	"quick-input/internal/store"
)

func snippet(id int64, title, content string, groupID *int64, tagIDs ...int64) store.Snippet {
	sn := store.Snippet{ID: id, Title: title, Content: content, GroupID: groupID}
	for _, tagID := range tagIDs {
		sn.Tags = append(sn.Tags, store.Tag{ID: tagID, Name: "t"})
	}
	return sn
}

func TestFilterSnippets(t *testing.T) {
	g1, g2 := int64(1), int64(2)
	all := []store.Snippet{
		snippet(1, "Greeting", "Hello world", &g1, 10),
		snippet(2, "Farewell", "Goodbye\nworld", &g2, 10, 20),
		snippet(3, "Ungrouped", "hello there", nil, 20),
		snippet(4, "Mixed", "HELLO", &g1, 10, 20),
	}

	cases := []struct {
		name   string
		filter FilterState
		want   []int64
	}{
		{"关键字 trim 与大小写", FilterState{Keyword: "  hello  "}, []int64{1, 3, 4}},
		{"标题与内容换行拼接", FilterState{Keyword: "goodbye\nworld"}, []int64{2}},
		{"未分组", FilterState{Scope: scopeUngrouped}, []int64{3}},
		{"指定分组", FilterState{Scope: scopeGroup, GroupID: &g1}, []int64{1, 4}},
		{"标签交集", FilterState{TagIDs: []int64{10, 20}}, []int64{2, 4}},
		{"组合条件", FilterState{Keyword: "hello", Scope: scopeGroup, GroupID: &g1, TagIDs: []int64{10}}, []int64{1, 4}},
		{"分组缺少 ID", FilterState{Scope: scopeGroup}, nil},
	}
	for _, tc := range cases {
		if tc.filter.Scope == "" {
			tc.filter.Scope = scopeAll
		}
		got := filterSnippets(all, tc.filter)
		ids := make([]int64, 0, len(got))
		for _, sn := range got {
			ids = append(ids, sn.ID)
		}
		if len(ids) == 0 && len(tc.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(ids, tc.want) {
			t.Fatalf("%s: ids = %v, want %v", tc.name, ids, tc.want)
		}
	}
}

func TestNextIndex(t *testing.T) {
	cases := []struct {
		current, total, step, want int
	}{
		{0, 3, 1, 1},
		{2, 3, 1, 0},
		{0, 3, -1, 2},
		{9, 0, 1, 0},
		{-1, 3, 1, 0},
		{3, 3, 1, 0},
	}
	for _, tc := range cases {
		if got := nextIndex(tc.current, tc.total, tc.step); got != tc.want {
			t.Fatalf("nextIndex(%d,%d,%d) = %d, want %d", tc.current, tc.total, tc.step, got, tc.want)
		}
	}
}

func TestToggleID(t *testing.T) {
	ids := []int64{1, 2}
	ids = toggleID(ids, 3)
	if !reflect.DeepEqual(ids, []int64{1, 2, 3}) {
		t.Fatalf("add = %v", ids)
	}
	ids = toggleID(ids, 2)
	if !reflect.DeepEqual(ids, []int64{1, 3}) {
		t.Fatalf("remove = %v", ids)
	}
	if !hasID(ids, 3) || hasID(ids, 2) {
		t.Fatalf("hasID mismatch: %v", ids)
	}
}

func TestPruneFilter(t *testing.T) {
	g1 := int64(1)
	filter := FilterState{Scope: scopeGroup, GroupID: &g1, TagIDs: []int64{10, 99}}
	pruneFilter(&filter, []store.Group{{ID: 2}}, []store.Tag{{ID: 10}})
	if filter.Scope != scopeAll || filter.GroupID != nil {
		t.Fatalf("stale group not pruned: %+v", filter)
	}
	if !reflect.DeepEqual(filter.TagIDs, []int64{10}) {
		t.Fatalf("stale tag not pruned: %v", filter.TagIDs)
	}
}
