//go:build !jsoniter_sloppy
// +build !jsoniter_sloppy

package jsoniter

func (iter *Iterator) skipNumber() {
	iter.readNumberRaw(nil)
}

func (iter *Iterator) skipString() {
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
