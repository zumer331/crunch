// Package app связывает GUI и установщик.
package app

import (
	"fmt"
	"os"

	"crunch/internal/installer"
	"crunch/internal/setup"
	"crunch/internal/ui"
)

const Name = "Crunch"
const Version = "1.0"

// RunGUI проверяет окружение и запускает GUI.
// Должна вызываться только с правами root.
//
// EnsureWget здесь не вызывается — это делает installer.Run() в момент
// нажатия «Начать установку». Так логика подготовки wget живёт в одном
// месте и срабатывает ровно один раз за установку.
func RunGUI() {
	if !setup.IsFedora() {
		fmt.Fprintln(os.Stderr, "Crunch работает только на Fedora.")
		os.Exit(1)
	}
	ui.Run(installer.StepNames())
}