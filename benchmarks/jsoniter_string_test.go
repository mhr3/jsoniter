package test

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func TestReadLongString(t *testing.T) {
	const strSize = 10240

	chars := make([]byte, strSize)
	for i := range chars {
		// ascii 48-122 ('0'-'z')
		chars[i] = byte(rand.Intn(75)) + 48
	}
	testString1 := string(chars)
	testString1Quoted := strconv.Quote(testString1)

	chars = make([]byte, strSize*3/4)
	_, err := rand.Read(chars)
	if err != nil {
		t.Fail()
	}
	testString2 := base64.StdEncoding.EncodeToString(chars)
	testString2Quoted := strconv.Quote(testString2)

	testCases := []struct {
		Name       string
		BufSize    int
		TestString string
		Quoted     string
	}{
		{
			Name:       "simple",
			BufSize:    1,
			TestString: testString2,
			Quoted:     testString2Quoted,
		},
		{
			Name:       "simple",
			BufSize:    512,
			TestString: testString2,
			Quoted:     testString2Quoted,
		},
		{
			Name:       "simple",
			BufSize:    4096,
			TestString: testString2,
			Quoted:     testString2Quoted,
		},
		{
			Name:       "simple",
			BufSize:    65536,
			TestString: testString2,
			Quoted:     testString2Quoted,
		},
		{
			Name:       "escaped-chars",
			BufSize:    1,
			TestString: testString1,
			Quoted:     testString1Quoted,
		},
		{
			Name:       "escaped-chars",
			BufSize:    512,
			TestString: testString1,
			Quoted:     testString1Quoted,
		},
		{
			Name:       "escaped-chars",
			BufSize:    4096,
			TestString: testString1,
			Quoted:     testString1Quoted,
		},
		{
			Name:       "escaped-chars",
			BufSize:    65536,
			TestString: testString1,
			Quoted:     testString1Quoted,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-buf-%d", tc.Name, tc.BufSize), func(t *testing.T) {
			iter := jsoniter.Parse(jsoniter.ConfigDefault, strings.NewReader(tc.Quoted), tc.BufSize)
			str := iter.ReadString()
			assert.Equal(t, tc.TestString, str)
		})
	}
}

func BenchmarkU4Decode(b *testing.B) {
	const manyU4s = `"\u30CB\u30B3\u52D5\u3067\u8E0A\u308A\u624B\u3084\u3063\u3066\u307E\u3059!!\u5FDC\u63F4\u672C\u5F53\u306B\u5B09\u3057\u3044\u3067\u3059\u3042\u308A\u304C\u3068\u3046\u3054\u3056\u3044\u307E\u3059!!\u3000\u307D\u3063\u3061\u3083\u308A\u3060\u3051\u3069\u524D\u5411\u304D\u306B\u9811\u5F35\u308B\u8150\u5973\u5B50\u3067\u3059\u3002\u5D50\u3068\u5F31\u866B\u30DA\u30C0\u30EB\u304C\u5927\u597D\u304D\uFF01\u3010\u304A\u8FD4\u4E8B\u3011\u308A\u3077(\u57FA\u672C\u306F)\u201D\u25CB\u201D\u3000DM (\u540C\u696D\u8005\u69D8\u3092\u9664\u3044\u3066)\u201D\u00D7\u201D\u3000\u52D5\u753B\u306E\u8EE2\u8F09\u306F\u7D76\u5BFE\u306B\u3084\u3081\u3066\u304F\u3060\u3055\u3044\u3002 \u30D6\u30ED\u30B0\u2192http://t.co/8E91tqoeKX\u3000\u3000"`
	data := []byte(manyU4s)

	iter := jsoniter.ParseBytes(jsoniter.ConfigDefault, nil)

	for i := 0; i < b.N; i++ {
		iter.Error = nil
		iter.ResetBytes(data)

		rs := iter.ReadString()
		_ = rs
		//_ = rs.String()
	}
}

func BenchmarkReadLongString(b *testing.B) {
	const strSize = 10240
	const bufSize = 4096

	b.Run("simple", func(b *testing.B) {
		chars := make([]byte, strSize*3/4)
		_, err := rand.Read(chars)
		if err != nil {
			b.Fail()
		}
		testString := strconv.Quote(base64.StdEncoding.EncodeToString(chars))
		rdr := strings.NewReader(testString)
		iter := jsoniter.Parse(jsoniter.ConfigDefault, rdr, bufSize)

		for i := 0; i < b.N; i++ {
			rdr.Reset(testString)
			iter.Reset(rdr)
			value := iter.ReadString()
			if value == "" {
				b.Fail()
			}
		}
	})

	b.Run("simple-raw", func(b *testing.B) {
		chars := make([]byte, strSize*3/4)
		_, err := rand.Read(chars)
		if err != nil {
			b.Fail()
		}
		testString := strconv.Quote(base64.StdEncoding.EncodeToString(chars))
		rdr := strings.NewReader(testString)
		iter := jsoniter.Parse(jsoniter.ConfigDefault, rdr, bufSize)

		for i := 0; i < b.N; i++ {
			rdr.Reset(testString)
			iter.Reset(rdr)
			value := iter.ReadRawString()
			if value.IsNil() {
				b.Fail()
			}
		}
	})

	b.Run("escaped", func(b *testing.B) {
		chars := make([]byte, strSize)
		for i := range chars {
			// ascii 32-122 (' '-'z')
			chars[i] = byte(rand.Intn(91)) + 32
		}
		testString := strconv.Quote(string(chars))
		rdr := strings.NewReader(testString)
		iter := jsoniter.Parse(jsoniter.ConfigDefault, rdr, bufSize)

		for i := 0; i < b.N; i++ {
			rdr.Reset(testString)
			iter.Reset(rdr)
			value := iter.ReadString()
			if value == "" {
				b.Fail()
			}
		}
	})

	b.Run("escaped-raw", func(b *testing.B) {
		chars := make([]byte, strSize)
		for i := range chars {
			// ascii 32-122 (' '-'z')
			chars[i] = byte(rand.Intn(91)) + 32
		}
		testString := strconv.Quote(string(chars))
		rdr := strings.NewReader(testString)
		iter := jsoniter.Parse(jsoniter.ConfigDefault, rdr, bufSize)

		for i := 0; i < b.N; i++ {
			rdr.Reset(testString)
			iter.Reset(rdr)
			value := iter.ReadRawString()
			if value.IsNil() {
				b.Fail()
			}
		}
	})
}
