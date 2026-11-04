//go:build windows

package platform

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Win32 常量。
const (
	wmHotkey     = 0x0312
	wmApp        = 0x8000
	wmTrayIcon   = wmApp + 1
	wmClose      = 0x0010
	wmDestroy    = 0x0002
	wmNull       = 0x0000
	wmRButtonUp  = 0x0205
	wmContext    = 0x007B
	wmLButtonDbl = 0x0203

	modAlt  = 0x0001
	vkSpace = 0x20
	vkQ     = 0x51
	vkMenu  = 0x12
	vkShift = 0x10
	vkCtrl  = 0x11

	swHide     = 0
	swShow     = 5
	swMinimize = 6

	gwlExStyle    = ^uintptr(19) // -20
	wsExTopmost   = 0x00000008
	wsExToolWnd   = 0x00000080
	swpShowWindow = 0x0040

	monitorDefaultToNearest = 2
	monitorDefaultToPrimary = 1

	mfString       = 0x0000
	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100
	tpmNoNotify    = 0x0080

	cmdTrayNewSnippet  = 1001
	cmdTrayOpenManager = 1002
	cmdTrayQuit        = 1003

	nimAdd    = 0
	nimDelete = 2
	nifMsg    = 0x00000001
	nifIcon   = 0x00000002
	nifTip    = 0x00000004

	idiApplication = 32512

	cfUnicodeText = 13
	gmemMoveable  = 0x0002

	inputKeyboard = 1
	keyEventKeyUp = 0x0002
	hotkeyID      = 1
	trayIconID    = 1
	messageWindow = "QuickInputMessageWindow"
	defaultDPI    = 96
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")

	procRegisterClassExW       = user32.NewProc("RegisterClassExW")
	procCreateWindowExW        = user32.NewProc("CreateWindowExW")
	procDefWindowProcW         = user32.NewProc("DefWindowProcW")
	procDestroyWindow          = user32.NewProc("DestroyWindow")
	procRegisterHotKey         = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey       = user32.NewProc("UnregisterHotKey")
	procGetMessageW            = user32.NewProc("GetMessageW")
	procTranslateMessage       = user32.NewProc("TranslateMessage")
	procDispatchMessageW       = user32.NewProc("DispatchMessageW")
	procPostQuitMessage        = user32.NewProc("PostQuitMessage")
	procPostMessageW           = user32.NewProc("PostMessageW")
	procEnumWindows            = user32.NewProc("EnumWindows")
	procGetWindowThreadProcess = user32.NewProc("GetWindowThreadProcessId")
	procGetForegroundWindow    = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	procGetGUIThreadInfo       = user32.NewProc("GetGUIThreadInfo")
	procClientToScreen         = user32.NewProc("ClientToScreen")
	procGetCursorPos           = user32.NewProc("GetCursorPos")
	procGetDpiForWindow        = user32.NewProc("GetDpiForWindow")
	procMonitorFromPoint       = user32.NewProc("MonitorFromPoint")
	procMonitorFromWindow      = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW        = user32.NewProc("GetMonitorInfoW")
	procGetSystemMetrics       = user32.NewProc("GetSystemMetrics")
	procSetWindowPos           = user32.NewProc("SetWindowPos")
	procShowWindow             = user32.NewProc("ShowWindow")
	procGetWindowLongPtrW      = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW      = user32.NewProc("SetWindowLongPtrW")
	procCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	procAppendMenuW            = user32.NewProc("AppendMenuW")
	procTrackPopupMenu         = user32.NewProc("TrackPopupMenu")
	procDestroyMenu            = user32.NewProc("DestroyMenu")
	procGetAsyncKeyState       = user32.NewProc("GetAsyncKeyState")
	procGetClipboardSequence   = user32.NewProc("GetClipboardSequenceNumber")
	procOpenClipboard          = user32.NewProc("OpenClipboard")
	procCloseClipboard         = user32.NewProc("CloseClipboard")
	procEmptyClipboard         = user32.NewProc("EmptyClipboard")
	procGetClipboardData       = user32.NewProc("GetClipboardData")
	procSetClipboardData       = user32.NewProc("SetClipboardData")
	procSendInput              = user32.NewProc("SendInput")
	procIsWindow               = user32.NewProc("IsWindow")
	procLoadIconW              = user32.NewProc("LoadIconW")

	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
	procGlobalFree   = kernel32.NewProc("GlobalFree")

	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
)

type point struct {
	X int32
	Y int32
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   windows.Handle
	icon       windows.Handle
	cursor     windows.Handle
	background windows.Handle
	menuName   *uint16
	className  *uint16
	iconSm     windows.Handle
}

type guiThreadInfo struct {
	cbSize        uint32
	flags         uint32
	hwndActive    windows.Handle
	hwndFocus     windows.Handle
	hwndCapture   windows.Handle
	hwndMenuOwner windows.Handle
	hwndMoveSize  windows.Handle
	hwndCaret     windows.Handle
	rcCaret       rect
}

type monitorInfo struct {
	cbSize    uint32
	rcMonitor rect
	rcWork    rect
	dwFlags   uint32
}

type msg struct {
	hwnd    windows.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type notifyIconData struct {
	cbSize           uint32
	hWnd             windows.Handle
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            windows.Handle
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uVersion         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     windows.Handle
}

type keybdInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

// input 对应 Win32 INPUT，amd64 下固定 40 字节（含联合体补齐）。
type input struct {
	typ uint32
	_   uint32
	ki  keybdInput
	_   [8]byte
}

type winBridge struct {
	title  string
	cb     Callbacks
	hwnd   windows.Handle
	own    windows.Handle
	target windows.Handle
}

var (
	activeBridge *winBridge
	classOnce    sync.Once
	classErr     error
	procWndProc  = windows.NewCallback(wndProc)
)

// NewBridge 创建 Windows 原生桥接实现。
func NewBridge(windowTitle string) Bridge {
	return &winBridge{title: windowTitle}
}

// Start 在独立的消息线程上创建消息窗口、注册全局热键与系统托盘。
func (b *winBridge) Start(cb Callbacks) error {
	b.cb = cb
	activeBridge = b

	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		if err := b.initMessageWindow(); err != nil {
			ready <- err
			return
		}
		ready <- nil
		b.runMessageLoop()
	}()
	return <-ready
}

// Stop 通知消息线程注销热键与托盘图标后退出。
func (b *winBridge) Stop() {
	if b.hwnd != 0 {
		procPostMessageW.Call(uintptr(b.hwnd), wmClose, 0, 0)
	}
}

func (b *winBridge) initMessageWindow() error {
	if err := ensureWindowClass(); err != nil {
		return err
	}
	hInstance, err := moduleHandle()
	if err != nil {
		return err
	}
	classPtr, err := windows.UTF16PtrFromString(messageWindow)
	if err != nil {
		return err
	}
	hwnd, _, callErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(classPtr)),
		0,
		0,
		0, 0, 0, 0,
		^uintptr(2), // HWND_MESSAGE
		0,
		uintptr(hInstance),
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("创建消息窗口失败: %v", callErr)
	}
	b.hwnd = windows.Handle(hwnd)

	// 默认 Alt+Space，被占用时回退 Alt+Q；两者都失败仍保留托盘入口。
	if ok, _, _ := procRegisterHotKey.Call(hwnd, hotkeyID, modAlt, vkSpace); ok == 0 {
		if ok, _, _ := procRegisterHotKey.Call(hwnd, hotkeyID, modAlt, vkQ); ok == 0 {
			_, _, _ = procRegisterHotKey.Call(hwnd, hotkeyID, modAlt, vkSpace)
		}
	}
	if err := b.addTrayIcon(); err != nil {
		return err
	}
	return nil
}

func (b *winBridge) addTrayIcon() error {
	var nid notifyIconData
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = b.hwnd
	nid.uID = trayIconID
	nid.uFlags = nifMsg | nifIcon | nifTip
	nid.uCallbackMessage = wmTrayIcon

	icon, _, _ := procLoadIconW.Call(0, idiApplication)
	nid.hIcon = windows.Handle(icon)
	tip := b.title
	if tip == "" {
		tip = "Quick Input"
	}
	copy(nid.szTip[:], windows.StringToUTF16(tip))

	if ok, _, callErr := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid))); ok == 0 {
		return fmt.Errorf("注册系统托盘图标失败: %v", callErr)
	}
	return nil
}

func (b *winBridge) removeTrayIcon() {
	if b.hwnd == 0 {
		return
	}
	var nid notifyIconData
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = b.hwnd
	nid.uID = trayIconID
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func (b *winBridge) runMessageLoop() {
	var m msg
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func wndProc(hwnd windows.Handle, message uint32, wParam, lParam uintptr) uintptr {
	b := activeBridge
	switch message {
	case wmHotkey:
		if b != nil && b.cb.OnToggleOverlay != nil {
			go b.cb.OnToggleOverlay()
		}
		return 0
	case wmTrayIcon:
		if b != nil {
			switch uint32(lParam) & 0xffff {
			case wmRButtonUp, wmContext:
				b.showTrayMenu(hwnd)
			case wmLButtonDbl:
				if b.cb.OnTrayOpenManager != nil {
					go b.cb.OnTrayOpenManager()
				}
			}
		}
		return 0
	case wmClose:
		if b != nil {
			b.cleanupMessageWindow()
		}
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(message), wParam, lParam)
	return ret
}

func (b *winBridge) showTrayMenu(hwnd windows.Handle) {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	appendMenu(menu, cmdTrayNewSnippet, "新建预设文本")
	appendMenu(menu, cmdTrayOpenManager, "打开管理窗口")
	appendMenu(menu, cmdTrayQuit, "退出")

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	procSetForegroundWindow.Call(uintptr(hwnd))
	cmd, _, _ := procTrackPopupMenu.Call(
		menu,
		tpmReturnCmd|tpmRightButton|tpmNoNotify,
		uintptr(pt.X), uintptr(pt.Y),
		0,
		uintptr(hwnd),
		0,
	)
	// 托盘菜单关闭后补发空消息，避免菜单残留。
	procPostMessageW.Call(uintptr(hwnd), wmNull, 0, 0)

	switch cmd {
	case cmdTrayNewSnippet:
		if b.cb.OnTrayNewSnippet != nil {
			go b.cb.OnTrayNewSnippet()
		}
	case cmdTrayOpenManager:
		if b.cb.OnTrayOpenManager != nil {
			go b.cb.OnTrayOpenManager()
		}
	case cmdTrayQuit:
		if b.cb.OnTrayQuit != nil {
			go b.cb.OnTrayQuit()
		}
	}
}

func appendMenu(menu uintptr, id uintptr, label string) {
	text, err := windows.UTF16PtrFromString(label)
	if err != nil {
		return
	}
	procAppendMenuW.Call(menu, mfString, id, uintptr(unsafe.Pointer(text)))
}

func (b *winBridge) cleanupMessageWindow() {
	removeHotkey := b.hwnd
	b.removeTrayIcon()
	if removeHotkey != 0 {
		procUnregisterHotKey.Call(uintptr(removeHotkey), hotkeyID)
		procDestroyWindow.Call(uintptr(removeHotkey))
	}
	b.hwnd = 0
}

func moduleHandle() (windows.Handle, error) {
	var handle windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &handle); err != nil {
		return 0, err
	}
	return handle, nil
}

func ensureWindowClass() error {
	classOnce.Do(func() {
		hInstance, err := moduleHandle()
		if err != nil {
			classErr = err
			return
		}
		namePtr, err := windows.UTF16PtrFromString(messageWindow)
		if err != nil {
			classErr = err
			return
		}
		wc := wndClassEx{
			size:      uint32(unsafe.Sizeof(wndClassEx{})),
			wndProc:   procWndProc,
			instance:  hInstance,
			className: namePtr,
		}
		if atom, _, callErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
			classErr = fmt.Errorf("注册消息窗口类失败: %v", callErr)
		}
	})
	return classErr
}

// ShowOverlayAtCursor 快照目标窗口后，把悬浮窗定位到输入光标（回退鼠标）处并激活。
func (b *winBridge) ShowOverlayAtCursor(logicalWidth, logicalHeight int) Point {
	own := b.ownHWND()
	if own == 0 {
		return Point{}
	}
	if fg := b.foregroundWindow(); fg != 0 && fg != own {
		b.target = fg
	}

	anchor := b.caretAnchor()
	dpi := b.windowDPI(own)
	width := ScaleByDPI(logicalWidth, dpi)
	height := ScaleByDPI(logicalHeight, dpi)
	work := b.workAreaAt(anchor)
	pos := ClampWindowPosition(anchor, width, height, work)

	b.applyChrome(own, true)
	procSetWindowPos.Call(uintptr(own), ^uintptr(0), uintptr(int32(pos.X)), uintptr(int32(pos.Y)),
		uintptr(width), uintptr(height), swpShowWindow)
	procShowWindow.Call(uintptr(own), swShow)
	procSetForegroundWindow.Call(uintptr(own))
	return pos
}

// ShowManagerWindow 取消置顶工具窗样式，按 DPI 换算后居中于主显示器工作区。
func (b *winBridge) ShowManagerWindow(logicalWidth, logicalHeight int) {
	own := b.ownHWND()
	if own == 0 {
		return
	}
	dpi := b.windowDPI(own)
	width := ScaleByDPI(logicalWidth, dpi)
	height := ScaleByDPI(logicalHeight, dpi)
	work := b.primaryWorkArea(own)

	x := work.Left + (work.Right-work.Left-width)/2
	y := work.Top + (work.Bottom-work.Top-height)/2
	if x < work.Left {
		x = work.Left
	}
	if y < work.Top {
		y = work.Top
	}

	b.applyChrome(own, false)
	procSetWindowPos.Call(uintptr(own), ^uintptr(1), uintptr(int32(x)), uintptr(int32(y)),
		uintptr(width), uintptr(height), swpShowWindow)
	procShowWindow.Call(uintptr(own), swShow)
	procSetForegroundWindow.Call(uintptr(own))
}

// HideWindow 隐藏当前窗口。
func (b *winBridge) HideWindow() {
	if own := b.ownHWND(); own != 0 {
		procShowWindow.Call(uintptr(own), swHide)
	}
}

// MinimizeWindow 最小化当前窗口。
func (b *winBridge) MinimizeWindow() {
	if own := b.ownHWND(); own != 0 {
		procShowWindow.Call(uintptr(own), swMinimize)
	}
}

// PasteTextToTarget 归还焦点后以剪贴板粘贴写入文本，并在序列号未变时还原原文本。
func (b *winBridge) PasteTextToTarget(text string) error {
	if text == "" {
		return errors.New("待键入文本为空")
	}
	if own := b.ownHWND(); own != 0 {
		procShowWindow.Call(uintptr(own), swHide)
	}

	if target := b.target; target != 0 && b.isWindow(target) {
		procSetForegroundWindow.Call(uintptr(target))
		waitUntil(100*time.Millisecond, func() bool { return b.foregroundWindow() == target })
	}

	waitModifiersReleased(300 * time.Millisecond)

	backup, hadText := readClipboardText()
	if err := writeClipboardText(text); err != nil {
		return err
	}
	seq := clipboardSequence()

	sendPasteKeystroke()

	time.AfterFunc(150*time.Millisecond, func() {
		// 仅恢复纯文本，且用户在此期间未复制新内容时才覆盖，避免抢占新剪贴板数据。
		if clipboardSequence() == seq && hadText {
			_ = writeClipboardText(backup)
		}
	})
	return nil
}

func (b *winBridge) ownHWND() windows.Handle {
	if b.own != 0 && b.isWindow(b.own) {
		return b.own
	}
	pid := uint32(windows.GetCurrentProcessId())
	found := windows.Handle(0)
	callback := windows.NewCallback(func(hwnd windows.Handle, _ uintptr) uintptr {
		var windowPID uint32
		procGetWindowThreadProcess.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&windowPID)))
		if windowPID == pid && found == 0 {
			found = hwnd
			return 0
		}
		return 1
	})
	procEnumWindows.Call(callback, 0)
	b.own = found
	return found
}

func (b *winBridge) foregroundWindow() windows.Handle {
	hwnd, _, _ := procGetForegroundWindow.Call()
	return windows.Handle(hwnd)
}

func (b *winBridge) isWindow(hwnd windows.Handle) bool {
	ok, _, _ := procIsWindow.Call(uintptr(hwnd))
	return ok != 0
}

// caretAnchor 优先取输入光标矩形，无法获取时回退鼠标物理坐标。
func (b *winBridge) caretAnchor() Point {
	var info guiThreadInfo
	info.cbSize = uint32(unsafe.Sizeof(info))
	if ok, _, _ := procGetGUIThreadInfo.Call(0, uintptr(unsafe.Pointer(&info))); ok != 0 {
		if info.hwndCaret != 0 &&
			info.rcCaret.Right > info.rcCaret.Left &&
			info.rcCaret.Bottom > info.rcCaret.Top {
			pt := point{X: info.rcCaret.Left, Y: info.rcCaret.Bottom}
			if ok, _, _ := procClientToScreen.Call(uintptr(info.hwndCaret), uintptr(unsafe.Pointer(&pt))); ok != 0 {
				return Point{X: int(pt.X), Y: int(pt.Y)}
			}
		}
	}

	var pt point
	if ok, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))); ok != 0 {
		return Point{X: int(pt.X), Y: int(pt.Y)}
	}
	return Point{}
}

func (b *winBridge) windowDPI(hwnd windows.Handle) int {
	if procGetDpiForWindow.Find() == nil {
		if dpi, _, _ := procGetDpiForWindow.Call(uintptr(hwnd)); dpi > 0 {
			return int(dpi)
		}
	}
	return defaultDPI
}

func (b *winBridge) workAreaAt(anchor Point) Rect {
	pt := point{X: int32(anchor.X), Y: int32(anchor.Y)}
	monitor, _, _ := procMonitorFromPoint.Call(uintptr(unsafe.Pointer(&pt)), monitorDefaultToNearest)
	return monitorWorkArea(monitor)
}

func (b *winBridge) primaryWorkArea(hwnd windows.Handle) Rect {
	monitor, _, _ := procMonitorFromWindow.Call(uintptr(hwnd), monitorDefaultToPrimary)
	return monitorWorkArea(monitor)
}

func monitorWorkArea(monitor uintptr) Rect {
	if monitor != 0 {
		var info monitorInfo
		info.cbSize = uint32(unsafe.Sizeof(info))
		if ok, _, _ := procGetMonitorInfoW.Call(monitor, uintptr(unsafe.Pointer(&info))); ok != 0 {
			work := info.rcWork
			if work.Right > work.Left && work.Bottom > work.Top {
				return Rect{Left: int(work.Left), Top: int(work.Top), Right: int(work.Right), Bottom: int(work.Bottom)}
			}
		}
	}
	width, _, _ := procGetSystemMetrics.Call(0)
	height, _, _ := procGetSystemMetrics.Call(1)
	return Rect{Left: 0, Top: 0, Right: int(width), Bottom: int(height)}
}

func (b *winBridge) applyChrome(hwnd windows.Handle, overlay bool) {
	style, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), gwlExStyle)
	if overlay {
		style |= wsExTopmost | wsExToolWnd
	} else {
		style &^= wsExTopmost | wsExToolWnd
	}
	procSetWindowLongPtrW.Call(uintptr(hwnd), gwlExStyle, style)
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
		alt, _, _ := procGetAsyncKeyState.Call(vkMenu)
		shift, _, _ := procGetAsyncKeyState.Call(vkShift)
		return alt&0x8000 == 0 && shift&0x8000 == 0
	}
	waitUntil(timeout, released)
}

// pointerAt 把 Win32 返回的地址转为指针；该内存属于系统全局堆而非 Go 托管堆。
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
	// 剪贴板句柄绑定线程，Open/Close 必须落在同一 OS 线程上。
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
