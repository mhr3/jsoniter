package jsoniter

import (
	"encoding/json"
	"io"
	"math/big"
	"strconv"
)

var floatDigits []int8

const invalidCharForNumber = int8(-1)
const endOfNumber = int8(-2)
const dotInNumber = int8(-3)

func init() {
	floatDigits = make([]int8, 256)
	for i := 0; i < len(floatDigits); i++ {
		floatDigits[i] = invalidCharForNumber
	}
	for i := int8('0'); i <= int8('9'); i++ {
		floatDigits[i] = i - int8('0')
	}
	floatDigits[','] = endOfNumber
	floatDigits[']'] = endOfNumber
	floatDigits['}'] = endOfNumber
	floatDigits[' '] = endOfNumber
	floatDigits['\t'] = endOfNumber
	floatDigits['\r'] = endOfNumber
	floatDigits['\n'] = endOfNumber
	floatDigits['.'] = dotInNumber
}

// ReadBigFloat read big.Float
func (iter *Iterator) ReadBigFloat() (ret *big.Float) {
	str := iter.readNumberAsString()
	if iter.Error != nil && iter.Error != io.EOF {
		return nil
	}
	prec := 64
	if len(str) > prec {
		prec = len(str)
	}
	val, _, err := big.ParseFloat(str, 10, uint(prec), big.ToZero)
	if err != nil {
		iter.Error = err
		return nil
	}
	return val
}

// ReadBigInt read big.Int
func (iter *Iterator) ReadBigInt() (ret *big.Int) {
	str := iter.readNumberAsString()
	if iter.Error != nil && iter.Error != io.EOF {
		return nil
	}
	ret = big.NewInt(0)
	var success bool
	ret, success = ret.SetString(str, 10)
	if !success {
		iter.ReportError("ReadBigInt", "invalid big int")
		return nil
	}
	return ret
}

//ReadFloat32 read float32
func (iter *Iterator) ReadFloat32() (ret float32) {
	c := iter.nextToken()
	if c == '-' {
		return -iter.readPositiveFloat32()
	}
	iter.unreadByte()
	return iter.readPositiveFloat32()
}

func (iter *Iterator) readPositiveFloat32() (ret float32) {
	i := iter.head
	// first char
	if i == iter.tail {
		return iter.readFloat32SlowPath()
	}
	c := iter.buf[i]
	i++
	ind := floatDigits[c]
	switch ind {
	case invalidCharForNumber:
		return iter.readFloat32SlowPath()
	case endOfNumber:
		iter.ReportError("readFloat32", "empty number")
		return
	case dotInNumber:
		iter.ReportError("readFloat32", "leading dot is invalid")
		return
	case 0:
		if i == iter.tail {
			return iter.readFloat32SlowPath()
		}
		c = iter.buf[i]
		switch c {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			iter.ReportError("readFloat32", "leading zero is invalid")
			return
		}
	}
	value := uint64(ind)
	// chars before dot
non_decimal_loop:
	for ; i < iter.tail; i++ {
		c = iter.buf[i]
		ind := floatDigits[c]
		switch ind {
		case invalidCharForNumber:
			return iter.readFloat32SlowPath()
		case endOfNumber:
			iter.head = i
			return float32(value)
		case dotInNumber:
			break non_decimal_loop
		}
		if value > uint64SafeToMultiple10 {
			return iter.readFloat32SlowPath()
		}
		value = (value << 3) + (value << 1) + uint64(ind) // value = value * 10 + ind;
	}
	// chars after dot
	if c == '.' {
		i++
		decimalPlaces := 0
		if i == iter.tail {
			return iter.readFloat32SlowPath()
		}
		for ; i < iter.tail; i++ {
			c = iter.buf[i]
			ind := floatDigits[c]
			switch ind {
			case endOfNumber:
				if decimalPlaces > 0 && decimalPlaces < len(pow10) {
					iter.head = i
					return float32(float64(value) / float64(pow10[decimalPlaces]))
				}
				// too many decimal places
				return iter.readFloat32SlowPath()
			case invalidCharForNumber, dotInNumber:
				return iter.readFloat32SlowPath()
			}
			decimalPlaces++
			if value > uint64SafeToMultiple10 {
				return iter.readFloat32SlowPath()
			}
			value = (value << 3) + (value << 1) + uint64(ind)
		}
	}
	return iter.readFloat32SlowPath()
}

const (
	numberParseState0 byte = iota
	numberParseState1
	numberParseState2
	numberParseState3
	numberParseState4
	numberParseState5
	numberParseState6
	numberParseState7
	numberParseStateEnd
	numberParseStateError
)

func (iter *Iterator) readNumberAsBytes(buf []byte) []byte {
	end := -1
	state := numberParseState0
load_loop:
	for {
		// eliminate bounds check
		if iter.head < 0 || iter.tail > len(iter.buf) {
			break
		}

		end = iter.head
		for i := iter.head; i < iter.tail; i++ {
			c := iter.buf[i]

			switch state {
			case numberParseState0:
				switch c {
				case '-':
					state = numberParseState1
					end++
				case '0':
					state = numberParseState2
					end++
				case '1', '2', '3', '4', '5', '6', '7', '8', '9':
					state = numberParseState3
					end++
				default:
					state = numberParseStateError
					break load_loop
				}
			case numberParseState1:
				switch c {
				case '0':
					state = numberParseState2
					end++
				case '1', '2', '3', '4', '5', '6', '7', '8', '9':
					state = numberParseState3
					end++
				default:
					state = numberParseStateError
					break load_loop
				}
			case numberParseState2:
				switch c {
				case '.':
					state = numberParseState4
					end++
				case 'e', 'E':
					state = numberParseState6
					end++
				default:
					state = numberParseStateEnd
					break load_loop
				}
			case numberParseState3:
				switch c {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					end++
				case '.':
					state = numberParseState4
					end++
				case 'e', 'E':
					state = numberParseState6
					end++
				default:
					state = numberParseStateEnd
					break load_loop
				}
			case numberParseState4:
				switch c {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					state = numberParseState5
					end++
				default:
					state = numberParseStateError
					break load_loop
				}
			case numberParseState5:
				switch c {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					end++
				case 'e', 'E':
					state = numberParseState6
					end++
				default:
					state = numberParseStateEnd
					break load_loop
				}
			case numberParseState6:
				switch c {
				case '+', '-':
					state = numberParseState7
					end++
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					state = numberParseState7
					end++
				default:
					state = numberParseStateError
					break load_loop
				}
			case numberParseState7:
				switch c {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					end++
				default:
					state = numberParseStateEnd
					break load_loop
				}
			}
		}
		buf = append(buf, iter.buf[iter.head:end]...)
		if !iter.loadMore() {
			break
		}
	}
	if iter.Error != nil && iter.Error != io.EOF {
		return nil
	}

	// are we in an accepting state?
	switch state {
	case numberParseState2, numberParseState3, numberParseState5, numberParseState7:
		// yep
		buf = append(buf, iter.buf[iter.head:end]...)
		iter.head = end
	case numberParseStateEnd:
		// we need to ensure the next character is something that terminates the number
		if floatDigits[iter.buf[end]] != endOfNumber {
			iter.ReportError("readNumberAsBytes", "unexpected character after number")
			return nil
		}
		buf = append(buf, iter.buf[iter.head:end]...)
		iter.head = end
	default:
		iter.ReportError("readNumberAsBytes", "invalid number")
		return nil
	}

	return buf
}

func (iter *Iterator) readNumberAsString() (ret string) {
	// this will save one alloc in most cases
	buf := [24]byte{}
	res := iter.readNumberAsBytes(buf[:0])

	return string(res)
}

func (iter *Iterator) readFloat32SlowPath() (ret float32) {
	buf := [24]byte{}
	strBuf := iter.readNumberAsBytes(buf[:0])
	if iter.Error != nil && iter.Error != io.EOF {
		return
	}
	val, err := strconv.ParseFloat(string(strBuf), 32)
	if err != nil {
		iter.Error = err
		return
	}
	return float32(val)
}

// ReadFloat64 read float64
func (iter *Iterator) ReadFloat64() (ret float64) {
	c := iter.nextToken()
	if c == '-' {
		return -iter.readPositiveFloat64()
	}
	iter.unreadByte()
	return iter.readPositiveFloat64()
}

func (iter *Iterator) readPositiveFloat64() (ret float64) {
	i := iter.head
	// first char
	if i == iter.tail {
		return iter.readFloat64SlowPath()
	}
	c := iter.buf[i]
	i++
	ind := floatDigits[c]
	switch ind {
	case invalidCharForNumber:
		return iter.readFloat64SlowPath()
	case endOfNumber:
		iter.ReportError("readFloat64", "empty number")
		return
	case dotInNumber:
		iter.ReportError("readFloat64", "leading dot is invalid")
		return
	case 0:
		if i == iter.tail {
			return iter.readFloat64SlowPath()
		}
		c = iter.buf[i]
		switch c {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			iter.ReportError("readFloat64", "leading zero is invalid")
			return
		}
	}
	value := uint64(ind)
	// chars before dot
non_decimal_loop:
	for ; i < iter.tail; i++ {
		c = iter.buf[i]
		ind := floatDigits[c]
		switch ind {
		case invalidCharForNumber:
			return iter.readFloat64SlowPath()
		case endOfNumber:
			iter.head = i
			return float64(value)
		case dotInNumber:
			break non_decimal_loop
		}
		if value > uint64SafeToMultiple10 {
			return iter.readFloat64SlowPath()
		}
		value = (value << 3) + (value << 1) + uint64(ind) // value = value * 10 + ind;
	}
	// chars after dot
	if c == '.' {
		i++
		decimalPlaces := 0
		if i == iter.tail {
			return iter.readFloat64SlowPath()
		}
		for ; i < iter.tail; i++ {
			c = iter.buf[i]
			ind := floatDigits[c]
			switch ind {
			case endOfNumber:
				if decimalPlaces > 0 && decimalPlaces < len(pow10) {
					iter.head = i
					return float64(value) / float64(pow10[decimalPlaces])
				}
				// too many decimal places
				return iter.readFloat64SlowPath()
			case invalidCharForNumber, dotInNumber:
				return iter.readFloat64SlowPath()
			}
			decimalPlaces++
			if value > uint64SafeToMultiple10 {
				return iter.readFloat64SlowPath()
			}
			value = (value << 3) + (value << 1) + uint64(ind)
			if value > maxFloat64 {
				return iter.readFloat64SlowPath()
			}
		}
	}
	return iter.readFloat64SlowPath()
}

func (iter *Iterator) readFloat64SlowPath() (ret float64) {
	buf := [24]byte{}
	strBuf := iter.readNumberAsBytes(buf[:0])
	if iter.Error != nil && iter.Error != io.EOF {
		return
	}
	val, err := strconv.ParseFloat(string(strBuf), 64)
	if err != nil {
		iter.Error = err
		return
	}
	return val
}

// ReadNumber read json.Number
func (iter *Iterator) ReadNumber() (ret json.Number) {
	return json.Number(iter.readNumberAsString())
}

// ReadNumberAsSlice reads a json number into the provided byte slice (can be nil)
func (iter *Iterator) ReadNumberAsSlice(buf []byte) []byte {
	return iter.readNumberAsBytes(buf)
}
