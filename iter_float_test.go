package jsoniter

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadNumber(t *testing.T) {
	input := []byte(`{"num":1234567890}`)

	iter := ParseBytes(ConfigDefault, input)
	key := iter.ReadObject()
	require.Equal(t, "num", key)
	num := iter.ReadNumber()
	require.Equal(t, "1234567890", num.String())
	n, _ := num.Int64()
	require.EqualValues(t, 1234567890, n)
}

func TestParseNumber(t *testing.T) {
	testCases := map[string]interface{}{
		"-1":                   -1,
		"0":                    0,
		"400":                  uint(400),
		"1234567890":           1234567890,
		"0.0125":               float32(0.0125),
		"-64.5":                float32(-64.5),
		"-0.00625":             float64(-0.00625),
		"12.3e8":               float64(12.3e8),
		"18446744073709551616": float64(math.MaxUint64 + 1),
		"-9223372036854775808": int64(math.MinInt64),
		"9223372036854775807":  int64(math.MaxInt64),
		"18446744073709551615": uint64(math.MaxUint64),
	}

	iter := NewIterator(ConfigDefault)

	for input, expected := range testCases {
		iter.ResetBytes([]byte(input))
		iter.Error = nil

		switch val := expected.(type) {
		case int:
			require.Equal(t, val, iter.ReadInt())
		case int64:
			require.Equal(t, val, iter.ReadInt64())
		case uint:
			require.Equal(t, val, iter.ReadUint())
		case uint64:
			require.Equal(t, val, iter.ReadUint64())
		case float64:
			require.Equal(t, val, iter.ReadFloat64())
		case float32:
			require.Equal(t, val, iter.ReadFloat32())
		}
	}
}

func BenchmarkAllocs(b *testing.B) {
	type tcPair struct {
		Value interface{}
		Slice []byte
	}
	testCases := map[string]*tcPair{
		"0.0125":               {Value: float32(0.0125)},
		"-64.5":                {Value: float32(-64.5)},
		"-0.00625":             {Value: float64(-0.00625)},
		"12.3e8":               {Value: float64(12.3e8)},
		"18446744073709551616": {Value: float64(18446744073709551616)},
	}

	for k, v := range testCases {
		v.Slice = []byte(k)
	}

	iter := NewIterator(ConfigDefault)

	for i := 0; i < b.N; i++ {
		for _, expected := range testCases {
			iter.ResetBytes(expected.Slice)
			iter.Error = nil

			switch val := expected.Value.(type) {
			case int:
				if iter.ReadInt() != val {
					b.Fatal("mismatch @", val)
				}
			case int64:
				if iter.ReadInt64() != val {
					b.Fatal("mismatch @", val)
				}
			case uint:
				if iter.ReadUint() != val {
					b.Fatal("mismatch @", val)
				}
			case uint64:
				if iter.ReadUint64() != val {
					b.Fatal("mismatch @", val)
				}
			case float64:
				if iter.ReadFloat64() != val {
					b.Fatal("mismatch @", val)
				}
			case float32:
				if iter.ReadFloat32() != val {
					b.Fatal("mismatch @", val)
				}
			}
		}
	}
}

func BenchmarkNumberAllocs(b *testing.B) {
	type tcPair struct {
		Value interface{}
		Slice []byte
	}
	testCases := map[string]*tcPair{
		"0.0125":               {Value: float32(0.0125)},
		"-64.5":                {Value: float32(-64.5)},
		"-0.00625":             {Value: float64(-0.00625)},
		"12.3e8":               {Value: float64(12.3e8)},
		"18446744073709551616": {Value: float64(18446744073709551616)},
	}

	for k, v := range testCases {
		v.Slice = []byte(k)
	}

	iter := NewIterator(ConfigDefault)

	for i := 0; i < b.N; i++ {
		for _, expected := range testCases {
			iter.ResetBytes(expected.Slice)
			iter.Error = nil

			scratch := [24]byte{}

			switch val := expected.Value.(type) {
			case float64:
				buf := iter.ReadNumberAsSlice(scratch[:0])
				if res, _ := strconv.ParseFloat(string(buf), 64); res != val {
					b.Fatal("mismatch @", val, "!=", res)
				}
			case float32:
				buf := iter.ReadNumberAsSlice(scratch[:0])
				if res, _ := strconv.ParseFloat(string(buf), 64); float32(res) != val {
					b.Fatal("mismatch @", val, "!=", res)
				}
			}
		}
	}
}
