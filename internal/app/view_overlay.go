package app

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"quick-input/internal/store"
)

// OverlayView builds the 560x380 quick panel. The search input keeps the
// focus; Up/Down/Enter/Escape go through it or the root's single fallback
// path, never a second keyboard handler in the list.
func (a *App) OverlayView(c *ui.Context) {
	t := useMD3(c)
	p := mdColors(t)
	items := a.filteredSnippets(a.overlay.filter)
	if a.overlay.chosen >= len(items) {
		a.overlay.chosen = max(len(items)-1, 0)
	}
	if a.overlay.chosen < 0 {
		a.overlay.chosen = 0
	}

	a.overlay.list.Key = func(i int) any { return items[i].ID }
	a.overlay.list.Label = func(i int) string { return items[i].Title }

	root := ui.Column(c).Fill().Background(p.Surface).Radius(radiusSheet).Clip()
	root.Children(func() {
		a.overlaySearch(c, t, items)
		a.overlayBody(c, t, p, items)
	})
}

func (a *App) overlaySearch(c *ui.Context, t *ui.Theme, items []store.Snippet) {
	row := ui.Row(c).Height(44).Padding(0, 12).Gap(8).
		Background(t.Background).BorderWidth(0, 0, 1, 0).BorderColor(t.Border)
	row.Children(func() {
		ui.Icon(c, iconSearch).FontSize(16).TextColor(t.TextMuted)
		in := ui.TextInput(c, &a.overlay.filter.Keyword).
			Grow(1).FontSize(13).Radius(radiusField).Background(ui.Transparent).Border(0, ui.Transparent).
			Placeholder("搜索文本或内容").AutoFocus().Bind(&a.overlay.search)
		in.OnTextKey(func(k ui.TextKey) bool {
			switch {
			case k.Key == ui.KeyUp || k.Key == ui.KeyDown:
				if len(items) == 0 {
					return true
				}
				step := 1
				if k.Key == ui.KeyUp {
					step = -1
				}
				a.overlay.chosen = nextIndex(a.overlay.chosen, len(items), step)
				a.overlay.list.ScrollIntoView(a.overlay.chosen)
				return true
			case k.Key == ui.KeyEscape && a.overlay.filter.Keyword != "":
				k.SetText("")
				a.overlay.chosen = 0
				a.overlay.list.ScrollTo(0, ui.Start)
				return true
			}
			return false
		})
		if in.Changed() {
			a.overlay.chosen = 0
			a.overlay.list.ScrollTo(0, ui.Start)
		}
		if in.Submitted() {
			a.insertChosen(items)
		}
		if a.overlay.filter.Keyword != "" {
			if iconButton(c, iconClose, "清除搜索").Clicked() {
				a.overlay.filter.Keyword = ""
				a.overlay.chosen = 0
				a.overlay.search.Focus()
			}
		}
	})

	// Enter when the focus left the field, and Escape when the field is
	// empty and therefore does not consume it.
	if !a.overlay.search.FocusWithin(c) && c.Shortcut(0, ui.KeyEnter) {
		a.insertChosen(items)
	}
	if c.Shortcut(0, ui.KeyEscape) {
		switch {
		case a.overlay.filter.Keyword != "":
			a.overlay.filter.Keyword = ""
			a.overlay.chosen = 0
			a.overlay.list.ScrollTo(0, ui.Start)
		case a.overlay.filter.Scope != scopeAll || len(a.overlay.filter.TagIDs) > 0:
			a.overlay.filter.Scope = scopeAll
			a.overlay.filter.GroupID = nil
			a.overlay.filter.TagIDs = nil
			a.overlay.chosen = 0
			a.overlay.list.ScrollTo(0, ui.Start)
		default:
			a.HideOverlay()
		}
	}
}

func (a *App) insertChosen(items []store.Snippet) {
	if len(items) == 0 {
		return
	}
	a.insert(items[a.overlay.chosen].ID)
}

func (a *App) overlayBody(c *ui.Context, t *ui.Theme, p mdPalette, items []store.Snippet) {
	ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Grow(1).Children(func() {
			list := ui.List(c, &a.overlay.list, len(items), func(i int) {
				a.overlayRow(c, t, p, items, i)
			}).Grow(1).Padding(6)
			list.Children(func() {
				if len(items) == 0 {
					a.overlayEmpty(c, t, p)
				}
			})
		})
		a.overlayFilters(c, t, p)
	})
}

func (a *App) overlayRow(c *ui.Context, t *ui.Theme, p mdPalette, items []store.Snippet, i int) {
	snippet := items[i]
	row := ui.Row(c).Key(snippet.ID).Height(56).Padding(6, 10).Gap(10).Radius(radiusChip)
	if row.Hovered() && a.overlay.chosen != i {
		a.overlay.chosen = i
		c.Invalidate()
	}
	chosen := a.overlay.chosen == i
	switch {
	case chosen:
		row.Background(p.PrimaryContainer).TextColor(p.OnPrimaryContainer)
	case row.Hovered():
		row.Background(p.SurfaceHigh)
	}
	row.Children(func() {
		ui.Column(c).Grow(1).Justify(ui.Center).Gap(2).Children(func() {
			ui.Text(c, snippet.Title).FontSize(13).Bold().SingleLine()
			ui.Text(c, preview(snippet.Content, 96)).FontSize(11.5).TextColor(p.OnSurfaceVariant).SingleLine()
		})
		ui.Column(c).AlignItems(ui.End).Justify(ui.Center).Gap(3).Children(func() {
			meta := strings.Join(overlayMeta(snippet), " · ")
			if meta != "" {
				ui.Text(c, meta).FontSize(10.5).TextColor(p.OnSurfaceVariant).SingleLine()
			}
		})
	})
	if row.Clicked() {
		a.insert(snippet.ID)
	}
}

func overlayMeta(snippet store.Snippet) []string {
	parts := make([]string, 0, 1+len(snippet.Tags))
	if snippet.GroupName != "" {
		parts = append(parts, snippet.GroupName)
	}
	for _, tag := range snippet.Tags {
		parts = append(parts, tag.Name)
	}
	return parts
}

func (a *App) overlayFilters(c *ui.Context, t *ui.Theme, p mdPalette) {
	side := ui.Column(c).Width(184).Background(p.SurfaceLow).
		BorderWidth(1, 0, 0, 0).BorderColor(p.OutlineVariant)
	side.Children(func() {
		ui.Scroll(c).Grow(1).Padding(10).Gap(6).Children(func() {
			sectionTitle(c, "分组")
			a.groupRow(c, t, p, "全部", scopeAll, nil)
			a.groupRow(c, t, p, "未分组", scopeUngrouped, nil)
			for _, group := range a.groups {
				group := group
				a.groupRow(c, t, p, group.Name, scopeGroup, &group.ID)
			}
			ui.Box(c).Height(4)
			sectionTitle(c, "标签")
			if len(a.tags) == 0 {
				ui.Text(c, "暂无标签").FontSize(10.5).TextColor(p.OnSurfaceVariant).Padding(2, 6)
			}
			for _, tag := range a.tags {
				tag := tag
				on := hasID(a.overlay.filter.TagIDs, tag.ID)
				chip := ui.Toggle(c, &on, tag.Name).Height(26).Radius(radiusChip).
					Padding(0, 10).FontSize(11).Justify(ui.Start).AlignSelf(ui.Start)
				if on {
					chip.Background(p.PrimaryContainer).TextColor(p.OnPrimaryContainer)
				} else {
					chip.Background(p.Surface).TextColor(p.OnSurfaceVariant).Border(1, p.OutlineVariant)
				}
				if chip.Changed() {
					a.overlay.filter.TagIDs = toggleID(a.overlay.filter.TagIDs, tag.ID)
					a.overlay.chosen = 0
					a.overlay.list.ScrollTo(0, ui.Start)
				}
			}
		})
	})
}

func (a *App) groupRow(c *ui.Context, t *ui.Theme, p mdPalette, label, scope string, id *int64) {
	row := ui.Row(c).Height(28).Padding(0, 8).Gap(7).Radius(radiusChip)
	active := a.overlay.filter.Scope == scope &&
		(scope != scopeGroup || (id != nil && a.overlay.filter.GroupID != nil && *id == *a.overlay.filter.GroupID))
	switch {
	case active:
		row.Background(p.PrimaryContainer).TextColor(p.OnPrimaryContainer)
	case row.Hovered():
		row.Background(p.SurfaceHigh)
	}
	row.Children(func() {
		ui.Text(c, label).FontSize(11.5).SingleLine().Grow(1)
	})
	if row.Clicked() {
		a.overlay.filter.Scope = scope
		a.overlay.filter.GroupID = nil
		if scope == scopeGroup {
			a.overlay.filter.GroupID = id
		}
		a.overlay.chosen = 0
		a.overlay.list.ScrollTo(0, ui.Start)
	}
}

func (a *App) overlayEmpty(c *ui.Context, t *ui.Theme, p mdPalette) {
	filtered := a.overlay.filter.Keyword != "" || a.overlay.filter.Scope != scopeAll || len(a.overlay.filter.TagIDs) > 0
	ui.Column(c).Fill().Center().Gap(8).Children(func() {
		ui.Icon(c, iconInbox).FontSize(22).TextColor(p.OnSurfaceVariant)
		if filtered {
			ui.Text(c, "没有匹配的预设文本").FontSize(12).TextColor(p.OnSurfaceVariant)
			if textButton(c, "清除筛选").Clicked() {
				a.overlay.filter = FilterState{Scope: scopeAll}
			}
			return
		}
		ui.Text(c, "暂无预设文本").FontSize(12).TextColor(p.OnSurfaceVariant)
		if textButton(c, "新建预设文本").Clicked() {
			a.ShowManager(true)
		}
	})
}
