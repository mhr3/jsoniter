package jsoniter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func parseJson(iter *Iterator) {
	switch iter.WhatIsNext() {
	case StringValue:
		iter.ReadString()
	case NumberValue:
		iter.ReadFloat64()
	case NilValue:
		iter.ReadNil()
	case BoolValue:
		iter.ReadBool()
	case ArrayValue:
		for iter.ReadArray() {
			parseJson(iter)
		}
	case ObjectValue:
		for rs := iter.ReadObjectRaw(); !rs.IsNil(); rs = iter.ReadObjectRaw() {
			parseJson(iter)
		}
	case InvalidValue:
		return
	default:
		panic("error parsing json")
	}
}

func stdFromHexChar(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}

	return 0, false
}

func TestU4(t *testing.T) {
	for i := 0; i <= 255; i++ {
		b := byte(i)
		v, ok := fromHexChar(b)
		w, okk := stdFromHexChar(b)

		require.Equal(t, ok, okk)
		require.Equal(t, v, w)
	}
}

func BenchmarkSkipLiterals(b *testing.B) {
	cases := []struct {
		name string
		json string
	}{
		{"null", `null`},
		{"true", `true`},
		{"false", `false`},
		{"obj with null", `{"a": null}`},
		{"obj with true", `{"a": true}`},
		{"obj with false", `{"a": false}`},
		{"array with null", `[null]`},
		{"array with true", `[true]`},
		{"array with false", `[false]`},
		{"mixed array", `[null, true, false]`},
		{"more mixed array", `[null, true, false, null, true, false]`},
	}

	for _, c := range cases {
		payload := []byte(c.json)
		b.Run(c.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				iter := ParseBytes(ConfigDefault, payload)
				parseJson(iter)
			}
		})
	}
}
