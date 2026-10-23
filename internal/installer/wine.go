package installer

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

// runWinecfg запускает winecfg в фоне от имени SUDO_USER.
// Создаёт Wine-префикс ~/.wine. Ошибки не фатальны — установка продолжается.
func runWinecfg() {
	sudoUser, home, uid, err := sudoUserInfo()
	if err != nil {
		warn(err.Error())
		return
	}

	info(fmt.Sprintf("Запуск winecfg от пользователя %s в фоне (без прав root)...", sudoUser))
	info(fmt.Sprintf("Префикс будет создан в %s/.wine. Завершите диалоги wine-mono/gecko вручную.", home))

	args := append([]string{"-u", sudoUser, "-H", "env"}, winecfgEnv(home, uid)...)
	args = append(args, "bash", "-c", "winecfg >/tmp/winecfg-$USER.log 2>&1 &")

	cmd := exec.Command("sudo", args...)
	if err := cmd.Start(); err != nil {
		warn(fmt.Sprintf("Не удалось запустить winecfg: %v", err))
		return
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			warn(fmt.Sprintf("winecfg завершился с ошибкой: %v", err))
		}
	}()
}

// sudoUserInfo возвращает имя, домашний каталог и uid пользователя,
// который вызвал sudo. Ошибка — если SUDO_USER не определён.
func sudoUserInfo() (name, home, uid string, err error) {
	name = os.Getenv("SUDO_USER")
	if name == "" || name == "root" {
		return "", "", "", fmt.Errorf("SUDO_USER не определён, запустите winecfg вручную")
	}
	u, lookupErr := user.Lookup(name)
	if lookupErr != nil {
		return "", "", "", fmt.Errorf("не удалось найти пользователя %s: %w", name, lookupErr)
	}
	return name, u.HomeDir, u.Uid, nil
}

// winecfgEnv собирает переменные окружения для winecfg.
func winecfgEnv(home, uid string) []string {
	display := os.Getenv("DISPLAY")
	if display == "" {
		display = ":0"
	}
	xauth := os.Getenv("XAUTHORITY")
	if xauth == "" {
		xauth = filepath.Join(home, ".Xauthority")
	}
	env := []string{
		"DISPLAY=" + display,
		"XAUTHORITY=" + xauth,
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/" + uid + "/bus",
	}
	if wd := os.Getenv("WAYLAND_DISPLAY"); wd != "" {
		env = append(env, "WAYLAND_DISPLAY="+wd)
	}
	return env
}