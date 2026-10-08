package domain_test

import (
	"testing"

	"github.com/F3dosik/GophKeeper/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestKDFParams_Validate(t *testing.T) {
	assert.NoError(t, domain.DefaultKDFParams.Validate())
	assert.NoError(t, domain.LegacyKDFParams.Validate())

	invalid := map[string]domain.KDFParams{
		"zero":          {},
		"too weak mem":  {Time: 3, MemoryKiB: 1024, Threads: 4},
		"too much time": {Time: 1000, MemoryKiB: 64 * 1024, Threads: 4},
		"too much mem":  {Time: 3, MemoryKiB: 8 * 1024 * 1024, Threads: 4},
		"no threads":    {Time: 3, MemoryKiB: 64 * 1024, Threads: 0},
	}
	for name, p := range invalid {
		assert.ErrorIs(t, p.Validate(), domain.ErrInvalidArgument, name)
	}
}
