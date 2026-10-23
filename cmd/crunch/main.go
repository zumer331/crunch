// Command crunch — установщик аудио-инструментов для Fedora.
package main

import (
	"flag"
	"fmt"
	"os"

	"crunch/internal/app"
	"crunch/internal/installer"
)

func main() {
	installerMode := flag.Bool("installer", false, "run installer (requires root)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = usage
	flag.Parse()

	switch {
	case *showVersion:
		fmt.Println(app.Name, app.Version)
		return
	case *installerMode:
		if os.Geteuid() != 0 {
			fmt.Fprintln(os.Stderr, "installer requires root privileges")
			os.Exit(1)
		}
		if err := installer.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	// GUI-режим. Программа работает только от root.
	if os.Geteuid() != 0 {
		if err := app.RelaunchWithSudo(); err != nil {
			fmt.Fprintln(os.Stderr, "Ошибка:", err)
			os.Exit(1)
		}
		return
	}
	app.RunGUI()
}

func usage() {
	fmt.Fprintf(os.Stderr, "%s %s\n\n", app.Name, app.Version)
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintf(os.Stderr, "  %s              запуск GUI (требует sudo)\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --installer  установщик без GUI (нужен root)\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s --version    версия\n", os.Args[0])
}