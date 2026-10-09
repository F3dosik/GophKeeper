package backend

import (
	"math"

	"github.com/F3dosik/GophKeeper/internal/client/passgen"
)

// GeneratorOptions — параметры пароля из символов.
type GeneratorOptions struct {
	Length      int  `json:"length"`
	Lower       bool `json:"lower"`
	Upper       bool `json:"upper"`
	Digits      bool `json:"digits"`
	Symbols     bool `json:"symbols"`
	NoAmbiguous bool `json:"noAmbiguous"`
}

// Generated — сгенерированный пароль и его стойкость.
type Generated struct {
	Password    string `json:"password"`
	EntropyBits int    `json:"entropyBits"`
	Strength    string `json:"strength"`
}

// GeneratePassword генерирует пароль из случайных символов. Генерация выполняется на
// клиенте, пароль никуда не отправляется.
func (a *App) GeneratePassword(opts GeneratorOptions) (*Generated, error) {
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
func (a *App) GeneratePassphrase(words int) (*Generated, error) {
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
