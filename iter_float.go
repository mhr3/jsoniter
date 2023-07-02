package jsoniter

import (
	"encoding/json"
	"io"
	"math/big"
	"strconv"
)

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
	numberParseStateInitial byte = iota
	numberParseStateNegative
	numberParseStateZero   // terminal
	numberParseStateDigits // terminal
	numberParseStateFloat
	numberParseStateFloatDigit // terminal
	numberParseStateExponent
	numberParseStateExponentDigit
	numberParseStateExponentExtraDigits // terminal
	numberParseStateEnd
	numberParseStateError
)

func (iter *Iterator) readNumberRaw(copied []byte) RawString {
	var (
		state  = numberParseStateInitial
		endIdx = 0
	)

load_loop:
	for {
		// eliminate bounds check
		if iter.head < 0 || iter.tail > len(iter.buf) {
			break
		}

		for i := iter.head; i < iter.tail; i++ {
			c := iter.buf[i]

			switch state {
			case numberParseStateInitial:
				switch c {
				case '-':
					state = numberParseStateNegative
				case '0':
					state = numberParseStateZero
				case '1', '2', '3', '4', '5', '6', '7', '8', '9':
					state = numberParseStateDigits
				default:
					state = numberParseStateError
					break load_loop
				}
			case numberParseStateNegative:
				switch c {
				case '0':
					state = numberParseStateZero
				case '1', '2', '3', '4', '5', '6', '7', '8', '9':
					state = numberParseStateDigits
				default:
					state = numberParseStateError
					break load_loop
				}
			case numberParseStateZero:
				switch c {
				case '.':
					state = numberParseStateFloat
				case 'e', 'E':
					state = numberParseStateExponent
				default:
					state = numberParseStateEnd
					endIdx = i
					break load_loop
				}
			case numberParseStateDigits:
				switch c {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				case '.':
					state = numberParseStateFloat
				case 'e', 'E':
					state = numberParseStateExponent
				default:
					state = numberParseStateEnd
					endIdx = i
					break load_loop
				}
			case numberParseStateFloat:
				switch c {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					state = numberParseStateFloatDigit
				default:
					state = numberParseStateError
					break load_loop
				}
			case numberParseStateFloatDigit:
				switch c {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				case 'e', 'E':
					state = numberParseStateExponent
				default:
					state = numberParseStateEnd
					endIdx = i
					break load_loop
				}
			case numberParseStateExponent:
				switch c {
				case '+', '-':
					state = numberParseStateExponentDigit
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					state = numberParseStateExponentExtraDigits
				default:
					state = numberParseStateError
					break load_loop
				}
			case numberParseStateExponentDigit:
				switch c {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					state = numberParseStateExponentExtraDigits
				default:
					state = numberParseStateError
					break load_loop
				}
			case numberParseStateExponentExtraDigits:
				switch c {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				default:
					state = numberParseStateEnd
					endIdx = i
					break load_loop
				}
			}
		}
		copied = append(copied, iter.buf[iter.head:iter.tail]...)
		if !iter.loadMore() {
			// we just did the copy, so set end to head
			endIdx = iter.head
			break
		}
	}
	if iter.Error != nil && iter.Error != io.EOF {
		return RawString{}
	}

	// are we in an accepting state?
	switch state {
	case numberParseStateZero, numberParseStateDigits, numberParseStateFloatDigit, numberParseStateExponentExtraDigits:
		// yep
	case numberParseStateEnd:
		// we need to ensure the next character is something that terminates the number
		switch iter.buf[endIdx] {
		case ' ', '\t', '\r', '\n', ',', '}', ']':
			// all good
		default:
			iter.ReportError("readNumberAsBytes", "unexpected character after number")
			return RawString{}
		}
	default:
		iter.ReportError("readNumberAsBytes", "invalid number")
		return RawString{}
	}

	prevHead := iter.head
	iter.head = endIdx

	if len(copied) == 0 {
		return RawString{isRaw: true, buf: iter.buf[prevHead:iter.head]}
	}
	copied = append(copied, iter.buf[prevHead:iter.head]...)
	return RawString{buf: copied}
}

func (iter *Iterator) readNumberAsBytes(buf []byte) []byte {
	rs := iter.readNumberRaw(buf)
	if rs.isRaw {
		return append(buf, rs.buf...)
	}
	return rs.buf
}

func (iter *Iterator) readNumberAsString() (ret string) {
	rs := iter.readNumberRaw(nil)
	return string(rs.buf)
}

func parseFloatBytes(b []byte, bitSize int) (float64, error) {
	// the []byte to string conversion used to cause an alloc,
	// but that's no longer the case as of go1.20
	return strconv.ParseFloat(string(b), bitSize)
}

// ReadFloat32 read float32
func (iter *Iterator) ReadFloat32() (ret float32) {
	rs := iter.readNumberRaw(nil)
	if iter.Error != nil && iter.Error != io.EOF {
		return
	}

	res, err := parseFloatBytes(rs.buf, 32)
	if err != nil {
		iter.ReportError("ReadFloat32", err.Error())
		return
	}
	return float32(res)
}

// ReadFloat64 read float64
func (iter *Iterator) ReadFloat64() (ret float64) {
	rs := iter.readNumberRaw(nil)
	if iter.Error != nil && iter.Error != io.EOF {
		return
	}

	res, err := parseFloatBytes(rs.buf, 64)
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
