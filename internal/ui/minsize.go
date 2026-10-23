package ui

import (
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
)

const (
	minWidth  float32 = 600
	minHeight float32 = 400

	// minSizeCheckInterval — как часто проверяем размер окна.
	// 200 мс = 5 просыпаний/сек. На слабом железе снижает нагрузку
	// на CPU в 4 раза по сравнению с 50 мс.
	minSizeCheckInterval = 200 * time.Millisecond
)

// enforceMinSize удерживает окно не меньше 600×400.
func enforceMinSize(w fyne.Window, closing *atomic.Bool) {
	go func() {
		ticker := time.NewTicker(minSizeCheckInterval)
		defer ticker.Stop()
		for range ticker.C {
			if closing.Load() {
				return
			}

			canvas := w.Canvas()
			if canvas == nil {
				continue
			}

			size := canvas.Size()
			if size.Width >= minWidth && size.Height >= minHeight {
				continue
			}

			newW := maxF32(size.Width, minWidth)
			newH := maxF32(size.Height, minHeight)

			fyne.Do(func() {
				defer func() { _ = recover() }()
				w.Resize(fyne.NewSize(newW, newH))
			})
		}
	}()
}

func maxF32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}