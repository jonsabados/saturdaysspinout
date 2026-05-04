package driver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInt64Slice(t *testing.T) {
	testCases := []struct {
		name            string
		values          []string
		expectedInts    []int64
		expectedInvalid []string
	}{
		{
			name:         "empty input",
			values:       []string{},
			expectedInts: nil,
		},
		{
			name:         "valid integers",
			values:       []string{"1", "2", "3"},
			expectedInts: []int64{1, 2, 3},
		},
		{
			name:            "mixed valid and invalid",
			values:          []string{"1", "abc", "3", "def"},
			expectedInts:    []int64{1, 3},
			expectedInvalid: []string{"abc", "def"},
		},
		{
			name:            "all invalid",
			values:          []string{"abc", "def"},
			expectedInvalid: []string{"abc", "def"},
		},
		{
			name:         "large int64 values",
			values:       []string{"9223372036854775807"},
			expectedInts: []int64{9223372036854775807},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ints, invalid := parseInt64Slice(tc.values)
			assert.Equal(t, tc.expectedInts, ints)
			assert.Equal(t, tc.expectedInvalid, invalid)
		})
	}
}