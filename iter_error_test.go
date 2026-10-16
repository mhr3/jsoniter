package jsoniter

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestReportErrorIsValidUTF8(t *testing.T) {
	iter := ParseString(ConfigDefault, "(\xb5/\xfd")
	iter.ReadObject()
	require.True(t, utf8.ValidString(iter.Error.Error()))
}
