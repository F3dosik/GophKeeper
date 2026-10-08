package command

import (
	"testing"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKDFFlags(t *testing.T) {
	parse := func(args ...string) (domain.KDFParams, error) {
		var f kdfFlags
		cmd := &cobra.Command{}
		f.register(cmd)
		require.NoError(t, cmd.ParseFlags(args))
		return f.params()
	}

	t.Run("defaults", func(t *testing.T) {
		params, err := parse()
		require.NoError(t, err)
		assert.Equal(t, domain.DefaultKDFParams, params)
	})

	t.Run("custom", func(t *testing.T) {
		params, err := parse("--kdf-time=5", "--kdf-memory=128")
		require.NoError(t, err)
		assert.Equal(t, domain.KDFParams{Time: 5, MemoryKiB: 128 * 1024, Threads: domain.DefaultKDFParams.Threads}, params)
	})

	for name, args := range map[string][]string{
		"time too high":   {"--kdf-time=11"},
		"time zero":       {"--kdf-time=0"},
		"memory too low":  {"--kdf-memory=8"},
		"memory too high": {"--kdf-memory=2048"},
		"memory overflow": {"--kdf-memory=4294967295"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parse(args...)
			assert.ErrorIs(t, err, domain.ErrInvalidArgument)
		})
	}
}
