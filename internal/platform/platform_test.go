package platform

import "testing"

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
