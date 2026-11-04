//go:build !windows

package platform

import "sync"

// StubBridge 是非 Windows 环境下的内存实现，仅保留可断言的状态供开发与单测使用。
type StubBridge struct {
	mu sync.Mutex

	LastPastedText string
	Visible        bool
	CurrentMode    string
	LastPosition   Point
}

// NewBridge 返回非 Windows 平台的内存桩。
func NewBridge(windowTitle string) Bridge {
	return &StubBridge{}
}

func (s *StubBridge) Start(cb Callbacks) error { return nil }

func (s *StubBridge) Stop() {}

func (s *StubBridge) ShowOverlayAtCursor(logicalWidth, logicalHeight int) Point {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Visible = true
	s.CurrentMode = ModeOverlay
	s.LastPosition = Point{X: 120, Y: 120}
	return s.LastPosition
}

func (s *StubBridge) ShowManagerWindow(logicalWidth, logicalHeight int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Visible = true
	s.CurrentMode = ModeManager
	s.LastPosition = Point{X: 0, Y: 0}
}

func (s *StubBridge) HideWindow() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Visible = false
}

func (s *StubBridge) MinimizeWindow() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Visible = false
}

func (s *StubBridge) PasteTextToTarget(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastPastedText = text
	s.Visible = false
	return nil
}
