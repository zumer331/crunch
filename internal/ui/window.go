// Package ui реализует GUI на Fyne в стиле libadwaita.
package ui

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"crunch/internal/logtheme"
	"crunch/internal/runner"
)

// logFlushInterval — период сброса буфера лога в TextGrid.
// 200 мс = 5 обновлений/сек. На слабом железе снижает нагрузку на рендер
// в 2 раза; пользователь видит лог с задержкой 0.2 с — незаметно.
const logFlushInterval = 200 * time.Millisecond

type logEntry struct {
	line string
	tag  logtheme.Tag
}

type stepState int

const (
	stepPending stepState = iota
	stepRunning
	stepDone
	stepFailed
)

type window struct {
	app fyne.App
	win fyne.Window

	btnInstall *PillButton
	progress   *widget.ProgressBar
	status     *widget.Label

	stepLabels  []*widget.Label
	currentStep int

	logGrid   *widget.TextGrid
	logScroll *container.Scroll

	worker   *runner.Runner
	running  bool
	steps    []string

	startTime time.Time
	closing   atomic.Bool

	logMu  sync.Mutex
	logBuf []logEntry
	logRow int

	logStyles map[logtheme.Tag]*widget.CustomTextGridStyle
}

// Run создаёт окно и запускает event loop Fyne.
func Run(steps []string) {
	a := app.NewWithID("crunch")
	a.Settings().SetTheme(&AdwaitaTheme{})

	w := a.NewWindow("Crunch")
	w.Resize(fyne.NewSize(1000, 720))
	w.CenterOnScreen()

	win := &window{
		app:         a,
		win:         w,
		steps:       steps,
		currentStep: -1,
	}
	win.build()
	win.installCloseHook()
	w.Show()

	go win.flushLoop()
	a.Run()
}

// installCloseHook перехватывает закрытие главного окна.
func (w *window) installCloseHook() {
	w.win.SetCloseIntercept(func() {
		w.closing.Store(true)
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "crunch: паника при закрытии: %v\n", r)
			}
			os.Exit(0)
		}()
		w.shutdown()
		w.win.Close()
		w.app.Quit()
	})
}

func (w *window) shutdown() {
	if w.worker != nil {
		w.worker.Cancel()
		if !w.worker.Wait(3 * time.Second) {
			w.worker.ForceKill()
			w.worker.Wait(2 * time.Second)
		}
	}
}

// ── Построение UI ─────────────────────────────────────────────────────

func (w *window) build() {
	banner := NewInfoBanner(
		"Создаём рабочее окружение для твоего творчества",
		"Скачается около 1,5 ГБ данных",
	)

	w.status = widget.NewLabel("Готов к запуску")
	w.status.Importance = widget.MediumImportance

	w.progress = widget.NewProgressBar()
	w.progress.Min = 0
	w.progress.Max = 100

	progressBlock := container.NewVBox(w.status, w.progress)

	stepsPanel := w.buildStepsPanel()
	logPanel := w.buildLogPanel()

	responsive := newResponsiveBody(stepsPanel, logPanel)

	w.btnInstall = NewPillButton("Начать установку", true, w.start)

	bottomBar := container.NewHBox(
		layout.NewSpacer(),
		w.btnInstall,
	)

	top := container.NewVBox(
		container.NewPadded(banner),
		vSpacer(4),
		container.NewPadded(progressBlock),
		vSpacer(4),
	)

	body := container.NewPadded(responsive)
	bottom := container.NewPadded(bottomBar)

	w.win.SetContent(container.NewBorder(top, bottom, nil, nil, body))
	enforceMinSize(w.win, &w.closing)
}

func (w *window) buildStepsPanel() fyne.CanvasObject {
	content := container.NewVBox()
	w.stepLabels = make([]*widget.Label, len(w.steps))
	for i, name := range w.steps {
		lbl := widget.NewLabel(fmt.Sprintf("○  %s", name))
		lbl.Importance = widget.LowImportance
		w.stepLabels[i] = lbl
		content.Add(lbl)
	}

	header := widget.NewLabelWithStyle("Шаги",
		fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	inner := container.NewBorder(
		header, nil, nil, nil,
		container.NewVScroll(content),
	)
	return NewCard(inner)
}

func (w *window) buildLogPanel() fyne.CanvasObject {
	w.logStyles = make(map[logtheme.Tag]*widget.CustomTextGridStyle)
	for _, tag := range []logtheme.Tag{
		logtheme.Default, logtheme.Info, logtheme.Warn,
		logtheme.Error, logtheme.Step, logtheme.OK,
	} {
		w.logStyles[tag] = &widget.CustomTextGridStyle{
			FGColor: logtheme.ColorFor(tag),
			BGColor: color.White,
		}
	}

	w.logGrid = widget.NewTextGrid()
	w.logScroll = container.NewVScroll(w.logGrid)

	header := widget.NewLabelWithStyle("Журнал",
		fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	inner := container.NewBorder(
		header, nil, nil, nil,
		w.logScroll,
	)
	return NewCard(inner)
}

// ── Установка ────────────────────────────────────────────────────────

func (w *window) start() {
	if w.closing.Load() || w.running {
		return
	}
	w.beginInstall()
}

func (w *window) beginInstall() {
	if w.closing.Load() {
		return
	}

	w.progress.SetValue(0)
	w.status.SetText("Запуск…")
	w.status.Importance = widget.MediumImportance
	w.clearLog()
	w.resetSteps()
	w.startTime = time.Now()

	w.running = true
	w.btnInstall.Disable()

	cmd, err := w.buildInstallCmd()
	if err != nil {
		dialog.ShowError(err, w.win)
		w.running = false
		w.btnInstall.Enable()
		return
	}

	w.appendLog("$ "+strings.Join(cmd.Args, " "), logtheme.Step)

	wk := runner.New(cmd, w.steps, runner.Callbacks{
		OnLine: func(line string, tag logtheme.Tag) {
			w.appendLog(line, tag)
		},
		OnStep: func(index int, marker string) {
			fyne.Do(func() { w.updateStep(index, marker) })
		},
		OnDone: w.onDone,
	})

	w.worker = wk
	go wk.Run()
}

// buildInstallCmd — программа уже работает от root, sudo не нужен.
func (w *window) buildInstallCmd() (*exec.Cmd, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить путь к программе: %w", err)
	}
	self, err = filepath.Abs(self)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить абсолютный путь: %w", err)
	}
	return exec.Command(self, "--installer"), nil
}

func (w *window) onDone(rc int) {
	if w.closing.Load() {
		return
	}

	w.flushLog()
	fyne.Do(func() {
		w.running = false
		w.btnInstall.Enable()

		switch {
		case rc == 0:
			w.markAllStepsDone()
			w.progress.SetValue(100)
			w.appendLog("", logtheme.Default)
			w.appendLog("═══ Установка завершена успешно ═══", logtheme.OK)

		case rc < 0:
			w.status.SetText("⛔ Установка отменена")
			w.status.Importance = widget.WarningImportance
			w.appendLog("═══ Установка отменена ═══", logtheme.Error)

		default:
			if w.currentStep >= 0 {
				w.setStepState(w.currentStep, stepFailed)
			}
			w.appendLog(fmt.Sprintf("═══ Ошибка установки (код %d) ═══", rc), logtheme.Error)
		}

		w.finalizeStatus(rc)

		if rc == 0 {
			w.offerReboot()
		}
	})
}

// offerReboot показывает диалог с предложением перезагрузить систему
// после успешной установки.
func (w *window) offerReboot() {
	dialog.ShowConfirm(
		"Установка завершена",
		"Все компоненты установлены.\n\n"+
			"Перезагрузить систему сейчас?\n\n"+
			"Это нужно, чтобы применились:\n"+
			"  • low-latency профиль tuned\n"+
			"  • драйверы аппаратного ускорения\n"+
			"  • WineASIO",
		func(ok bool) {
			if !ok {
				return
			}
			w.reboot()
		}, w.win)
}

// reboot инициирует перезагрузку системы.
func (w *window) reboot() {
	w.status.SetText("Перезагрузка…")
	w.status.Importance = widget.MediumImportance
	w.appendLog("$ systemctl reboot", logtheme.Step)

	if err := exec.Command("systemctl", "reboot").Run(); err != nil {
		w.appendLog(fmt.Sprintf("[WARN] systemctl reboot: %v", err), logtheme.Warn)
		if err2 := exec.Command("reboot").Run(); err2 != nil {
			dialog.ShowError(
				fmt.Errorf("не удалось перезагрузить систему: %v", err2),
				w.win,
			)
		}
	}
}

func (w *window) finalizeStatus(rc int) {
	switch {
	case rc == 0:
		w.status.SetText("✓ Готово. Перезагрузите систему")
		w.status.Importance = widget.SuccessImportance
	case rc > 0:
		w.status.SetText(fmt.Sprintf("❌ Ошибка установки (код %d)", rc))
		w.status.Importance = widget.DangerImportance
	}
}

func (w *window) setStepState(index int, state stepState) {
	if index < 0 || index >= len(w.stepLabels) {
		return
	}
	lbl := w.stepLabels[index]
	name := w.steps[index]

	switch state {
	case stepPending:
		lbl.SetText(fmt.Sprintf("○  %s", name))
		lbl.Importance = widget.LowImportance
	case stepRunning:
		lbl.SetText(fmt.Sprintf("▶  %s", name))
		lbl.Importance = widget.HighImportance
	case stepDone:
		lbl.SetText(fmt.Sprintf("✓  %s", name))
		lbl.Importance = widget.SuccessImportance
	case stepFailed:
		lbl.SetText(fmt.Sprintf("✗  %s", name))
		lbl.Importance = widget.DangerImportance
	}
	lbl.Refresh()
}

func (w *window) resetSteps() {
	for i := range w.steps {
		w.setStepState(i, stepPending)
	}
	w.currentStep = -1
}

func (w *window) markAllStepsDone() {
	for i := range w.steps {
		w.setStepState(i, stepDone)
	}
}

func (w *window) updateStep(index int, marker string) {
	for i := 0; i < index-1; i++ {
		w.setStepState(i, stepDone)
	}
	if index >= 1 && index <= len(w.steps) {
		w.setStepState(index-1, stepRunning)
		w.currentStep = index - 1
	}

	total := len(w.steps)
	pct := float64(index) / float64(total)
	if pct > 1 {
		pct = 1
	}
	w.progress.SetValue(pct * 100)
	w.status.SetText(fmt.Sprintf("Шаг %d из %d: %s", index, total, marker))
	w.status.Importance = widget.MediumImportance
}

func (w *window) appendLog(line string, tag logtheme.Tag) {
	w.logMu.Lock()
	w.logBuf = append(w.logBuf, logEntry{line, tag})
	w.logMu.Unlock()
}

func (w *window) flushLoop() {
	ticker := time.NewTicker(logFlushInterval)
	defer ticker.Stop()
	for range ticker.C {
		if w.closing.Load() {
			return
		}
		w.flushLog()
	}
}

func (w *window) flushLog() {
	w.logMu.Lock()
	if len(w.logBuf) == 0 {
		w.logMu.Unlock()
		return
	}
	buf := w.logBuf
	w.logBuf = nil
	startRow := w.logRow
	w.logRow += len(buf)
	w.logMu.Unlock()

	fyne.Do(func() {
		for i, e := range buf {
			w.logGrid.SetRow(startRow+i, e.line, w.logStyles[e.tag])
		}
		w.logScroll.ScrollToBottom()
	})
}

func (w *window) clearLog() {
	w.logMu.Lock()
	w.logBuf = nil
	w.logRow = 0
	w.logMu.Unlock()
	fyne.Do(func() { w.logGrid.SetText("") })
}