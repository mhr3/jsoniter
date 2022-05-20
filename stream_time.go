package jsoniter

import (
	"errors"
	"time"
)

var errCannotMarshalTime = errors.New("stream.WriteTime: year outside of range [0,9999]")

// WriteTime writes a time.Time to stream
func (stream *Stream) WriteTime(val time.Time) {
	// this mimics Time.MarshalJSON()
	if y := val.Year(); y < 0 || y >= 10000 {
		// RFC 3339 is clear that years are 4 digits exactly.
		// See golang.org/issue/4556#c15 for more discussion.
		stream.Error = errCannotMarshalTime
		return
	}

	var buf [len(time.RFC3339Nano) + 2]byte

	b := buf[:0]
	b = append(b, '"')
	b = val.AppendFormat(b, time.RFC3339Nano)
	b = append(b, '"')

	stream.buf = append(stream.buf, b...)
}
