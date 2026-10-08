package grpcclient

import "sync"

// TokenStore хранит текущий JWT-токен, который authInterceptor прикрепляет к запросам.
// Токен можно заменить во время работы, например после повторной проверки пароля,
// и последующие запросы того же соединения пойдут уже с новым токеном.
// Методы безопасны для конкурентного вызова и для nil-указателя.
type TokenStore struct {
	mu    sync.RWMutex
	token string
}

// NewTokenStore создаёт хранилище с начальным токеном (может быть пустым).
func NewTokenStore(token string) *TokenStore {
	return &TokenStore{token: token}
}

// Token возвращает текущий токен.
func (s *TokenStore) Token() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

// SetToken заменяет текущий токен.
func (s *TokenStore) SetToken(token string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
}
