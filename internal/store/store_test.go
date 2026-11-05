package store

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore(:memory:): %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestNewStoreCreatesParentDirectories(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "dir", "quick_input.db")
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore(%q): %v", dbPath, err)
	}
	defer s.Close()

	if _, err := s.SaveGroup(0, "工作"); err != nil {
		t.Fatalf("父目录创建后写入分组失败: %v", err)
	}
}

func TestGroupAndTagCRUDWithCaseInsensitiveUniqueness(t *testing.T) {
	s := newTestStore(t)

	work, err := s.SaveGroup(0, "工作")
	if err != nil {
		t.Fatalf("新建分组: %v", err)
	}
	if _, err := s.SaveGroup(0, "工作"); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("分组重名应返回 ErrDuplicateName, got %v", err)
	}
	if _, err := s.SaveGroup(0, ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("空分组名应返回 ErrInvalidInput, got %v", err)
	}

	renamed, err := s.SaveGroup(work.ID, "Work")
	if err != nil {
		t.Fatalf("重命名分组: %v", err)
	}
	if renamed.Name != "Work" {
		t.Fatalf("分组重命名后名称为 %q", renamed.Name)
	}
	if _, err := s.SaveGroup(0, "WORK"); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("大小写不敏感唯一约束失效, got %v", err)
	}

	urgent, err := s.SaveTag(0, "紧急")
	if err != nil {
		t.Fatalf("新建标签: %v", err)
	}
	if _, err := s.SaveTag(0, "紧急"); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("标签重名应返回 ErrDuplicateName, got %v", err)
	}
	if _, err := s.SaveTag(urgent.ID, "常用"); err != nil {
		t.Fatalf("重命名标签: %v", err)
	}

	groups, err := s.ListGroups()
	if err != nil || len(groups) != 1 {
		t.Fatalf("ListGroups = %v, %v", groups, err)
	}
	tags, err := s.ListTags()
	if err != nil || len(tags) != 1 || tags[0].Name != "常用" {
		t.Fatalf("ListTags = %v, %v", tags, err)
	}
}

func TestSaveSnippetReplacesTagBindings(t *testing.T) {
	s := newTestStore(t)
	a := mustTag(t, s, "A")
	b := mustTag(t, s, "B")
	c := mustTag(t, s, "C")

	sn := mustSnippet(t, s, SnippetInput{Title: "问候", Content: "你好", TagIDs: []int64{a.ID, b.ID}})
	if len(sn.Tags) != 2 {
		t.Fatalf("新建后标签数 = %d, want 2", len(sn.Tags))
	}

	updated, err := s.SaveSnippet(SnippetInput{
		ID: sn.ID, Title: "问候", Content: "你好呀", TagIDs: []int64{b.ID, c.ID},
	})
	if err != nil {
		t.Fatalf("编辑预设文本: %v", err)
	}
	if len(updated.Tags) != 2 || updated.Tags[0].Name != "B" || updated.Tags[1].Name != "C" {
		t.Fatalf("标签全量替换后 = %v, want [B C]", updated.Tags)
	}
	if updated.Content != "你好呀" {
		t.Fatalf("内容未更新: %q", updated.Content)
	}
}

func TestListSnippetsFilters(t *testing.T) {
	s := newTestStore(t)
	groupA := mustGroup(t, s, "A组")
	groupB := mustGroup(t, s, "B组")
	tagX := mustTag(t, s, "X")
	tagY := mustTag(t, s, "Y")

	mustSnippet(t, s, SnippetInput{Title: "会议纪要", Content: "周会结论", GroupID: &groupA.ID, TagIDs: []int64{tagX.ID, tagY.ID}})
	mustSnippet(t, s, SnippetInput{Title: "邮件模板", Content: "尊敬的客户", GroupID: &groupB.ID, TagIDs: []int64{tagX.ID}})
	mustSnippet(t, s, SnippetInput{Title: "收货地址", Content: "北京市海淀区", TagIDs: []int64{tagY.ID}})

	cases := []struct {
		name   string
		filter SnippetFilter
		want   int
	}{
		{"全部", SnippetFilter{GroupScope: "all"}, 3},
		{"关键字命中内容", SnippetFilter{GroupScope: "all", Keyword: "周会"}, 1},
		{"未分组", SnippetFilter{GroupScope: "ungrouped"}, 1},
		{"指定分组", SnippetFilter{GroupScope: "group", GroupID: &groupA.ID}, 1},
		{"单标签", SnippetFilter{GroupScope: "all", TagIDs: []int64{tagX.ID}}, 2},
		{"多标签交集", SnippetFilter{GroupScope: "all", TagIDs: []int64{tagX.ID, tagY.ID}}, 1},
		{"组合无结果", SnippetFilter{GroupScope: "group", GroupID: &groupB.ID, TagIDs: []int64{tagY.ID}}, 0},
	}
	for _, tc := range cases {
		got, err := s.ListSnippets(tc.filter)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(got) != tc.want {
			t.Fatalf("%s: 命中 %d 条, want %d", tc.name, len(got), tc.want)
		}
	}

	if _, err := s.ListSnippets(SnippetFilter{GroupScope: "group"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("group 作用域缺少 GroupID 应返回 ErrInvalidInput, got %v", err)
	}
}

func TestBatchMoveGroup(t *testing.T) {
	s := newTestStore(t)
	group := mustGroup(t, s, "目标组")
	first := mustSnippet(t, s, SnippetInput{Title: "一", Content: "1"})
	second := mustSnippet(t, s, SnippetInput{Title: "二", Content: "2"})

	if err := s.BatchMoveGroup([]int64{first.ID, second.ID}, &group.ID); err != nil {
		t.Fatalf("批量移入分组: %v", err)
	}
	moved, err := s.ListSnippets(SnippetFilter{GroupScope: "group", GroupID: &group.ID})
	if err != nil || len(moved) != 2 {
		t.Fatalf("移入分组后 = %d 条, %v", len(moved), err)
	}

	if err := s.BatchMoveGroup([]int64{first.ID, second.ID}, nil); err != nil {
		t.Fatalf("批量重置为未分组: %v", err)
	}
	ungrouped, err := s.ListSnippets(SnippetFilter{GroupScope: "ungrouped"})
	if err != nil || len(ungrouped) != 2 {
		t.Fatalf("重置未分组后 = %d 条, %v", len(ungrouped), err)
	}
}

func TestBatchUpdateTags(t *testing.T) {
	s := newTestStore(t)
	x := mustTag(t, s, "X")
	y := mustTag(t, s, "Y")
	first := mustSnippet(t, s, SnippetInput{Title: "一", Content: "1"})
	second := mustSnippet(t, s, SnippetInput{Title: "二", Content: "2"})
	ids := []int64{first.ID, second.ID}

	if err := s.BatchUpdateTags(ids, []int64{x.ID, y.ID}, nil); err != nil {
		t.Fatalf("批量新增标签: %v", err)
	}
	if err := s.BatchUpdateTags(ids, []int64{x.ID}, nil); err != nil {
		t.Fatalf("重复添加标签应幂等: %v", err)
	}
	both, err := s.ListSnippets(SnippetFilter{GroupScope: "all", TagIDs: []int64{x.ID, y.ID}})
	if err != nil || len(both) != 2 {
		t.Fatalf("批量新增后交集 = %d 条, %v", len(both), err)
	}

	if err := s.BatchUpdateTags(ids, nil, []int64{y.ID}); err != nil {
		t.Fatalf("批量移除标签: %v", err)
	}
	onlyY, err := s.ListSnippets(SnippetFilter{GroupScope: "all", TagIDs: []int64{y.ID}})
	if err != nil || len(onlyY) != 0 {
		t.Fatalf("批量移除后 Y 命中 = %d 条, %v", len(onlyY), err)
	}
}

func TestDeleteCleanup(t *testing.T) {
	s := newTestStore(t)
	group := mustGroup(t, s, "待删组")
	tag := mustTag(t, s, "待删标签")
	sn := mustSnippet(t, s, SnippetInput{Title: "文本", Content: "内容", GroupID: &group.ID, TagIDs: []int64{tag.ID}})

	if err := s.DeleteGroup(group.ID); err != nil {
		t.Fatalf("删除分组: %v", err)
	}
	afterGroup, err := s.GetSnippetByID(sn.ID)
	if err != nil {
		t.Fatalf("读取文本: %v", err)
	}
	if afterGroup.GroupID != nil {
		t.Fatalf("删除分组后 GroupID = %v, want nil", *afterGroup.GroupID)
	}

	if err := s.DeleteTag(tag.ID); err != nil {
		t.Fatalf("删除标签: %v", err)
	}
	afterTag, err := s.GetSnippetByID(sn.ID)
	if err != nil {
		t.Fatalf("读取文本: %v", err)
	}
	if len(afterTag.Tags) != 0 {
		t.Fatalf("删除标签后关联未清理: %v", afterTag.Tags)
	}

	if err := s.DeleteSnippets([]int64{sn.ID}); err != nil {
		t.Fatalf("删除文本: %v", err)
	}
	if _, err := s.GetSnippetByID(sn.ID); err == nil {
		t.Fatal("删除后仍能读取到文本")
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	s := newTestStore(t)

	value, err := s.GetSetting(SettingHotkey)
	if err != nil {
		t.Fatalf("读取未设置项: %v", err)
	}
	if value != "" {
		t.Fatalf("未设置项 = %q, want 空串", value)
	}

	if err := s.SetSetting(SettingHotkey, "Alt+Space"); err != nil {
		t.Fatalf("写入设置: %v", err)
	}
	if value, err = s.GetSetting(SettingHotkey); err != nil || value != "Alt+Space" {
		t.Fatalf("读取设置 = %q, %v", value, err)
	}

	if err := s.SetSetting(SettingHotkey, "Ctrl+Shift+Q"); err != nil {
		t.Fatalf("覆盖设置: %v", err)
	}
	if value, err = s.GetSetting(SettingHotkey); err != nil || value != "Ctrl+Shift+Q" {
		t.Fatalf("覆盖后读取设置 = %q, %v", value, err)
	}
}

func TestSnippetPositionMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quick_input.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("打开旧库: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE snippets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		group_id INTEGER,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("建旧版 snippets: %v", err)
	}
	old := []struct{ title, updatedAt string }{
		{"oldest", "2026-01-01T00:00:00Z"},
		{"middle", "2026-01-02T00:00:00Z"},
		{"newest", "2026-01-03T00:00:00Z"},
	}
	for _, row := range old {
		if _, err := db.Exec(`INSERT INTO snippets (title, content, group_id, updated_at) VALUES (?, ?, NULL, ?)`,
			row.title, row.title, row.updatedAt); err != nil {
			t.Fatalf("插入旧记录: %v", err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("关闭旧库: %v", err)
	}

	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("迁移旧库: %v", err)
	}
	defer s.Close()

	if got := snippetTitles(t, s); strings.Join(got, ",") != "newest,middle,oldest" {
		t.Fatalf("迁移后顺序 = %v, want [newest middle oldest]", got)
	}

	rows, err := s.db.Query(`SELECT position FROM snippets ORDER BY position`)
	if err != nil {
		t.Fatalf("读取 position: %v", err)
	}
	defer rows.Close()
	var positions []int
	for rows.Next() {
		var position int
		if err := rows.Scan(&position); err != nil {
			t.Fatalf("扫描 position: %v", err)
		}
		positions = append(positions, position)
	}
	if len(positions) != 3 || positions[0] != 0 || positions[1] != 1 || positions[2] != 2 {
		t.Fatalf("回填 position = %v, want [0 1 2]", positions)
	}
}

func TestSnippetOrderingAndReorder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quick_input.db")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	a := mustSnippet(t, s, SnippetInput{Title: "A", Content: "a"})
	b := mustSnippet(t, s, SnippetInput{Title: "B", Content: "b"})

	expect := func(name string, want ...string) {
		t.Helper()
		if got := snippetTitles(t, s); strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("%s: 顺序 = %v, want %v", name, got, want)
		}
	}

	expect("新建置顶", "B", "A")
	if _, err := s.SaveSnippet(SnippetInput{ID: a.ID, Title: "A2", Content: "a"}); err != nil {
		t.Fatalf("编辑 A: %v", err)
	}
	expect("编辑不移动", "B", "A2")

	if err := s.ReorderSnippets([]int64{a.ID}, &b.ID); err != nil {
		t.Fatalf("A 到 B 前: %v", err)
	}
	expect("单条移到首行之前", "A2", "B")

	if err := s.ReorderSnippets([]int64{b.ID}, &a.ID); err != nil {
		t.Fatalf("B 到 A 前: %v", err)
	}
	expect("单条移到中间", "B", "A2")

	if err := s.ReorderSnippets([]int64{a.ID}, nil); err != nil {
		t.Fatalf("A 到末尾: %v", err)
	}
	expect("nil 锚点落末尾", "B", "A2")

	unknown := int64(9999)
	if err := s.ReorderSnippets([]int64{b.ID}, &unknown); err != nil {
		t.Fatalf("未知锚点: %v", err)
	}
	expect("未知锚点落末尾", "A2", "B")

	if err := s.ReorderSnippets(nil, &a.ID); err != nil {
		t.Fatalf("空 ids: %v", err)
	}
	expect("空 ids 不改顺序", "A2", "B")

	if err := s.Close(); err != nil {
		t.Fatalf("关闭库: %v", err)
	}
	reopened, err := NewStore(path)
	if err != nil {
		t.Fatalf("重新打开库: %v", err)
	}
	defer reopened.Close()
	if got := snippetTitles(t, reopened); strings.Join(got, ",") != "A2,B" {
		t.Fatalf("重启后顺序 = %v, want [A2 B]", got)
	}
}

func TestReorderSnippetsMultipleIDs(t *testing.T) {
	s := newTestStore(t)
	a := mustSnippet(t, s, SnippetInput{Title: "A", Content: "a"})
	b := mustSnippet(t, s, SnippetInput{Title: "B", Content: "b"})
	c := mustSnippet(t, s, SnippetInput{Title: "C", Content: "c"})
	if got := snippetTitles(t, s); strings.Join(got, ",") != "C,B,A" {
		t.Fatalf("初始顺序 = %v, want [C B A]", got)
	}

	if err := s.ReorderSnippets([]int64{a.ID, c.ID}, &b.ID); err != nil {
		t.Fatalf("多 id 重排: %v", err)
	}
	if got := snippetTitles(t, s); strings.Join(got, ",") != "A,C,B" {
		t.Fatalf("多 id 重排后 = %v, want [A C B]", got)
	}
}

func snippetTitles(t *testing.T, s *Store) []string {
	t.Helper()
	items, err := s.ListSnippets(SnippetFilter{GroupScope: "all"})
	if err != nil {
		t.Fatalf("ListSnippets: %v", err)
	}
	titles := make([]string, 0, len(items))
	for _, item := range items {
		titles = append(titles, item.Title)
	}
	return titles
}

func mustGroup(t *testing.T, s *Store, name string) Group {
	t.Helper()
	g, err := s.SaveGroup(0, name)
	if err != nil {
		t.Fatalf("SaveGroup(%q): %v", name, err)
	}
	return g
}

func mustTag(t *testing.T, s *Store, name string) Tag {
	t.Helper()
	tg, err := s.SaveTag(0, name)
	if err != nil {
		t.Fatalf("SaveTag(%q): %v", name, err)
	}
	return tg
}

func mustSnippet(t *testing.T, s *Store, in SnippetInput) Snippet {
	t.Helper()
	sn, err := s.SaveSnippet(in)
	if err != nil {
		t.Fatalf("SaveSnippet(%q): %v", in.Title, err)
	}
	return sn
}
