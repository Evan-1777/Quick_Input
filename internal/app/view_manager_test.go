package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestManagerReorderSnippets(t *testing.T) {
	fx := newFixture(t)
	fx.app.ShowManager(false)
	tt := ui.NewTester(fx.app.ManagerView, 960, 640)
	tt.SetSize(960, 640)

	mixed, ok := tt.Find("Mixed")
	if !ok {
		t.Fatalf("Mixed not found: %v", tt.Texts())
	}
	farewell, ok := tt.Find("Farewell")
	if !ok {
		t.Fatalf("Farewell not found: %v", tt.Texts())
	}
	dragRow(tt, farewell.X+farewell.W/2, farewell.Y+farewell.H/2, mixed.X+mixed.W/2, mixed.Y+2)

	if fx.app.editor != nil {
		t.Fatal("拖动不应打开编辑侧边页")
	}
	if got := appTitles(fx.app); strings.Join(got, ",") != "Farewell,Mixed,Ungrouped,Greeting" {
		t.Fatalf("拖动后顺序 = %v, want [Farewell Mixed Ungrouped Greeting]", got)
	}
}

func TestManagerReorderWithinFilter(t *testing.T) {
	fx := newFixture(t)
	fx.app.ShowManager(false)
	fx.app.lib.filter = FilterState{Scope: scopeGroup, GroupID: &fx.groupA}
	tt := ui.NewTester(fx.app.ManagerView, 960, 640)
	tt.SetSize(960, 640)

	mixed, ok := tt.Find("Mixed")
	if !ok {
		t.Fatalf("Mixed not found: %v", tt.Texts())
	}
	greeting, ok := tt.Find("Greeting")
	if !ok {
		t.Fatalf("Greeting not found: %v", tt.Texts())
	}
	dragRow(tt, greeting.X+greeting.W/2, greeting.Y+greeting.H/2, mixed.X+mixed.W/2, mixed.Y+2)

	if got := appTitles(fx.app); strings.Join(got, ",") != "Greeting,Mixed,Ungrouped,Farewell" {
		t.Fatalf("过滤视图拖动后顺序 = %v, want [Greeting Mixed Ungrouped Farewell]", got)
	}

	tt = ui.NewTester(fx.app.ManagerView, 960, 640)
	tt.SetSize(960, 640)
	greeting, ok = tt.Find("Greeting")
	mixed, ok2 := tt.Find("Mixed")
	if !ok || !ok2 || greeting.Y >= mixed.Y {
		t.Fatalf("过滤视图可见顺序错误: Greeting=%v Mixed=%v", greeting, mixed)
	}
}

func TestManagerReorderMultipleSelection(t *testing.T) {
	fx := newFixture(t)
	fx.app.ShowManager(false)
	tt := ui.NewTester(fx.app.ManagerView, 960, 640)
	tt.SetSize(960, 640)

	if err := tt.Click("Mixed"); err != nil {
		t.Fatalf("click Mixed: %v", err)
	}
	if err := tt.ClickWith(ui.Cmd, "Greeting"); err != nil {
		t.Fatalf("cmd click Greeting: %v", err)
	}
	if fx.app.lib.chosen.Len() != 2 {
		t.Fatalf("选中数量 = %d, want 2", fx.app.lib.chosen.Len())
	}

	mixed, ok := tt.Find("Mixed")
	if !ok {
		t.Fatalf("Mixed not found: %v", tt.Texts())
	}
	farewell, ok := tt.Find("Farewell")
	if !ok {
		t.Fatalf("Farewell not found: %v", tt.Texts())
	}
	dragRow(tt, mixed.X+mixed.W/2, mixed.Y+mixed.H/2, farewell.X+farewell.W/2, farewell.Y+2)

	if got := appTitles(fx.app); strings.Join(got, ",") != "Ungrouped,Mixed,Greeting,Farewell" {
		t.Fatalf("多选拖动后顺序 = %v, want [Ungrouped Mixed Greeting Farewell]", got)
	}
	if fx.app.lib.chosen.Len() != 2 {
		t.Fatalf("拖动后选中数量 = %d, want 2", fx.app.lib.chosen.Len())
	}
}

func TestManagerReorderStoreError(t *testing.T) {
	fx := newFixture(t)
	before := strings.Join(appTitles(fx.app), ",")
	if err := fx.app.store.Close(); err != nil {
		t.Fatalf("关闭 store: %v", err)
	}
	fx.app.reorderSnippets(fx.app.snippets, []int{0}, 2)

	if fx.app.lib.err == "" {
		t.Fatal("写库失败应写入错误条")
	}
	if got := strings.Join(appTitles(fx.app), ","); got != before {
		t.Fatalf("写库失败改变了内存顺序: %v, want %v", got, before)
	}
}

func appTitles(a *App) []string {
	titles := make([]string, 0, len(a.snippets))
	for _, snippet := range a.snippets {
		titles = append(titles, snippet.Title)
	}
	return titles
}

func dragRow(tt *ui.Tester, fromX, fromY, toX, toY float32) {
	tt.Press(fromX, fromY)
	for k := 1; k <= 4; k++ {
		f := float32(k) / 4
		tt.Move(fromX+(toX-fromX)*f, fromY+(toY-fromY)*f)
	}
	tt.Release(toX, toY)
}

func TestManagerCreateEditDelete(t *testing.T) {
	fx := newFixture(t)
	fx.app.ShowManager(false)
	tt := ui.NewTester(fx.app.ManagerView, 960, 640)
	tt.SetSize(960, 640)

	if err := tt.Click("新建"); err != nil {
		t.Fatalf("click 新建: %v", err)
	}
	tt.Type("周报模板")
	if err := tt.Click("内容"); err != nil {
		t.Fatalf("focus 内容: %v", err)
	}
	tt.Type("本周进展：无")
	if err := tt.Click("保存"); err != nil {
		t.Fatalf("click 保存: %v", err)
	}
	if !tt.HasText("周报模板") {
		t.Fatalf("new snippet not listed: %v", tt.Texts())
	}

	if err := tt.Click("周报模板"); err != nil {
		t.Fatalf("select row: %v", err)
	}
	tt.Frame()
	if err := tt.Click("删除所选"); err != nil {
		t.Fatalf("click 删除所选: %v", err)
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatalf("click 取消: %v", err)
	}
	if !tt.HasText("周报模板") {
		t.Fatal("cancel must keep the snippet")
	}

	if err := tt.Click("周报模板"); err != nil {
		t.Fatalf("click row: %v", err)
	}
	if err := tt.Click("周报模板"); err != nil {
		t.Fatalf("double click row: %v", err)
	}
	if !tt.HasText("编辑预设文本") {
		t.Fatalf("double click did not open the editor: %v", tt.Texts())
	}
	if err := tt.Click("标题"); err != nil {
		t.Fatalf("focus title field: %v", err)
	}
	tt.Command("selectAll")
	tt.Type("周报模板改")
	if err := tt.Click("保存"); err != nil {
		t.Fatalf("save edit: %v", err)
	}
	if !tt.HasText("周报模板改") {
		t.Fatalf("edit did not replace the title: %v", tt.Texts())
	}

	if err := tt.Click("周报模板改"); err != nil {
		t.Fatalf("select edited row: %v", err)
	}
	tt.Frame()
	if err := tt.Click("删除所选"); err != nil {
		t.Fatalf("click 删除所选: %v", err)
	}
	if err := tt.Click("删除"); err != nil {
		t.Fatalf("confirm delete: %v", err)
	}
	if tt.HasText("周报模板改") {
		t.Fatalf("confirmed delete left the row: %v", tt.Texts())
	}
}

func TestManagerFiltersAndSettings(t *testing.T) {
	fx := newFixture(t)
	fx.app.ShowManager(false)
	fx.app.lib.filter = FilterState{Scope: scopeGroup, GroupID: &fx.groupA}
	tt := ui.NewTester(fx.app.ManagerView, 960, 640)
	tt.SetSize(960, 640)
	if !tt.HasText("Greeting") || !tt.HasText("Mixed") {
		t.Fatalf("group filter missing rows: %v", tt.Texts())
	}
	if tt.HasText("Farewell") || tt.HasText("Ungrouped") {
		t.Fatalf("group filter kept other rows: %v", tt.Texts())
	}

	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click 设置: %v", err)
	}
	if !tt.HasText("唤出快捷键") {
		t.Fatalf("settings page missing: %v", tt.Texts())
	}

	fx.keys.fail["Ctrl+Alt+Q"] = true
	if err := tt.Click("Ctrl+Alt+Q"); err != nil {
		t.Fatalf("choose taken hotkey: %v", err)
	}
	if fx.app.lib.err == "" || !strings.Contains(fx.app.lib.err, "注册失败") {
		t.Fatalf("taken hotkey error missing: %q", fx.app.lib.err)
	}
	if fx.app.saved != "Alt+Space" {
		t.Fatalf("failed switch changed saved value: %q", fx.app.saved)
	}

	if err := tt.Click("Alt+Q"); err != nil {
		t.Fatalf("choose Alt+Q: %v", err)
	}
	if fx.app.saved != "Alt+Q" || fx.keys.registered["Alt+Q"] == nil {
		t.Fatalf("hotkey not switched: saved=%q registered=%v", fx.app.saved, fx.keys.calls)
	}
	if err := tt.Click("Alt+Q"); err != nil {
		t.Fatalf("click current hotkey: %v", err)
	}
	if fx.app.lib.err != "" {
		t.Fatalf("clicking the current hotkey set an error: %q", fx.app.lib.err)
	}
}

func TestSettingsShowUngrouped(t *testing.T) {
	fx := newFixture(t)
	fx.app.ShowManager(false)
	tt := ui.NewTester(fx.app.ManagerView, 960, 640)
	tt.SetSize(960, 640)

	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click 设置: %v", err)
	}
	if !tt.HasText("显示未分组") {
		t.Fatalf("设置页缺少显示未分组: %v", tt.Texts())
	}
	if err := tt.Click("显示未分组开关"); err != nil {
		t.Fatalf("click 显示未分组开关: %v", err)
	}
	if !fx.app.showUngrouped {
		t.Fatal("点击后开关未开启")
	}
	if err := tt.Click("显示未分组开关"); err != nil {
		t.Fatalf("再次 click 显示未分组开关: %v", err)
	}
	if fx.app.showUngrouped {
		t.Fatal("再次点击后开关未关闭")
	}
}

func TestSetHotkeyRollback(t *testing.T) {
	fx := newFixture(t)
	fx.keys.fail["Ctrl+Alt+Q"] = true
	if err := fx.app.SetHotkey("Ctrl+Alt+Q"); err == nil {
		t.Fatal("taken hotkey should fail")
	}
	if fx.app.hotkey != "Alt+Space" || fx.app.saved != "Alt+Space" {
		t.Fatalf("failed switch changed state: %q/%q", fx.app.hotkey, fx.app.saved)
	}
}

func TestManagerSidebarCRUD(t *testing.T) {
	fx := newFixture(t)
	fx.app.ShowManager(false)
	tt := ui.NewTester(fx.app.ManagerView, 960, 640)
	tt.SetSize(960, 640)

	if err := tt.Click("新建分组"); err != nil {
		t.Fatalf("click 新建分组: %v", err)
	}
	tt.Type("项目")
	tt.Key(0, ui.KeyEnter)
	if !tt.HasText("项目") {
		t.Fatalf("new group missing: %v", tt.Texts())
	}

	rect, ok := tt.Find("项目")
	if !ok {
		t.Fatalf("group text not found: %v", tt.Texts())
	}
	tt.Move(rect.X+rect.W/2, rect.Y+rect.H/2)
	tt.Frame()
	if err := tt.Click("重命名分组"); err != nil {
		t.Fatalf("click 重命名分组: %v", err)
	}
	tt.Command("selectAll")
	tt.Type("项目改")
	tt.Key(0, ui.KeyEnter)
	if !tt.HasText("项目改") {
		t.Fatalf("renamed group missing: %v", tt.Texts())
	}

	rect, ok = tt.Find("项目改")
	if !ok {
		t.Fatalf("renamed group text not found: %v", tt.Texts())
	}
	tt.Move(rect.X+rect.W/2, rect.Y+rect.H/2)
	tt.Frame()
	if err := tt.Click("删除分组"); err != nil {
		t.Fatalf("click 删除分组: %v", err)
	}
	if !tt.HasText("「项目改」下的预设文本将变为未分组，且不可撤销。") {
		t.Fatalf("group confirmation missing: %v", tt.Texts())
	}
	if err := tt.Click("删除"); err != nil {
		t.Fatalf("confirm group delete: %v", err)
	}
	if tt.HasText("项目改") {
		t.Fatalf("group still present: %v", tt.Texts())
	}

	if err := tt.Click("新建标签"); err != nil {
		t.Fatalf("click 新建标签: %v", err)
	}
	tt.Type("临时")
	tt.Key(0, ui.KeyEnter)
	if !tt.HasText("临时") {
		t.Fatalf("new tag missing: %v", tt.Texts())
	}
	rect, ok = tt.Find("临时")
	if !ok {
		t.Fatalf("tag text not found: %v", tt.Texts())
	}
	tt.Move(rect.X+rect.W/2, rect.Y+rect.H/2)
	tt.Frame()
	if err := tt.Click("删除标签"); err != nil {
		t.Fatalf("click 删除标签: %v", err)
	}
	if err := tt.Click("删除"); err != nil {
		t.Fatalf("confirm tag delete: %v", err)
	}
	if tt.HasText("临时") {
		t.Fatalf("tag still present: %v", tt.Texts())
	}
}
