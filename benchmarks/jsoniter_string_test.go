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

func BenchmarkReadLongString(b *testing.B) {
	const strSize = 10240

	b.Run("simple", func(b *testing.B) {
		chars := make([]byte, strSize*3/4)
		_, err := rand.Read(chars)
		if err != nil {
			b.Fail()
		}
		testString := strconv.Quote(base64.StdEncoding.EncodeToString(chars))
		rdr := strings.NewReader(testString)
		iter := jsoniter.Parse(jsoniter.ConfigDefault, nil, 4096)

		for i := 0; i < b.N; i++ {
			rdr.Reset(testString)
			iter.Reset(rdr)
			value := iter.ReadString()
			if value == "" {
				b.Fail()
			}
		}
	})

	b.Run("escaped", func(b *testing.B) {
		chars := make([]byte, strSize)
		for i := range chars {
			// ascii 48-122 ('0'-'z')
			chars[i] = byte(rand.Intn(75)) + 48
		}
		testString := strconv.Quote(string(chars))
		rdr := strings.NewReader(testString)
		iter := jsoniter.Parse(jsoniter.ConfigDefault, rdr, 4096)

		for i := 0; i < b.N; i++ {
			rdr.Reset(testString)
			iter.Reset(rdr)
			value := iter.ReadString()
			if value == "" {
				b.Fail()
			}
		}
	})
}
