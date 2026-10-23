// Package ui — тема и компоненты в стиле libadwaita.
package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AdwaitaTheme struct{}

var (
	colorBg        = color.NRGBA{0xf6, 0xf5, 0xf4, 0xff}
	colorCard      = color.NRGBA{0xff, 0xff, 0xff, 0xff}
	colorBorder    = color.NRGBA{0xdc, 0xdc, 0xdc, 0xff}
	colorText      = color.NRGBA{0x2e, 0x34, 0x36, 0xff}
	colorTextDim   = color.NRGBA{0x6a, 0x6a, 0x6a, 0xff}
	colorAccent    = color.NRGBA{0x35, 0x84, 0xe4, 0xff}
	colorAccentHov = color.NRGBA{0x2b, 0x70, 0xc4, 0xff}
	colorBtn       = color.NRGBA{0xe8, 0xe8, 0xe8, 0xff}
	colorBtnHov    = color.NRGBA{0xdc, 0xdc, 0xdc, 0xff}
	colorSuccess   = color.NRGBA{0x2e, 0xc2, 0x5e, 0xff}
	colorDanger    = color.NRGBA{0xe0, 0x1b, 0x24, 0xff}
	colorWarn      = color.NRGBA{0xe5, 0xa5, 0x00, 0xff}
	colorInfoBg    = color.NRGBA{0xe8, 0xf1, 0xfc, 0xff}
)

func (AdwaitaTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colorBg
	case theme.ColorNameForeground:
		return colorText
	case theme.ColorNameButton:
		return colorBtn
	case theme.ColorNameDisabledButton:
		return color.NRGBA{0xf0, 0xf0, 0xf0, 0xff}
	case theme.ColorNamePrimary:
		return colorAccent
	case theme.ColorNameHover:
		return color.NRGBA{0x00, 0x00, 0x00, 0x08}
	case theme.ColorNameDisabled:
		return colorTextDim
	case theme.ColorNameSeparator:
		return colorBorder
	case theme.ColorNameSuccess:
		return colorSuccess
	case theme.ColorNameError:
		return colorDanger
	case theme.ColorNameWarning:
		return colorWarn
	case theme.ColorNameInputBackground:
		return colorCard
	case theme.ColorNameForegroundOnPrimary:
		return color.White
	}
	return theme.DefaultTheme().Color(name, theme.VariantLight)
}

func (AdwaitaTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (AdwaitaTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (AdwaitaTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 10
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 20
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNameInlineIcon:
		return 20
	case theme.SizeNameScrollBar:
		return 10
	case theme.SizeNameScrollBarSmall:
		return 4
	case theme.SizeNameInputRadius:
		return 8
	case theme.SizeNameSelectionRadius:
		return 6
	}
	return theme.DefaultTheme().Size(name)
}

func NewCard(content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(colorCard)
	bg.CornerRadius = 10
	bg.StrokeColor = colorBorder
	bg.StrokeWidth = 1
	return container.NewStack(bg, container.NewPadded(content))
}

func NewInfoBanner(mainText, warningText string) fyne.CanvasObject {
	bg := canvas.NewRectangle(colorInfoBg)
	bg.CornerRadius = 12

	icon := widget.NewIcon(theme.InfoIcon())

	mainLabel := widget.NewLabel(mainText)
	mainLabel.Wrapping = fyne.TextWrapWord
	mainLabel.TextStyle = fyne.TextStyle{Bold: true}

	warningLabel := widget.NewLabel(warningText)
	warningLabel.Wrapping = fyne.TextWrapWord
	warningLabel.Importance = widget.MediumImportance

	texts := container.NewVBox(mainLabel, warningLabel)
	row := container.NewBorder(nil, nil, icon, nil, texts)
	return container.NewStack(bg, container.NewPadded(row))
}

type PillButton struct {
	widget.BaseWidget

	Label    string
	Accent   bool
	OnTapped func()

	disabled bool
	hovered  bool

	bg   *canvas.Rectangle
	text *canvas.Text
}

func NewPillButton(label string, accent bool, onTap func()) *PillButton {
	b := &PillButton{Label: label, Accent: accent, OnTapped: onTap}
	b.ExtendBaseWidget(b)
	return b
}

func (b *PillButton) Disable() {
	b.disabled = true
	b.Refresh()
}

func (b *PillButton) Enable() {
	b.disabled = false
	b.Refresh()
}

func (b *PillButton) CreateRenderer() fyne.WidgetRenderer {
	b.bg = canvas.NewRectangle(b.bgColor())
	b.bg.CornerRadius = 20

	b.text = canvas.NewText(b.Label, b.fgColor())
	b.text.Alignment = fyne.TextAlignCenter
	b.text.TextStyle = fyne.TextStyle{Bold: true}
	b.text.TextSize = 14

	return &pillRenderer{btn: b, objs: []fyne.CanvasObject{b.bg, b.text}}
}

func (b *PillButton) Tapped(_ *fyne.PointEvent) {
	if b.disabled || b.OnTapped == nil {
		return
	}
	b.OnTapped()
}

func (b *PillButton) TappedSecondary(_ *fyne.PointEvent) {}

func (b *PillButton) MouseIn(_ *desktop.MouseEvent) {
	if b.disabled {
		return
	}
	b.hovered = true
	b.Refresh()
}

func (b *PillButton) MouseMoved(_ *desktop.MouseEvent) {}

func (b *PillButton) MouseOut() {
	b.hovered = false
	b.Refresh()
}

func (b *PillButton) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (b *PillButton) bgColor() color.Color {
	if b.disabled {
		if b.Accent {
			return color.NRGBA{0xc7, 0xd9, 0xf3, 0xff}
		}
		return color.NRGBA{0xf2, 0xf2, 0xf2, 0xff}
	}
	if b.Accent {
		if b.hovered {
			return colorAccentHov
		}
		return colorAccent
	}
	if b.hovered {
		return colorBtnHov
	}
	return colorBtn
}

func (b *PillButton) fgColor() color.Color {
	if b.disabled {
		return colorTextDim
	}
	if b.Accent {
		return color.White
	}
	return colorText
}

type pillRenderer struct {
	btn  *PillButton
	objs []fyne.CanvasObject
}

func (r *pillRenderer) Layout(size fyne.Size) {
	r.btn.bg.Resize(size)
	tSize := r.btn.text.MinSize()
	r.btn.text.Resize(tSize)
	r.btn.text.Move(fyne.NewPos(
		(size.Width-tSize.Width)/2,
		(size.Height-tSize.Height)/2,
	))
}

func (r *pillRenderer) MinSize() fyne.Size {
	t := r.btn.text.MinSize()
	return fyne.NewSize(t.Width+56, t.Height+22)
}

func (r *pillRenderer) Refresh() {
	r.btn.bg.FillColor = r.btn.bgColor()
	r.btn.bg.CornerRadius = 20
	r.btn.text.Text = r.btn.Label
	r.btn.text.Color = r.btn.fgColor()
	canvas.Refresh(r.btn.bg)
	canvas.Refresh(r.btn.text)
}

func (r *pillRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *pillRenderer) Destroy()                     {}

func vSpacer(h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(0, h))
	return r
}