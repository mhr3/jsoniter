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

func FuzzDecodeStrings(f *testing.F) {
	f.Add([]byte(`"\u30CB\u30B3\u52D5\u3067\u8E0A\u308A\u624B\u3084\u3063\u3066\u307E\u3059!!\u5FDC\u63F4\u672C\u5F53\u306B\u5B09\u3057\u3044\u3067\u3059\u3042\u308A\u304C\u3068\u3046\u3054\u3056\u3044\u307E\u3059!!\u3000\u307D\u3063\u3061\u3083\u308A\u3060\u3051\u3069\u524D\u5411\u304D\u306B\u9811\u5F35\u308B\u8150\u5973\u5B50\u3067\u3059\u3002\u5D50\u3068\u5F31\u866B\u30DA\u30C0\u30EB\u304C\u5927\u597D\u304D\uFF01\u3010\u304A\u8FD4\u4E8B\u3011\u308A\u3077(\u57FA\u672C\u306F)\u201D\u25CB\u201D\u3000DM (\u540C\u696D\u8005\u69D8\u3092\u9664\u3044\u3066)\u201D\u00D7\u201D\u3000\u52D5\u753B\u306E\u8EE2\u8F09\u306F\u7D76\u5BFE\u306B\u3084\u3081\u3066\u304F\u3060\u3055\u3044\u3002 \u30D6\u30ED\u30B0\u2192http://t.co/8E91tqoeKX\u3000\u3000"`))
	f.Add([]byte(`"foo\tbar\nbaz\u0020qux"`))

	f.Fuzz(func(t *testing.T, b []byte) {
		t.Run("decode", func(t *testing.T) {
			var (
				v interface{}
			)

			iterErr := jsoniter.Unmarshal(b, &v)
			if iterErr != nil {
				return
			}
			if _, isString := v.(string); !isString {
				return
			}

			iter := jsoniter.ParseBytes(jsoniter.ConfigDefault, b)
			streamStr := iter.ReadString()
			iter.WhatIsNext()
			iterStreamErr := iter.Error
			if iterStreamErr == io.EOF {
				iterStreamErr = nil
			} else if iterStreamErr != nil {
				t.Error(iterStreamErr)
			} else if iterStreamErr == nil {
				t.Error("expected EOF")
			}

			require.Equal(t, v, streamStr)

			iter = jsoniter.Parse(jsoniter.ConfigDefault, bytes.NewReader(b), 1)
			//iter.ResetBytes(b)
			rs := iter.ReadRawString()
			streamStr = rs.String()

			require.Equal(t, v, streamStr)
		})
	})
}
