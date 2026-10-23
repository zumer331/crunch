package installer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
)

// reaperFallbackRelease — версия REAPER, если не удалось определить
// последнюю с reaper.fm 782 гарантированно существует.
const reaperFallbackRelease = "782"

// reaperReleaseRe ищет ссылку на Linux-сборку REAPER в HTML страницы.
var reaperReleaseRe = regexp.MustCompile(
	`reaper(\d+)_linux_(?:x86_64|aarch64)\.tar\.xz`,
)

// resolveReaperRelease возвращает номер последней релизной сборки REAPER.
// При ошибке сети, парсинга или HTTP-статуса — fallback.
func resolveReaperRelease() string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://www.reaper.fm/download.php")
	if err != nil {
		warn(fmt.Sprintf("Не удалось проверить последнюю версию REAPER: %v. Используем %s.",
			err, reaperFallbackRelease))
		return reaperFallbackRelease
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		warn(fmt.Sprintf("reaper.fm вернул статус %d. Используем %s.",
			resp.StatusCode, reaperFallbackRelease))
		return reaperFallbackRelease
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		warn(fmt.Sprintf("Не удалось прочитать страницу REAPER: %v. Используем %s.",
			err, reaperFallbackRelease))
		return reaperFallbackRelease
	}

	m := reaperReleaseRe.FindSubmatch(body)
	if len(m) < 2 {
		warn(fmt.Sprintf("Не удалось найти ссылку REAPER на странице. Используем %s.",
			reaperFallbackRelease))
		return reaperFallbackRelease
	}

	release := string(m[1])
	info(fmt.Sprintf("Обнаружена последняя версия REAPER: %s", release))
	return release
}

// reaperMajor возвращает major-версию для URL каталога на reaper.fm.
// 3-значные версии (7xx, 8xx, 9xx) → первая цифра.
// 4-значные (10xx, 11xx) → первые две.
func reaperMajor(release string) string {
	switch {
	case len(release) == 3:
		return release[:1]
	case len(release) >= 4:
		return release[:2]
	default:
		return "7"
	}
}

// reaperDownloadURL собирает URL архива для заданной release+arch.
func reaperDownloadURL(release, arch string) string {
	tarName := fmt.Sprintf("reaper%s_linux_%s.tar.xz", release, arch)
	return fmt.Sprintf("https://www.reaper.fm/files/%s.x/%s",
		reaperMajor(release), tarName)
}

// reaperArchName отображает runtime.GOARCH на имя сборки REAPER.
func reaperArchName() (string, bool) {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64", true
	case "arm64":
		return "aarch64", true
	}
	return "", false
}

// installReaper скачивает и устанавливает последнюю версию REAPER.
func installReaper() error {
	arch, ok := reaperArchName()
	if !ok {
		warn(fmt.Sprintf("Архитектура %s не поддерживается. REAPER пропущен.", runtime.GOARCH))
		return nil
	}

	release := resolveReaperRelease()
	url := reaperDownloadURL(release, arch)
	tarName := fmt.Sprintf("reaper%s_linux_%s.tar.xz", release, arch)

	tmp, err := os.MkdirTemp("", "reaper-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	info(fmt.Sprintf("Скачивание REAPER %s для %s...", release, arch))
	if err := run("wget", "-q", "--show-progress",
		"-O", filepath.Join(tmp, tarName), url); err != nil {
		return err
	}

	info("Распаковка и установка в /opt/REAPER...")
	if err := run("tar", "-xf", filepath.Join(tmp, tarName), "-C", tmp); err != nil {
		return err
	}

	installReaperDir(tmp, release, arch)
	return nil
}

// installReaperDir перемещает распакованный REAPER в /opt и ставит симлинк.
func installReaperDir(tmp, release, arch string) {
	if err := os.RemoveAll("/opt/REAPER"); err != nil {
		warn(fmt.Sprintf("Не удалось удалить старый /opt/REAPER: %v", err))
	}

	src := filepath.Join(tmp, fmt.Sprintf("reaper%s_linux_%s", release, arch))
	if err := run("mv", src, "/opt/REAPER"); err != nil {
		warn(fmt.Sprintf("Не удалось установить REAPER в /opt: %v. Закройте REAPER и повторите.", err))
		return
	}

	link := "/usr/local/bin/reaper"
	os.Remove(link)
	if err := os.Symlink("/opt/REAPER/reaper", link); err != nil {
		warn(fmt.Sprintf("Не удалось создать симлинк %s: %v", link, err))
	}
}