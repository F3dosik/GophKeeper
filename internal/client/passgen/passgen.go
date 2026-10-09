// Package passgen генерирует стойкие пароли и парольные фразы на клиенте.
//
// Случайность берётся из crypto/rand, символы и слова выбираются без статистического
// перекоса (rand.Int использует отбор, а не взятие остатка).
//
// Парольные фразы составляются из словаря EFF Large Wordlist (7776 слов, 12,9 бита
// на слово): https://www.eff.org/dice. Словарь распространяется по лицензии
// Creative Commons Attribution 3.0 US (CC BY 3.0 US), автор — Electronic Frontier Foundation.
package passgen

import (
	"crypto/rand"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Наборы символов для паролей.
const (
	lowerChars  = "abcdefghijklmnopqrstuvwxyz"
	upperChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars  = "0123456789"
	symbolChars = "!@#$%^&*()-_=+[]{};:,.?/~"
	// ambiguousChars — символы, которые легко спутать при чтении и вводе вручную.
	ambiguousChars = "0O1lI|"
)

// Ограничения.
const (
	MinLength     = 8
	MaxLength     = 128
	DefaultLength = 20
	MinWords      = 4
	MaxWords      = 12
	DefaultWords  = 6
)

// Ошибки параметров.
var (
	ErrLength    = fmt.Errorf("длина пароля должна быть от %d до %d", MinLength, MaxLength)
	ErrNoSets    = errors.New("выберите хотя бы один набор символов")
	ErrWords     = fmt.Errorf("число слов должно быть от %d до %d", MinWords, MaxWords)
	ErrSeparator = errors.New("разделитель — от 1 до 3 символов без переводов строки и табуляции")
)

// Options — параметры пароля из символов.
type Options struct {
	Length  int
	Lower   bool
	Upper   bool
	Digits  bool
	Symbols bool
	// NoAmbiguous исключает похожие символы (0/O, 1/l/I, |).
	NoAmbiguous bool
}

// DefaultOptions — все наборы символов, длина 20.
func DefaultOptions() Options {
	return Options{Length: DefaultLength, Lower: true, Upper: true, Digits: true, Symbols: true}
}

// Result — сгенерированный пароль и его стойкость.
type Result struct {
	Password string
	// EntropyBits — стойкость в битах при условии, что атакующий знает способ генерации.
	EntropyBits float64
}

//go:embed eff_large_wordlist.txt
var wordlistData string

// wordlist — слова словаря EFF.
var wordlist = strings.Fields(wordlistData)

// Password генерирует пароль из символов выбранных наборов. В пароле обязательно есть
// хотя бы один символ каждого выбранного набора: некоторые сайты этого требуют.
func Password(opts Options) (Result, error) {
	if opts.Length < MinLength || opts.Length > MaxLength {
		return Result{}, ErrLength
	}

	var sets []string
	for _, set := range []struct {
		on    bool
		chars string
	}{
		{opts.Lower, lowerChars},
		{opts.Upper, upperChars},
		{opts.Digits, digitChars},
		{opts.Symbols, symbolChars},
	} {
		if !set.on {
			continue
		}
		chars := set.chars
		if opts.NoAmbiguous {
			chars = strings.Map(func(r rune) rune {
				if strings.ContainsRune(ambiguousChars, r) {
					return -1
				}
				return r
			}, chars)
		}
		sets = append(sets, chars)
	}
	if len(sets) == 0 {
		return Result{}, ErrNoSets
	}
	alphabet := strings.Join(sets, "")

	// Повторяем генерацию, пока в пароле не окажется символа из каждого набора.
	// При длине от 8 это почти всегда первая же попытка; отбор сохраняет равномерность
	// среди подходящих паролей, в отличие от подстановки символов на случайные места.
	for {
		password, err := randomString(alphabet, opts.Length)
		if err != nil {
			return Result{}, err
		}
		if containsEach(password, sets) {
			return Result{
				Password:    password,
				EntropyBits: passwordEntropy(len(alphabet), opts.Length, sets),
			}, nil
		}
	}
}

// Passphrase генерирует парольную фразу из words случайных слов словаря EFF,
// разделённых separator.
func Passphrase(words int, separator string) (Result, error) {
	if words < MinWords || words > MaxWords {
		return Result{}, ErrWords
	}
	if n := utf8.RuneCountInString(separator); n < 1 || n > 3 || strings.IndexFunc(separator, unicode.IsControl) >= 0 {
		return Result{}, ErrSeparator
	}

	chosen := make([]string, words)
	for i := range chosen {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(wordlist))))
		if err != nil {
			return Result{}, fmt.Errorf("passgen: %w", err)
		}
		chosen[i] = wordlist[n.Int64()]
	}
	return Result{
		Password:    strings.Join(chosen, separator),
		EntropyBits: float64(words) * math.Log2(float64(len(wordlist))),
	}, nil
}

// randomString возвращает строку длины n из символов alphabet.
func randomString(alphabet string, n int) (string, error) {
	max := big.NewInt(int64(len(alphabet)))
	var b strings.Builder
	b.Grow(n)
	for range n {
		i, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("passgen: %w", err)
		}
		b.WriteByte(alphabet[i.Int64()])
	}
	return b.String(), nil
}

// containsEach сообщает, есть ли в s хотя бы один символ каждого набора.
func containsEach(s string, sets []string) bool {
	for _, set := range sets {
		if !strings.ContainsAny(s, set) {
			return false
		}
	}
	return true
}

// passwordEntropy — log2 числа паролей длины length из алфавита, где есть символ каждого
// набора. Считается по формуле включений-исключений; для одного набора равна
// length·log2(alphabet).
func passwordEntropy(alphabet, length int, sets []string) float64 {
	// Число строк, в которых есть символы всех наборов:
	// Σ по подмножествам S исключённых наборов (−1)^|S| · (alphabet − Σ|S|)^length.
	total := new(big.Int)
	k := len(sets)
	for mask := 0; mask < 1<<k; mask++ {
		size, sign := alphabet, 1
		for j := range k {
			if mask&(1<<j) != 0 {
				size -= len(sets[j])
				sign = -sign
			}
		}
		term := new(big.Int).Exp(big.NewInt(int64(size)), big.NewInt(int64(length)), nil)
		if sign < 0 {
			total.Sub(total, term)
		} else {
			total.Add(total, term)
		}
	}
	f, _ := new(big.Float).SetInt(total).Float64()
	if math.IsInf(f, 1) {
		// Для очень длинных паролей поправка пренебрежимо мала.
		return float64(length) * math.Log2(float64(alphabet))
	}
	return math.Log2(f)
}

// Strength описывает стойкость по числу бит.
func Strength(bits float64) string {
	switch {
	case bits >= 100:
		return "очень высокая"
	case bits >= 75:
		return "высокая"
	case bits >= 60:
		return "достаточная"
	default:
		return "низкая"
	}
}
