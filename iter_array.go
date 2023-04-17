package jsoniter

// ReadArray read array element, tells if the array has more element to read.
func (iter *Iterator) ReadArray() (ret bool) {
	c := iter.nextToken()
	switch c {
	case 'n':
		iter.ensureLiteral(nullLiteral)
		return false // null
	case '[':
		c = iter.nextToken()
		if c != ']' {
			iter.unreadByte()
			return true
		}
		return false
	case ']':
		return false
	case ',':
		return true
	default:
		iter.ReportError("ReadArray", "expect [ or , or ] or n, but found "+string(c))
		return
	}
}

// ReadArrayCB read array with callback
func (iter *Iterator) ReadArrayCB(callback func(*Iterator) bool) {
	c := iter.nextToken()
	if c == '[' {
		if !iter.incrementDepth() {
			return
		}
		c = iter.nextToken()
		if c != ']' {
			iter.unreadByte()
			if !callback(iter) {
				iter.decrementDepth()
				return
			}
			c = iter.nextToken()
			for c == ',' {
				if !callback(iter) {
					iter.decrementDepth()
					return
				}
				c = iter.nextToken()
			}
			if c != ']' {
				iter.ReportError("ReadArrayCB", "expect ] in the end, but found "+string(c))
				iter.decrementDepth()
				return
			}
			iter.decrementDepth()
			return
		}
		iter.decrementDepth()
		return
	}
	if c == 'n' {
		iter.ensureLiteral(nullLiteral)
		return
	}

	iter.ReportError("ReadArrayCB", "expect [ or n, but found "+string(c))
}
