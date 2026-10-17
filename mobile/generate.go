package mobile

import (
	"math"

	"github.com/F3dosik/Shron/internal/client/passgen"
)

// GeneratorOptions — параметры пароля из символов.
type GeneratorOptions struct {
	Length      int
	Lower       bool
	Upper       bool
	Digits      bool
	Symbols     bool
	NoAmbiguous bool
}

// Generated — сгенерированный пароль и его стойкость.
type Generated struct {
	Password    string
	EntropyBits int
	Strength    string
}

// GeneratePassword генерирует пароль из случайных символов. Генерация выполняется на
// устройстве, пароль никуда не отправляется. В Kotlin: Mobile.generatePassword(opts).
func GeneratePassword(opts *GeneratorOptions) (*Generated, error) {
	if opts == nil {
		return nil, validationError("Нет параметров генерации")
	}
	result, err := passgen.Password(passgen.Options{
		Length:      opts.Length,
		Lower:       opts.Lower,
		Upper:       opts.Upper,
		Digits:      opts.Digits,
		Symbols:     opts.Symbols,
		NoAmbiguous: opts.NoAmbiguous,
	})
	if err != nil {
		return nil, validationError(err.Error())
	}
	return toGenerated(result), nil
}

// GeneratePassphrase генерирует парольную фразу из words случайных слов словаря EFF,
// разделённых дефисом.
func GeneratePassphrase(words int) (*Generated, error) {
	result, err := passgen.Passphrase(words, "-")
	if err != nil {
		return nil, validationError(err.Error())
	}
	return toGenerated(result), nil
}

func toGenerated(r passgen.Result) *Generated {
	return &Generated{
		Password:    r.Password,
		EntropyBits: int(math.Floor(r.EntropyBits)),
		Strength:    passgen.Strength(r.EntropyBits),
	}
}
