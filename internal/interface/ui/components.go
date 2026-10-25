package ui

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// colorFn reads a color when the widget refreshes, so light/dark changes apply.
type colorFn func() color.Color

func named(n fyne.ThemeColorName) colorFn {
	return func() color.Color { return theme.Color(n) }
}

func tint(n fyne.ThemeColorName, alpha uint8) colorFn {
	return func() color.Color { return themeColor(n, alpha) }
}

// panel is a rounded background.
type panel struct {
	widget.BaseWidget
	fill, stroke colorFn
	strokeWidth  float32
	radius       float32
	rect         *canvas.Rectangle
}

func newPanel(fill, stroke colorFn, radius float32) *panel {
	p := &panel{fill: fill, stroke: stroke, strokeWidth: 1, radius: radius, rect: canvas.NewRectangle(color.Transparent)}
	p.ExtendBaseWidget(p)
	p.applyColors()
	return p
}

func (p *panel) applyColors() {
	p.rect.FillColor = color.Transparent
	if p.fill != nil {
		p.rect.FillColor = p.fill()
	}
	p.rect.StrokeColor = color.Transparent
	p.rect.StrokeWidth = 0
	if p.stroke != nil {
		p.rect.StrokeColor = p.stroke()
		p.rect.StrokeWidth = p.strokeWidth
	}
	p.rect.CornerRadius = p.radius
}

func (p *panel) Refresh() {
	p.applyColors()
	p.BaseWidget.Refresh()
}

func (p *panel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.rect)
}

// card is a raised box that groups one part of the screen.
func card(content fyne.CanvasObject) fyne.CanvasObject {
	return container.NewStack(
		newPanel(named(colorSurface), named(colorSurfaceBorder), 16),
		container.New(layout.NewCustomPaddedLayout(14, 16, 18, 18), content),
	)
}

// tight removes most of a label's top and bottom space.
func tight(o fyne.CanvasObject) fyne.CanvasObject {
	return container.New(layout.NewCustomPaddedLayout(-5, -5, 0, 0), o)
}

func label(text string, size fyne.ThemeSizeName, bold bool, imp widget.Importance) *widget.Label {
	l := widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: bold})
	l.SizeName = size
	l.Importance = imp
	return l
}

// stepTitle is a numbered heading: (1) What do you want?
func stepTitle(step, text string) fyne.CanvasObject {
	num := widget.NewLabelWithStyle(step, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	num.Importance = widget.HighImportance
	num.SizeName = theme.SizeNameCaptionText
	badge := container.NewGridWrap(fyne.NewSize(24, 24),
		container.NewStack(newPanel(tint(theme.ColorNamePrimary, 0x30), nil, 12), container.NewCenter(num)))

	return container.NewHBox(container.NewCenter(badge), label(text, theme.SizeNameSubHeadingText, true, widget.MediumImportance))
}

// iconBadge is an icon on a soft rounded square.
type iconBadge struct {
	widget.BaseWidget
	bg      *panel
	icon    *widget.Icon
	content fyne.CanvasObject
}

func newIconBadge(size, iconSize float32) *iconBadge {
	b := &iconBadge{bg: newPanel(nil, nil, size/3.2), icon: widget.NewIcon(nil)}
	b.content = container.NewGridWrap(fyne.NewSize(size, size),
		container.NewStack(b.bg, container.NewCenter(container.NewGridWrap(fyne.NewSize(iconSize, iconSize), b.icon))))
	b.ExtendBaseWidget(b)
	return b
}

func (b *iconBadge) set(icon fyne.Resource, c fyne.ThemeColorName) {
	b.bg.fill = tint(c, 0x2c)
	b.bg.Refresh()
	b.icon.SetResource(icon)
}

func (b *iconBadge) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(b.content)
}

// chip is a small colored status pill.
type chip struct {
	widget.BaseWidget
	bg      *panel
	label   *widget.Label
	content fyne.CanvasObject
}

func newChip() *chip {
	c := &chip{bg: newPanel(nil, nil, 11), label: label("", theme.SizeNameCaptionText, true, widget.MediumImportance)}
	c.content = container.NewStack(c.bg, container.New(layout.NewCustomPaddedLayout(-7, -7, 2, 2), c.label))
	c.ExtendBaseWidget(c)
	return c
}

func (c *chip) set(text string, col fyne.ThemeColorName, imp widget.Importance) {
	c.label.Text = text
	c.label.Importance = imp
	c.label.Refresh()
	c.bg.fill = tint(col, 0x2c)
	c.bg.Refresh()
}

func (c *chip) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.content)
}

// thinBar is a 4px progress line. Negative values animate (progress unknown).
// When idle it draws nothing but keeps its space, so rows never jump.
type thinBar struct {
	widget.BaseWidget
	value  float64
	idle   bool
	offset float32
	anim   *fyne.Animation
}

func newThinBar() *thinBar {
	b := &thinBar{idle: true}
	b.ExtendBaseWidget(b)
	return b
}

// SetValue takes 0-1, or a negative number when the amount is not known.
func (b *thinBar) SetValue(v float64) {
	b.idle = false
	b.value = v
	if v < 0 {
		b.startAnim()
	} else {
		b.stopAnim()
	}
	b.Refresh()
}

func (b *thinBar) SetIdle() {
	b.idle = true
	b.stopAnim()
	b.Refresh()
}

func (b *thinBar) startAnim() {
	if b.anim != nil {
		return
	}
	b.anim = fyne.NewAnimation(1400*time.Millisecond, func(f float32) {
		b.offset = f
		b.Refresh()
	})
	b.anim.RepeatCount = fyne.AnimationRepeatForever
	b.anim.Curve = fyne.AnimationEaseInOut
	b.anim.Start()
}

func (b *thinBar) stopAnim() {
	if b.anim != nil {
		b.anim.Stop()
		b.anim = nil
	}
}

func (b *thinBar) CreateRenderer() fyne.WidgetRenderer {
	r := &thinBarRenderer{bar: b, track: canvas.NewRectangle(color.Transparent), fill: canvas.NewRectangle(color.Transparent)}
	r.Refresh()
	return r
}

type thinBarRenderer struct {
	bar         *thinBar
	track, fill *canvas.Rectangle
}

func (r *thinBarRenderer) Layout(size fyne.Size) {
	// a squeezed row can give a negative size, which the painter cannot draw
	size = fyne.NewSize(max(size.Width, 0), max(size.Height, 0))

	r.track.Move(fyne.NewPos(0, 0))
	r.track.Resize(size)

	w := size.Width
	if r.bar.value >= 0 {
		r.fill.Move(fyne.NewPos(0, 0))
		r.fill.Resize(fyne.NewSize(w*float32(r.bar.value), size.Height))
		return
	}

	// unknown: a segment slides from left to right
	seg := w * 0.3
	x := (w+seg)*r.bar.offset - seg
	left, right := max(x, 0), min(x+seg, w)
	r.fill.Move(fyne.NewPos(left, 0))
	r.fill.Resize(fyne.NewSize(max(right-left, 0), size.Height))
}

func (r *thinBarRenderer) MinSize() fyne.Size { return fyne.NewSize(20, 4) }

func (r *thinBarRenderer) Refresh() {
	if r.bar.idle {
		r.track.FillColor = color.Transparent
		r.fill.FillColor = color.Transparent
	} else {
		r.track.FillColor = themeColor(theme.ColorNameForeground, 0x1e)
		r.fill.FillColor = theme.Color(theme.ColorNamePrimary)
	}
	r.track.CornerRadius = 2
	r.fill.CornerRadius = 2
	r.Layout(r.bar.Size())
	r.track.Refresh()
	r.fill.Refresh()
}

func (r *thinBarRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.track, r.fill} }

func (r *thinBarRenderer) Destroy() { r.bar.stopAnim() }

// modeTile is a big choice button: an icon, a name and a short line.
type modeTile struct {
	widget.BaseWidget
	selected bool
	hovered  bool
	onTap    func()

	iconRes fyne.Resource
	bg      *panel
	icon    *widget.Icon
	title   *widget.Label
	caption *widget.Label
	content fyne.CanvasObject
}

func newModeTile(icon fyne.Resource, title, caption string, onTap func()) *modeTile {

	t := &modeTile{iconRes: icon, onTap: onTap}
	t.bg = newPanel(nil, nil, 14)
	t.icon = widget.NewIcon(icon)
	t.title = widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	t.caption = widget.NewLabelWithStyle(caption, fyne.TextAlignCenter, fyne.TextStyle{})
	t.caption.SizeName = theme.SizeNameCaptionText
	t.caption.Importance = widget.LowImportance

	iconBox := container.NewCenter(container.NewGridWrap(fyne.NewSize(30, 30), t.icon))
	body := container.NewVBox(iconBox, tight(t.title), tight(t.caption))
	t.content = container.NewStack(t.bg, container.New(layout.NewCustomPaddedLayout(12, 10, 6, 6), body))

	t.ExtendBaseWidget(t)
	t.applyStyle()
	return t
}

func (t *modeTile) applyStyle() {
	switch {
	case t.selected:
		t.bg.fill = tint(theme.ColorNamePrimary, 0x26)
		t.bg.stroke = named(theme.ColorNamePrimary)
		t.bg.strokeWidth = 2
		t.icon.Resource = theme.NewPrimaryThemedResource(t.iconRes)
		t.title.Importance = widget.HighImportance
	case t.hovered:
		t.bg.fill = tint(theme.ColorNamePrimary, 0x10)
		t.bg.stroke = tint(theme.ColorNamePrimary, 0x80)
		t.bg.strokeWidth = 1
		t.icon.Resource = t.iconRes
		t.title.Importance = widget.MediumImportance
	default:
		t.bg.fill = named(colorRaised)
		t.bg.stroke = named(colorSurfaceBorder)
		t.bg.strokeWidth = 1
		t.icon.Resource = t.iconRes
		t.title.Importance = widget.MediumImportance
	}
	t.bg.Refresh()
	t.icon.Refresh()
	t.title.Refresh()
}

func (t *modeTile) setSelected(on bool) {
	t.selected = on
	t.Refresh()
}

func (t *modeTile) Refresh() {
	t.applyStyle()
	t.BaseWidget.Refresh()
}

func (t *modeTile) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.content)
}

func (t *modeTile) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *modeTile) MouseIn(*desktop.MouseEvent) {
	t.hovered = true
	t.Refresh()
}

func (t *modeTile) MouseMoved(*desktop.MouseEvent) {}

func (t *modeTile) MouseOut() {
	t.hovered = false
	t.Refresh()
}

func (t *modeTile) Cursor() desktop.Cursor { return desktop.PointerCursor }

// segmented is a row of buttons where one is selected.
type segmented struct {
	widget.BaseWidget
	OnChanged func(string)

	options  []string
	selected string
	buttons  []*widget.Button
	content  fyne.CanvasObject
}

func newSegmented(options []string) *segmented {

	s := &segmented{options: options}
	objs := make([]fyne.CanvasObject, len(options))
	for i, o := range options {
		b := widget.NewButton(o, func() {
			s.SetSelected(o)
			if s.OnChanged != nil {
				s.OnChanged(o)
			}
		})
		s.buttons = append(s.buttons, b)
		objs[i] = b
	}
	s.content = container.NewGridWithColumns(len(options), objs...)

	s.ExtendBaseWidget(s)
	return s
}

func (s *segmented) SetSelected(o string) {
	s.selected = o
	for i, b := range s.buttons {
		if s.options[i] == o {
			b.Importance = widget.HighImportance
		} else {
			b.Importance = widget.MediumImportance
		}
		b.Refresh()
	}
}

func (s *segmented) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.content)
}
