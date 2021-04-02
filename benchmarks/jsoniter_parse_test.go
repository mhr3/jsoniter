package test

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
)

func getTestDataReader(name string) (io.ReadCloser, io.Closer, error) {
	p := filepath.Join("testdata", name+".json.gz")

	f, err := os.Open(p)
	if err != nil {
		return nil, nil, err
	}

	gr, err := gzip.NewReader(f)
	return gr, f, err
}

func getTestDataBytes(name string) ([]byte, error) {
	rdr, cl, err := getTestDataReader(name)
	if err != nil {
		return nil, err
	}
	defer cl.Close()
	defer rdr.Close()

	return ioutil.ReadAll(rdr)
}

func parseJson(iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		iter.ReadString()
	case jsoniter.NumberValue:
		iter.ReadFloat64()
	case jsoniter.NilValue:
		iter.ReadNil()
	case jsoniter.BoolValue:
		iter.ReadBool()
	case jsoniter.ArrayValue:
		for iter.ReadArray() {
			parseJson(iter)
		}
	case jsoniter.ObjectValue:
		for iter.ReadObject() != "" {
			parseJson(iter)
		}
	case jsoniter.InvalidValue:
		return
	default:
		panic("error parsing json")
	}
}

func runBenchmark(b *testing.B, dataName string) {
	data, err := getTestDataBytes(dataName)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("bytes", func(b *testing.B) {
		startTime := time.Now()
		var bytesProcessed int64

		for i := 0; i < b.N; i++ {
			iter := jsoniter.ParseBytes(jsoniter.ConfigDefault, data)
			parseJson(iter)

			bytesProcessed += int64(len(data))
		}

		b.ReportMetric(float64(bytesProcessed)/time.Since(startTime).Seconds()/1024/1024, "MB/s")
	})

	/*
		b.Run("stream", func(b *testing.B) {
			startTime := time.Now()
			var bytesProcessed int64

			for i := 0; i < b.N; i++ {
				iter := jsoniter.Parse(jsoniter.ConfigDefault, bytes.NewReader(data), 4096)
				parseJson(iter)

				bytesProcessed += int64(len(data))
			}

			b.ReportMetric(float64(bytesProcessed)/time.Since(startTime).Seconds()/1024/1024, "MB/s")
		})
	*/
}

func BenchmarkDatasetCanada(b *testing.B)         { runBenchmark(b, "canada") }
func BenchmarkDatasetGithub(b *testing.B)         { runBenchmark(b, "github_events") }
func BenchmarkDatasetGsoc2018(b *testing.B)       { runBenchmark(b, "gsoc-2018") }
func BenchmarkDatasetTwitterEscaped(b *testing.B) { runBenchmark(b, "twitterescaped") }
