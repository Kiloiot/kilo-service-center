package postgres

import (
	"database/sql"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const attachCounterTestEndpoint = int64(7)

// A stored attach counter reads back over its full 32-bit range; a value
// outside it, which the column check refuses, is an error, never a wrapped
// counter.
func TestAttachCounterReadsTheStoredCounter(t *testing.T) {
	counter, err := attachCounter(sql.NullInt64{}, attachCounterTestEndpoint)
	require.NoError(t, err)
	assert.Nil(t, counter, "no stored counter")

	counter, err = attachCounter(sql.NullInt64{Int64: math.MaxUint32, Valid: true}, attachCounterTestEndpoint)
	require.NoError(t, err)
	require.NotNil(t, counter)
	assert.Equal(t, uint32(math.MaxUint32), *counter)

	for _, stored := range []int64{-1, math.MaxUint32 + 1} {
		_, err = attachCounter(sql.NullInt64{Int64: stored, Valid: true}, attachCounterTestEndpoint)
		assert.Error(t, err, "attach_cnt %d is outside the counter's range", stored)
	}
}
