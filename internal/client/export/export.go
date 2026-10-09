// Package export — зашифрованный экспорт хранилища в файл и импорт из него.
//
// Экспорт не зависит от сервера и мастер-пароля: секреты шифруются отдельным паролем
// экспорта. Файл — JSON с заголовком (формат, версия, параметры Argon2id, соль) и
// шифротекстом: ключ = Argon2id(пароль экспорта, соль), шифр — AES-256-GCM. Внутри —
// JSON со списком секретов в том же виде, что и на сервере до шифрования (имя, тип,
// данные, комментарий). Такой файл можно хранить где угодно и импортировать в любую
// учётку на любом сервере.
package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/F3dosik/GophKeeper/pkg/crypto"
)

// Format и Version — идентификатор и версия формата файла.
const (
	Format  = "gophkeeper-export"
	Version = 1
)

// MinPasswordLength — минимальная длина пароля экспорта.
const MinPasswordLength = 8

// Ошибки экспорта и импорта.
var (
	// ErrNotExport — файл не является экспортом GophKeeper.
	ErrNotExport = errors.New("файл не является экспортом GophKeeper")
	// ErrUnsupportedVersion — экспорт создан более новой версией программы.
	ErrUnsupportedVersion = errors.New("экспорт создан более новой версией GophKeeper — обновите программу")
	// ErrWrongPassword — неверный пароль экспорта или файл повреждён (AES-GCM не различает эти случаи).
	ErrWrongPassword = errors.New("неверный пароль экспорта или файл повреждён")
	// ErrWeakPassword — слишком короткий пароль экспорта.
	ErrWeakPassword = fmt.Errorf("пароль экспорта должен быть не короче %d символов", MinPasswordLength)
)

// envelope — содержимое файла экспорта.
type envelope struct {
	Format  string    `json:"format"`
	Version int       `json:"version"`
	KDF     kdfParams `json:"kdf"`
	Salt    []byte    `json:"salt"`
	// Data — nonce || шифротекст AES-256-GCM от JSON-представления Archive.
	Data []byte `json:"data"`
}

type kdfParams struct {
	Time      uint32 `json:"time"`
	MemoryKiB uint32 `json:"memoryKiB"`
	Threads   uint8  `json:"threads"`
}

// Archive — расшифрованное содержимое экспорта.
type Archive struct {
	// ExportedAt — время создания экспорта.
	ExportedAt time.Time `json:"exportedAt"`
	// Login — учётка, из которой сделан экспорт (для информации).
	Login   string                 `json:"login"`
	Secrets []domain.SecretPayload `json:"secrets"`
}

// Encrypt шифрует архив паролем экспорта и возвращает содержимое файла.
func Encrypt(archive *Archive, password string) ([]byte, error) {
	if len(password) < MinPasswordLength {
		return nil, ErrWeakPassword
	}
	plaintext, err := json.Marshal(archive)
	if err != nil {
		return nil, fmt.Errorf("export: encode: %w", err)
	}

	salt, err := crypto.GenerateSalt()
	if err != nil {
		return nil, fmt.Errorf("export: %w", err)
	}
	kdf := domain.DefaultKDFParams
	cipher, err := newCipher(password, salt, kdf)
	if err != nil {
		return nil, err
	}
	data, err := cipher.Encrypt(plaintext)
	clear(plaintext)
	if err != nil {
		return nil, fmt.Errorf("export: encrypt: %w", err)
	}

	return json.MarshalIndent(envelope{
		Format:  Format,
		Version: Version,
		KDF:     kdfParams{Time: kdf.Time, MemoryKiB: kdf.MemoryKiB, Threads: kdf.Threads},
		Salt:    salt,
		Data:    data,
	}, "", "  ")
}

// Decrypt разбирает и расшифровывает файл экспорта.
func Decrypt(file []byte, password string) (*Archive, error) {
	var env envelope
	if err := json.Unmarshal(file, &env); err != nil || env.Format != Format {
		return nil, ErrNotExport
	}
	if env.Version > Version {
		return nil, ErrUnsupportedVersion
	}
	kdf := domain.KDFParams{Time: env.KDF.Time, MemoryKiB: env.KDF.MemoryKiB, Threads: env.KDF.Threads}
	// Параметры берутся из файла, поэтому проверяем границы: файл мог быть подделан,
	// чтобы заставить программу считать Argon2id часами или съесть всю память.
	if err := kdf.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotExport, err)
	}
	if len(env.Salt) == 0 {
		return nil, ErrNotExport
	}

	cipher, err := newCipher(password, env.Salt, kdf)
	if err != nil {
		return nil, err
	}
	plaintext, err := cipher.Decrypt(env.Data)
	if err != nil {
		return nil, ErrWrongPassword
	}
	defer clear(plaintext)

	var archive Archive
	if err := json.Unmarshal(plaintext, &archive); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotExport, err)
	}
	return &archive, nil
}

// newCipher выводит ключ из пароля экспорта и создаёт AES-256-GCM.
func newCipher(password string, salt []byte, kdf domain.KDFParams) (crypto.Cipher, error) {
	key := crypto.DeriveKey(password, salt, kdf)
	defer clear(key)
	cipher, err := crypto.NewAESCipher(key)
	if err != nil {
		return nil, fmt.Errorf("export: %w", err)
	}
	return cipher, nil
}
