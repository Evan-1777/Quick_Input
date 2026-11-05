//go:build windows

package platform

import (
	"errors"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// winPaster 使用 Win32 前台窗口与剪贴板把文本键入到目标应用。
type winPaster struct {
	target windows.Handle
}

// NewPaster 创建 Windows 键入实现。
func NewPaster() Paster { return &winPaster{} }

// CaptureTarget 记录唤出前的目标窗口；前台已是本进程窗口时保留旧快照。
func (p *winPaster) CaptureTarget() {
	fg, _, _ := procGetForegroundWindow.Call()
	hwnd := windows.Handle(fg)
	if hwnd == 0 {
		return
	}
	var pid uint32
	procGetWindowThreadProcess.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	if pid == windows.GetCurrentProcessId() {
		return
	}
	p.target = hwnd
}

// Paste 隐藏面板后归还焦点并以 Ctrl+V 键入文本，序列号未变时还原剪贴板。
func (p *winPaster) Paste(text string) error {
	if text == "" {
		return errors.New("待键入文本为空")
	}
	if p.target != 0 {
		procSetForegroundWindow.Call(uintptr(p.target))
		waitUntil(100*time.Millisecond, func() bool {
			hwnd, _, _ := procGetForegroundWindow.Call()
			return windows.Handle(hwnd) == p.target
		})
	}

	waitModifiersReleased(300 * time.Millisecond)

	backup, hadText := readClipboardText()
	if err := writeClipboardText(text); err != nil {
		return err
	}
	seq := clipboardSequence()
	sendPasteKeystroke()

	time.AfterFunc(150*time.Millisecond, func() {
		if restoreClipboard(hadText, seq, clipboardSequence()) {
			_ = writeClipboardText(backup)
		}
	})
	return nil
}
