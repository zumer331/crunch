// Package setup готовит окружение перед запуском.
package setup

import (
	"fmt"
	"os"
	"os/exec"
)

// IsFedora проверяет, что система — Fedora.
func IsFedora() bool {
	_, err := os.Stat("/etc/fedora-release")
	return err == nil
}

// EnsureWget ставит wget через dnf, если его нет в PATH.
// Требует root. Возвращает ошибку, если dnf не смог установить
// или wget так и не появился в PATH.
func EnsureWget() error {
	if _, err := exec.LookPath("wget"); err == nil {
		return nil
	}

	cmd := exec.Command("dnf", "-y", "install", "wget")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dnf install wget: %w", err)
	}

	if _, err := exec.LookPath("wget"); err != nil {
		return fmt.Errorf("wget не появился в PATH после установки")
	}
	return nil
}