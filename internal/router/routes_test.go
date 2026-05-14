package router

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseTrustedSubnet проверяет, что parseTrustedSubnet работает корректно
func TestParseTrustedSubnet(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		n, err := parseTrustedSubnet("")
		require.NoError(t, err)
		assert.Nil(t, n)
		n, err = parseTrustedSubnet("   ")
		require.NoError(t, err)
		assert.Nil(t, n)
	})

	t.Run("valid", func(t *testing.T) {
		n, err := parseTrustedSubnet("192.168.0.0/16")
		require.NoError(t, err)
		require.NotNil(t, n)
		assert.True(t, n.Contains(net.ParseIP("192.168.1.1")))
		assert.False(t, n.Contains(net.ParseIP("10.0.0.1")))
	})

	t.Run("invalid", func(t *testing.T) {
		n, err := parseTrustedSubnet("not-a-cidr")
		assert.Error(t, err)
		assert.Nil(t, n)
	})
}
