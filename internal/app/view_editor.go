package app

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"quick-input/internal/store"
)

// editorSheet builds the right-side sheet for a new or existing snippet.
func (a *App) editorSheet(c *ui.Context, t *ui.Theme, p mdPalette) {
	draft := a.editor
	if draft == nil {
		return
	}

	ui.Overlay(c, func() {
		back := ui.Row(c).Fill().Background(mdScrim).Modal().Justify(ui.End)
		if back.Clicked() {
			a.editor = nil
		}
		back.Children(func() {
			sheet := ui.Column(c).Absolute().Top(8).Right(8).Bottom(8).Width(420).
				Radius(radiusSheet).Background(p.SurfaceHigh).Padding(20).Gap(14).
				Role(ui.RoleDialog).Label("预设文本")
			sheet.Children(func() {
				head := ui.Row(c).Height(28).Gap(8)
				head.Children(func() {
					title := "新建预设文本"
					if draft.id != 0 {
						title = "编辑预设文本"
					}
					ui.Text(c, title).FontSize(13.5).Bold().Grow(1)
					if iconButton(c, iconClose, "关闭").Clicked() {
						a.editor = nil
					}
				})

				ui.Scroll(c).Grow(1).Gap(12).Children(func() {
					ui.Field(c, "标题", func() {
						ui.TextInput(c, &draft.title).AutoFocus().FontSize(12.5).Radius(radiusField)
					})
					ui.Field(c, "内容", func() {
						ui.TextArea(c, &draft.content).Lines(6, 12).FontSize(12.5).Radius(radiusField)
					})
					ui.Field(c, "分组", func() {
						a.editorGroupSelect(c, draft)
					})
					ui.Field(c, "标签", func() {
						a.editorTags(c, t, p, draft)
					})
					a.editorNewTag(c, t, draft)
				})

				footer := ui.Row(c).Gap(8)
				footer.Children(func() {
					if draft.err != "" {
						ui.Text(c, draft.err).FontSize(11).TextColor(p.Error).Grow(1).Wrap()
					} else {
						ui.Spacer(c)
					}
					if textButton(c, "取消").Clicked() {
						a.editor = nil
					}
					valid := strings.TrimSpace(draft.title) != "" && strings.TrimSpace(draft.content) != ""
					if filledButton(c, nil, "保存").Disabled(!valid).Clicked() {
						a.saveDraft()
					}
				})
			})
		})
	})

	if c.Shortcut(0, ui.KeyEscape) {
		a.editor = nil
	}
}

func (a *App) editorGroupSelect(c *ui.Context, draft *editorState) {
	name := "未分组"
	if draft.groupID != nil {
		if group, ok := a.groupByID(*draft.groupID); ok {
			name = group.Name
		}
	}
	options := make([]string, 0, len(a.groups)+1)
	options = append(options, "未分组")
	for _, group := range a.groups {
		options = append(options, group.Name)
	}
	if ui.Select(c, &name, options).Radius(radiusField).Changed() {
		if name == "未分组" {
			draft.groupID = nil
			return
		}
		for _, group := range a.groups {
			if group.Name == name {
				id := group.ID
				draft.groupID = &id
				return
			}
		}
	}
}

func (a *App) editorTags(c *ui.Context, t *ui.Theme, p mdPalette, draft *editorState) {
	if len(a.tags) == 0 {
		ui.Text(c, "暂无标签").FontSize(11).TextColor(p.OnSurfaceVariant)
		return
	}
	ui.Row(c).Wrap().Gap(6).Children(func() {
		for _, tag := range a.tags {
			tag := tag
			on := hasID(draft.tagIDs, tag.ID)
			box := ui.Toggle(c, &on, tag.Name).Height(26).Radius(radiusChip).
				Padding(0, 8).FontSize(11)
			if on {
				box.Background(p.PrimaryContainer).TextColor(p.OnPrimaryContainer)
			} else {
				box.Background(ui.Transparent).Border(1, p.OutlineVariant).TextColor(p.OnSurfaceVariant)
			}
			if box.Changed() {
				draft.tagIDs = toggleID(draft.tagIDs, tag.ID)
			}
		}
	})
}

func (a *App) editorNewTag(c *ui.Context, t *ui.Theme, draft *editorState) {
	ui.Row(c).Gap(8).Children(func() {
		in := ui.TextInput(c, &draft.newTag).Placeholder("新建标签").FontSize(11.5).Radius(radiusField).Grow(1)
		if in.Submitted() {
			a.addTagToDraft()
		}
		if tonalButton(c, "添加").Clicked() {
			a.addTagToDraft()
		}
	})
}

func (a *App) addTagToDraft() {
	draft := a.editor
	if draft == nil {
		return
	}
	name := strings.TrimSpace(draft.newTag)
	if name == "" {
		return
	}
	if err := a.saveTag(0, name); err != nil {
		draft.err = err.Error()
		return
	}
	for _, tag := range a.tags {
		if strings.EqualFold(tag.Name, name) {
			if !hasID(draft.tagIDs, tag.ID) {
				draft.tagIDs = append(draft.tagIDs, tag.ID)
			}
			break
		}
	}
	draft.newTag = ""
	draft.err = ""
}

func (a *App) saveDraft() {
	draft := a.editor
	if draft == nil {
		return
	}
	title := strings.TrimSpace(draft.title)
	content := strings.TrimSpace(draft.content)
	if title == "" || content == "" {
		draft.err = "标题与内容不能为空"
		return
	}
	err := a.saveSnippet(store.SnippetInput{
		ID:      draft.id,
		Title:   title,
		Content: content,
		GroupID: draft.groupID,
		TagIDs:  draft.tagIDs,
	})
	if err != nil {
		draft.err = err.Error()
		return
	}
	a.editor = nil
	a.lib.err = ""
}
