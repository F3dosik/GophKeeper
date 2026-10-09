package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Файлы в каталоге данных приложения.
const (
	settingsFile = "settings.json"
	caCertFile   = "ca.crt"
	sessionFile  = "session"
)

// defaultAutoLockMinutes — таймаут автоблокировки по умолчанию.
const defaultAutoLockMinutes = 5

// Settings — сохраняемые настройки приложения.
type Settings struct {
	// ServerAddress — адрес сервера (host:port); пустой — приложение не настроено.
	ServerAddress string `json:"serverAddress"`
	// AutoLockMinutes — блокировка после бездействия, минут; 0 — не блокировать.
	AutoLockMinutes int `json:"autoLockMinutes"`
}

// loadSettings читает настройки; отсутствие файла — не ошибка.
func loadSettings(dir string) (Settings, error) {
	settings := Settings{AutoLockMinutes: defaultAutoLockMinutes}
	data, err := os.ReadFile(filepath.Join(dir, settingsFile))
	if errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return settings, fmt.Errorf("read settings: %w", err)
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return settings, fmt.Errorf("decode settings: %w", err)
	}
	return settings, nil
}

// saveSettings записывает настройки и CA-сертификат (пустой caPEM удаляет файл:
// тогда используются системные корневые сертификаты).
func saveSettings(dir string, settings Settings, caPEM []byte) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, settingsFile), data, 0o600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}

	caPath := filepath.Join(dir, caCertFile)
	if len(caPEM) == 0 {
		if err := os.Remove(caPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove CA certificate: %w", err)
		}
		return nil
	}
	if err := os.WriteFile(caPath, caPEM, 0o600); err != nil {
		return fmt.Errorf("write CA certificate: %w", err)
	}
	return nil
}

// loadCACert читает сохранённый CA-сертификат; отсутствие файла — пустой результат.
func loadCACert(dir string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(dir, caCertFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}
