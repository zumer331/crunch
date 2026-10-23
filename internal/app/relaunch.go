package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// RelaunchWithSudo показывает диалог пароля и перезапускает программу
// через sudo. Возвращает nil при успехе, error при отмене или ошибке sudo.
func RelaunchWithSudo() error {
	var password string
	var confirmed bool

	a := app.NewWithID("crunch-sudo")
	w := a.NewWindow(Name + " — нужны права администратора")
	w.Resize(fyne.NewSize(440, 260))
	w.CenterOnScreen()

	info := widget.NewLabel(
		Name + " запускается только с правами администратора.\n\n" +
			"Введите пароль sudo, чтобы продолжить:")
	info.Wrapping = fyne.TextWrapWord

	entry := widget.NewPasswordEntry()
	entry.SetPlaceHolder("Пароль sudo")

	errLabel := widget.NewLabel("")
	errLabel.Importance = widget.DangerImportance

	btnCancel := widget.NewButton("Отмена", func() { a.Quit() })
	btnOK := widget.NewButton("Продолжить", func() {
		if entry.Text == "" {
			errLabel.SetText("Пароль не введён")
			return
		}
		password = entry.Text
		confirmed = true
		a.Quit()
	})

	w.SetContent(container.NewPadded(container.NewVBox(
		info, entry, errLabel,
		layout.NewSpacer(),
		container.NewHBox(layout.NewSpacer(), btnCancel, btnOK),
	)))
	w.Show()
	a.Run()

	if !confirmed {
		return errors.New("отменено пользователем")
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("не удалось получить путь к программе: %w", err)
	}
	self, _ = filepath.Abs(self)

	args := []string{"-S", "-p", ""}
	args = append(args, "env")
	for _, k := range []string{"DISPLAY", "XAUTHORITY", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR"} {
		if v := os.Getenv(k); v != "" {
			args = append(args, k+"="+v)
		}
	}
	args = append(args, self)

	cmd := exec.Command("sudo", args...)

	var stderrBuf bytes.Buffer
	cmd.Stdin = strings.NewReader(password + "\n")
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	runErr := cmd.Run()
	password = ""

	if runErr == nil {
		return nil
	}

	msg := strings.TrimSpace(stderrBuf.String())
	switch {
	case strings.Contains(msg, "Sorry, try again"),
		strings.Contains(msg, "incorrect password"),
		strings.Contains(msg, "неверный пароль"):
		return errors.New("неверный пароль sudo")
	case msg != "":
		return fmt.Errorf("sudo: %s", msg)
	default:
		return fmt.Errorf("sudo: %w", runErr)
	}
}