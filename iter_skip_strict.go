//go:build !jsoniter_sloppy
// +build !jsoniter_sloppy

package jsoniter

import (
	"io"
)

func (iter *Iterator) skipNumber() {
	if !iter.trySkipNumber() {
		iter.unreadByte()
		if iter.Error != nil && iter.Error != io.EOF {
			return
		}
		iter.ReadFloat64()
		if iter.Error != nil && iter.Error != io.EOF {
			iter.Error = nil
			iter.ReadBigFloat()
		}
	}
}

func (iter *Iterator) trySkipNumber() bool {
	dotFound := false

	if iter.head < 0 || iter.head >= iter.tail || iter.tail > len(iter.buf) {
		return false
	}

	for i := iter.head; i < iter.tail; i++ {
		c := iter.buf[i]
		switch c {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		case '.':
			if dotFound {
				iter.ReportError("validateNumber", `more than one dot found in number`)
				return true // already failed
			}
			if i+1 == iter.tail {
				return false
			}
			c = iter.buf[i+1]
			switch c {
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			default:
				iter.ReportError("validateNumber", `missing digit after dot`)
				return true // already failed
			}
			dotFound = true
		default:
			switch c {
			case ',', ']', '}', ' ', '\t', '\n', '\r':
				if iter.head == i {
					return false // if - without following digits
				}
				iter.head = i
				return true // must be valid
			}
			return false // may be invalid
		}
	}
	return false
}

func (iter *Iterator) skipString() {
	if iter.head >= 0 && iter.head < iter.tail && iter.tail <= len(iter.buf) {
		for i := iter.head; i < iter.tail; i++ {
			c := iter.buf[i]
			if c == '"' {
				iter.head = i + 1
				return // skipped the entire string
			} else if c == '\\' {
				iter.head = i
				break
			} else if c < ' ' {
				iter.head = i
				break
			}
		}
	}

	iter.readRawStringInner()
}

func (iter *Iterator) skipObject() {
	iter.unreadByte()
	iter.ReadObjectRawCB(func(i *Iterator, rs RawString) bool {
		i.Skip()
		return true
	})
}

func (iter *Iterator) skipArray() {
	iter.unreadByte()
	iter.ReadArrayCB(func(iter *Iterator) bool {
		iter.Skip()
		return true
	})
}
