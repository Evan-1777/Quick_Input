package app

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// iconButton is a compact icon-only action. It keeps a stable accessible
// name so views and tests can find it without visible text.
func iconButton(c *ui.Context, glyph *ui.SVG, label string) ui.Element {
	t := c.Theme()
	b := ui.Button(c, "").Label(label).Tooltip(label).
		Size(28, 28).Padding(0).Radius(radiusField).
		Background(ui.Transparent).Border(0, ui.Transparent).TextColor(t.TextMuted)
	b.Children(func() { ui.Icon(c, glyph).FontSize(16) })
	if b.Hovered() {
		b.Background(t.SurfaceHover).TextColor(t.Text)
	}
	return b
}

// filledButton is the primary action: a pill with the accent color.
func filledButton(c *ui.Context, glyph *ui.SVG, label string) ui.Element {
	b := ui.PrimaryButton(c, "").Radius(radiusPill).Padding(8, 16).Height(34)
	b.Children(func() {
		if glyph != nil {
			ui.Icon(c, glyph).FontSize(16)
		}
		ui.Text(c, label).SingleLine()
	})
	return b
}

// textButton is a low-emphasis pill with accent text.
func textButton(c *ui.Context, label string) ui.Element {
	t := c.Theme()
	return ui.Button(c, label).Radius(radiusPill).Padding(8, 12).
		Background(ui.Transparent).Border(0, ui.Transparent).TextColor(t.Accent)
}

// tonalButton is a secondary action on a primary container.
func tonalButton(c *ui.Context, label string) ui.Element {
	t := c.Theme()
	p := mdColors(t)
	return ui.Button(c, label).Radius(radiusPill).Padding(8, 14).
		Background(p.PrimaryContainer).Border(0, ui.Transparent).
		TextColor(p.OnPrimaryContainer)
}

// dangerButton is a destructive action on an error container.
func dangerButton(c *ui.Context, label string) ui.Element {
	t := c.Theme()
	p := mdColors(t)
	return ui.Button(c, label).Radius(radiusPill).Padding(8, 14).
		Background(p.ErrorContainer).Border(0, ui.Transparent).
		TextColor(p.OnErrorContainer)
}

// chip is a small read-only label for a snippet's tags or group.
func chip(c *ui.Context, label string) ui.Element {
	t := c.Theme()
	p := mdColors(t)
	box := ui.Row(c).Padding(2, 7).Radius(radiusChip).
		Border(1, p.OutlineVariant).Background(ui.Transparent)
	box.Children(func() {
		ui.Text(c, label).FontSize(10.5).TextColor(p.OnSurfaceVariant).SingleLine()
	})
	return box
}

// kbdChip shows one key of a shortcut.
func kbdChip(c *ui.Context, label string) ui.Element {
	t := c.Theme()
	p := mdColors(t)
	box := ui.Row(c).Padding(2, 6).Radius(4).
		Border(1, p.OutlineVariant).Background(t.Surface)
	box.Children(func() {
		ui.Text(c, label).FontSize(10.5).TextColor(p.OnSurfaceVariant).SingleLine()
	})
	return box
}

// sectionTitle labels a sidebar section.
func sectionTitle(c *ui.Context, label string) ui.Element {
	t := c.Theme()
	p := mdColors(t)
	return ui.Text(c, label).FontSize(10.5).Bold().TextColor(p.OnSurfaceVariant).Padding(4, 6)
}

// preview flattens a snippet body to one line, bounded for stable row heights.
func preview(text string, maxRunes int) string {
	flat := strings.Join(strings.Fields(text), " ")
	runes := []rune(flat)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "…"
	}
	return flat
}
