package jsoniter

import (
	"strconv"
	"strings"
	"unicode/utf16"
)

// ReadString read string from iterator
func (iter *Iterator) ReadString() string {
	c := iter.nextToken()
	if c == '"' {
		return iter.readStringInner()
	} else if c == 'n' {
		iter.skipThreeBytes('u', 'l', 'l')
		return ""
	}
	iter.ReportError("ReadString", `expects " or n, but found `+string([]byte{c}))
	return ""
}

func (iter *Iterator) readStringInner() string {
	sb := strings.Builder{}

outerLoop:
	for iter.Error == nil {
		for i := iter.head; i < iter.tail; i++ {
			c := iter.buf[i]
			if c == '"' {
				if sb.Len() == 0 {
					// super fast path
					res := string(iter.buf[iter.head:i])
					iter.head = i + 1
					return res
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
		iter.ReportError("readEscapedChar",
			`invalid escape char after \`)
	}
}

// ReadStringAsSlice read string from iterator without copying into string form.
// The []byte can not be kept, as it will change after next iterator call.
func (iter *Iterator) ReadStringAsSlice() (ret []byte) {
	c := iter.nextToken()
	if c == '"' {
		for i := iter.head; i < iter.tail; i++ {
			// require ascii string and no escape
			// for: field name, base64, number
			if iter.buf[i] == '"' {
				// fast path: reuse the underlying buffer
				ret = iter.buf[iter.head:i]
				iter.head = i + 1
				return ret
			}
		}
		readLen := iter.tail - iter.head
		copied := make([]byte, readLen, readLen*2)
		copy(copied, iter.buf[iter.head:iter.tail])
		iter.head = iter.tail
		for iter.Error == nil {
			c := iter.readByte()
			if c == '"' {
				return copied
			}
			copied = append(copied, c)
		}
		return copied
	}
	iter.ReportError("ReadStringAsSlice", `expects " or n, but found `+string([]byte{c}))
	return
}

func (iter *Iterator) readU4() (ret rune) {
	for i := 0; i < 4; i++ {
		c := iter.readByte()
		if iter.Error != nil {
			return
		}
		if c >= '0' && c <= '9' {
			ret = ret*16 + rune(c-'0')
		} else if c >= 'a' && c <= 'f' {
			ret = ret*16 + rune(c-'a'+10)
		} else if c >= 'A' && c <= 'F' {
			ret = ret*16 + rune(c-'A'+10)
		} else {
			iter.ReportError("readU4", "expects 0~9 or a~f, but found "+string([]byte{c}))
			return
		}
	}
	return ret
}
