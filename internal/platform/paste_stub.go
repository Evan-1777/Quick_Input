//go:build !windows

package platform

// stubPaster 在非 Windows 平台空实现，供本地开发与测试。
type stubPaster struct{}

// NewPaster 创建非 Windows 键入桩。
func NewPaster() Paster { return stubPaster{} }

func (stubPaster) CaptureTarget() {}

func (stubPaster) Paste(string) error { return nil }
