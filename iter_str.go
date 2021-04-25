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

func (iter *Iterator) PeekString() StringPeeker {
	c := iter.nextToken()
	switch c {
	case '"':
	case 'n':
		iter.skipThreeBytes('u', 'l', 'l')
		return StringPeeker{}
	default:
		iter.ReportError("PeekString", `expects " or n, but found `+string([]byte{c}))
		return StringPeeker{}
	}

	return iter.readRawStringInner()
}

func (iter *Iterator) readRawStringInner() StringPeeker {
	var copied []byte
	for iter.Error == nil {
		for i := iter.head; i < iter.tail; i++ {
			c := iter.buf[i]
			if c == '"' {
				// careful, we're copying the ending double quote into the buffer
				if copied == nil {
					// super fast path
					prevHead := iter.head
					iter.head = i + 1
					return StringPeeker{buf: iter.buf[prevHead:iter.head], isRaw: true}
				}
				prevHead := iter.head
				iter.head = i + 1
				copied = append(copied, iter.buf[prevHead:iter.head]...)
				return StringPeeker{buf: copied}
			} else if c == '\\' && i+1 < iter.tail {
				// could be escaped double quote, need to skip it
				i++
				switch iter.buf[i] {
				case 'u', '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				default:
					iter.ReportError("PeekString", `invalid escape char after \`)
					return StringPeeker{}
				}
				continue
			} else if c < ' ' {
				iter.ReportError("PeekString",
					"invalid control character found: "+strconv.Itoa(int(c)))
				return StringPeeker{}
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

	iter.ReportError("PeekString", "unexpected end of input")
	return StringPeeker{}
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
		peeker := iter.readRawStringInner()
		buf, _ := peeker.Bytes()
		return buf
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

type StringPeeker struct {
	buf   []byte
	isRaw bool
}

func (p StringPeeker) IsEmpty() bool {
	return p.buf == nil
}

func (p *StringPeeker) Realize() {
	if p.isRaw {
		bufCpy := make([]byte, len(p.buf))
		copy(bufCpy, p.buf)
		p.buf = bufCpy
		p.isRaw = false
	}
}

func (p *StringPeeker) String() string {
	if p.buf == nil {
		return ""
	}

	iter := Iterator{
		buf:  p.buf,
		tail: len(p.buf),
	}
	return iter.readStringInner()
}

// Bytes returns a buffer and true if this is a direct view into iter,
// or false if the buffer is a copy.
// Note that the buffer is only valid until the next read from Iterator.
// Use Realize before reading further from the Iterator.
func (p *StringPeeker) Bytes() ([]byte, bool) {
	raw := p.buf
	if len(raw) > 0 {
		raw = raw[:len(raw)-1]
	}
	return raw, p.isRaw
}
