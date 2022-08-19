package jsoniter

import (
	"encoding/json"
	"io"
	"math/big"
	"strconv"
	"unsafe"
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

//go:nosplit
//go:nocheckptr
func noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

func parseFloatBytes(b []byte, bitSize int) (float64, error) {
	s := *(*string)(noescape(unsafe.Pointer(&b)))

	res, err := strconv.ParseFloat(s, bitSize)
	if err != nil {
		if nErr, ok := err.(*strconv.NumError); ok {
			nErr.Num = string(b)
			err = nErr
		}
	}
	return res, err
}

//ReadFloat32 read float32
func (iter *Iterator) ReadFloat32() (ret float32) {
	buf := [24]byte{}
	numBuf := iter.readNumberAsBytes(buf[:0])
	if iter.Error != nil && iter.Error != io.EOF {
		return
	}

	res, err := parseFloatBytes(numBuf, 32)
	if err != nil {
		iter.ReportError("ReadFloat32", err.Error())
		return
	}
	return float32(res)
}

// ReadFloat64 read float64
func (iter *Iterator) ReadFloat64() (ret float64) {
	buf := [24]byte{}
	numBuf := iter.readNumberAsBytes(buf[:0])
	if iter.Error != nil && iter.Error != io.EOF {
		return
	}

	res, err := parseFloatBytes(numBuf, 64)
	if err != nil {
		iter.ReportError("ReadFloat64", err.Error())
		return
	}
	return res
}

// ReadNumber read json.Number
func (iter *Iterator) ReadNumber() (ret json.Number) {
	return json.Number(iter.readNumberAsString())
}

// ReadNumberAsSlice reads a json number into the provided byte slice (can be nil)
func (iter *Iterator) ReadNumberAsSlice(buf []byte) []byte {
	return iter.readNumberAsBytes(buf)
}
