package test

import (
	"bytes"
	"encoding/json"
	"testing"

	jsoniter "github.com/json-iterator/go"
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
		})
	})
}
