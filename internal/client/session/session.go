// Package session отвечает за хранение и загрузку пользовательской сессии
// (логин + JWT токен) на стороне клиента. Сессия сериализуется в JSON
// и сохраняется на диск с правами 0600.
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Session представляет состояние аутентифицированного клиента: логин
// пользователя и его JWT токен, выданный сервером.
type Session struct {
	Login string `json:"login"`
	Token string `json:"token"`
}

// Load читает и декодирует сессию из файла по указанному пути.
// Возвращает ошибку, если файл отсутствует, недоступен или содержит
// некорректный JSON.
func Load(path string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("session.Load: read session: %w", err)
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("session.Load: decode session: %w", err)
	}
	return &session, nil
}

// Save сериализует сессию в JSON и записывает в файл по указанному пути.
// При отсутствии родительской директории создаёт её с правами 0700.
// Файл сессии записывается с правами 0600 (только владелец может читать/писать).
func Save(path string, s *Session) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("session.Save: create dir: %w", err)
	}
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("session.Save: encode session: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("session.Save: write session: %w", err)
	}
	return nil
}
