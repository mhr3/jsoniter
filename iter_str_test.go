package jsoniter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringParseU4(t *testing.T) {
	testCases := []struct {
		input     string
		expect    string
		expectErr bool
	}{
		{
			input:  `"\u0020"`,
			expect: " ",
		},
		{
			input:  `"\u4e2d"`,
			expect: "中",
		},
		{
			input:  `"\u4E2D"`,
			expect: "中",
		},
		{
			input:  `"\u6587"`,
			expect: "文",
		},
		{
			input:     `"\u658U"`,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		iter := ParseString(ConfigDefault, tc.input)
		actual := iter.ReadString()
		if tc.expectErr {
			require.Error(t, iter.Error)
			continue
		}
		require.Equal(t, tc.expect, actual)
	}
}
