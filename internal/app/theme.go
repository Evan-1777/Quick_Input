package app

import "github.com/egoist/mygo/ui"

// MD3 shape tokens. Containers keep the sheet/card radii, fields a tighter
// radius, and buttons are pills.
const (
	radiusSheet = float32(16)
	radiusCard  = float32(12)
	radiusField = float32(8)
	radiusChip  = float32(6)
	radiusPill  = float32(999)
)

// mdPalette is the compact Material 3 tonal palette of one appearance.
type mdPalette struct {
	Surface            ui.Color
	SurfaceLow         ui.Color
	SurfaceContainer   ui.Color
	SurfaceHigh        ui.Color
	SurfaceHighest     ui.Color
	OnSurface          ui.Color
	OnSurfaceVariant   ui.Color
	Outline            ui.Color
	OutlineVariant     ui.Color
	Primary            ui.Color
	OnPrimary          ui.Color
	PrimaryContainer   ui.Color
	OnPrimaryContainer ui.Color
	Error              ui.Color
	ErrorContainer     ui.Color
	OnErrorContainer   ui.Color
	InverseSurface     ui.Color
	InverseOnSurface   ui.Color
}

// mdScrim dims the window behind a side sheet or dialog.
var mdScrim = ui.RGBA(0, 0, 0, 0.32)

var lightPalette = mdPalette{
	Surface:            ui.Hex("#F9F9FF"),
	SurfaceLow:         ui.Hex("#F3F3FA"),
	SurfaceContainer:   ui.Hex("#EDEDF4"),
	SurfaceHigh:        ui.Hex("#E7E8EE"),
	SurfaceHighest:     ui.Hex("#E1E2E8"),
	OnSurface:          ui.Hex("#1A1C1E"),
	OnSurfaceVariant:   ui.Hex("#44474E"),
	Outline:            ui.Hex("#74777F"),
	OutlineVariant:     ui.Hex("#C4C6D0"),
	Primary:            ui.Hex("#0B57D0"),
	OnPrimary:          ui.Hex("#FFFFFF"),
	PrimaryContainer:   ui.Hex("#D3E3FD"),
	OnPrimaryContainer: ui.Hex("#041E49"),
	Error:              ui.Hex("#BA1A1A"),
	ErrorContainer:     ui.Hex("#FFDAD6"),
	OnErrorContainer:   ui.Hex("#410002"),
	InverseSurface:     ui.Hex("#2F3033"),
	InverseOnSurface:   ui.Hex("#F1F0F4"),
}

var darkPalette = mdPalette{
	Surface:            ui.Hex("#111318"),
	SurfaceLow:         ui.Hex("#1A1C20"),
	SurfaceContainer:   ui.Hex("#1F2125"),
	SurfaceHigh:        ui.Hex("#292C31"),
	SurfaceHighest:     ui.Hex("#33353A"),
	OnSurface:          ui.Hex("#E2E2E9"),
	OnSurfaceVariant:   ui.Hex("#C4C6D0"),
	Outline:            ui.Hex("#8E9099"),
	OutlineVariant:     ui.Hex("#44474E"),
	Primary:            ui.Hex("#A8C7FA"),
	OnPrimary:          ui.Hex("#062E6F"),
	PrimaryContainer:   ui.Hex("#0842A0"),
	OnPrimaryContainer: ui.Hex("#D3E3FD"),
	Error:              ui.Hex("#FFB4AB"),
	ErrorContainer:     ui.Hex("#93000A"),
	OnErrorContainer:   ui.Hex("#FFDAD6"),
	InverseSurface:     ui.Hex("#E2E2E9"),
	InverseOnSurface:   ui.Hex("#2F3033"),
}

// mdTheme returns a compact Material 3 copy of the platform's theme.
func mdTheme(dark bool) *ui.Theme {
	t := ui.LightTheme()
	if dark {
		t = ui.DarkTheme()
	}
	p := mdColors(t)

	t.Background = p.Surface
	t.Surface = p.SurfaceContainer
	t.SurfaceHover = p.SurfaceHigh
	t.SurfacePressed = p.SurfaceHighest
	t.Border = p.OutlineVariant
	t.Text = p.OnSurface
	t.TextMuted = p.OnSurfaceVariant
	t.Accent = p.Primary
	t.AccentText = p.OnPrimary
	t.AccentHover = p.Primary.Mix(p.OnPrimary, 0.08)
	t.AccentPressed = p.Primary.Mix(p.OnPrimary, 0.14)
	t.Danger = p.Error
	t.Warning = p.Error
	t.Success = p.Primary
	t.Selection = p.PrimaryContainer
	t.Focus = p.Primary
	t.Inverse = p.InverseSurface
	t.InverseText = p.InverseOnSurface
	t.Scrollbar = p.OutlineVariant
	t.ScrollbarWidth = 3
	t.Radius = radiusCard
	t.Spacing = 4
	t.FontSize = 13
	return t
}

// useMD3 rebuilds and installs the theme for the current frame. The context
// starts every frame from the platform theme, so this must run in every view
// root, not once at startup.
func useMD3(c *ui.Context) *ui.Theme {
	t := mdTheme(c.Theme().Dark)
	c.SetTheme(t)
	return t
}

func mdColors(t *ui.Theme) mdPalette {
	if t != nil && t.Dark {
		return darkPalette
	}
	return lightPalette
}
