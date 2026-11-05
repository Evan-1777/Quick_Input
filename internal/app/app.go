package app

import (
	"errors"
	"fmt"
	"sort"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"quick-input/internal/platform"
	"quick-input/internal/store"
)

// Shortcuts is the global shortcut registry the app drives.
type Shortcuts interface {
	Register(spec string, fn func()) error
	Unregister(spec string)
}

type mygoShortcuts struct{}

func (mygoShortcuts) Register(spec string, fn func()) error {
	return mygo.GlobalShortcut.Register(spec, fn)
}
func (mygoShortcuts) Unregister(spec string) { mygo.GlobalShortcut.Unregister(spec) }

// NewShortcuts returns the real MyGo shortcut registry.
func NewShortcuts() Shortcuts { return mygoShortcuts{} }

type page int

const (
	pageList page = iota
	pageSettings
)

// overlayState is the quick panel's model. Handles and list state live here,
// never elements.
type overlayState struct {
	filter  FilterState
	chosen  int
	visible bool
	list    ui.ListState
	search  ui.Handle
}

// libraryState is the manager's model.
type libraryState struct {
	filter FilterState
	chosen ui.Selection[int64]
	list   ui.ListState
	page   page

	err string

	addingKind  string // "group" or "tag"
	addingValue string

	renamingKind  string
	renamingID    int64
	renamingValue string

	confirm *confirmState
}

type confirmState struct {
	title   string
	message string
	run     func()
}

// editorState is the side sheet's draft. It is replaced as a whole when the
// sheet opens so a failed save keeps every field.
type editorState struct {
	id      int64
	title   string
	content string
	groupID *int64
	tagIDs  []int64
	newTag  string
	err     string
}

// App owns the store, the platform paster and both window views.
type App struct {
	store     *store.Store
	paster    platform.Paster
	shortcuts Shortcuts

	panel   *mygo.Window
	manager *mygo.Window

	groups   []store.Group
	tags     []store.Tag
	snippets []store.Snippet

	// showUngrouped keeps ungrouped snippets in the quick panel's list and
	// group cycle. The manager always shows them.
	showUngrouped bool

	// hotkey is what the OS registered; saved is what the settings page
	// shows. A failed startup fallback keeps them apart, as before.
	hotkey string
	saved  string

	overlay overlayState
	lib     libraryState
	editor  *editorState
}

// New builds the app model. Windows are injected later by Attach.
func New(st *store.Store, paster platform.Paster, shortcuts Shortcuts) *App {
	if shortcuts == nil {
		shortcuts = NewShortcuts()
	}
	a := &App{store: st, paster: paster, shortcuts: shortcuts}
	a.lib.filter.Scope = scopeAll
	a.overlay.filter.Scope = scopeAll
	if value, err := st.GetSetting(store.SettingOverlayShowUngrouped); err == nil && value == "1" {
		a.showUngrouped = true
	}
	return a
}

// SetShowUngrouped persists the panel's ungrouped preference, then updates
// the in-memory value. Turning it off while the panel filters ungrouped
// resets that filter, which would otherwise hide every row.
func (a *App) SetShowUngrouped(show bool) error {
	value := "0"
	if show {
		value = "1"
	}
	if err := a.store.SetSetting(store.SettingOverlayShowUngrouped, value); err != nil {
		return fmt.Errorf("保存设置失败: %w", err)
	}
	a.showUngrouped = show
	if !show && a.overlay.filter.Scope == scopeUngrouped {
		a.overlay.filter = FilterState{Scope: scopeAll}
	}
	return nil
}

// SetStartupHotkey records the combination that actually registered and the
// persisted one shown in settings.
func (a *App) SetStartupHotkey(registered, saved string) {
	a.hotkey, a.saved = registered, saved
}

// RegisterHotkey registers spec for toggling the overlay.
func (a *App) RegisterHotkey(spec string) error {
	return a.shortcuts.Register(spec, a.ToggleOverlay)
}

// Attach injects the windows. Both may be nil in tests.
func (a *App) Attach(panel, manager *mygo.Window) {
	a.panel, a.manager = panel, manager
}

// Reload replaces every snapshot from SQLite and prunes stale filters.
func (a *App) Reload() error {
	groups, err := a.store.ListGroups()
	if err != nil {
		return err
	}
	tags, err := a.store.ListTags()
	if err != nil {
		return err
	}
	snippets, err := a.store.ListSnippets(store.SnippetFilter{GroupScope: scopeAll})
	if err != nil {
		return err
	}
	a.groups, a.tags, a.snippets = groups, tags, snippets
	pruneFilter(&a.overlay.filter, groups, tags)
	pruneFilter(&a.lib.filter, groups, tags)
	a.pruneSelection()
	return nil
}

func (a *App) reload() {
	if err := a.Reload(); err != nil {
		a.lib.err = err.Error()
	}
}

func (a *App) pruneSelection() {
	if a.lib.chosen.Len() == 0 {
		return
	}
	exists := make(map[int64]bool, len(a.snippets))
	for _, snippet := range a.snippets {
		exists[snippet.ID] = true
	}
	var stale []int64
	for id := range a.lib.chosen.All() {
		if !exists[id] {
			stale = append(stale, id)
		}
	}
	for _, id := range stale {
		a.lib.chosen.Remove(id)
	}
}

func (a *App) filteredSnippets(f FilterState) []store.Snippet {
	return filterSnippets(a.snippets, f)
}

// reorderSnippets moves rows to before the first visible row from to on
// that is not moving; no such row puts them at the end. Hidden rows keep
// their relative order around the visible anchor.
func (a *App) reorderSnippets(items []store.Snippet, rows []int, to int) {
	if len(rows) == 0 {
		return
	}
	ids := make([]int64, 0, len(rows))
	moving := make(map[int64]bool, len(rows))
	for _, row := range rows {
		if row < 0 || row >= len(items) {
			return
		}
		ids = append(ids, items[row].ID)
		moving[items[row].ID] = true
	}
	var before *int64
	for i := max(to, 0); i < len(items); i++ {
		if !moving[items[i].ID] {
			id := items[i].ID
			before = &id
			break
		}
	}
	if err := a.store.ReorderSnippets(ids, before); err != nil {
		a.lib.err = err.Error()
		return
	}
	a.reload()
}

func (a *App) groupByID(id int64) (store.Group, bool) {
	for _, group := range a.groups {
		if group.ID == id {
			return group, true
		}
	}
	return store.Group{}, false
}

// ShowOverlay snapshots the target, reloads, resets the panel and places it
// on the display under the pointer: horizontally centered, a third down.
func (a *App) ShowOverlay() {
	a.paster.CaptureTarget()
	a.reload()
	a.resetOverlay()
	a.overlay.visible = true
	if a.panel == nil {
		return
	}
	display := mygo.Screen.DisplayNearestPoint(mygo.Screen.CursorScreenPoint())
	width, height := a.panel.Size()
	work := display.WorkArea
	a.panel.SetPosition(work.X+(work.Width-width)/2, work.Y+(work.Height-height)/3)
	a.panel.Show()
	a.panel.Focus()
	a.overlay.search.Focus()
}

// HideOverlay hides the quick panel.
func (a *App) HideOverlay() {
	a.overlay.visible = false
	if a.panel != nil {
		a.panel.Hide()
	}
}

// ToggleOverlay shows or hides the quick panel.
func (a *App) ToggleOverlay() {
	if a.panel != nil && a.panel.IsVisible() {
		a.HideOverlay()
		return
	}
	a.ShowOverlay()
}

// ShowManager opens the manager, optionally straight into a new snippet.
func (a *App) ShowManager(create bool) {
	a.reload()
	a.lib.page = pageList
	if create {
		a.openEditor(nil)
	} else {
		a.editor = nil
	}
	if a.manager == nil {
		return
	}
	a.manager.Show()
	a.manager.Focus()
}

// ShowSettings switches the manager to the settings subpage.
func (a *App) ShowSettings() {
	a.lib.page = pageSettings
	a.lib.err = ""
	a.editor = nil
	if a.manager != nil {
		a.manager.Show()
		a.manager.Focus()
	}
}

func (a *App) resetOverlay() {
	a.overlay.filter = FilterState{Scope: scopeAll}
	a.overlay.chosen = 0
	a.overlay.list.ScrollTo(0, ui.Start)
}

// insert hides the panel, then pastes the snippet's latest content in the
// background so the UI thread is never blocked by focus waits.
func (a *App) insert(id int64) {
	snippet, err := a.store.GetSnippetByID(id)
	if err != nil {
		a.overlay.visible = false
		return
	}
	a.HideOverlay()
	paste := func() {
		if err := a.paster.Paste(snippet.Content); err != nil {
			mygo.RunOnMain(func() {
				a.lib.err = err.Error()
				if a.manager != nil {
					a.manager.Invalidate()
				}
			})
		}
	}
	if a.panel == nil {
		paste()
		return
	}
	go paste()
}

// SetHotkey registers the new combination first, then persists it. A save
// failure rolls the registration back, leaving the old combination active.
func (a *App) SetHotkey(spec string) error {
	if spec == a.hotkey {
		if spec == a.saved {
			return nil
		}
		if err := a.store.SetSetting(store.SettingHotkey, spec); err != nil {
			return fmt.Errorf("保存设置失败: %w", err)
		}
		a.saved = spec
		return nil
	}
	if err := a.shortcuts.Register(spec, a.ToggleOverlay); err != nil {
		return fmt.Errorf("快捷键 %s 注册失败，可能已被其他程序占用", spec)
	}
	if err := a.store.SetSetting(store.SettingHotkey, spec); err != nil {
		a.shortcuts.Unregister(spec)
		var rollback error
		if a.hotkey != "" {
			rollback = a.shortcuts.Register(a.hotkey, a.ToggleOverlay)
		}
		if rollback != nil {
			return fmt.Errorf("保存设置失败且无法恢复原快捷键: %w", errors.Join(err, rollback))
		}
		return fmt.Errorf("保存设置失败，已恢复原快捷键: %w", err)
	}
	old := a.hotkey
	a.hotkey, a.saved = spec, spec
	if old != "" && old != spec {
		a.shortcuts.Unregister(old)
	}
	return nil
}

func (a *App) saveSnippet(in store.SnippetInput) error {
	_, err := a.store.SaveSnippet(in)
	if err != nil {
		return err
	}
	a.reload()
	return nil
}

func (a *App) deleteSnippets(ids []int64) error {
	if err := a.store.DeleteSnippets(ids); err != nil {
		return err
	}
	for _, id := range ids {
		a.lib.chosen.Remove(id)
	}
	a.reload()
	return nil
}

func (a *App) saveGroup(id int64, name string) error {
	if _, err := a.store.SaveGroup(id, name); err != nil {
		return err
	}
	a.reload()
	return nil
}

func (a *App) deleteGroup(id int64) error {
	if err := a.store.DeleteGroup(id); err != nil {
		return err
	}
	if a.lib.filter.Scope == scopeGroup && a.lib.filter.GroupID != nil && *a.lib.filter.GroupID == id {
		a.lib.filter = FilterState{Scope: scopeAll}
	}
	a.reload()
	return nil
}

func (a *App) saveTag(id int64, name string) error {
	if _, err := a.store.SaveTag(id, name); err != nil {
		return err
	}
	a.reload()
	return nil
}

func (a *App) deleteTag(id int64) error {
	if err := a.store.DeleteTag(id); err != nil {
		return err
	}
	a.lib.filter.TagIDs = removeID(a.lib.filter.TagIDs, id)
	a.overlay.filter.TagIDs = removeID(a.overlay.filter.TagIDs, id)
	a.reload()
	return nil
}

func (a *App) moveSnippets(ids []int64, groupID *int64) error {
	if err := a.store.BatchMoveGroup(ids, groupID); err != nil {
		return err
	}
	a.reload()
	return nil
}

func (a *App) updateTags(ids []int64, add, remove []int64) error {
	if err := a.store.BatchUpdateTags(ids, add, remove); err != nil {
		return err
	}
	a.reload()
	return nil
}

func removeID(ids []int64, id int64) []int64 {
	result := ids[:0]
	for _, existing := range ids {
		if existing != id {
			result = append(result, existing)
		}
	}
	return result
}

// openEditor starts a draft for a snippet, or a new one when snippet is nil.
func (a *App) openEditor(snippet *store.Snippet) {
	draft := &editorState{}
	if snippet != nil {
		draft.id = snippet.ID
		draft.title = snippet.Title
		draft.content = snippet.Content
		draft.groupID = snippet.GroupID
		for _, tag := range snippet.Tags {
			draft.tagIDs = append(draft.tagIDs, tag.ID)
		}
	}
	a.editor = draft
}

// chosenIDs returns the current multi-selection in a stable order.
func (a *App) chosenIDs() []int64 {
	ids := make([]int64, 0, a.lib.chosen.Len())
	for id := range a.lib.chosen.All() {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
