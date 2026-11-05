package app

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

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
