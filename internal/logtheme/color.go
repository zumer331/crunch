package logtheme

import "image/color"

// tagColors — цвета текста на светлом фоне (палитра GitHub Light).
var tagColors = map[Tag]color.Color{
	Default: color.NRGBA{0x2c, 0x2c, 0x2c, 0xff},
	Info:    color.NRGBA{0x1a, 0x7f, 0x37, 0xff},
	Warn:    color.NRGBA{0x9a, 0x67, 0x00, 0xff},
	Error:   color.NRGBA{0xcf, 0x22, 0x2e, 0xff},
	Step:    color.NRGBA{0x09, 0x69, 0xda, 0xff},
	OK:      color.NRGBA{0x1a, 0x7f, 0x37, 0xff},
}

// ColorFor возвращает цвет текста для тега.
func ColorFor(t Tag) color.Color {
	if c, ok := tagColors[t]; ok {
		return c
	}
	return tagColors[Default]
}