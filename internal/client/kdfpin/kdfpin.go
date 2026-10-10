// Package kdfpin запоминает параметры Argon2id учёток на этом устройстве.
//
// Соль и параметры Argon2id клиент получает от сервера перед каждым входом. Подменённый
// сервер мог бы прислать минимальные допустимые параметры: клиент вывел бы по ним ключ
// аутентификации и отправил его, а такой ключ перебирать в разы дешевле, чем ключ с
// настоящими параметрами. Поэтому клиент запоминает параметры, с которыми учётка уже
// входила с этого устройства (как SSH запоминает ключ сервера), и не выводит ключ, если
// сервер прислал более слабые.
//
// Соль не запоминается: она меняется при каждой смене пароля, а подмена соли без
// ослабления параметров перебор не удешевляет.
package kdfpin

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/F3dosik/GophKeeper/internal/domain"
)

// FileName — имя файла с параметрами; лежит рядом с файлом сессии.
const FileName = "known_kdf.json"

// Store — параметры учёток в JSON-файле. Нулевой (nil) Store ничего не хранит и
// ничего не запрещает: так работают клиенты без локального состояния.
type Store struct {
	path string
	mu   sync.Mutex
}

// entry — параметры одной учётки в файле.
type entry struct {
	Time      uint32 `json:"time"`
	MemoryKiB uint32 `json:"memoryKiB"`
	Threads   uint8  `json:"threads"`
}

// New возвращает Store, хранящий параметры в path.
func New(path string) *Store {
	return &Store{path: path}
}

// NextTo возвращает Store в каталоге файла сессии sessionPath; для пустого пути — nil.
func NextTo(sessionPath string) *Store {
	if sessionPath == "" {
		return nil
	}
	return New(filepath.Join(filepath.Dir(sessionPath), FileName))
}

// key — учётка на конкретном сервере. Логины на сервере регистронезависимы.
func key(server, login string) string {
	return server + "|" + strings.ToLower(login)
}

// Get возвращает запомненные параметры учётки; ok == false — учётка с этого
// устройства ещё не входила.
func (s *Store) Get(server, login string) (params domain.KDFParams, ok bool, err error) {
	if s == nil {
		return domain.KDFParams{}, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.load()
	if err != nil {
		return domain.KDFParams{}, false, err
	}
	e, ok := entries[key(server, login)]
	if !ok {
		return domain.KDFParams{}, false, nil
	}
	return domain.KDFParams{Time: e.Time, MemoryKiB: e.MemoryKiB, Threads: e.Threads}, true, nil
}

// Remember запоминает параметры учётки, заменяя прежние.
func (s *Store) Remember(server, login string, params domain.KDFParams) error {
	if s == nil {
		return nil
	}
	return s.update(func(entries map[string]entry) {
		entries[key(server, login)] = entry{Time: params.Time, MemoryKiB: params.MemoryKiB, Threads: params.Threads}
	})
}

// Forget удаляет параметры учётки: следующий вход примет любые допустимые параметры.
func (s *Store) Forget(server, login string) error {
	if s == nil {
		return nil
	}
	return s.update(func(entries map[string]entry) {
		delete(entries, key(server, login))
	})
}

func (s *Store) update(change func(map[string]entry)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.load()
	if err != nil {
		return err
	}
	change(entries)
	return s.save(entries)
}

// load читает файл. Повреждённый файл — ошибка, а не пустой список: иначе испорченный
// или подменённый файл молча отключил бы защиту.
func (s *Store) load() (map[string]entry, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]entry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("kdfpin: %w", err)
	}
	entries := map[string]entry{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("kdfpin: файл %s повреждён: %w", s.path, err)
	}
	return entries, nil
}

// save записывает файл атомарно: через временный файл и переименование, чтобы
// одновременный запуск нескольких клиентов или сбой не оставили его недописанным.
func (s *Store) save(entries map[string]entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("kdfpin: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("kdfpin: %w", err)
	}
	tmp, err := os.CreateTemp(dir, FileName+".*")
	if err != nil {
		return fmt.Errorf("kdfpin: %w", err)
	}
	defer os.Remove(tmp.Name()) // после успешного переименования файла уже нет
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("kdfpin: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("kdfpin: %w", err)
	}
	if err := os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("kdfpin: %w", err)
	}
	return nil
}
