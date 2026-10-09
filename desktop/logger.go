package main

import (
	"strings"

	"github.com/wailsapp/wails/v2/pkg/logger"
)

// ipcLogPrefixes — сообщения Wails, в которые попадает содержимое вызовов между
// интерфейсом и Go: аргументы (мастер-пароль, данные секрета) или результаты
// (расшифрованные секреты). Например, если метод паникует, Wails пишет на уровне ERROR
// весь вызов вместе с аргументами — и в релизной сборке тоже.
var ipcLogPrefixes = []string{
	"process message error: ",
	"json call result data: ",
	"Unknown message returned from dispatcher: ",
	"unknown Browser message: ",
	"unknown Window message: ",
}

// redactedLogger передаёт сообщения дальше, вырезая из них содержимое вызовов.
type redactedLogger struct {
	next logger.Logger
}

func newRedactedLogger(next logger.Logger) logger.Logger {
	return &redactedLogger{next: next}
}

// redact заменяет содержимое вызова в сообщении на заглушку. У ошибки обработки
// сообщения сохраняется текст ошибки после « -> », чтобы паника оставалась диагностируемой.
func redact(message string) string {
	for _, prefix := range ipcLogPrefixes {
		if !strings.HasPrefix(message, prefix) {
			continue
		}
		redacted := prefix + "[содержимое скрыто]"
		if prefix == "process message error: " {
			if i := strings.LastIndex(message, " -> "); i >= len(prefix) {
				redacted += message[i:]
			}
		}
		return redacted
	}
	return message
}

func (l *redactedLogger) Print(message string)   { l.next.Print(redact(message)) }
func (l *redactedLogger) Trace(message string)   { l.next.Trace(redact(message)) }
func (l *redactedLogger) Debug(message string)   { l.next.Debug(redact(message)) }
func (l *redactedLogger) Info(message string)    { l.next.Info(redact(message)) }
func (l *redactedLogger) Warning(message string) { l.next.Warning(redact(message)) }
func (l *redactedLogger) Error(message string)   { l.next.Error(redact(message)) }
func (l *redactedLogger) Fatal(message string)   { l.next.Fatal(redact(message)) }
