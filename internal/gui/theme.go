// Package gui provides a compact theme variant to reduce toolbar button height.
package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// compactTheme wraps the default theme but reduces inner padding for smaller buttons.
type compactTheme struct {
	fyne.Theme
}

// NewCompactTheme returns a theme with reduced padding for a denser layout.
func NewCompactTheme() fyne.Theme {
	return &compactTheme{Theme: theme.DefaultTheme()}
}

func (t *compactTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return t.Theme.Color(name, variant)
}

func (t *compactTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.Theme.Icon(name)
}

func (t *compactTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.Theme.Font(style)
}

func (t *compactTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameInnerPadding:
		return 2 // default is 4; halved for compact buttons
	case theme.SizeNamePadding:
		return 3 // default is 4
	}
	return t.Theme.Size(name)
}
