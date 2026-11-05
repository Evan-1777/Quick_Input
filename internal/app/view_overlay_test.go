package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestOverlayFilterAndKeyboard(t *testing.T) {
	fx := newFixture(t)
	fx.app.ShowOverlay()
	tt := ui.NewTester(fx.app.OverlayView, 560, 380)

	if tt.HasText("Search") {
		t.Fatal("placeholder must be Chinese, found English Search")
	}
	if !tt.HasText("Greeting") || !tt.HasText("Farewell") {
		t.Fatalf("initial rows missing: %v", tt.Texts())
	}

	tt.Type("gree")
	if !tt.HasText("Greeting") || tt.HasText("Farewell") || tt.HasText("Ungrouped") {
		t.Fatalf("keyword filter wrong: %v", tt.Texts())
	}

	tt.Key(0, ui.KeyEscape)
	if fx.app.overlay.filter.Keyword != "" {
		t.Fatalf("Escape did not clear the keyword: %q", fx.app.overlay.filter.Keyword)
	}
	if !tt.HasText("Farewell") {
		t.Fatalf("Escape should keep the panel open: %v", tt.Texts())
	}

	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if len(fx.paster.texts) != 1 || fx.paster.texts[0] != "Goodbye world" {
		t.Fatalf("pasted %v, want Goodbye world", fx.paster.texts)
	}
	if fx.app.overlay.visible {
		t.Fatal("insert must hide the panel")
	}
}

func TestOverlayWrapUpAndTagIntersection(t *testing.T) {
	fx := newFixture(t)
	fx.app.ShowOverlay()
	tt := ui.NewTester(fx.app.OverlayView, 560, 380)
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyEnter)
	if len(fx.paster.texts) != 1 || fx.paster.texts[0] != "Hello world" {
		t.Fatalf("wrap up pasted %v, want Hello world", fx.paster.texts)
	}

	fx = newFixture(t)
	fx.app.ShowOverlay()
	fx.app.overlay.filter.TagIDs = []int64{fx.tagA, fx.tagB}
	tt = ui.NewTester(fx.app.OverlayView, 560, 380)
	if !tt.HasText("Farewell") || !tt.HasText("Mixed") {
		t.Fatalf("tag intersection missing rows: %v", tt.Texts())
	}
	if tt.HasText("Greeting") || tt.HasText("Ungrouped") {
		t.Fatalf("tag intersection kept non-matching rows: %v", tt.Texts())
	}

	tt.Key(0, ui.KeyEscape)
	if len(fx.app.overlay.filter.TagIDs) != 0 {
		t.Fatalf("Escape did not clear tag filter: %v", fx.app.overlay.filter.TagIDs)
	}
	tt.Key(0, ui.KeyEscape)
	if fx.app.overlay.visible {
		t.Fatal("third Escape should hide the panel")
	}
}

func TestOverlayGroupBracketsAndUngrouped(t *testing.T) {
	fx := newFixture(t)
	fx.app.ShowOverlay()
	tt := ui.NewTester(fx.app.OverlayView, 560, 380)

	if tt.HasText("Ungrouped") {
		t.Fatalf("默认应隐藏未分组: %v", tt.Texts())
	}
	if tt.HasText("未分组") {
		t.Fatalf("默认不应渲染未分组筛选项: %v", tt.Texts())
	}
	if !tt.HasText("Greeting") || !tt.HasText("Farewell") || !tt.HasText("Mixed") {
		t.Fatalf("初始行缺失: %v", tt.Texts())
	}

	tt.TypeKey(0, ui.KeyBracketRight, "]")
	assertOverlayScope(t, fx, scopeGroup, &fx.groupA)
	if tt.HasText("Farewell") || !tt.HasText("Greeting") || !tt.HasText("Mixed") {
		t.Fatalf("] 后未切到工作分组: %v", tt.Texts())
	}

	tt.TypeKey(0, ui.KeyBracketRight, "]")
	assertOverlayScope(t, fx, scopeGroup, &fx.groupB)
	if !tt.HasText("Farewell") || tt.HasText("Greeting") {
		t.Fatalf("] 后未切到生活分组: %v", tt.Texts())
	}

	tt.TypeKey(0, ui.KeyBracketRight, "]")
	assertOverlayScope(t, fx, scopeAll, nil)
	if !tt.HasText("Greeting") || !tt.HasText("Farewell") {
		t.Fatalf("] 未循环回全部: %v", tt.Texts())
	}

	tt.TypeKey(0, ui.KeyBracketLeft, "[")
	assertOverlayScope(t, fx, scopeGroup, &fx.groupB)

	// Clicking a group restores the search focus, so the next keys land in
	// the field; brackets keep the keyword and move one group.
	if err := tt.Click("全部"); err != nil {
		t.Fatalf("click 全部: %v", err)
	}
	tt.Type("hello")
	if fx.app.overlay.filter.Keyword != "hello" {
		t.Fatalf("点击分组后按键未进入搜索框: %q", fx.app.overlay.filter.Keyword)
	}
	tt.TypeKey(0, ui.KeyBracketRight, "]")
	assertOverlayScope(t, fx, scopeGroup, &fx.groupA)
	if fx.app.overlay.filter.Keyword != "hello" {
		t.Fatalf("切组后关键字 = %q, want hello", fx.app.overlay.filter.Keyword)
	}
	if tt.HasText("没有匹配的预设文本") {
		t.Fatalf("方括号不应产生空结果帧: %v", tt.Texts())
	}
	if !tt.HasText("Greeting") || !tt.HasText("Mixed") {
		t.Fatalf("切组后缺少匹配行: %v", tt.Texts())
	}

	// A tag click hands the focus to the toggle; it must come back too.
	if err := tt.Click("urgent"); err != nil {
		t.Fatalf("click urgent: %v", err)
	}
	tt.Type("x")
	if fx.app.overlay.filter.Keyword != "hellox" {
		t.Fatalf("点击标签后按键未进入搜索框: %q", fx.app.overlay.filter.Keyword)
	}

	tt.Type("zzz")
	if !tt.HasText("没有匹配的预设文本") {
		t.Fatalf("无匹配空态缺失: %v", tt.Texts())
	}
	if err := tt.Click("清除筛选"); err != nil {
		t.Fatalf("click 清除筛选: %v", err)
	}
	tt.Type("q")
	if fx.app.overlay.filter.Keyword != "q" {
		t.Fatalf("清除筛选后按键未进入搜索框: %q", fx.app.overlay.filter.Keyword)
	}

	// Several brackets in one frame move only one group and leave no
	// brackets in the keyword.
	tt.Type("[[[[")
	assertOverlayScope(t, fx, scopeGroup, &fx.groupB)
	if fx.app.overlay.filter.Keyword != "q" {
		t.Fatalf("多枚方括号残留: %q", fx.app.overlay.filter.Keyword)
	}
	tt.Type("]]")
	assertOverlayScope(t, fx, scopeAll, nil)
	if fx.app.overlay.filter.Keyword != "q" {
		t.Fatalf("多枚方括号残留: %q", fx.app.overlay.filter.Keyword)
	}

	// Shown ungrouped joins the list and the cycle right after 全部.
	fx = newFixture(t)
	if err := fx.app.SetShowUngrouped(true); err != nil {
		t.Fatalf("开启显示未分组: %v", err)
	}
	fx.app.ShowOverlay()
	tt = ui.NewTester(fx.app.OverlayView, 560, 380)
	if !tt.HasText("Ungrouped") || !tt.HasText("未分组") {
		t.Fatalf("开启后未分组未出现: %v", tt.Texts())
	}
	tt.TypeKey(0, ui.KeyBracketRight, "]")
	assertOverlayScope(t, fx, scopeUngrouped, nil)
	if !tt.HasText("Ungrouped") || tt.HasText("Greeting") {
		t.Fatalf("] 后应只显示未分组: %v", tt.Texts())
	}

	// With every snippet ungrouped and the preference off, the empty state
	// says why and offers the manager.
	fx = newFixture(t)
	ids := make([]int64, 0, len(fx.app.snippets))
	for _, snippet := range fx.app.snippets {
		ids = append(ids, snippet.ID)
	}
	if err := fx.app.store.BatchMoveGroup(ids, nil); err != nil {
		t.Fatalf("全部移为未分组: %v", err)
	}
	fx.app.ShowOverlay()
	tt = ui.NewTester(fx.app.OverlayView, 560, 380)
	if !tt.HasText("暂无分组预设文本") || !tt.HasText("打开管理窗口") {
		t.Fatalf("未分组空态缺失: %v", tt.Texts())
	}
}

func assertOverlayScope(t *testing.T, fx *fixture, scope string, groupID *int64) {
	t.Helper()
	filter := fx.app.overlay.filter
	if filter.Scope != scope {
		t.Fatalf("Scope = %q, want %q", filter.Scope, scope)
	}
	switch {
	case groupID == nil && filter.GroupID != nil:
		t.Fatalf("GroupID = %d, want nil", *filter.GroupID)
	case groupID != nil && (filter.GroupID == nil || *filter.GroupID != *groupID):
		t.Fatalf("GroupID = %v, want %d", filter.GroupID, *groupID)
	}
}

func TestOverlayEmptyState(t *testing.T) {
	fx := newFixture(t)
	fx.app.ShowOverlay()
	fx.app.overlay.filter.Keyword = "zzz"
	tt := ui.NewTester(fx.app.OverlayView, 560, 380)
	if !tt.HasText("没有匹配的预设文本") {
		t.Fatalf("filtered empty state missing: %v", tt.Texts())
	}
	if err := tt.Click("清除筛选"); err != nil {
		t.Fatalf("clear filter: %v", err)
	}
	if fx.app.overlay.filter.Keyword != "" {
		t.Fatalf("clear filter kept keyword %q", fx.app.overlay.filter.Keyword)
	}
	if !tt.HasText("Greeting") {
		t.Fatalf("rows not restored: %v", tt.Texts())
	}
}
