package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// recordingLogger запоминает переданные сообщения.
type recordingLogger struct {
	messages []string
}

func (r *recordingLogger) Print(m string)   { r.messages = append(r.messages, m) }
func (r *recordingLogger) Trace(m string)   { r.messages = append(r.messages, m) }
func (r *recordingLogger) Debug(m string)   { r.messages = append(r.messages, m) }
func (r *recordingLogger) Info(m string)    { r.messages = append(r.messages, m) }
func (r *recordingLogger) Warning(m string) { r.messages = append(r.messages, m) }
func (r *recordingLogger) Error(m string)   { r.messages = append(r.messages, m) }
func (r *recordingLogger) Fatal(m string)   { r.messages = append(r.messages, m) }

func TestRedact(t *testing.T) {
	const secret = "correct-horse-battery-staple"

	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			// Так Wails пишет панику в привязанном методе — с аргументами вызова.
			name:    "process message error",
			message: `process message error: C{"name":"main.App.Login","args":["alice","` + secret + `"],"callbackID":"1"} -> runtime error: index out of range`,
			want:    "process message error: [содержимое скрыто] -> runtime error: index out of range",
		},
		{
			name:    "process message error without error text",
			message: `process message error: C{"args":["` + secret + `"]}`,
			want:    "process message error: [содержимое скрыто]",
		},
		{
			name:    "call result",
			message: `json call result data: {"result":{"password":"` + secret + `"}}`,
			want:    "json call result data: [содержимое скрыто]",
		},
		{
			name:    "unknown dispatcher message",
			message: "Unknown message returned from dispatcher: " + secret,
			want:    "Unknown message returned from dispatcher: [содержимое скрыто]",
		},
		{
			name:    "unrelated message",
			message: "Blocked request from unauthorized origin: http://evil",
			want:    "Blocked request from unauthorized origin: http://evil",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redact(tt.message)
			assert.Equal(t, tt.want, got)
			assert.NotContains(t, got, secret)
		})
	}
}

func TestRedactedLoggerAllLevels(t *testing.T) {
	const secret = "s3cr3t"
	rec := &recordingLogger{}
	l := newRedactedLogger(rec)
	msg := `process message error: C{"args":["` + secret + `"]} -> boom`

	l.Print(msg)
	l.Trace(msg)
	l.Debug(msg)
	l.Info(msg)
	l.Warning(msg)
	l.Error(msg)
	l.Fatal(msg)

	assert.Len(t, rec.messages, 7)
	for _, m := range rec.messages {
		assert.False(t, strings.Contains(m, secret), m)
	}
}
