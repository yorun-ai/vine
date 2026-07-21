package embedded

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseScanOptionRejectsInvalidCursor(t *testing.T) {
	_, err := parseScanOption([][]byte{[]byte("SCAN"), []byte("bad")})
	assert.EqualError(t, err, "ERR invalid scan cursor")
}

func TestParseScanOptionUsesDefaultCount(t *testing.T) {
	option, err := parseScanOption([][]byte{[]byte("SCAN"), []byte("0")})

	require.NoError(t, err)
	assert.Equal(t, scanDefaultCount, option.count)
}

func TestParseScanOptionAcceptsLargeCount(t *testing.T) {
	option, err := parseScanOption([][]byte{[]byte("SCAN"), []byte("0"), []byte("COUNT"), []byte("1001")})

	require.NoError(t, err)
	assert.Equal(t, 1001, option.count)
}

func TestParseScanOptionParsesCursor(t *testing.T) {
	option, err := parseScanOption([][]byte{[]byte("SCAN"), []byte(strconv.FormatUint(42, 10)), []byte("COUNT"), []byte("10")})

	require.NoError(t, err)
	assert.Equal(t, uint64(42), option.cursor)
	assert.Equal(t, 10, option.count)
}
