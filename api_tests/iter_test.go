package test

import (
	"io"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
)

func Test_iterator_offsets(t *testing.T) {
	json := `{ "foo": "bar", "num": 123 }
	{ "num" : 27 }
	{ "arr": [1.1,2.2,3.3], "obj": { "key": "val"}, "num": 68 }
	{ "foo.name": "quiz", "num": "321"}`

	should := require.New(t)
	startOffsets := []int64{}
	for i, r := range json {
		if r == '{' {
			startOffsets = append(startOffsets, int64(i))
		}
	}
	should.Len(startOffsets, 5)

	iter := jsoniter.Parse(jsoniter.ConfigDefault, strings.NewReader(json), 8)
	should.NotNil(iter)

	should.EqualValues(0, iter.InputOffset())
	should.Equal(startOffsets[0], iter.InputOffset())

	iter.ReadObjectCB(func(iter *jsoniter.Iterator, key string) bool {
		switch key {
		case "foo":
			should.Equal(jsoniter.StringValue, iter.WhatIsNext())
			should.EqualValues(9, iter.InputOffset())
		case "num":
			should.Equal(jsoniter.NumberValue, iter.WhatIsNext())
			should.EqualValues(23, iter.InputOffset())
		default:
			should.NotNil(nil, "unexpected key: %s", key)
		}

		// skip the value
		iter.Skip()

		return true
	})
	should.NoError(iter.Error)
	should.EqualValues(28, iter.InputOffset())
	should.EqualValues('}', json[iter.InputOffset()-1])
	// there's still some whitespace to get to the next object
	should.NotEqual(startOffsets[1], iter.InputOffset())

	// read second line

	should.Equal(jsoniter.ObjectValue, iter.WhatIsNext())
	should.Equal(startOffsets[1], iter.InputOffset())
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, key string) bool {
		switch key {
		case "num":
			should.Equal(jsoniter.NumberValue, iter.WhatIsNext())
			should.EqualValues(40, iter.InputOffset())
		default:
			should.NotNil(nil, "unexpected key: %s", key)
		}

		// skip the value
		iter.Skip()

		return true
	})
	should.NoError(iter.Error)

	// read third line
	should.Equal(jsoniter.ObjectValue, iter.WhatIsNext())
	should.Equal(startOffsets[2], iter.InputOffset())
	for _, ok := iter.ReadObject(); ok; _, ok = iter.ReadObject() {
		iter.Skip()
	}
	should.NoError(iter.Error)

	// read fourth line
	should.Equal(jsoniter.ObjectValue, iter.WhatIsNext())
	should.Equal(startOffsets[4], iter.InputOffset())
	for _, ok := iter.ReadObject(); ok; _, ok = iter.ReadObject() {
		iter.Skip()
	}
	should.NoError(iter.Error)

	should.Equal(jsoniter.InvalidValue, iter.WhatIsNext())
	should.EqualValues(len(json), iter.InputOffset())
	should.EqualError(iter.Error, io.EOF.Error())
}

func TestIterator_Buffered(t *testing.T) {
	should := require.New(t)

	t.Run("empty buffer", func(t *testing.T) {
		iter := jsoniter.ParseBytes(jsoniter.ConfigDefault, []byte{})
		buffered := iter.Buffered()
		should.NotNil(buffered)

		// Read from empty buffered reader should return 0 and EOF
		buf, err := io.ReadAll(buffered)
		should.NoError(err)
		should.Len(buf, 0)
	})

	t.Run("no data consumed", func(t *testing.T) {
		input := []byte(`{"key": "value"}`)
		iter := jsoniter.ParseBytes(jsoniter.ConfigDefault, input)
		should.Equal(jsoniter.ObjectValue, iter.WhatIsNext())
		buffered := iter.Buffered()
		should.NotNil(buffered)

		// All data should be available in buffered reader
		buf, err := io.ReadAll(buffered)
		should.NoError(err)
		should.Equal(string(input), string(buf))
	})

	t.Run("partial data consumed", func(t *testing.T) {
		input := []byte(`{"key": "value", "num": 123}`)
		iter := jsoniter.ParseBytes(jsoniter.ConfigDefault, input)

		// Consume first key-value pair
		iter.ReadObjectCB(func(iter *jsoniter.Iterator, key string) bool {
			iter.Skip()
			return false
		})
		should.NoError(iter.Error)

		// Get buffered reader after partial consumption
		buffered := iter.Buffered()
		should.NotNil(buffered)

		// Should contain remaining unread data
		buf, err := io.ReadAll(buffered)
		should.NoError(err)
		// After consuming first key-value pair, remaining should contain the rest
		should.Equal(`, "num": 123}`, string(buf))
	})

	t.Run("all data consumed", func(t *testing.T) {
		input := []byte(`{"key": "value"}`)
		iter := jsoniter.ParseBytes(jsoniter.ConfigDefault, input)

		// Consume all data
		iter.Skip()
		should.NoError(iter.Error)

		// Get buffered reader after all data consumed
		buffered := iter.Buffered()
		should.NotNil(buffered)

		// Should be empty
		buf, err := io.ReadAll(buffered)
		should.Len(buf, 0)
		should.NoError(err)
	})

	t.Run("iterator reset", func(t *testing.T) {
		input := []byte(`{"key": "value", "num": 123}`)
		iter := jsoniter.ParseBytes(jsoniter.ConfigDefault, input)

		// Consume first key-value pair
		iter.ReadObjectCB(func(iter *jsoniter.Iterator, key string) bool {
			iter.Skip()
			return false
		})
		should.NoError(iter.Error)

		// Get buffered reader after partial consumption
		buffered := iter.Buffered()
		should.NotNil(buffered)

		iter.ResetBytes([]byte(`null`))
		should.Equal(jsoniter.NilValue, iter.WhatIsNext())
		should.EqualValues(0, iter.InputOffset())

		buf, err := io.ReadAll(buffered)
		should.NoError(err)
		should.Equal(`, "num": 123}`, string(buf))
	})

	t.Run("with streaming reader", func(t *testing.T) {
		input := `{"key1": "value1", "key2": "value2", "key3": "value3"}`
		reader := strings.NewReader(input)
		iter := jsoniter.Parse(jsoniter.ConfigDefault, reader, 1024)

		// Consume first key-value pair
		iter.ReadObjectCB(func(iter *jsoniter.Iterator, key string) bool {
			iter.Skip()
			return false
		})
		should.NoError(iter.Error)

		// Get buffered reader
		buffered := iter.Buffered()
		should.NotNil(buffered)

		// Should contain remaining data
		buf, err := io.ReadAll(buffered)
		should.NoError(err)
		should.Equal(`, "key2": "value2", "key3": "value3"}`, string(buf))
	})

	t.Run("buffered reader after error", func(t *testing.T) {
		input := []byte(`{"key": "value"  123`) // Invalid JSON - missing closing brace
		iter := jsoniter.ParseBytes(jsoniter.ConfigDefault, input)

		// Try to read which should cause an error
		iter.ReadObjectCB(func(iter *jsoniter.Iterator, key string) bool {
			iter.Skip()
			return true
		})
		// Iterator should have an error due to malformed JSON
		should.Error(iter.Error)

		// Buffered reader should still work
		buffered := iter.Buffered()
		should.NotNil(buffered)

		// Should be able to read whatever is buffered
		buf, err := io.ReadAll(buffered)
		should.NoError(err)
		should.Equal("123", string(buf))
	})

	t.Run("consistency with standard library behavior", func(t *testing.T) {
		input := []byte(`{"key": "value", "num": 123}`)
		iter := jsoniter.ParseBytes(jsoniter.ConfigDefault, input)

		// Consume some data
		iter.ReadObjectCB(func(iter *jsoniter.Iterator, key string) bool {
			iter.Skip()
			return false
		})
		should.NoError(iter.Error)

		// Get buffered reader
		buffered1 := iter.Buffered()
		buffered2 := iter.Buffered()

		// Each call should return a new reader
		should.NotSame(buffered1, buffered2)

		// Both should have the same content
		buf1, err1 := io.ReadAll(buffered1)
		buf2, err2 := io.ReadAll(buffered2)

		should.NoError(err1)
		should.NoError(err2)
		should.Equal(string(buf1), string(buf2))
	})
}
