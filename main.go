package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"quick-input/internal/app"
	"quick-input/internal/platform"
	"quick-input/internal/store"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	st, err := store.NewStore(databasePath())
	if err != nil {
		log.Fatalf("初始化数据库: %v", err)
	}
	defer st.Close()

	hotkey, err := st.GetSetting(store.SettingHotkey)
	if err != nil {
		log.Printf("读取快捷键设置失败: %v", err)
	}
	bridge := platform.NewBridge("Quick Input", hotkey)
	application := app.NewApp(st, bridge)

	err = wails.Run(&options.App{
		Title:             "Quick Input",
		Width:             platform.OverlayLogicalWidth,
		Height:            platform.OverlayLogicalHeight,
		Frameless:         true,
		StartHidden:       true,
		HideWindowOnClose: true,
		BackgroundColour:  &options.RGBA{R: 250, G: 250, B: 250, A: 255},
		AssetServer:       &assetserver.Options{Assets: assets},
		Bind:              []interface{}{application},
		OnStartup: func(ctx context.Context) {
			app.AttachContext(application, ctx)
			if err := bridge.Start(platform.Callbacks{
				OnToggleOverlay:   application.ToggleOverlay,
				OnTrayNewSnippet:  func() { application.SwitchToManager(true) },
				OnTrayOpenManager: func() { application.SwitchToManager(false) },
				OnTrayQuit:        application.Quit,
			}); err != nil {
				log.Printf("平台层启动失败: %v", err)
			}
		},
		OnShutdown: func(context.Context) {
			bridge.Stop()
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "quick-input-win11-lock",
			OnSecondInstanceLaunch: func(options.SecondInstanceData) {
				application.SwitchToManager(false)
			},
		},
	})
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

// databasePath 返回 %APPDATA%\QuickInput\quick_input.db（非 Windows 平台为配置目录）。
func databasePath() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	return filepath.Join(base, "QuickInput", "quick_input.db")
}
