package jsoniter

import (
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
			if c == '"' {
				if sb.Len() == 0 {
					// super fast path
					res := iter.buf[iter.head:i]
					iter.head = i + 1
					return string(res)
				}
				sb.Write(iter.buf[iter.head:i])
				iter.head = i + 1
				return sb.String()
			} else if c == '\\' {
				sb.Write(iter.buf[iter.head:i])
				iter.head = i + 1
				iter.readEscapedChar(&sb)
				continue outerLoop
			} else if c < ' ' {
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
		copied     []byte
		hasEscapes bool
	)

outerLoop:
	for iter.Error == nil {
		for i := iter.head; i < iter.tail; i++ {
			c := iter.buf[i]
			if c == '"' {
				// careful, we're copying the ending double quote into the buffer
				if copied == nil {
					// super fast path
					prevHead := iter.head
					iter.head = i + 1
					return RawString{buf: iter.buf[prevHead:iter.head], isRaw: true, hasEscapes: hasEscapes}
				}
				prevHead := iter.head
				iter.head = i + 1
				copied = append(copied, iter.buf[prevHead:iter.head]...)
				return RawString{buf: copied, hasEscapes: hasEscapes}
			} else if c == '\\' {
				// could be escaped double quote, need to skip it
				hasEscapes = true

				if i+1 >= iter.tail {
					copied = append(copied, iter.buf[iter.head:i+1]...)
					iter.head = i + 1
					// load next chunk, resets head, tail and buf
					if iter.readByte() == 0 {
						return RawString{}
					}
					iter.unreadByte()
					i = iter.head
				} else {
					// look at the next byte directly
					i++
				}

				switch iter.buf[i] {
				case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				case 'u':
					prevHead := iter.head
					iter.head = i + 1
					// are we about to change iter.buf?
					if i+4 >= iter.tail {
						copied = append(copied, iter.buf[prevHead:iter.head]...)
						var buf [4]byte
						iter.readAndFillU4(buf[:])
						copied = append(copied, buf[:]...)
						continue outerLoop
					}

					iter.readU4()
					if iter.Error != nil {
						return RawString{}
					}
					// ensure we copy this escaped section
					iter.head = prevHead
					i += 4
				default:
					iter.ReportError("ReadRawString", `invalid escape char after \`)
					return RawString{}
				}
				continue
			} else if c < ' ' {
				iter.ReportError("ReadRawString",
					"invalid control character found: "+strconv.Itoa(int(c)))
				return RawString{}
			}
		}

		// copy buffer and load more
		copied = append(copied, iter.buf[iter.head:iter.tail]...)
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

func (iter *Iterator) readU4() (ret rune) {
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
