package jsoniter

import (
	"fmt"
	"strings"
)

// ReadObject reads one field from object.
// If object ended, returns empty string and false.
// Otherwise, returns the field name.
func (iter *Iterator) ReadObject() (string, bool) {
	rs := iter.ReadObjectRaw()
	if rs.IsNil() {
		return "", false
	}
	return rs.String(), true
}

// ReadObjectRaw reads one field from object and returns
// the field name as RawString.
func (iter *Iterator) ReadObjectRaw() RawString {
	c := iter.nextToken()
	switch c {
	case 'n':
		iter.skipThreeBytes('u', 'l', 'l')
		return RawString{} // null
	case '{':
		c = iter.nextToken()
		if c == '"' {
			peeker := iter.readRawStringInner()
			if !iter.isNextTokenBuffered() {
				peeker.Realize()
			}
			c = iter.nextToken()
			if c != ':' {
				iter.ReportError("ReadObject", "expect : after object field, but found "+string([]byte{c}))
			}
			return peeker
		}
		if c == '}' {
			return RawString{} // end of object
		}
		iter.ReportError("ReadObject", `expect " after {, but found `+string([]byte{c}))
		return RawString{}
	case ',':
		peeker := iter.ReadRawString()
		if !iter.isNextTokenBuffered() {
			peeker.Realize()
		}
		c = iter.nextToken()
		if c != ':' {
			iter.ReportError("ReadObject", "expect : after object field, but found "+string([]byte{c}))
		}
		return peeker
	case '}':
		return RawString{} // end of object
	default:
		iter.ReportError("ReadObject", fmt.Sprintf(`expect { or , or } or n, but found %s`, string([]byte{c})))
		return RawString{}
	}
}

// CaseInsensitive
func (iter *Iterator) readFieldHash() int64 {
	hash := int64(0x811c9dc5)
	c := iter.nextToken()
	if c != '"' {
		iter.ReportError("readFieldHash", `expect ", but found `+string([]byte{c}))
		return 0
	}
	for {
		for i := iter.head; i < iter.tail; i++ {
			// require ascii string and no escape
			b := iter.buf[i]
			if b == '\\' {
				iter.head = i
				for _, b := range iter.readStringInner() {
					if 'A' <= b && b <= 'Z' && !iter.cfg.caseSensitive {
						b += 'a' - 'A'
					}
					hash ^= int64(b)
					hash *= 0x1000193
				}
				c = iter.nextToken()
				if c != ':' {
					iter.ReportError("readFieldHash", `expect :, but found `+string([]byte{c}))
					return 0
				}
				return hash
			}
			if b == '"' {
				iter.head = i + 1
				c = iter.nextToken()
				if c != ':' {
					iter.ReportError("readFieldHash", `expect :, but found `+string([]byte{c}))
					return 0
				}
				return hash
			}
			if 'A' <= b && b <= 'Z' && !iter.cfg.caseSensitive {
				b += 'a' - 'A'
			}
			hash ^= int64(b)
			hash *= 0x1000193
		}
		if !iter.loadMore() {
			iter.ReportError("readFieldHash", `incomplete field name`)
			return 0
		}
	}
}

func calcHash(str string, caseSensitive bool) int64 {
	if !caseSensitive {
		str = strings.ToLower(str)
	}
	hash := int64(0x811c9dc5)
	for _, b := range []byte(str) {
		hash ^= int64(b)
		hash *= 0x1000193
	}
	return int64(hash)
}

// ReadObjectCB read map with callback, the key can be any string
func (iter *Iterator) ReadObjectCB(callback func(*Iterator, string) bool) bool {
	return iter.ReadObjectRawCB(func(i *Iterator, rs RawString) bool {
		return callback(i, rs.String())
	})
}

// ReadObjectCB read map with callback, the key can be any string
func (iter *Iterator) ReadObjectRawCB(callback func(*Iterator, RawString) bool) bool {
	c := iter.nextToken()
	if c == '{' {
		if !iter.incrementDepth() {
			return false
		}
		c = iter.nextToken()
		if c == '"' {
			rs := iter.readRawStringInner()
			c = iter.nextToken()
			if c != ':' {
				iter.ReportError("ReadObject", "expect : after object field, but found "+string([]byte{c}))
				return false
			}
			if !callback(iter, rs) {
				iter.decrementDepth()
				return false
			}
			c = iter.nextToken()
			for c == ',' {
				rs = iter.ReadRawString()
				c = iter.nextToken()
				if c != ':' {
					iter.ReportError("ReadObject", "expect : after object field, but found "+string([]byte{c}))
					return false
				}
				if !callback(iter, rs) {
					iter.decrementDepth()
					return false
				}
				c = iter.nextToken()
			}
			if c != '}' {
				iter.ReportError("ReadObjectCB", `object not ended with }`)
				return false
			}
			return iter.decrementDepth()
		}
		if c == '}' {
			return iter.decrementDepth()
		}
		iter.ReportError("ReadObjectCB", `expect " after {, but found `+string([]byte{c}))
		iter.decrementDepth()
		return false
	}
	if c == 'n' {
		iter.skipThreeBytes('u', 'l', 'l')
		return true // null
	}
	iter.ReportError("ReadObjectCB", `expect { or n, but found `+string([]byte{c}))
	return false
}

func (iter *Iterator) readObjectStart() bool {
	c := iter.nextToken()
	if c == '{' {
		c = iter.nextToken()
		if c == '}' {
			return false
		}
		iter.unreadByte()
		return true
	} else if c == 'n' {
		iter.skipThreeBytes('u', 'l', 'l')
		return false
	}
	iter.ReportError("readObjectStart", "expect { or n, but found "+string([]byte{c}))
	return false
}

func (iter *Iterator) isObjectEnd() bool {
	c := iter.nextToken()
	if c == ',' {
		return false
	}
	if c == '}' {
		return true
	}

	iter.ReportError("isObjectEnd", "object ended prematurely, unexpected char "+string([]byte{c}))
	return true
}
