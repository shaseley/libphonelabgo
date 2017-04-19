package libphonelabgo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFrameDiffOffset(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	// Test MS
	tests := []struct {
		ts       int64
		offset   int64
		expected float64
	}{
		{123645302254, 0, 123645302.254},
		{123645302254, 70000000, 123715302.254},
	}

	for _, test := range tests {
		assert.Equal(test.expected, adjustTimestampMsToS(test.ts, test.offset))
	}
}
