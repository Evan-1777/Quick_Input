package platform

import (
	"errors"
	"testing"
)

func TestScaleByDPI(t *testing.T) {
	cases := []struct {
		logical int
		dpi     int
		want    int
	}{
		{560, 96, 560},
		{560, 120, 700},
		{560, 144, 840},
		{380, 120, 475},
		{560, 0, 560},
		{560, -96, 560},
	}
	for _, tc := range cases {
		if got := ScaleByDPI(tc.logical, tc.dpi); got != tc.want {
			t.Fatalf("ScaleByDPI(%d, %d) = %d, want %d", tc.logical, tc.dpi, got, tc.want)
		}
	}
}

func TestClampWindowPosition(t *testing.T) {
	work := Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

	cases := []struct {
		name   string
		anchor Point
		w, h   int
		work   Rect
		want   Point
	}{
		{"常规位置", Point{X: 400, Y: 300}, 560, 380, work, Point{X: 408, Y: 308}},
		{"右边缘溢出", Point{X: 1900, Y: 300}, 560, 380, work, Point{X: 1360, Y: 308}},
		{"下边缘溢出", Point{X: 400, Y: 1070}, 560, 380, work, Point{X: 408, Y: 700}},
		{"左上越界", Point{X: -40, Y: -30}, 560, 380, work, Point{X: 0, Y: 0}},
		{"副屏工作区偏移", Point{X: -700, Y: 100}, 560, 380,
			Rect{Left: -1920, Top: 0, Right: 0, Bottom: 1080}, Point{X: -692, Y: 108}},
	}
	for _, tc := range cases {
		if got := ClampWindowPosition(tc.anchor, tc.w, tc.h, tc.work); got != tc.want {
			t.Fatalf("%s: ClampWindowPosition = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestParseHotkey(t *testing.T) {
	valid := []struct {
		spec string
		want Hotkey
		out  string
	}{
		{"Alt+Space", Hotkey{Modifiers: ModAlt, Key: 0x20}, "Alt+Space"},
		{"ctrl+shift+q", Hotkey{Modifiers: ModControl | ModShift, Key: 0x51}, "Ctrl+Shift+Q"},
		{"F12+Ctrl", Hotkey{Modifiers: ModControl, Key: 0x7B}, "Ctrl+F12"},
		{"Ctrl+Alt+1", Hotkey{Modifiers: ModControl | ModAlt, Key: 0x31}, "Ctrl+Alt+1"},
		{" Win + Enter ", Hotkey{Modifiers: ModWin, Key: 0x0D}, "Win+Enter"},
	}
	for _, tc := range valid {
		got, err := ParseHotkey(tc.spec)
		if err != nil {
			t.Fatalf("ParseHotkey(%q): %v", tc.spec, err)
		}
		if got != tc.want {
			t.Fatalf("ParseHotkey(%q) = %+v, want %+v", tc.spec, got, tc.want)
		}
		if got.String() != tc.out {
			t.Fatalf("ParseHotkey(%q).String() = %q, want %q", tc.spec, got.String(), tc.out)
		}
	}

	invalid := []string{"", "Space", "Ctrl+", "Alt+Q+Space", "Alt+Alt+Q", "Alt+F13", "Hyper+Q"}
	for _, spec := range invalid {
		if _, err := ParseHotkey(spec); !errors.Is(err, ErrInvalidHotkey) {
			t.Fatalf("ParseHotkey(%q) err = %v, want ErrInvalidHotkey", spec, err)
		}
	}

	if got := (Hotkey{Modifiers: ModAlt, Key: 0x99}).String(); got != "" {
		t.Fatalf("未知键码 String() = %q, want 空串", got)
	}
}

func TestHotkeyFallbacks(t *testing.T) {
	cases := []struct {
		name    string
		initial string
		want    []string
	}{
		{"空串", "", []string{"Alt+Space", "Alt+Q"}},
		{"默认组合", "Alt+Space", []string{"Alt+Space", "Alt+Q"}},
		{"自定义组合", "Ctrl+Shift+Q", []string{"Ctrl+Shift+Q", "Alt+Space", "Alt+Q"}},
		{"小写输入", "ctrl+alt+k", []string{"Ctrl+Alt+K", "Alt+Space", "Alt+Q"}},
		{"非法组合", "Hyper+Q", []string{"Alt+Space", "Alt+Q"}},
		{"回退组合", "Alt+Q", []string{"Alt+Q", "Alt+Space"}},
	}
	for _, tc := range cases {
		got := hotkeyFallbacks(tc.initial)
		if len(got) != len(tc.want) {
			t.Fatalf("%s: hotkeyFallbacks(%q) = %v, want %v", tc.name, tc.initial, got, tc.want)
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Fatalf("%s: hotkeyFallbacks(%q) = %v, want %v", tc.name, tc.initial, got, tc.want)
			}
		}
	}
}
