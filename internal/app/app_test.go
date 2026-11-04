package app

import (
	"testing"

	"quick-input/internal/platform"
	"quick-input/internal/store"
)

type fakeBridge struct {
	lastText string
	visible  bool
	mode     string
}

func (f *fakeBridge) Start(platform.Callbacks) error { return nil }

func (f *fakeBridge) Stop() {}

func (f *fakeBridge) ShowOverlayAtCursor(int, int) platform.Point {
	f.visible = true
	f.mode = platform.ModeOverlay
	return platform.Point{X: 10, Y: 20}
}

func (f *fakeBridge) ShowManagerWindow(int, int) {
	f.visible = true
	f.mode = platform.ModeManager
}

func (f *fakeBridge) HideWindow() { f.visible = false }

func (f *fakeBridge) MinimizeWindow() { f.visible = false }

func (f *fakeBridge) PasteTextToTarget(text string) error {
	f.lastText = text
	f.visible = false
	return nil
}

func newTestApp(t *testing.T) (*App, *fakeBridge, *store.Store, *[]WindowStatePayload) {
	t.Helper()
	st, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	bridge := &fakeBridge{}
	application := NewApp(st, bridge)

	events := &[]WindowStatePayload{}
	application.emit = func(name string, data any) {
		if name != EventWindowMode {
			t.Fatalf("收到未定义事件 %q", name)
		}
		state, ok := data.(WindowStatePayload)
		if !ok {
			t.Fatalf("窗口状态负载类型错误: %T", data)
		}
		*events = append(*events, state)
	}
	return application, bridge, st, events
}

func TestWindowModeEvents(t *testing.T) {
	application, bridge, _, events := newTestApp(t)

	application.ToggleOverlay()
	if !bridge.visible || bridge.mode != platform.ModeOverlay {
		t.Fatalf("热键唤出后 bridge = %+v", bridge)
	}

	// 再次按热键应收起悬浮窗，而非重复唤出。
	application.ToggleOverlay()
	if bridge.visible {
		t.Fatal("悬浮窗未收起")
	}

	application.SwitchToManager(true)
	application.SwitchToManager(false)

	want := []WindowStatePayload{
		{Mode: platform.ModeOverlay, OpenCreator: false},
		{Mode: platform.ModeManager, OpenCreator: true},
		{Mode: platform.ModeManager, OpenCreator: false},
	}
	if len(*events) != len(want) {
		t.Fatalf("事件数 = %d, want %d: %+v", len(*events), len(want), *events)
	}
	for i, state := range want {
		if (*events)[i] != state {
			t.Fatalf("事件 %d = %+v, want %+v", i, (*events)[i], state)
		}
	}
}

func TestTriggerPasteUsesLatestContent(t *testing.T) {
	application, bridge, st, _ := newTestApp(t)

	snippet, err := st.SaveSnippet(store.SnippetInput{Title: "问候", Content: "初始内容"})
	if err != nil {
		t.Fatalf("SaveSnippet: %v", err)
	}
	if _, err := st.SaveSnippet(store.SnippetInput{ID: snippet.ID, Title: "问候", Content: "修改后的内容"}); err != nil {
		t.Fatalf("编辑内容: %v", err)
	}

	if err := application.TriggerPaste(snippet.ID); err != nil {
		t.Fatalf("TriggerPaste: %v", err)
	}
	if bridge.lastText != "修改后的内容" {
		t.Fatalf("键入内容 = %q, want 修改后的内容", bridge.lastText)
	}
}

func TestBatchOperationsReflectedInBootstrap(t *testing.T) {
	application, _, st, _ := newTestApp(t)

	group, err := st.SaveGroup(0, "工作")
	if err != nil {
		t.Fatalf("SaveGroup: %v", err)
	}
	tag, err := st.SaveTag(0, "常用")
	if err != nil {
		t.Fatalf("SaveTag: %v", err)
	}
	first, err := st.SaveSnippet(store.SnippetInput{Title: "一", Content: "1"})
	if err != nil {
		t.Fatalf("SaveSnippet: %v", err)
	}
	second, err := st.SaveSnippet(store.SnippetInput{Title: "二", Content: "2"})
	if err != nil {
		t.Fatalf("SaveSnippet: %v", err)
	}
	ids := []int64{first.ID, second.ID}

	if err := application.BatchMoveGroup(ids, &group.ID); err != nil {
		t.Fatalf("BatchMoveGroup: %v", err)
	}
	if err := application.BatchUpdateTags(ids, []int64{tag.ID}, nil); err != nil {
		t.Fatalf("BatchUpdateTags: %v", err)
	}

	data, err := application.GetBootstrapData()
	if err != nil {
		t.Fatalf("GetBootstrapData: %v", err)
	}
	if len(data.Snippets) != 2 {
		t.Fatalf("Bootstrap 文本数 = %d", len(data.Snippets))
	}
	for _, sn := range data.Snippets {
		if sn.GroupID == nil || *sn.GroupID != group.ID {
			t.Fatalf("文本 %q 分组未同步: %+v", sn.Title, sn.GroupID)
		}
		if len(sn.Tags) != 1 || sn.Tags[0].ID != tag.ID {
			t.Fatalf("文本 %q 标签未同步: %+v", sn.Title, sn.Tags)
		}
	}
	if data.WindowState.Mode != platform.ModeOverlay {
		t.Fatalf("初始窗口模式 = %q", data.WindowState.Mode)
	}
}
