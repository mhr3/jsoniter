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
		iter.skipThreeBytes('u', 'l', 'l')
		return ""
	default:
		iter.ReportError("ReadString", `expects " or n, but found `+string([]byte{c}))
		return ""
	}

	return iter.readStringInner()
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
		if iter.readByte() == 0 {
			break
		}
		iter.unreadByte()
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
		iter.skipThreeBytes('u', 'l', 'l')
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
					prevHead := iter.head
					iter.head = i + 1
					return RawString{buf: iter.buf[prevHead:iter.head], isRaw: true, hasEscapes: hasEscapes}
				}
				prevHead := iter.head
				iter.head = i + 1
				copied.Write(iter.buf[prevHead:iter.head])
				return RawString{buf: copied.Bytes(), hasEscapes: hasEscapes}
			case c == '\\':
				// toggle readingEscape
				readingEscape = readingEscape != true
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
				prevHead := iter.head
				iter.head = i + 1
				// are we about to change iter.buf?
				if i+4 >= iter.tail {
					copied.Write(iter.buf[prevHead:iter.head])
					var buf [4]byte
					iter.readAndFillU4(buf[:])
					copied.Write(buf[:])
					continue outerLoop
				}

				if iter.parseU4() == -1 {
					iter.ReportError("ReadRawString", "invalid unicode escape sequence")
					return RawString{}
				}
				// it shouldn't be necessary to break out of the loop, but
				// for some reason the compiler doesn't like this branch
				// and inserts slice bounds check around the iter.buf[i] read
				// without the "continue outerLoop"
				copied.Write(iter.buf[prevHead:iter.head])
				continue outerLoop
			default:
				iter.ReportError("ReadRawString", `invalid escape char after \`)
				return RawString{}
			}
		}

		// copy buffer and load more
		copied.Write(iter.buf[iter.head:iter.tail])
		iter.head = iter.tail

		// load next chunk
		if iter.readByte() == 0 {
			break
		}
		iter.unreadByte()
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

func (iter *Iterator) parseU4() (ret rune) {
	// eliminate bounds check inside the loop
	end := iter.head + 4
	if iter.head < 0 || end > len(iter.buf) {
		return -1
	}

	for i := iter.head; i < end; i++ {
		c := iter.buf[i]
		c -= '0'
		if c <= 9 {
			ret = ret*16 + rune(c)
			continue
		}
		c -= 'A' - '0'
		if c <= 5 {
			ret = ret*16 + rune(c+10)
			continue
		}
		c -= 'a' - 'A'
		if c <= 5 {
			ret = ret*16 + rune(c+10)
			continue
		}

		return -1
	}
	iter.head = end
	return ret
}

func (iter *Iterator) readU4() (ret rune) {
	if iter.tail-iter.head >= 4 {
		if ret = iter.parseU4(); ret < 0 {
			iter.ReportError("readU4", "invalid hex char")
			return 0
		}
		return ret
	}

	for i := 0; i < 4; i++ {
		c := iter.readByte()
		if iter.Error != nil {
			return
		}
		c -= '0'
		if c <= 9 {
			ret = ret*16 + rune(c)
			continue
		}
		c -= 'A' - '0'
		if c <= 5 {
			ret = ret*16 + rune(c+10)
			continue
		}
		c -= 'a' - 'A'
		if c <= 5 {
			ret = ret*16 + rune(c+10)
			continue
		}

		iter.ReportError("readU4", "invalid hex char")
		return
	}
	return ret
}

func (iter *Iterator) readAndFillU4(buf []byte) (ret rune) {
	if len(buf) < 4 {
		panic("buffer too small")
	}

	for i := 0; i < 4; i++ {
		c := iter.readByte()
		if iter.Error != nil {
			return
		}
		buf[i] = c
		c -= '0'
		if c <= 9 {
			ret = ret*16 + rune(c)
			continue
		}
		c -= 'A' - '0'
		if c <= 5 {
			ret = ret*16 + rune(c+10)
			continue
		}
		c -= 'a' - 'A'
		if c <= 5 {
			ret = ret*16 + rune(c+10)
			continue
		}

		iter.ReportError("readU4", "invalid hex char")
		return
	}
	return ret
}
