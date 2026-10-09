package passgen

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Хеш словаря EFF (по одному слову в строке): изменение или порча словаря изменят
// стойкость фраз, поэтому это должно быть заметно.
const wordlistSHA256 = "6d557f0693958fb5e650b68b5bee585eb82cf4da32965505c789e924743bc522"

func TestWordlist(t *testing.T) {
	sum := sha256.Sum256([]byte(wordlistData))
	assert.Equal(t, wordlistSHA256, hex.EncodeToString(sum[:]))
	assert.Len(t, wordlist, 7776)

	seen := make(map[string]bool, len(wordlist))
	for _, w := range wordlist {
		assert.False(t, seen[w], "duplicate %q", w)
		seen[w] = true
	}
}

func TestPassword(t *testing.T) {
	t.Run("default contains every set", func(t *testing.T) {
		for range 200 {
			r, err := Password(DefaultOptions())
			require.NoError(t, err)
			assert.Len(t, r.Password, DefaultLength)
			assert.True(t, strings.ContainsAny(r.Password, lowerChars))
			assert.True(t, strings.ContainsAny(r.Password, upperChars))
			assert.True(t, strings.ContainsAny(r.Password, digitChars))
			assert.True(t, strings.ContainsAny(r.Password, symbolChars))
		}
	})

	t.Run("only selected sets", func(t *testing.T) {
		r, err := Password(Options{Length: 64, Digits: true})
		require.NoError(t, err)
		assert.Regexp(t, `^[0-9]{64}$`, r.Password)
	})

	t.Run("no ambiguous characters", func(t *testing.T) {
		opts := DefaultOptions()
		opts.NoAmbiguous = true
		opts.Length = MaxLength
		for range 50 {
			r, err := Password(opts)
			require.NoError(t, err)
			assert.False(t, strings.ContainsAny(r.Password, ambiguousChars), r.Password)
		}
	})

	t.Run("passwords differ", func(t *testing.T) {
		seen := map[string]bool{}
		for range 1000 {
			r, err := Password(DefaultOptions())
			require.NoError(t, err)
			assert.False(t, seen[r.Password])
			seen[r.Password] = true
		}
	})

	t.Run("invalid options", func(t *testing.T) {
		_, err := Password(Options{Length: MinLength - 1, Lower: true})
		assert.ErrorIs(t, err, ErrLength)
		_, err = Password(Options{Length: MaxLength + 1, Lower: true})
		assert.ErrorIs(t, err, ErrLength)
		_, err = Password(Options{Length: 20})
		assert.ErrorIs(t, err, ErrNoSets)
	})
}

// Символы должны встречаться равновероятно: при выборе по остатку от деления
// часть алфавита выпадала бы чаще. Проверка — критерий хи-квадрат.
func TestPassword_Uniform(t *testing.T) {
	opts := Options{Length: MaxLength, Lower: true, Upper: true, Digits: true, Symbols: true}
	alphabet := lowerChars + upperChars + digitChars + symbolChars
	counts := make(map[rune]int)
	total := 0
	for range 400 {
		r, err := Password(opts)
		require.NoError(t, err)
		for _, c := range r.Password {
			counts[c]++
			total++
		}
	}
	expected := float64(total) / float64(len(alphabet))
	chi2 := 0.0
	for _, c := range alphabet {
		d := float64(counts[c]) - expected
		chi2 += d * d / expected
	}
	// 86 степеней свободы: критическое значение для p = 0.001 около 132.
	assert.Less(t, chi2, 140.0, "characters are not uniformly distributed (chi2 = %.1f)", chi2)
}

func TestPasswordEntropy(t *testing.T) {
	// Один набор: ровно length·log2(alphabet).
	assert.InDelta(t, 20*math.Log2(10), passwordEntropy(10, 20, []string{digitChars}), 1e-9)

	// Требование «символ каждого набора» немного уменьшает число паролей.
	r, err := Password(DefaultOptions())
	require.NoError(t, err)
	full := 20 * math.Log2(float64(len(lowerChars+upperChars+digitChars+symbolChars)))
	assert.Less(t, r.EntropyBits, full)
	assert.Greater(t, r.EntropyBits, full-1)

	// Очень длинный пароль не переполняет вычисление.
	r, err = Password(Options{Length: MaxLength, Lower: true, Upper: true, Digits: true, Symbols: true})
	require.NoError(t, err)
	assert.False(t, math.IsInf(r.EntropyBits, 0) || math.IsNaN(r.EntropyBits))
	assert.Greater(t, r.EntropyBits, 800.0)
}

func TestPassphrase(t *testing.T) {
	inList := make(map[string]bool, len(wordlist))
	for _, w := range wordlist {
		inList[w] = true
	}

	r, err := Passphrase(DefaultWords, " ")
	require.NoError(t, err)
	words := strings.Split(r.Password, " ")
	require.Len(t, words, DefaultWords)
	for _, w := range words {
		assert.True(t, inList[w], "%q is not from the EFF list", w)
	}
	assert.InDelta(t, 6*math.Log2(7776), r.EntropyBits, 1e-9)
	assert.InDelta(t, 77.5, r.EntropyBits, 0.1)

	_, err = Passphrase(MinWords-1, "-")
	assert.ErrorIs(t, err, ErrWords)
	_, err = Passphrase(MaxWords+1, "-")
	assert.ErrorIs(t, err, ErrWords)
	for _, sep := range []string{"", "\n", "\t", "----"} {
		_, err = Passphrase(6, sep)
		assert.ErrorIs(t, err, ErrSeparator, "%q", sep)
	}
	_, err = Passphrase(6, "-")
	assert.NoError(t, err)
}

func TestStrength(t *testing.T) {
	assert.Equal(t, "низкая", Strength(40))
	assert.Equal(t, "достаточная", Strength(64))
	assert.Equal(t, "высокая", Strength(77.5))
	assert.Equal(t, "очень высокая", Strength(129))
}
