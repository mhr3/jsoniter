package jsoniter

import "fmt"

type jsonLiteral byte

const (
	nullLiteral jsonLiteral = iota
	trueLiteral
	falseLiteral

	firstByteSkippedOffset = 4
)

var literalTable = [...]string{
	nullLiteral:  "null",
	trueLiteral:  "true",
	falseLiteral: "false",
}

var literalTablePayloads = [...][8]byte{
	nullLiteral:  {'n', 'u', 'l', 'l'},
	trueLiteral:  {'t', 'r', 'u', 'e'},
	falseLiteral: {'f', 'a', 'l', 's', 'e'},

	nullLiteral + firstByteSkippedOffset:  {'u', 'l', 'l'},
	trueLiteral + firstByteSkippedOffset:  {'r', 'u', 'e'},
	falseLiteral + firstByteSkippedOffset: {'a', 'l', 's', 'e'},
}

func (lit jsonLiteral) String() string {
	return literalTable[lit]
}

func (lit jsonLiteral) Len() int {
	return len(literalTable[lit])
}

func (lit jsonLiteral) As8Bytes(headOffset int) [8]byte {
	var offset jsonLiteral
	if headOffset > 0 {
		offset = firstByteSkippedOffset
	}
	return literalTablePayloads[lit+offset]
}

// note that this function expects the head to be at the first byte of the literal
func (iter *Iterator) ensureLiteral(lit jsonLiteral) {
	iter.skipLiteralBytes(lit, 1)
}

func (iter *Iterator) ensureLiteralFull(lit jsonLiteral) {
	iter.skipLiteralBytes(lit, 0)
}

func (iter *Iterator) skipLiteralBytes(lit jsonLiteral, litOffset int) {
	comparerLen := lit.Len() - litOffset

	if iter.tail-iter.head >= comparerLen {
		// the compiler can do this without allocs
		startIdx := iter.head
		endIdx := startIdx + comparerLen

		var buf [8]byte
		copy(buf[:], iter.buf[startIdx:endIdx])

		if buf == lit.As8Bytes(litOffset) {
			//if string(iter.buf[startIdx:endIdx]) == comparer {
			iter.head = endIdx
			return
		}

		iter.ReportError("readLiteral", fmt.Sprintf("expected %s", lit.String()))
		return
	}

	comparer := lit.String()
	comparer = comparer[litOffset:]

	for i := 0; i < comparerLen; i++ {
		if iter.readByte() != comparer[i] {
			iter.ReportError("readLiteral", fmt.Sprintf("expected %s", lit.String()))
			return
		}
	}
}
