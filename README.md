# Crunch

GUI-утилита для Fedora 44, упрощающая установку программного обеспечения необходимого музыканту на  Linux. 
Один бинарник на Go + Fyne, c интерфейс в стиле libadwaita.

## Что устанавливает утилита?

- RPM Fusion (free + nonfree + tainted) + Cisco OpenH264
- Полный FFmpeg и кодеки GStreamer
- Проприетарные кодеки: GStreamer freeworld (bad, ugly, libav), MP3 (lame),
  H.264 (OpenH264), DVD (libdvdcss)
- Драйверы аппаратного ускорения (VA-API, Vulkan)
- Wine (32/64-бит) + дополнения (mono, gecko, dxvk, winetricks)
- WineASIO (ASIO-драйвер для Wine через PipeWire JACK)
- Аудио-стек: PipeWire + JACK + low-latency (`tuned`), `pavucontrol`
- yabridge + yabridgectl (COPR)
- VLC + vlc-plugins-freeworld, MuseScore, REAPER (с reaper.fm)

## Требования для сборки

```bash
sudo dnf install golang gcc gcc-c++ pkg-config gtk3-devel
```

## Сборка
```bash
make tidy
make build
make vet
```

Бинарник: 
```bash
./crunch
```
## Запуск

Программа работает только от root. Запускайте без sudo — Crunch сам попросит пароль и перезапустится:
```bash
./crunch
```

Или сразу:
```bash
sudo ./crunch
```

Установщик без GUI:
```bash
sudo ./crunch --installer
```

Версия:
```bash
./crunch --version
```

## После установки
1. Перезагрузить систему (для low-latency профиля, драйверов, WineASIO).
2. Дождаться диалогов wine-mono/wine-gecko в winecfg.
3. Для Windows-VST: yabridgectl add <путь> и yabridgectl sync.
4. Для WineASIO зарегистрировать DLL в нужном Wine-префиксе:
```bash
regsvr32 wineasio.dll
# или для 64-бит:
wine64 regsvr32 wineasio.dll
```