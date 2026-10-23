// Package installer выполняет установку пакетов и настройку системы.
// Запускается только с правами root через флаг --installer.
package installer

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"crunch/internal/setup"
)

const (
	colorGreen  = "\x1b[0;32m"
	colorYellow = "\x1b[1;33m"
	colorReset  = "\x1b[0m"
)

var (
	hwAccelPackages = []string{
		"mesa-va-drivers-freeworld", "libva-utils",
	}
	audioPackages = []string{
		"pipewire", "pipewire-jack", "pipewire-utils",
		"qpwgraph", "wireplumber",
		"kernel-tools", "tuned",
		"pavucontrol",
	}
)

type step struct {
	name string
	fn   func() error
}

// steps — упорядоченный список шагов установки.
//
// Инвариант: name каждого шага — подстрока первой info-строки,
// которую печатает fn. По этой подстроке runner двигает прогресс-бар.
var steps = []step{
	{"Обновление системы", stepUpgrade},
	{"Подключение RPM Fusion", stepRpmfusion},
	{"Установка драйверов аппаратного ускорения", stepHWAccel},
	{"Установка кодеков и мультимедиа", stepCodecsAndMedia},
	{"Установка и настройка Wine (32/64-бит) и WineASIO", stepWine},
	{"Установка аудио-стека", stepAudioStack},
	{"Установка yabridge", stepYabridge},
	{"Установка программ", stepAudioPrograms},
}

// StepNames возвращает имена шагов в порядке выполнения.
func StepNames() []string {
	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.name
	}
	return names
}

// Run выполняет все шаги установки последовательно, затем — очистку кэша
// и финальный статус. При SIGINT/SIGTERM немедленно выходит с кодом 130.
func Run() error {
	if err := setup.EnsureWget(); err != nil {
		return fmt.Errorf("подготовка wget: %w", err)
	}

	stop := handleSignals()
	defer stop()

	for _, s := range steps {
		if err := s.fn(); err != nil {
			return fmt.Errorf("step %q: %w", s.name, err)
		}
	}

	if err := run("dnf", "-y", "clean", "all"); err != nil {
		warn(fmt.Sprintf("Не удалось очистить кэш DNF: %v", err))
	}

	info("Готово! Перезагрузите систему, чтобы применились low-latency настройки и драйверы.")
	return nil
}

func handleSignals() func() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		warn("Получен сигнал завершения, выходим…")
		os.Exit(130)
	}()
	return func() { signal.Stop(sigCh) }
}

// ── Вывод ────────────────────────────────────────────────────────────

func info(msg string) {
	fmt.Printf("%s[INFO]%s %s\n", colorGreen, colorReset, msg)
}

func warn(msg string) {
	fmt.Printf("%s[WARN]%s %s\n", colorYellow, colorReset, msg)
}

// ── Хелперы dnf ──────────────────────────────────────────────────────

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func pkg(msg string, packages ...string) error {
	info(msg + "...")
	return run("dnf", append([]string{"-y", "install"}, packages...)...)
}

func swap(msg, from, to string) error {
	info(msg + "...")
	return run("dnf", "-y", "swap", from, to, "--allowerasing")
}

func coprEnable(repo string) error {
	return run("dnf", "-y", "copr", "enable", repo)
}

func fedoraVersion() (string, error) {
	out, err := exec.Command("rpm", "-E", "%fedora").Output()
	if err != nil {
		return "", fmt.Errorf("rpm -E %%fedora: %w", err)
	}
	ver := strings.TrimSpace(string(out))
	if ver == "" {
		return "", fmt.Errorf("rpm -E %%fedora вернул пустую строку")
	}
	return ver, nil
}