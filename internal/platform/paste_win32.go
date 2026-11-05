//go:build windows

package platform

import (
	"errors"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002

	inputKeyboard = 1
	keyEventKeyUp = 0x0002

	vkMenu  = 0x12
	vkShift = 0x10
	vkCtrl  = 0x11
	vkLWin  = 0x5B
	vkRWin  = 0x5C
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procGetForegroundWindow    = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	procGetWindowThreadProcess = user32.NewProc("GetWindowThreadProcessId")
	procGetAsyncKeyState       = user32.NewProc("GetAsyncKeyState")
	procGetClipboardSequence   = user32.NewProc("GetClipboardSequenceNumber")
	procOpenClipboard          = user32.NewProc("OpenClipboard")
	procCloseClipboard         = user32.NewProc("CloseClipboard")
	procEmptyClipboard         = user32.NewProc("EmptyClipboard")
	procGetClipboardData       = user32.NewProc("GetClipboardData")
	procSetClipboardData       = user32.NewProc("SetClipboardData")
	procSendInput              = user32.NewProc("SendInput")

	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
	procGlobalFree   = kernel32.NewProc("GlobalFree")
)

type keybdInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

// input corresponds to Win32 INPUT, 40 bytes on amd64 including padding.
type input struct {
	typ uint32
	_   uint32
	ki  keybdInput
	_   [8]byte
}

func waitUntil(timeout time.Duration, done func() bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if done() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitModifiersReleased(timeout time.Duration) {
	released := func() bool {
		ctrl, _, _ := procGetAsyncKeyState.Call(vkCtrl)
		alt, _, _ := procGetAsyncKeyState.Call(vkMenu)
		shift, _, _ := procGetAsyncKeyState.Call(vkShift)
		lwin, _, _ := procGetAsyncKeyState.Call(vkLWin)
		rwin, _, _ := procGetAsyncKeyState.Call(vkRWin)
		const down = 0x8000
		return ctrl&down == 0 && alt&down == 0 && shift&down == 0 && lwin&down == 0 && rwin&down == 0
	}
	waitUntil(timeout, released)
}

// pointerAt turns a Win32 address into a pointer; the memory belongs to the
// system heap, not Go's.
func pointerAt(addr uintptr) unsafe.Pointer {
	base := unsafe.Pointer(&addr)
	return unsafe.Pointer(uintptr(base) + (addr - uintptr(base)))
}

func clipboardSequence() uint32 {
	seq, _, _ := procGetClipboardSequence.Call()
	return uint32(seq)
}

func openClipboard() bool {
	for i := 0; i < 10; i++ {
		if ok, _, _ := procOpenClipboard.Call(0); ok != 0 {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func readClipboardText() (string, bool) {
	// Clipboard handles are bound to a thread: Open and Close must run on
	// the same OS thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !openClipboard() {
		return "", false
	}
	defer procCloseClipboard.Call()

	handle, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if handle == 0 {
		return "", false
	}
	ptr, _, _ := procGlobalLock.Call(handle)
	if ptr == 0 {
		return "", false
	}
	defer procGlobalUnlock.Call(handle)
	return windows.UTF16PtrToString((*uint16)(pointerAt(ptr))), true
}

func writeClipboardText(text string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	utf16, err := windows.UTF16FromString(text)
	if err != nil {
		return err
	}
	size := uintptr(len(utf16) * 2)
	handle, _, _ := procGlobalAlloc.Call(gmemMoveable, size)
	if handle == 0 {
		return errors.New("分配剪贴板内存失败")
	}
	ptr, _, _ := procGlobalLock.Call(handle)
	if ptr == 0 {
		procGlobalFree.Call(handle)
		return errors.New("锁定剪贴板内存失败")
	}
	copy(unsafe.Slice((*uint16)(pointerAt(ptr)), len(utf16)), utf16)
	procGlobalUnlock.Call(handle)

	if !openClipboard() {
		procGlobalFree.Call(handle)
		return errors.New("打开剪贴板失败")
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()
	if ok, _, _ := procSetClipboardData.Call(cfUnicodeText, handle); ok == 0 {
		procGlobalFree.Call(handle)
		return errors.New("写入剪贴板失败")
	}
	return nil
}

func sendPasteKeystroke() {
	inputs := [4]input{
		{typ: inputKeyboard, ki: keybdInput{wVk: vkCtrl}},
		{typ: inputKeyboard, ki: keybdInput{wVk: 'V'}},
		{typ: inputKeyboard, ki: keybdInput{wVk: 'V', dwFlags: keyEventKeyUp}},
		{typ: inputKeyboard, ki: keybdInput{wVk: vkCtrl, dwFlags: keyEventKeyUp}},
	}
	procSendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(inputs[0]))
}
