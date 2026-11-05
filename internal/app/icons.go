package app

import "github.com/egoist/mygo/ui"

const iconSVGHead = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">`
const iconSVGFoot = `</svg>`

func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(iconSVGHead + body + iconSVGFoot))
}

var (
	iconSearch      = icon(`<circle cx="11" cy="11" r="7"/><path d="m20 20-3.6-3.6"/>`)
	iconClose       = icon(`<path d="M6 6l12 12M18 6 6 18"/>`)
	iconPlus        = icon(`<path d="M12 5v14M5 12h14"/>`)
	iconPencil      = icon(`<path d="M4 20h4L20 8l-4-4L4 16v4Z"/><path d="m14 6 4 4"/>`)
	iconTrash       = icon(`<path d="M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13M10 11v5M14 11v5"/>`)
	iconSettings    = icon(`<path d="M4 7h10M18 7h2M4 17h2M10 17h10"/><circle cx="16" cy="7" r="2"/><circle cx="8" cy="17" r="2"/>`)
	iconChevronLeft = icon(`<path d="m15 18-6-6 6-6"/>`)
	iconChevronDown = icon(`<path d="m6 9 6 6 6-6"/>`)
	iconCheck       = icon(`<path d="m20 6-11 11-5-5"/>`)
	iconPanel       = icon(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 9h18"/>`)
	iconFolder      = icon(`<path d="M3 6h6l2 2h10v10H3V6Z"/>`)
	iconTag         = icon(`<path d="M4 4h7l9 9-7 7-9-9V4Z"/><circle cx="8" cy="8" r="1.2"/>`)
	iconInbox       = icon(`<path d="M4 5h16v14H4V5Z"/><path d="M4 13h4l2 3h4l2-3h4"/>`)
	iconAlert       = icon(`<path d="M12 3 2 21h20L12 3Z"/><path d="M12 9v5M12 18h.01"/>`)
)

// appMark is the product mark: a primary rounded tile with two text bars.
func appMark(c *ui.Context, size float32) ui.Element {
	t := c.Theme()
	mark := ui.Box(c).Size(size, size).Radius(size * 0.28).Background(t.Accent).Clip()
	mark.Children(func() {
		bar := size * 0.12
		ui.Box(c).Size(size*0.46, bar).Radius(bar / 2).Background(t.AccentText).
			Absolute().Top(size * 0.32).Left(size * 0.27)
		ui.Box(c).Size(size*0.32, bar).Radius(bar / 2).Background(t.AccentText).
			Absolute().Top(size * 0.56).Left(size * 0.27)
	})
	return mark
}
