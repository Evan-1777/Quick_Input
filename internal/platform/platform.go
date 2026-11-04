// Package platform 封装与操作系统窗口、剪贴板交互相关的原生能力。
// Windows 实现使用 Win32 API，其余平台使用可编译的内存测试桩。
package platform

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
