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

func TestGroupedOnly(t *testing.T) {
	g1 := int64(1)
	all := []store.Snippet{
		snippet(1, "a", "a", &g1),
		snippet(2, "b", "b", nil),
		snippet(3, "c", "c", &g1),
	}
	got := groupedOnly(all)
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 3 {
		t.Fatalf("groupedOnly = %v, want ids [1 3]", got)
	}
	if len(groupedOnly(nil)) != 0 {
		t.Fatal("groupedOnly(nil) should be empty")
	}
}

func TestGroupSteps(t *testing.T) {
	groups := []store.Group{{ID: 10}, {ID: 20}}

	steps := groupSteps(groups, false)
	if len(steps) != 3 || steps[0].Scope != scopeAll || steps[1].Scope != scopeGroup {
		t.Fatalf("隐藏未分组时步骤 = %+v", steps)
	}
	if steps[1].GroupID == nil || *steps[1].GroupID != 10 || steps[2].GroupID == nil || *steps[2].GroupID != 20 {
		t.Fatalf("分组步骤顺序 = %+v", steps)
	}

	steps = groupSteps(groups, true)
	if len(steps) != 4 || steps[1].Scope != scopeUngrouped || steps[2].Scope != scopeGroup {
		t.Fatalf("显示未分组时步骤 = %+v", steps)
	}
}

func TestCycleGroup(t *testing.T) {
	a, b := int64(10), int64(20)
	steps := []FilterState{
		{Scope: scopeAll},
		{Scope: scopeGroup, GroupID: &a},
		{Scope: scopeGroup, GroupID: &b},
	}
	base := FilterState{Scope: scopeAll, Keyword: "hello", TagIDs: []int64{7}}

	if got := cycleGroup(base, steps, 1); got.Scope != scopeGroup || got.GroupID == nil || *got.GroupID != a {
		t.Fatalf("向后一步 = %+v", got)
	}
	if got := cycleGroup(base, steps, -1); got.Scope != scopeGroup || got.GroupID == nil || *got.GroupID != b {
		t.Fatalf("向前一步 = %+v", got)
	}
	if got := cycleGroup(FilterState{Scope: scopeGroup, GroupID: &b}, steps, 1); got.Scope != scopeAll || got.GroupID != nil {
		t.Fatalf("末组向后回绕 = %+v", got)
	}

	got := cycleGroup(base, steps, 1)
	if got.Keyword != "hello" || len(got.TagIDs) != 1 || got.TagIDs[0] != 7 {
		t.Fatalf("循环后未保留关键字与标签: %+v", got)
	}

	for _, step := range []int{4, -4, 99} {
		got := cycleGroup(base, steps, step)
		want := a
		if step < 0 {
			want = b
		}
		if got.GroupID == nil || *got.GroupID != want {
			t.Fatalf("step=%d 未钳制为一步: %+v", step, got)
		}
	}

	if got := cycleGroup(base, nil, 1); got.Scope != scopeAll || got.Keyword != "hello" {
		t.Fatalf("空步骤应归一化返回: %+v", got)
	}
	if got := cycleGroup(FilterState{}, steps, 0); got.Scope != scopeAll {
		t.Fatalf("空 Scope 应归一为 all: %+v", got)
	}
	if got := cycleGroup(FilterState{Scope: scopeAll}, steps[:1], 1); got.Scope != scopeAll {
		t.Fatalf("单元素步骤不应越界: %+v", got)
	}
	if got := cycleGroup(FilterState{Scope: scopeAll}, steps[:2], -1); got.Scope != scopeGroup || got.GroupID == nil || *got.GroupID != a {
		t.Fatalf("双元素步骤回绕 = %+v", got)
	}
}

func TestSplitBrackets(t *testing.T) {
	cases := []struct {
		in    string
		want  string
		steps int
	}{
		{"hello", "hello", 0},
		{"hel]lo", "hello", 1},
		{"x[", "x", -1},
		{"[[a]]", "a", 0},
		{"]hello[", "hello", 0},
	}
	for _, tc := range cases {
		got, steps := splitBrackets(tc.in)
		if got != tc.want || steps != tc.steps {
			t.Fatalf("splitBrackets(%q) = %q, %d, want %q, %d", tc.in, got, steps, tc.want, tc.steps)
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
