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
