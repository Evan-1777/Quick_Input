package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"quick-input/internal/store"
)

// ManagerView builds the library window: app bar, sidebar, list, batch
// actions, and the editor or settings sheet above them.
func (a *App) ManagerView(c *ui.Context) {
	t := useMD3(c)
	p := mdColors(t)

	root := ui.Column(c).Fill().Background(t.Background)
	root.Children(func() {
		a.appBar(c, t, p)
		if a.lib.page == pageSettings {
			a.settingsView(c, t, p)
		} else {
			ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
				a.sidebar(c, t, p)
				a.libraryMain(c, t, p)
			})
		}
		if a.editor != nil {
			a.editorSheet(c, t, p)
		}
		a.confirmDialog(c)
	})
}

func (a *App) appBar(c *ui.Context, t *ui.Theme, p mdPalette) {
	bar := ui.Row(c).Height(52).Padding(0, 12).Gap(9).
		Background(p.Surface).BorderWidth(0, 0, 1, 0).BorderColor(p.OutlineVariant)
	bar.Children(func() {
		appMark(c, 22)
		ui.Text(c, "Quick Input").FontSize(13.5).Bold()
		ui.Spacer(c)
		settings := iconButton(c, iconSettings, "设置")
		if a.lib.page == pageSettings {
			settings.Background(p.PrimaryContainer).TextColor(p.OnPrimaryContainer)
		}
		if settings.Clicked() {
			if a.lib.page == pageSettings {
				a.lib.page = pageList
				a.lib.err = ""
			} else {
				a.ShowSettings()
			}
		}
		if iconButton(c, iconPanel, "唤出面板").Clicked() {
			a.ShowOverlay()
		}
	})
}

func (a *App) sidebar(c *ui.Context, t *ui.Theme, p mdPalette) {
	side := ui.Column(c).Width(232).FillHeight().Background(p.SurfaceLow).
		BorderWidth(0, 1, 0, 0).BorderColor(p.OutlineVariant)
	side.Children(func() {
		ui.Scroll(c).Grow(1).Padding(8).Gap(3).Children(func() {
			head := ui.Row(c).Height(30).Padding(4, 6).Gap(6)
			head.Children(func() {
				ui.Text(c, "分组").FontSize(10.5).Bold().TextColor(p.OnSurfaceVariant).Grow(1)
				if iconButton(c, iconPlus, "新建分组").Clicked() {
					a.startAdding("group")
				}
			})
			if a.lib.addingKind == "group" {
				a.inlineInput(c, t, "分组名称", func(value string) error { return a.saveGroup(0, value) })
			}
			a.sidebarGroup(c, t, p, nil, "全部", scopeAll)
			a.sidebarGroup(c, t, p, nil, "未分组", scopeUngrouped)
			for _, group := range a.groups {
				group := group
				a.sidebarGroup(c, t, p, &group, group.Name, scopeGroup)
			}

			ui.Box(c).Height(6)
			tagHead := ui.Row(c).Height(30).Padding(4, 6).Gap(6)
			tagHead.Children(func() {
				ui.Text(c, "标签").FontSize(10.5).Bold().TextColor(p.OnSurfaceVariant).Grow(1)
				if iconButton(c, iconPlus, "新建标签").Clicked() {
					a.startAdding("tag")
				}
			})
			if a.lib.addingKind == "tag" {
				a.inlineInput(c, t, "标签名称", func(value string) error { return a.saveTag(0, value) })
			}
			if len(a.tags) == 0 {
				ui.Text(c, "暂无标签").FontSize(10.5).TextColor(p.OnSurfaceVariant).Padding(4, 6)
			}
			for _, tag := range a.tags {
				tag := tag
				a.sidebarTag(c, t, p, tag)
			}
		})
	})
}

func (a *App) sidebarGroup(c *ui.Context, t *ui.Theme, p mdPalette, group *store.Group, label, scope string) {
	row := ui.Row(c).Height(30).Padding(0, 8).Gap(8).Radius(radiusChip)
	active := a.lib.filter.Scope == scope
	if scope == scopeGroup && group != nil {
		active = a.lib.filter.Scope == scopeGroup && a.lib.filter.GroupID != nil && *a.lib.filter.GroupID == group.ID
	}
	switch {
	case active:
		row.Background(p.PrimaryContainer).TextColor(p.OnPrimaryContainer)
	case row.Hovered():
		row.Background(p.SurfaceHigh)
	}

	if scope == scopeGroup && a.lib.renamingKind == "group" && a.lib.renamingID == group.ID {
		a.renameInput(c, t, func(value string) error { return a.saveGroup(group.ID, value) })
		return
	}

	row.Children(func() {
		if group == nil {
			ui.Icon(c, iconInbox).FontSize(14).TextColor(p.OnSurfaceVariant)
		} else {
			ui.Icon(c, iconFolder).FontSize(14).TextColor(p.OnSurfaceVariant)
		}
		ui.Text(c, label).FontSize(12).SingleLine().Grow(1)
		if group != nil && row.Hovered() {
			if iconButton(c, iconPencil, "重命名分组").Clicked() {
				a.startRenaming("group", group.ID, group.Name)
			}
			if iconButton(c, iconTrash, "删除分组").Clicked() {
				a.askDeleteGroup(*group)
			}
		}
	})
	if row.Clicked() {
		a.lib.filter.Scope = scope
		a.lib.filter.GroupID = nil
		if group != nil {
			a.lib.filter.GroupID = &group.ID
		}
	}
}

func (a *App) sidebarTag(c *ui.Context, t *ui.Theme, p mdPalette, tag store.Tag) {
	row := ui.Row(c).Height(30).Padding(0, 8).Gap(8).Radius(radiusChip)
	active := hasID(a.lib.filter.TagIDs, tag.ID)
	switch {
	case active:
		row.Background(p.PrimaryContainer).TextColor(p.OnPrimaryContainer)
	case row.Hovered():
		row.Background(p.SurfaceHigh)
	}
	if a.lib.renamingKind == "tag" && a.lib.renamingID == tag.ID {
		a.renameInput(c, t, func(value string) error { return a.saveTag(tag.ID, value) })
		return
	}
	row.Children(func() {
		ui.Icon(c, iconTag).FontSize(14).TextColor(p.OnSurfaceVariant)
		ui.Text(c, tag.Name).FontSize(12).SingleLine().Grow(1)
		if row.Hovered() {
			if iconButton(c, iconPencil, "重命名标签").Clicked() {
				a.startRenaming("tag", tag.ID, tag.Name)
			}
			if iconButton(c, iconTrash, "删除标签").Clicked() {
				a.askDeleteTag(tag)
			}
		}
	})
	if row.Clicked() {
		a.lib.filter.TagIDs = toggleID(a.lib.filter.TagIDs, tag.ID)
	}
}

func (a *App) startAdding(kind string) {
	a.lib.addingKind = kind
	a.lib.addingValue = ""
	a.lib.renamingKind = ""
}

func (a *App) startRenaming(kind string, id int64, value string) {
	a.lib.renamingKind = kind
	a.lib.renamingID = id
	a.lib.renamingValue = value
	a.lib.addingKind = ""
}

// inlineInput edits a new group or tag, submitted with Enter and canceled
// with Escape. The draft lives in the model, not in a per-frame variable.
func (a *App) inlineInput(c *ui.Context, t *ui.Theme, placeholder string, save func(string) error) {
	in := ui.TextInput(c, &a.lib.addingValue).AutoFocus().FontSize(11.5).
		Placeholder(placeholder).Height(28).Radius(radiusField)
	in.OnTextKey(func(k ui.TextKey) bool {
		if k.Key == ui.KeyEscape {
			a.lib.addingKind, a.lib.addingValue = "", ""
			return true
		}
		return false
	})
	if in.Submitted() {
		a.commitInline(save)
	}
}

func (a *App) commitInline(save func(string) error) {
	value := strings.TrimSpace(a.lib.addingValue)
	if value == "" {
		a.lib.addingKind, a.lib.addingValue = "", ""
		return
	}
	if err := save(value); err != nil {
		a.lib.err = err.Error()
		return
	}
	a.lib.addingKind, a.lib.addingValue = "", ""
	a.lib.err = ""
}

func (a *App) renameInput(c *ui.Context, t *ui.Theme, save func(string) error) {
	in := ui.TextInput(c, &a.lib.renamingValue).AutoFocus().FontSize(11.5).Height(28).Radius(radiusField)
	in.OnTextKey(func(k ui.TextKey) bool {
		if k.Key == ui.KeyEscape {
			a.lib.renamingKind = ""
			return true
		}
		return false
	})
	if in.Submitted() {
		value := strings.TrimSpace(a.lib.renamingValue)
		if value != "" {
			if err := save(value); err != nil {
				a.lib.err = err.Error()
				return
			}
		}
		a.lib.renamingKind = ""
		a.lib.err = ""
	}
}

func (a *App) libraryMain(c *ui.Context, t *ui.Theme, p mdPalette) {
	items := a.filteredSnippets(a.lib.filter)
	a.lib.list.Key = func(i int) any { return items[i].ID }
	a.lib.list.Label = func(i int) string { return items[i].Title }
	a.lib.list.Selection = &a.lib.chosen

	main := ui.Column(c).Grow(1).FillHeight()
	main.Children(func() {
		a.toolbar(c, t, p)
		if a.lib.err != "" {
			a.errorBar(c, t, p)
		}
		if a.lib.chosen.Len() > 0 {
			a.batchBar(c, t, p)
		}
		ui.List(c, &a.lib.list, len(items), func(i int) {
			a.libraryRow(c, t, p, items, i)
		}).Grow(1).Padding(8).Children(func() {
			if len(items) == 0 {
				a.libraryEmpty(c, t, p)
			}
		})
	})
}

func (a *App) toolbar(c *ui.Context, t *ui.Theme, p mdPalette) {
	bar := ui.Row(c).Height(56).Padding(12, 12).Gap(10)
	bar.Children(func() {
		ui.SearchField(c, &a.lib.filter.Keyword).Placeholder("搜索标题或内容").
			Grow(1).FontSize(12.5).Radius(radiusField)
		if filledButton(c, iconPlus, "新建").Clicked() {
			a.openEditor(nil)
		}
	})
}

func (a *App) errorBar(c *ui.Context, t *ui.Theme, p mdPalette) {
	bar := ui.Row(c).Padding(8, 12).Gap(8).Background(p.ErrorContainer)
	bar.Children(func() {
		ui.Icon(c, iconAlert).FontSize(15).TextColor(p.OnErrorContainer)
		ui.Text(c, a.lib.err).FontSize(11.5).TextColor(p.OnErrorContainer).Grow(1).Wrap()
		if iconButton(c, iconClose, "关闭错误").Clicked() {
			a.lib.err = ""
		}
	})
}

func (a *App) batchBar(c *ui.Context, t *ui.Theme, p mdPalette) {
	ids := a.chosenIDs()
	bar := ui.Row(c).Padding(8, 12).Gap(8).Background(p.PrimaryContainer)
	bar.Children(func() {
		ui.Text(c, fmt.Sprintf("已选 %d 项", len(ids))).FontSize(11.5).
			TextColor(p.OnPrimaryContainer)
		ui.PopoverMenuButton(c, "移入分组", func(m *ui.Menu) {
			if m.Item("未分组").Chosen() {
				a.report(a.moveSnippets(ids, nil))
			}
			for _, group := range a.groups {
				group := group
				if m.Item(group.Name).Chosen() {
					a.report(a.moveSnippets(ids, &group.ID))
				}
			}
		}).Radius(radiusPill).Padding(6, 12)
		ui.PopoverMenuButton(c, "添加标签", func(m *ui.Menu) {
			for _, tag := range a.tags {
				tag := tag
				if m.Item(tag.Name).Chosen() {
					a.report(a.updateTags(ids, []int64{tag.ID}, nil))
				}
			}
		}).Radius(radiusPill).Padding(6, 12)
		ui.PopoverMenuButton(c, "移除标签", func(m *ui.Menu) {
			for _, tag := range a.tags {
				tag := tag
				if m.Item(tag.Name).Chosen() {
					a.report(a.updateTags(ids, nil, []int64{tag.ID}))
				}
			}
		}).Radius(radiusPill).Padding(6, 12)
		ui.Spacer(c)
		if dangerButton(c, "删除所选").Clicked() {
			a.askDeleteSnippets(ids)
		}
		if textButton(c, "取消选择").Clicked() {
			a.lib.chosen.Clear()
		}
	})
}

func (a *App) libraryRow(c *ui.Context, t *ui.Theme, p mdPalette, items []store.Snippet, i int) {
	snippet := items[i]
	chosen := a.lib.chosen.Has(snippet.ID)
	row := ui.Row(c).Key(snippet.ID).Height(58).Padding(8, 16).Gap(10).Radius(radiusChip)
	switch {
	case chosen:
		row.Background(p.PrimaryContainer).TextColor(p.OnPrimaryContainer)
	case row.Hovered():
		row.Background(p.SurfaceHigh)
	}
	hovered := row.Hovered()
	row.Children(func() {
		ui.Column(c).Grow(1).Justify(ui.Center).Gap(3).Children(func() {
			ui.Text(c, snippet.Title).FontSize(13).Bold().SingleLine()
			ui.Text(c, preview(snippet.Content, 110)).FontSize(11.5).
				TextColor(p.OnSurfaceVariant).SingleLine()
		})
		ui.Row(c).Gap(5).Children(func() {
			chips := 0
			if snippet.GroupName != "" {
				chip(c, snippet.GroupName)
				chips++
			}
			for _, tag := range snippet.Tags {
				if chips >= 3 {
					break
				}
				chip(c, tag.Name)
				chips++
			}
		})
		if hovered {
			if iconButton(c, iconPencil, "编辑").Clicked() {
				a.openEditor(&snippet)
			}
			if iconButton(c, iconTrash, "删除").Clicked() {
				a.askDeleteSnippets([]int64{snippet.ID})
			}
		}
	})
	if row.RowDoubleClicked() {
		a.openEditor(&snippet)
	}
}

func (a *App) libraryEmpty(c *ui.Context, t *ui.Theme, p mdPalette) {
	filtered := a.lib.filter.Keyword != "" || a.lib.filter.Scope != scopeAll || len(a.lib.filter.TagIDs) > 0
	ui.Column(c).Fill().Center().Gap(8).Children(func() {
		ui.Icon(c, iconInbox).FontSize(24).TextColor(p.OnSurfaceVariant)
		if filtered {
			ui.Text(c, "没有匹配的预设文本").FontSize(12).TextColor(p.OnSurfaceVariant)
			if textButton(c, "清除筛选").Clicked() {
				a.lib.filter = FilterState{Scope: scopeAll}
			}
			return
		}
		ui.Text(c, "暂无预设文本").FontSize(12).TextColor(p.OnSurfaceVariant)
		if textButton(c, "新建预设文本").Clicked() {
			a.openEditor(nil)
		}
	})
}

// report shows a data-operation error without interrupting the view.
func (a *App) report(err error) {
	if err != nil {
		a.lib.err = err.Error()
	}
}

func (a *App) askDeleteSnippets(ids []int64) {
	if len(ids) == 0 {
		return
	}
	message := fmt.Sprintf("将删除 %d 条预设文本，且不可撤销。", len(ids))
	if len(ids) == 1 {
		for _, snippet := range a.snippets {
			if snippet.ID == ids[0] {
				message = fmt.Sprintf("「%s」将被删除，且不可撤销。", snippet.Title)
				break
			}
		}
	}
	a.lib.confirm = &confirmState{
		title:   "删除预设文本",
		message: message,
		run: func() {
			if err := a.deleteSnippets(ids); err != nil {
				a.lib.err = err.Error()
			}
		},
	}
}

func (a *App) askDeleteGroup(group store.Group) {
	a.lib.confirm = &confirmState{
		title:   "删除分组",
		message: fmt.Sprintf("「%s」下的预设文本将变为未分组，且不可撤销。", group.Name),
		run: func() {
			if err := a.deleteGroup(group.ID); err != nil {
				a.lib.err = err.Error()
				return
			}
			a.lib.filter.Scope = scopeAll
			a.lib.filter.GroupID = nil
		},
	}
}

func (a *App) askDeleteTag(tag store.Tag) {
	a.lib.confirm = &confirmState{
		title:   "删除标签",
		message: fmt.Sprintf("「%s」将从所有预设文本中解绑，且不可撤销。", tag.Name),
		run: func() {
			if err := a.deleteTag(tag.ID); err != nil {
				a.lib.err = err.Error()
			}
		},
	}
}

func (a *App) confirmDialog(c *ui.Context) {
	cf := a.lib.confirm
	if cf == nil {
		return
	}
	open := true
	switch ui.AlertDialog(c, &open, cf.title, cf.message, "取消", "删除") {
	case 1:
		a.lib.confirm = nil
		cf.run()
	case ui.AlertDismissed:
		a.lib.confirm = nil
	}
	if !open {
		a.lib.confirm = nil
	}
}
