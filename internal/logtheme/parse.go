// Package logtheme разбирает строки лога на теги и отдаёт цвета
// для подсветки в GUI.
package logtheme

import (
	"regexp"
	"strings"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Tag — тег строки лога.
type Tag int

const (
	Default Tag = iota
	Info
	Warn
	Error
	Step
	OK
)

// StripANSI удаляет ANSI escape-последовательности.
func StripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// errorSubs — предвычисленные подстроки для определения ошибок.
// Проверяются все падежи и оба регистра, чтобы не делать ToLower
// на каждой строке (аллокация + копирование).
var errorSubs = [...]string{
	"error", "Error", "ERROR",
	"failed", "Failed", "FAILED",
	"ошибка", "Ошибка", "ОШИБКА",
}

// ParseTag определяет тег строки лога.
func ParseTag(line string) Tag {
	if len(line) == 0 {
		return Default
	}

	if line[0] == '[' {
		if strings.HasPrefix(line, "[INFO]") {
			return Info
		}
		if strings.HasPrefix(line, "[WARN]") {
			return Warn
		}
	}

	for _, sub := range errorSubs {
		if strings.Contains(line, sub) {
			return Error
		}
	}
	return Default
}