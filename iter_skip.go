package jsoniter

import (
	"fmt"
	"io"
)

// ReadNil reads a json object as nil and
// returns whether it's a nil or not
func (iter *Iterator) ReadNil() (ret bool) {
	c := iter.nextToken()
	if c == 'n' {
		iter.ensureLiteral(nullLiteral)
		return iter.Error == nil
	}
	iter.unreadByte()
	return false
}

// ReadBool reads a json object as BoolValue
func (iter *Iterator) ReadBool() (ret bool) {
	c := iter.nextToken()
	if c == 't' {
		iter.ensureLiteral(trueLiteral)
		return true
	}
	if c == 'f' {
		iter.ensureLiteral(falseLiteral)
		return false
	}
	iter.ReportError("ReadBool", "expect t or f, but found "+string([]byte{c}))
	return
}

// SkipAndReturnBytes skip next JSON element, and return its content as []byte.
// The []byte can be kept, it is a copy of data.
func (iter *Iterator) SkipAndReturnBytes() []byte {
	iter.startCapture(iter.head)

	iter.Skip()
	if iter.Error != nil && iter.Error != io.EOF {
		iter.discardCapture()
		return nil
	}

	return iter.stopCapture()
}

// SkipAndAppendBytes skips next JSON element and appends its content to
// buffer, returning the result.
func (iter *Iterator) SkipAndAppendBytes(buf []byte) []byte {
	iter.startCaptureTo(buf, iter.head)
	iter.Skip()
	return iter.stopCapture()
}

func (iter *Iterator) startCaptureTo(buf []byte, captureStartedAt int) {
	if iter.captured != nil {
		panic("already in capture mode")
	}
	iter.captureStartedAt = captureStartedAt
	iter.captured = buf
}

func (iter *Iterator) startCapture(captureStartedAt int) {
	iter.startCaptureTo(make([]byte, 0, 32), captureStartedAt)
}

func (iter *Iterator) stopCapture() []byte {
	if iter.captured == nil {
		panic("not in capture mode")
	}
	captured := iter.captured
	remaining := iter.buf[iter.captureStartedAt:iter.head]
	iter.captureStartedAt = -1
	iter.captured = nil
	return append(captured, remaining...)
}

func (iter *Iterator) discardCapture() {
	iter.captured = nil
}

// Skip skips a json object and positions to relatively the next json object
func (iter *Iterator) Skip() {
	c := iter.nextToken()
	switch c {
	case '"':
		iter.skipString()
	case 'n':
		iter.ensureLiteral(nullLiteral)
	case 't':
		iter.ensureLiteral(trueLiteral)
	case 'f':
		iter.ensureLiteral(falseLiteral)
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		iter.unreadByte()
		iter.skipNumber()
	case '[':
		iter.skipArray()
	case '{':
		iter.skipObject()
	default:
		iter.ReportError("Skip", fmt.Sprintf("do not know how to skip: %v", c))
		return
	}
}
