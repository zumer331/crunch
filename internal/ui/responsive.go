// Package ui — адаптивная вёрстка.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	fullWidth  = 1000
	fullHeight = 720
	splitRatio = 0.40
)

func isCompact(size fyne.Size) bool {
	return size.Width < fullWidth || size.Height < fullHeight
}

type responsiveBody struct {
	widget.BaseWidget

	split   *container.Split
	logObj  fyne.CanvasObject
	compact bool
}

func newResponsiveBody(steps, log fyne.CanvasObject) *responsiveBody {
	r := &responsiveBody{
		split:  container.NewHSplit(steps, log),
		logObj: log,
	}
	r.split.SetOffset(splitRatio)
	r.ExtendBaseWidget(r)
	return r
}

func (r *responsiveBody) Resize(size fyne.Size) {
	want := isCompact(size)
	if want != r.compact {
		r.compact = want
		if want {
			r.split.SetOffset(1.0)
			r.logObj.Hide()
		} else {
			r.logObj.Show()
			r.split.SetOffset(splitRatio)
		}
	}
	r.BaseWidget.Resize(size)
}

func (r *responsiveBody) MinSize() fyne.Size {
	return r.split.MinSize()
}

func (r *responsiveBody) CreateRenderer() fyne.WidgetRenderer {
	return &responsiveRenderer{body: r}
}

type responsiveRenderer struct {
	body *responsiveBody
}

func (r *responsiveRenderer) Layout(size fyne.Size) {
	r.body.split.Resize(size)
	r.body.split.Move(fyne.NewPos(0, 0))
}

func (r *responsiveRenderer) MinSize() fyne.Size {
	return r.body.split.MinSize()
}

func (r *responsiveRenderer) Refresh() {
	r.body.split.Refresh()
}

func (r *responsiveRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.body.split}
}

func (r *responsiveRenderer) Destroy() {}