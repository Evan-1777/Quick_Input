package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// settingsView is the manager's settings subpage.
func (a *App) settingsView(c *ui.Context, t *ui.Theme, p mdPalette) {
	col := ui.Column(c).Grow(1).FillHeight().Background(p.Surface)
	col.Children(func() {
		head := ui.Row(c).Height(52).Padding(10, 20).Gap(8).
			BorderWidth(0, 0, 1, 0).BorderColor(p.OutlineVariant)
		head.Children(func() {
			if iconButton(c, iconChevronLeft, "返回").Clicked() {
				a.lib.page = pageList
			}
			ui.Text(c, "设置").FontSize(13).Bold()
		})
		ui.Scroll(c).Grow(1).Children(func() {
			page := ui.Column(c).FillWidth().MaxWidth(560).Margin(28, ui.Auto).Gap(14)
			page.Children(func() {
				ui.Text(c, "唤出快捷键").FontSize(12).Bold()
				card := ui.Column(c).Radius(radiusCard).Background(p.SurfaceLow).
					Border(1, p.OutlineVariant).Clip()
				card.Children(func() {
					for i, spec := range hotkeyOptions {
						a.hotkeyRow(c, t, p, spec, i == len(hotkeyOptions)-1)
					}
					if a.lib.err != "" {
						ui.Text(c, a.lib.err).FontSize(11).TextColor(p.Error).Padding(10, 16).Wrap()
					}
				})
			})
		})
	})
}

var hotkeyOptions = []string{
	"Alt+Space",
	"Alt+Q",
	"Ctrl+Alt+Space",
	"Ctrl+Alt+Q",
	"Ctrl+Shift+Space",
	"Ctrl+Shift+Q",
}

func (a *App) hotkeyRow(c *ui.Context, t *ui.Theme, p mdPalette, spec string, last bool) {
	selected := a.saved == spec
	row := ui.Button(c, "").Label(spec).Height(48).Padding(0, 16).Gap(6).
		Radius(0).Background(ui.Transparent).Border(0, ui.Transparent)
	switch {
	case selected:
		row.Background(p.PrimaryContainer).TextColor(p.OnPrimaryContainer)
	case row.Hovered():
		row.Background(p.SurfaceHigh)
	}
	if !last {
		row.BorderWidth(0, 0, 1, 0).BorderColor(p.OutlineVariant)
	}
	row.Children(func() {
		parts := strings.Split(spec, "+")
		for i, part := range parts {
			kbdChip(c, part)
			if i < len(parts)-1 {
				ui.Text(c, "+").FontSize(10.5).TextColor(p.OnSurfaceVariant)
			}
		}
		ui.Spacer(c)
		if selected {
			ui.Icon(c, iconCheck).FontSize(16).TextColor(p.Primary)
		}
	})
	if row.Clicked() {
		if err := a.SetHotkey(spec); err != nil {
			a.lib.err = err.Error()
		} else {
			a.lib.err = ""
		}
	}
}
