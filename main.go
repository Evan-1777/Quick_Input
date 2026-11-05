package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"quick-input/internal/app"
	"quick-input/internal/platform"
	"quick-input/internal/store"
)

// quitting lets both windows close without their OnClose listeners hiding
// them, which would otherwise cancel App.Quit.
var quitting bool

func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	mygo.App.SetName("Quick Input")
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.OnWindowAllClosed(func() {})

	st, err := store.NewStore(databasePath())
	if err != nil {
		log.Fatalf("初始化数据库: %v", err)
	}
	defer st.Close()

	saved, err := st.GetSetting(store.SettingHotkey)
	if err != nil {
		log.Printf("读取快捷键设置失败: %v", err)
	}
	if saved == "" {
		saved = "Alt+Space"
	}

	application := app.New(st, platform.NewPaster(), app.NewShortcuts())
	application.SetStartupHotkey("", saved)

	mygo.App.OnSecondInstance(func([]string, string) {
		application.ShowManager(false)
	})
	mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) {
		quitting = true
	})

	mygo.App.WhenReady(func() {
		panel := mygo.NewWindow(mygo.WindowOptions{
			Title:           "Quick Input",
			Panel:           true,
			Hidden:          true,
			Frameless:       true,
			Transparent:     true,
			CornerRadius:    14,
			Width:           560,
			Height:          380,
			DisableResize:   true,
			BackgroundColor: "#00000000",
			Content:         ui.View(application.OverlayView),
		})
		manager := mygo.NewWindow(mygo.WindowOptions{
			Title:           "Quick Input",
			Width:           960,
			Height:          640,
			MinWidth:        760,
			MinHeight:       480,
			Hidden:          true,
			StateKey:        "manager",
			BackgroundColor: "light-dark(#F9F9FF, #111318)",
			Content:         ui.View(application.ManagerView),
		})
		application.Attach(panel, manager)

		panel.OnClose(func(e *mygo.CloseEvent) {
			if quitting {
				return
			}
			e.PreventDefault()
			panel.Hide()
		})
		panel.OnBlur(func() { panel.Hide() })
		manager.OnClose(func(e *mygo.CloseEvent) {
			if quitting {
				return
			}
			e.PreventDefault()
			manager.Hide()
		})

		registerHotkey(application, saved)
		setUpTray(application)
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// registerHotkey tries the persisted combination, then the two defaults,
// recording whichever the system accepts.
func registerHotkey(application *app.App, saved string) {
	candidates := []string{saved, "Alt+Space", "Alt+Q"}
	seen := make(map[string]bool, len(candidates))
	for _, spec := range candidates {
		if seen[spec] || spec == "" {
			continue
		}
		seen[spec] = true
		if err := application.RegisterHotkey(spec); err != nil {
			log.Printf("注册快捷键 %s 失败: %v", spec, err)
			continue
		}
		application.SetStartupHotkey(spec, saved)
		return
	}
	application.SetStartupHotkey("", saved)
	log.Printf("没有可用的唤出快捷键，仍可通过托盘打开")
}

func setUpTray(application *app.App) {
	icon, err := trayIcon()
	if err != nil {
		log.Printf("生成托盘图标失败: %v", err)
		return
	}
	tray, err := mygo.NewTray(mygo.TrayOptions{
		ID:      "quick-input",
		Icon:    icon,
		ToolTip: "Quick Input",
		Menu: mygo.NewMenu([]*mygo.MenuItem{
			{Label: "打开 Quick Input", Click: func(*mygo.MenuItem, *mygo.Window) {
				application.ShowManager(false)
			}},
			{Label: "新建预设文本", Click: func(*mygo.MenuItem, *mygo.Window) {
				application.ShowManager(true)
			}},
			mygo.Separator(),
			{Label: "退出", Click: func(*mygo.MenuItem, *mygo.Window) {
				quitting = true
				mygo.App.Quit()
			}},
		}),
	})
	if err != nil {
		log.Printf("创建托盘图标失败: %v", err)
		return
	}
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		tray.Destroy()
	})
}

// databasePath keeps the existing %APPDATA%\QuickInput\quick_input.db
// location so upgrades reuse the current library.
func databasePath() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	return filepath.Join(base, "QuickInput", "quick_input.db")
}

// trayIcon draws the tray mark at 32x32: an accent rounded tile with two
// text bars, the same mark the app bar shows.
func trayIcon() ([]byte, error) {
	const size = 32
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	tile := color.RGBA{R: 0x0B, G: 0x57, B: 0xD0, A: 0xFF}
	white := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if roundedInside(x, y, 2, 2, 29, 29, 8) {
				img.SetRGBA(x, y, tile)
			}
		}
	}
	for _, bar := range []image.Rectangle{
		image.Rect(9, 11, 23, 14),
		image.Rect(9, 18, 19, 21),
	} {
		for y := bar.Min.Y; y < bar.Max.Y; y++ {
			for x := bar.Min.X; x < bar.Max.X; x++ {
				img.SetRGBA(x, y, white)
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func roundedInside(x, y, left, top, right, bottom, radius int) bool {
	if x < left || x > right || y < top || y > bottom {
		return false
	}
	cx := left + radius
	if x > right-radius {
		cx = right - radius
	}
	cy := top + radius
	if y > bottom-radius {
		cy = bottom - radius
	}
	dx, dy := x-cx, y-cy
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx*dx+dy*dy <= radius*radius || (x >= left+radius && x <= right-radius) || (y >= top+radius && y <= bottom-radius)
}
