package platform

import "testing"

func TestRestoreClipboard(t *testing.T) {
	cases := []struct {
		name    string
		hadText bool
		typed   uint32
		current uint32
		want    bool
	}{
		{"序列号未变", true, 7, 7, true},
		{"用户期间复制了新内容", true, 7, 8, false},
		{"键入前无文本", false, 7, 7, false},
	}
	for _, tc := range cases {
		if got := restoreClipboard(tc.hadText, tc.typed, tc.current); got != tc.want {
			t.Fatalf("%s: restoreClipboard = %v, want %v", tc.name, got, tc.want)
		}
	}
}
