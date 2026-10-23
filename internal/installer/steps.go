package installer

import (
	"fmt"
)

// ── Шаг 1. Обновление ────────────────────────────────────────────────

func stepUpgrade() error {
	info("Обновление системы...")
	return run("dnf", "-y", "upgrade", "--refresh")
}

// ── Шаг 2. RPM Fusion ────────────────────────────────────────────────

func stepRpmfusion() error {
	info("Подключение RPM Fusion...")
	ver, err := fedoraVersion()
	if err != nil {
		return err
	}
	if err := run("dnf", "-y", "install",
		fmt.Sprintf("https://mirrors.rpmfusion.org/free/fedora/rpmfusion-free-release-%s.noarch.rpm", ver),
		fmt.Sprintf("https://mirrors.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-%s.noarch.rpm", ver),
	); err != nil {
		return err
	}

	info("Подключение tainted-репозиториев RPM Fusion...")
	if err := run("dnf", "-y", "install",
		"rpmfusion-free-release-tainted",
		"rpmfusion-nonfree-release-tainted",
	); err != nil {
		return err
	}

	// config-manager — плагин dnf. Если его нет — OpenH264 пропускаем,
	// остальная установка продолжается.
	info("Установка dnf-plugins-core...")
	if err := run("dnf", "-y", "install", "dnf-plugins-core"); err != nil {
		warn(fmt.Sprintf("dnf-plugins-core не установился: %v. OpenH264 будет пропущен.", err))
		return nil
	}

	info("Включение репозитория Cisco OpenH264...")
	if err := run("dnf", "-y", "config-manager", "setopt",
		"fedora-cisco-openh264.enabled=1"); err != nil {
		warn(fmt.Sprintf("Не удалось включить fedora-cisco-openh264: %v", err))
	}
	return nil
}

// ── Шаг 3. Аппаратное ускорение ──────────────────────────────────────

func stepHWAccel() error {
	// pkg() сам печатает info — дублирующий info() не нужен.
	if err := pkg("Установка драйверов аппаратного ускорения", hwAccelPackages...); err != nil {
		return err
	}
	return swap("Замена mesa-vulkan-drivers на freeworld",
		"mesa-vulkan-drivers", "mesa-vulkan-drivers-freeworld")
}

// ── Шаг 4. Кодеки и мультимедиа ──────────────────────────────────────

func stepCodecsAndMedia() error {
	info("Установка кодеков и мультимедиа...")

	if err := codecsSwapFfmpeg(); err != nil {
		return err
	}
	if err := codecsInstallGStreamer(); err != nil {
		return err
	}
	return codecsInstallProprietary()
}

func codecsSwapFfmpeg() error {
	info("Замена ffmpeg-free на полный FFmpeg...")
	return run("dnf", "-y", "swap", "ffmpeg-free", "ffmpeg", "--allowerasing")
}

func codecsInstallGStreamer() error {
	info("Установка мультимедийной группы GStreamer...")
	return run("dnf", "-y", "install", "@multimedia",
		"--setopt=install_weak_deps=False",
		"--exclude=PackageKit-gstreamer-plugin")
}

func codecsInstallProprietary() error {
	info("Установка проприетарных кодеков...")

	info("GStreamer freeworld: bad, ugly, libav, lame...")
	if err := run("dnf", "-y", "install",
		"gstreamer1-plugins-bad-freeworld",
		"gstreamer1-plugins-ugly",
		"gstreamer1-libav",
		"lame",
	); err != nil {
		return err
	}

	info("Cisco OpenH264...")
	if err := run("dnf", "-y", "install",
		"gstreamer1-plugin-openh264",
		"mozilla-openh264",
	); err != nil {
		warn(fmt.Sprintf("OpenH264 не установился: %v", err))
	}

	info("libdvdcss для DVD...")
	if err := run("dnf", "-y", "install", "libdvdcss"); err != nil {
		warn(fmt.Sprintf("libdvdcss не установился: %v", err))
	}

	return nil
}

// ── Шаг 5. Wine ──────────────────────────────────────────────────────

func stepWine() error {
	info("Установка и настройка Wine (32/64-бит) и WineASIO...")

	if err := coprEnable("patrickl/wine-tkg"); err != nil {
		return err
	}
	if err := coprEnable("patrickl/pipewire-wineasio"); err != nil {
		return err
	}
	if err := run("dnf", "-y", "clean", "all"); err != nil {
		return err
	}

	info("Установка Wine 32/64-бит...")
	if err := run("dnf", "-y", "install", "wine.x86_64", "wine.i686"); err != nil {
		return err
	}

	info("Установка дополнений Wine (mono, gecko, dxvk, winetricks)...")
	if err := run("dnf", "-y", "install",
		"wine-mono-bin",
		"mingw32-wine-gecko", "mingw64-wine-gecko",
		"wine-dxvk",
		"winetricks",
		"libcurl-gnutls",
	); err != nil {
		return err
	}

	info("Установка WineASIO...")
	if err := run("dnf", "-y", "install",
		"pipewire-wineasio",
		"pipewire-jack-audio-connection-kit",
		"--allowerasing",
	); err != nil {
		return err
	}

	info("Настройка Wine (winecfg)...")
	runWinecfg()

	info("WineASIO установлен. Для активации в Wine-префиксе выполните от обычного пользователя:")
	info("  regsvr32 wineasio.dll")
	info("  (для 64-битного префикса: wine64 regsvr32 wineasio.dll)")
	info("  (для конкретного префикса: env WINEPREFIX=~/path regsvr32 wineasio.dll)")

	return nil
}

// ── Шаг 6. Аудио-стек ────────────────────────────────────────────────

func stepAudioStack() error {
	// pkg() сам печатает info.
	if err := pkg("Установка аудио-стека", audioPackages...); err != nil {
		return err
	}
	if err := run("tuned-adm", "profile", "latency-performance"); err != nil {
		warn("Профиль tuned latency-performance недоступен, пропускаем.")
	}
	return nil
}

// ── Шаг 7. Yabridge ──────────────────────────────────────────────────

func stepYabridge() error {
	info("Установка yabridge...")
	if err := coprEnable("patrickl/yabridge"); err != nil {
		return err
	}
	return run("dnf", "-y", "install", "yabridge")
}

// ── Шаг 8. Программы ─────────────────────────────────────────────────

func stepAudioPrograms() error {
	info("Установка программ...")

	if err := installVLC(); err != nil {
		return err
	}
	if err := installMuseScore(); err != nil {
		return err
	}
	return installReaper()
}

func installVLC() error {
	info("Установка VLC...")
	return run("dnf", "-y", "install",
		"vlc",
		"vlc-plugins-freeworld",
		"mozilla-vlc",
	)
}

func installMuseScore() error {
	info("Установка MuseScore...")
	return run("dnf", "-y", "install", "musescore")
}