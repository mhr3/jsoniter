package jsoniter

import (
	"bytes"
	"encoding/json"
	"io"
	"math/big"
	"strconv"
	"unsafe"
)

const invalidCharForNumber = int8(-1)

const invalidJSONNumberDotNoDigit = "missing digit after dot"

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

func (iter *Iterator) readNumberAsBytes(buf []byte) []byte {
load_loop:
	for {
		// eliminate bounds check
		if iter.head < 0 || iter.tail > len(iter.buf) {
			break
		}

		end := iter.head
		for i := iter.head; i < iter.tail; i++ {
			c := iter.buf[i]
			switch c {
			case '+', '-', '.', 'e', 'E', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				end++
			default:
				buf = append(buf, iter.buf[iter.head:end]...)
				iter.head = i
				break load_loop
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
	if len(buf) == 0 {
		iter.ReportError("readNumberAsBytes", "invalid number")
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

	if numBuf[len(numBuf)-1] == '.' {
		iter.ReportError("ReadFloat32", invalidJSONNumberDotNoDigit)
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

	if numBuf[len(numBuf)-1] == '.' {
		iter.ReportError("ReadFloat64", invalidJSONNumberDotNoDigit)
		return
	}

	res, err := parseFloatBytes(numBuf, 64)
	if err != nil {
		iter.ReportError("ReadFloat64", err.Error())
		return
	}
	return res
}

func validateJSONNumber(input []byte) string {
	// strconv.ParseFloat is not validating `1.` or `1.e1`
	if len(input) == 0 {
		return "empty number"
	}
	if input[0] == '-' {
		input = input[1:]
	}
	dotPos := bytes.IndexByte(input, '.')
	if dotPos != -1 {
		if dotPos == len(input)-1 {
			return invalidJSONNumberDotNoDigit
		}
		switch input[dotPos+1] {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		default:
			return invalidJSONNumberDotNoDigit
		}
	}
	return ""
}

// ReadNumber read json.Number
func (iter *Iterator) ReadNumber() (ret json.Number) {
	return json.Number(iter.readNumberAsString())
}

// ReadNumberAsSlice reads a json number into the provided byte slice (can be nil)
func (iter *Iterator) ReadNumberAsSlice(buf []byte) []byte {
	buf = iter.readNumberAsBytes(buf)

	if errMsg := validateJSONNumber(buf); errMsg != "" {
		iter.ReportError("ReadNumberAsSlice", errMsg)
		return nil
	}

	return buf
}
