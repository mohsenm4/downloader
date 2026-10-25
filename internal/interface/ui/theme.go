package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Extra color names used by the cards and tiles.
const (
	colorSurface       fyne.ThemeColorName = "surface"       // cards
	colorSurfaceBorder fyne.ThemeColorName = "surfaceBorder" // card edges
	colorRaised        fyne.ThemeColorName = "raised"        // rows and tiles inside a card
)

// palette: dark-first, with a violet accent. Every name not listed here
// falls back to Fyne's default theme.
var palettes = map[fyne.ThemeVariant]map[fyne.ThemeColorName]color.Color{
	theme.VariantDark: {
		theme.ColorNameBackground:          hex(0x0d0f14),
		colorSurface:                       hex(0x161921),
		colorSurfaceBorder:                 hex(0x252a35),
		colorRaised:                        hex(0x1d212b),
		theme.ColorNameInputBackground:     hex(0x1d212b),
		theme.ColorNameInputBorder:         hex(0x2f3543),
		theme.ColorNameButton:              hex(0x262b37),
		theme.ColorNameDisabledButton:      hex(0x1b1e26),
		theme.ColorNameHover:               rgba(0xffffff, 0x12),
		theme.ColorNamePressed:             rgba(0xffffff, 0x20),
		theme.ColorNameForeground:          hex(0xe9ebf1),
		theme.ColorNamePlaceHolder:         hex(0x8a90a0),
		theme.ColorNameDisabled:            hex(0x5b6170),
		theme.ColorNameSeparator:           hex(0x232733),
		theme.ColorNamePrimary:             hex(0x8b6dff),
		theme.ColorNameForegroundOnPrimary: hex(0xffffff),
		theme.ColorNameFocus:               rgba(0x8b6dff, 0x70),
		theme.ColorNameSelection:           rgba(0x8b6dff, 0x40),
		theme.ColorNameHyperlink:           hex(0xa48bff),
		theme.ColorNameSuccess:             hex(0x34d399),
		theme.ColorNameWarning:             hex(0xfbbf24),
		theme.ColorNameError:               hex(0xfb7185),
		theme.ColorNameScrollBar:           rgba(0xffffff, 0x30),
		theme.ColorNameShadow:              rgba(0x000000, 0x80),
		theme.ColorNameOverlayBackground:   hex(0x161921),
		theme.ColorNameMenuBackground:      hex(0x161921),
		theme.ColorNameHeaderBackground:    hex(0x161921),
	},
	theme.VariantLight: {
		theme.ColorNameBackground:          hex(0xf3f4f8),
		colorSurface:                       hex(0xffffff),
		colorSurfaceBorder:                 hex(0xe3e5ec),
		colorRaised:                        hex(0xf6f7fb),
		theme.ColorNameInputBackground:     hex(0xf1f2f7),
		theme.ColorNameInputBorder:         hex(0xd9dce5),
		theme.ColorNameButton:              hex(0xe9ebf2),
		theme.ColorNameDisabledButton:      hex(0xeceef3),
		theme.ColorNameHover:               rgba(0x000000, 0x0c),
		theme.ColorNamePressed:             rgba(0x000000, 0x18),
		theme.ColorNameForeground:          hex(0x1a1d26),
		theme.ColorNamePlaceHolder:         hex(0x6b7080),
		theme.ColorNameDisabled:            hex(0xa6abb8),
		theme.ColorNameSeparator:           hex(0xe6e8ee),
		theme.ColorNamePrimary:             hex(0x6a4cf0),
		theme.ColorNameForegroundOnPrimary: hex(0xffffff),
		theme.ColorNameFocus:               rgba(0x6a4cf0, 0x60),
		theme.ColorNameSelection:           rgba(0x6a4cf0, 0x30),
		theme.ColorNameHyperlink:           hex(0x5a3de0),
		theme.ColorNameSuccess:             hex(0x059669),
		theme.ColorNameWarning:             hex(0xd97706),
		theme.ColorNameError:               hex(0xe11d48),
		theme.ColorNameScrollBar:           rgba(0x000000, 0x30),
		theme.ColorNameShadow:              rgba(0x000000, 0x30),
		theme.ColorNameOverlayBackground:   hex(0xffffff),
		theme.ColorNameMenuBackground:      hex(0xffffff),
		theme.ColorNameHeaderBackground:    hex(0xffffff),
	},
}

type customTheme struct{}

// appTheme follows the Mac's light/dark setting.
func appTheme() fyne.Theme { return customTheme{} }

// NewTheme is the theme to set on the app.
func NewTheme() fyne.Theme { return appTheme() }

func (customTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if c, ok := palettes[v][n]; ok {
		return c
	}
	return theme.DefaultTheme().Color(n, v)
}

func (customTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameInputRadius:
		return 10
	case theme.SizeNameSelectionRadius:
		return 8
	case theme.SizeNameInnerPadding:
		return 10
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 22
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNameCaptionText:
		return 12
	}
	return theme.DefaultTheme().Size(n)
}

func (customTheme) Font(s fyne.TextStyle) fyne.Resource { return theme.DefaultTheme().Font(s) }

func (customTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }

// forceVariant pins light or dark, for screenshots.
type forceVariant struct {
	fyne.Theme
	variant fyne.ThemeVariant
}

func (f forceVariant) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(n, f.variant)
}

func hex(c uint32) color.Color { return rgba(c, 0xff) }

func rgba(c uint32, a uint8) color.Color {
	return color.NRGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: a}
}

// themeColor reads a color from the current theme, with alpha (0-255) applied.
func themeColor(n fyne.ThemeColorName, alpha uint8) color.Color {
	r, g, b, a := theme.Color(n).RGBA()
	if a == 0 {
		return color.Transparent
	}
	// un-premultiply, then apply the new alpha
	return color.NRGBA{
		R: uint8(r * 0xffff / a >> 8),
		G: uint8(g * 0xffff / a >> 8),
		B: uint8(b * 0xffff / a >> 8),
		A: uint8(uint32(alpha) * a / 0xffff),
	}
}
