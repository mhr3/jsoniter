package jsoniter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvalidLiteral(t *testing.T) {
	var (
		res interface{}
		err error
	)

	err = UnmarshalFromString("nul", &res)
	require.Error(t, err)

	err = UnmarshalFromString("falze", &res)
	require.Error(t, err)

	err = UnmarshalFromString("truth", &res)
	require.Error(t, err)
}

func BenchmarkLiterals(b *testing.B) {
	const literals = `[true, false, null, true, true, false, false, null, null]`

	soManyLiterals := "[" + strings.Repeat(literals[1:len(literals)-1]+", ", 100) + literals[1:]

	data := []byte(soManyLiterals)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		iter := ParseBytes(ConfigDefault, data)
		for iter.ReadArray() {
			switch iter.WhatIsNext() {
			case BoolValue:
				iter.ReadBool()
			case NilValue:
				iter.ReadNil()
			default:
				panic("unexpected value")
			}
		}
	}
}
