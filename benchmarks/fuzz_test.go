package test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"unicode"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
)

func FuzzUnmarshalJSON(f *testing.F) {
	f.Add([]byte(`{
"object": {
	"slice": [
		1,
		2.0,
		"3",
		[4],
		{5: {}}
	]
},
"slice": [[]],
"string": ":)",
"int": 1e5,
"float": 3e-9"
}`))
	f.Add([]byte(`"foo\tbar\nbaz\u0020qux"`))

	f.Fuzz(func(t *testing.T, b []byte) {
		t.Run("RawMessage", func(t *testing.T) {
			var (
				stdRawMsg  json.RawMessage
				iterRawMsg jsoniter.RawMessage
			)

			stdErr := json.Unmarshal(b, &stdRawMsg)
			iterErr := jsoniter.Unmarshal(b, &iterRawMsg)

			if stdErr != nil && iterErr == nil {
				t.Fatalf("std failed with %v, iter succeeded with %v", stdErr, iterErr)
			} else if stdErr == nil && iterErr != nil {
				t.Fatalf("std succeeded with %v, iter failed with %v", stdErr, iterErr)
			}

			if iterErr == nil && !bytes.Equal(stdRawMsg, iterRawMsg) {
				t.Fatalf("std raw message %s, iter raw message %s",
					string(stdRawMsg), string(iterRawMsg))
			}
		})

		t.Run("any", func(t *testing.T) {
			var (
				stdAny  interface{}
				iterAny interface{}
			)

			stdErr := json.Unmarshal(b, &stdAny)
			iterErr := jsoniter.Unmarshal(b, &iterAny)

			if stdErr != nil && iterErr == nil {
				t.Fatalf("std failed with %v, iter succeeded with %v", stdErr, iterErr)
			} else if stdErr == nil && iterErr != nil {
				t.Fatalf("std succeeded with %v, iter failed with %v", stdErr, iterErr)
			}

			if str, ok := stdAny.(string); ok && stdAny != iterAny {
				// jsoniter doesn't deal with invalid utf8 too well
				if !strings.ContainsRune(str, unicode.ReplacementChar) {
					require.Equal(t, str, iterAny)
				}
			}
		})

		t.Run("stream", func(t *testing.T) {
			var (
				v         interface{}
				streamAny interface{}
			)

			iterErr := jsoniter.Unmarshal(b, &v)

			iter := jsoniter.Parse(jsoniter.ConfigDefault, bytes.NewReader(b), 2)
			streamAny = iter.Read()
			iter.WhatIsNext()
			iterStreamErr := iter.Error
			if iterStreamErr == io.EOF {
				iterStreamErr = nil
			} else if iterStreamErr != nil {
				streamAny = nil
			} else if iterStreamErr == nil {
				iterStreamErr = errors.New("expected EOF")
			}

			if iterErr != nil && iterStreamErr == nil {
				require.Equal(t, iterErr, iterStreamErr)
			} else if iterErr == nil && iterStreamErr != nil {
				require.Equal(t, iterErr, iterStreamErr)
			}

			if iterErr == nil {
				require.Equal(t, v, streamAny)
			}
		})
	})
}
