// Package app 编排存储层与平台层，向前端暴露 Wails 绑定方法。
package app

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"quick-input/internal/platform"
	"quick-input/internal/store"
)

// EventWindowMode 是窗口模式切换事件名。
const EventWindowMode = "window:mode"

// WindowStatePayload 是前端切换视图所需的窗口状态。
type WindowStatePayload struct {
	Mode        string `json:"mode"`
	OpenCreator bool   `json:"openCreator"`
}

// SettingsPayload 是全局配置。
type SettingsPayload struct {
	Hotkey string `json:"hotkey"`
}

// BootstrapPayload 是前端初始化所需的全部数据。
type BootstrapPayload struct {
	Groups      []store.Group      `json:"groups"`
	Tags        []store.Tag        `json:"tags"`
	Snippets    []store.Snippet    `json:"snippets"`
	WindowState WindowStatePayload `json:"windowState"`
	Settings    SettingsPayload    `json:"settings"`
}

// App 是 Wails 绑定服务。
type App struct {
	mu     sync.Mutex
	ctx    context.Context
	store  *store.Store
	bridge platform.Bridge
	state  WindowStatePayload
	shown  bool

	emit func(eventName string, data any)
}

// NewApp 构造绑定服务；emit 默认丢弃事件，Startup 时替换为 Wails 运行时实现。
func NewApp(st *store.Store, bridge platform.Bridge) *App {
	return &App{
		store:  st,
		bridge: bridge,
		state:  WindowStatePayload{Mode: platform.ModeOverlay},
		emit:   func(string, any) {},
	}
}

// AttachContext 绑定 Wails 运行时上下文；作为包级函数而非方法，避免被暴露为前端绑定。
func AttachContext(a *App, ctx context.Context) {
	a.ctx = ctx
	a.emit = func(eventName string, data any) {
		runtime.EventsEmit(ctx, eventName, data)
	}
}

// ToggleOverlay 处理全局热键：悬浮窗已展示时收起，否则在光标处唤出。
func (a *App) ToggleOverlay() {
	a.mu.Lock()
	visible, mode := a.shown, a.state.Mode
	a.mu.Unlock()

	if visible && mode == platform.ModeOverlay {
		a.HideWindow()
		return
	}
	a.SwitchToOverlay()
}

// SwitchToOverlay 在光标处展示快捷悬浮窗。
func (a *App) SwitchToOverlay() {
	a.bridge.ShowOverlayAtCursor(platform.OverlayLogicalWidth, platform.OverlayLogicalHeight)
	a.applyState(WindowStatePayload{Mode: platform.ModeOverlay})
}

// SwitchToManager 展示主管理窗口，按需直接展开新建表单。
func (a *App) SwitchToManager(openCreator bool) {
	a.bridge.ShowManagerWindow(platform.ManagerLogicalWidth, platform.ManagerLogicalHeight)
	a.applyState(WindowStatePayload{Mode: platform.ModeManager, OpenCreator: openCreator})
}

// HideWindow 隐藏当前窗口。
func (a *App) HideWindow() {
	a.bridge.HideWindow()
	a.mu.Lock()
	a.shown = false
	a.mu.Unlock()
}

// MinimizeWindow 最小化当前窗口。
func (a *App) MinimizeWindow() {
	a.bridge.MinimizeWindow()
	a.mu.Lock()
	a.shown = false
	a.mu.Unlock()
}

// Quit 退出应用（由托盘菜单调用）。
func (a *App) Quit() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

// GetBootstrapData 返回前端初始化数据与当前窗口状态。
func (a *App) GetBootstrapData() (BootstrapPayload, error) {
	groups, err := a.store.ListGroups()
	if err != nil {
		return BootstrapPayload{}, err
	}
	tags, err := a.store.ListTags()
	if err != nil {
		return BootstrapPayload{}, err
	}
	snippets, err := a.store.ListSnippets(store.SnippetFilter{GroupScope: "all"})
	if err != nil {
		return BootstrapPayload{}, err
	}
	hotkey, err := a.store.GetSetting(store.SettingHotkey)
	if err != nil {
		return BootstrapPayload{}, err
	}
	if hotkey == "" {
		hotkey = platform.DefaultHotkey
	}

	a.mu.Lock()
	state := a.state
	a.mu.Unlock()

	return BootstrapPayload{
		Groups:      groups,
		Tags:        tags,
		Snippets:    snippets,
		WindowState: state,
		Settings:    SettingsPayload{Hotkey: hotkey},
	}, nil
}

// SetHotkey 切换全局唤出快捷键：注册成功后持久化，写库失败时回滚注册。
func (a *App) SetHotkey(spec string) error {
	prev, err := a.store.GetSetting(store.SettingHotkey)
	if err != nil {
		return err
	}
	if prev == "" {
		prev = platform.DefaultHotkey
	}

	h, err := platform.ParseHotkey(spec)
	if err != nil {
		return err
	}
	if err := a.bridge.SetHotkey(h.String()); err != nil {
		return err
	}
	if err := a.store.SetSetting(store.SettingHotkey, h.String()); err != nil {
		if rollbackErr := a.bridge.SetHotkey(prev); rollbackErr != nil {
			return fmt.Errorf("保存设置失败且无法恢复原快捷键: %w", errors.Join(err, rollbackErr))
		}
		return fmt.Errorf("保存设置失败，已恢复原快捷键: %w", err)
	}
	return nil
}

// SaveSnippet 新建或编辑预设文本。
func (a *App) SaveSnippet(in store.SnippetInput) (store.Snippet, error) {
	return a.store.SaveSnippet(in)
}

// DeleteSnippets 批量删除预设文本。
func (a *App) DeleteSnippets(ids []int64) error {
	return a.store.DeleteSnippets(ids)
}

// SaveGroup 新建或重命名分组。
func (a *App) SaveGroup(id int64, name string) (store.Group, error) {
	return a.store.SaveGroup(id, name)
}

// DeleteGroup 删除分组并将其下文本重置为未分组。
func (a *App) DeleteGroup(id int64) error {
	return a.store.DeleteGroup(id)
}

// SaveTag 新建或重命名标签。
func (a *App) SaveTag(id int64, name string) (store.Tag, error) {
	return a.store.SaveTag(id, name)
}

// DeleteTag 删除标签并解绑关联文本。
func (a *App) DeleteTag(id int64) error {
	return a.store.DeleteTag(id)
}

// BatchMoveGroup 批量移动分组。
func (a *App) BatchMoveGroup(snippetIDs []int64, groupID *int64) error {
	return a.store.BatchMoveGroup(snippetIDs, groupID)
}

// BatchUpdateTags 批量绑定与解绑标签。
func (a *App) BatchUpdateTags(snippetIDs []int64, addTagIDs []int64, removeTagIDs []int64) error {
	return a.store.BatchUpdateTags(snippetIDs, addTagIDs, removeTagIDs)
}

// TriggerPaste 从存储读取最新文本并键入到唤出前的目标窗口。
func (a *App) TriggerPaste(snippetID int64) error {
	snippet, err := a.store.GetSnippetByID(snippetID)
	if err != nil {
		return err
	}
	if err := a.bridge.PasteTextToTarget(snippet.Content); err != nil {
		return err
	}
	a.mu.Lock()
	a.shown = false
	a.mu.Unlock()
	return nil
}

func (a *App) applyState(state WindowStatePayload) {
	a.mu.Lock()
	a.state = state
	a.shown = true
	a.mu.Unlock()

	a.emit(EventWindowMode, state)
}
