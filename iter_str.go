package jsoniter

import (
	"bytes"
	"strconv"
	"strings"
	"unicode/utf16"
)

// ReadString read string from iterator
func (iter *Iterator) ReadString() string {
	c := iter.nextToken()
	switch c {
	case '"':
	case 'n':
		iter.ensureLiteral(nullLiteral)
		return ""
	default:
		iter.ReportError("ReadString", `expects " or n, but found `+string([]byte{c}))
		return ""
	}

	return iter.readStringInner()
}

// ReadStringAsSlice read string from iterator without copying into string form.
// The []byte can not be kept, as it will change after next iterator call.
// DEPRECATED: Use ReadRawString instead
func (iter *Iterator) ReadStringAsSlice() (ret []byte) {
	return iter.ReadRawString().buf
}

func (iter *Iterator) readStringInner() string {
	sb := strings.Builder{}

outerLoop:
	for iter.Error == nil {
		// eliminate bounds check inside the loop
		if iter.head < 0 || iter.tail > len(iter.buf) {
			return ""
		}
		for i := iter.head; i < iter.tail; i++ {
			c := iter.buf[i]
			switch {
			case c == '"':
				if sb.Len() == 0 {
					// super fast path
					res := iter.buf[iter.head:i]
					iter.head = i + 1
					return string(res)
				}
				sb.Write(iter.buf[iter.head:i])
				iter.head = i + 1
				return sb.String()
			case c == '\\':
				sb.Write(iter.buf[iter.head:i])
				iter.head = i + 1
				iter.readEscapedChar(&sb)
				continue outerLoop
			case c < ' ':
				iter.ReportError("ReadString",
					"invalid control character found: "+strconv.Itoa(int(c)))
				return ""
			}
		}

		// copy buffer and load more
		sb.Write(iter.buf[iter.head:iter.tail])
		iter.head = iter.tail

		// load next chunk
		if !iter.loadMore() {
			break
		}
	}

	iter.ReportError("ReadString", "unexpected end of input")
	return ""
}

// ReadRawString reads string from iterator without decoding escape sequences.
// Note that the returned RawString is only valid until the next read from the iterator.
func (iter *Iterator) ReadRawString() RawString {
	c := iter.nextToken()
	switch c {
	case '"':
	case 'n':
		iter.ensureLiteral(nullLiteral)
		return RawString{}
	default:
		iter.ReportError("ReadRawString", `expects " or n, but found `+string([]byte{c}))
		return RawString{}
	}

	return iter.readRawStringInner()
}

func (iter *Iterator) readRawStringInner() RawString {
	var (
		copied        bytes.Buffer
		readingEscape bool
		hasEscapes    bool
	)

	copyStart := iter.head

outerLoop:
	for iter.Error == nil {
		// eliminate bounds check inside the loop
		if iter.head < 0 || iter.tail > len(iter.buf) {
			return RawString{}
		}
		for i := iter.head; i < iter.tail; i++ {
			c := iter.buf[i]
			switch {
			case c == '"':
				if readingEscape {
					readingEscape = false
					continue
				}
				// careful, we're copying the ending double quote into the buffer
				if copied.Len() == 0 {
					// super fast path
					iter.head = i + 1
					return RawString{buf: iter.buf[copyStart:iter.head], isRaw: true, hasEscapes: hasEscapes}
				}
				iter.head = i + 1
				copied.Write(iter.buf[copyStart:iter.head])
				return RawString{buf: copied.Bytes(), hasEscapes: hasEscapes}
			case c == '\\':
				// toggle readingEscape
				readingEscape = !readingEscape
				hasEscapes = true
				continue
			case c < ' ':
				iter.ReportError("ReadRawString",
					"invalid control character found: "+strconv.Itoa(int(c)))
				return RawString{}
			default:
				if !readingEscape {
					continue
				}
			}

			readingEscape = false
			switch c {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			case 'u':
				iter.head = i + 1
				// are we about to change iter.buf?
				if i+4 >= iter.tail {
					copied.Write(iter.buf[copyStart:iter.head])
					buf := iter.readU4Buf()
					copied.Write(buf[:])
					copyStart = iter.head
					continue outerLoop
				}

				u4 := u4bufFromBytes(iter.buf[i+1 : i+5])
				if u4.Parse() < 0 {
					iter.ReportError("ReadRawString", "invalid unicode escape sequence")
					return RawString{}
				}
				iter.head += 4
				// it shouldn't be necessary to break out of the loop, but
				// for some reason the compiler doesn't like this branch
				// and inserts slice bounds check around the iter.buf[i] read
				// without the "continue outerLoop"
				continue outerLoop
			default:
				iter.ReportError("ReadRawString", `invalid escape char after \`)
				return RawString{}
			}
		}

		// copy buffer and load more
		copied.Write(iter.buf[copyStart:iter.tail])
		iter.head = iter.tail

		// load next chunk
		if !iter.loadMore() {
			break
		}
		copyStart = iter.head
	}

	iter.ReportError("ReadRawString", "unexpected end of input")
	return RawString{}
}

func (iter *Iterator) readEscapedChar(sb *strings.Builder) {
	c := iter.readByte()

start:
	switch c {
	case 'u':
		r := iter.readU4()
		if utf16.IsSurrogate(r) {
			c = iter.readByte()
			if iter.Error != nil {
				return
			}
			if c != '\\' {
				iter.unreadByte()
				sb.WriteRune(r)
				return
			}
			c = iter.readByte()
			if iter.Error != nil {
				return
			}
			if c != 'u' {
				sb.WriteRune(r)
				goto start
			}
			r2 := iter.readU4()
			if iter.Error != nil {
				return
			}
			combined := utf16.DecodeRune(r, r2)
			if combined == '\uFFFD' {
				sb.WriteRune(r)
				sb.WriteRune(r2)
			} else {
				sb.WriteRune(combined)
			}
		} else if iter.Error == nil {
			sb.WriteRune(r)
		}
	case '"':
		sb.WriteByte('"')
	case '\\':
		sb.WriteByte('\\')
	case '/':
		sb.WriteByte('/')
	case 'b':
		sb.WriteByte('\b')
	case 'f':
		sb.WriteByte('\f')
	case 'n':
		sb.WriteByte('\n')
	case 'r':
		sb.WriteByte('\r')
	case 't':
		sb.WriteByte('\t')
	default:
		iter.ReportError("readEscapedChar", `invalid escape char after \`)
	}
}

func fromHexChar(c byte) (byte, bool) {
	c -= '0'
	if c <= 9 {
		return c, true
	}
	c -= 'A' - '0'
	if c <= 5 {
		return c + 10, true
	}
	c -= 'a' - 'A'
	if c <= 5 {
		return c + 10, true
	}

	return 0, false
}

type u4buf [4]byte

func u4bufFromBytes(data []byte) u4buf {
	var u4 u4buf
	copy(u4[:], data[0:4])
	return u4
}

func (buf u4buf) Parse() rune {
	allOk := true

	a, ok := fromHexChar(buf[0])
	allOk = allOk && ok

	b, ok := fromHexChar(buf[1])
	allOk = allOk && ok
	ret := rune(a<<4 | b)

	a, ok = fromHexChar(buf[2])
	allOk = allOk && ok

	b, ok = fromHexChar(buf[3])
	allOk = allOk && ok

	if !allOk {
		return -1
	}

	return ret<<8 | rune(a<<4|b)
}

func (iter *Iterator) readU4() rune {
	var u4 u4buf

	startIdx := iter.head
	end := startIdx + 4

	if startIdx < 0 || end > len(iter.buf) || end < startIdx {
		u4 = iter.readU4Buf()
	} else {
		copy(u4[:], iter.buf[startIdx:end])

		iter.head += 4
	}

	ret := u4.Parse()
	if ret < 0 {
		iter.ReportError("readU4", "invalid hex char")
		return 0
	}

	return ret
}

func (iter *Iterator) readU4Buf() (buf u4buf) {
	for i := 0; i < 4; i++ {
		c := iter.readByte()
		if iter.Error != nil {
			return
		}
		buf[i] = c
		c -= '0'
		if c <= 9 {
			continue
		}
		c -= 'A' - '0'
		if c <= 5 {
			continue
		}
		c -= 'a' - 'A'
		if c <= 5 {
			continue
		}

		iter.ReportError("readU4", "invalid hex char")
		return
	}

	return buf
}
