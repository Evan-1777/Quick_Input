// Package platform 封装与操作系统窗口、剪贴板交互相关的原生能力。
// Windows 实现使用 Win32 API，其余平台使用可编译的内存测试桩。
package platform

import (
	"errors"
	"strconv"
	"strings"
)

// 全局唤出快捷键。
const (
	DefaultHotkey = "Alt+Space"
	// 修饰键位与 Win32 MOD_* 对齐。
	ModControl = 0x0002
	ModAlt     = 0x0001
	ModShift   = 0x0004
	ModWin     = 0x0008
)

// ErrInvalidHotkey 表示快捷键描述无法解析。
var ErrInvalidHotkey = errors.New("快捷键格式无效")

// Hotkey 是解析后的快捷键：修饰键位掩码与虚拟键码。
type Hotkey struct {
	Modifiers uint32
	Key       uint32
}

// ParseHotkey 解析形如 Alt+Space、Ctrl+Shift+Q 的组合，token 大小写不敏感、顺序任意。
func ParseHotkey(spec string) (Hotkey, error) {
	var h Hotkey
	for _, token := range strings.Split(spec, "+") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		var mod uint32
		switch strings.ToLower(token) {
		case "ctrl":
			mod = ModControl
		case "alt":
			mod = ModAlt
		case "shift":
			mod = ModShift
		case "win":
			mod = ModWin
		}
		if mod != 0 {
			if h.Modifiers&mod != 0 {
				return Hotkey{}, ErrInvalidHotkey
			}
			h.Modifiers |= mod
			continue
		}

		key, ok := keyCode(token)
		if !ok || h.Key != 0 {
			return Hotkey{}, ErrInvalidHotkey
		}
		h.Key = key
	}
	if h.Modifiers == 0 || h.Key == 0 {
		return Hotkey{}, ErrInvalidHotkey
	}
	return h, nil
}

// String 按 Ctrl+Alt+Shift+Win+Key 的固定顺序输出规范写法；未知键码返回空串。
func (h Hotkey) String() string {
	name, ok := keyName(h.Key)
	if !ok {
		return ""
	}
	parts := make([]string, 0, 5)
	if h.Modifiers&ModControl != 0 {
		parts = append(parts, "Ctrl")
	}
	if h.Modifiers&ModAlt != 0 {
		parts = append(parts, "Alt")
	}
	if h.Modifiers&ModShift != 0 {
		parts = append(parts, "Shift")
	}
	if h.Modifiers&ModWin != 0 {
		parts = append(parts, "Win")
	}
	return strings.Join(append(parts, name), "+")
}

// hotkeyFallbacks 返回启动注册的尝试序列：初始组合合法时优先，其后回退默认组合。
func hotkeyFallbacks(initial string) []string {
	candidates := make([]string, 0, 3)
	if h, err := ParseHotkey(initial); err == nil {
		candidates = append(candidates, h.String())
	}
	candidates = append(candidates, DefaultHotkey, "Alt+Q")

	seen := make(map[string]bool, len(candidates))
	result := make([]string, 0, len(candidates))
	for _, spec := range candidates {
		if seen[spec] {
			continue
		}
		seen[spec] = true
		result = append(result, spec)
	}
	return result
}

func keyCode(token string) (uint32, bool) {
	upper := strings.ToUpper(token)
	if len(upper) == 1 {
		switch c := upper[0]; {
		case c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			return uint32(c), true
		}
		return 0, false
	}
	switch upper {
	case "SPACE":
		return 0x20, true
	case "ENTER":
		return 0x0D, true
	case "TAB":
		return 0x09, true
	}
	if strings.HasPrefix(upper, "F") {
		if n, err := strconv.Atoi(upper[1:]); err == nil && n >= 1 && n <= 12 {
			return uint32(0x70 + n - 1), true
		}
	}
	return 0, false
}

func keyName(code uint32) (string, bool) {
	switch {
	case code >= 0x41 && code <= 0x5A, code >= 0x30 && code <= 0x39:
		return string(rune(code)), true
	case code == 0x20:
		return "Space", true
	case code == 0x0D:
		return "Enter", true
	case code == 0x09:
		return "Tab", true
	case code >= 0x70 && code <= 0x7B:
		return "F" + strconv.Itoa(int(code-0x70)+1), true
	}
	return "", false
}

// 窗口模式。
const (
	ModeOverlay = "overlay"
	ModeManager = "manager"
)

// 窗口逻辑尺寸（96 DPI 基准）。
const (
	OverlayLogicalWidth  = 560
	OverlayLogicalHeight = 380
	ManagerLogicalWidth  = 960
	ManagerLogicalHeight = 640
)

// Point 是物理像素坐标。
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Rect 是物理像素矩形，Right/Bottom 为开区间边界。
type Rect struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
}

// Callbacks 是由原生消息循环派发的用户动作。
type Callbacks struct {
	OnToggleOverlay   func()
	OnOverlayBlur     func()
	OnTrayNewSnippet  func()
	OnTrayOpenManager func()
	OnTrayQuit        func()
}

// Bridge 是需要按操作系统分别实现的窗口与键入能力。
type Bridge interface {
	Start(cb Callbacks) error
	Stop()
	ShowOverlayAtCursor(logicalWidth, logicalHeight int) Point
	ShowManagerWindow(logicalWidth, logicalHeight int)
	HideWindow()
	MinimizeWindow()
	PasteTextToTarget(text string) error
	// SetHotkey 可从任意 goroutine 调用；注册动作必须发生在消息线程。
	SetHotkey(spec string) error
}

// ScaleByDPI 将逻辑像素换算为物理像素，向上取整；dpi 非法时按 96 处理。
func ScaleByDPI(logicalPx, dpi int) int {
	if dpi <= 0 {
		dpi = 96
	}
	return (logicalPx*dpi + 48) / 96
}

// ClampWindowPosition 以锚点为基准偏移 8px 计算悬浮窗左上角，
// 并在超出显示器工作区右/下/左/上边界时回推至工作区内。
func ClampWindowPosition(anchor Point, physWidth, physHeight int, workArea Rect) Point {
	const offset = 8

	x := anchor.X + offset
	y := anchor.Y + offset

	if maxX := workArea.Right - physWidth; x > maxX {
		x = maxX
	}
	if maxY := workArea.Bottom - physHeight; y > maxY {
		y = maxY
	}
	if x < workArea.Left {
		x = workArea.Left
	}
	if y < workArea.Top {
		y = workArea.Top
	}
	return Point{X: x, Y: y}
}
