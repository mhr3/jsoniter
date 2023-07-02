package jsoniter

import (
	"fmt"
)

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

	nullLiteral + firstByteSkippedOffset:  "ull",
	trueLiteral + firstByteSkippedOffset:  "rue",
	falseLiteral + firstByteSkippedOffset: "alse",
}

func (lit jsonLiteral) String() string {
	return literalTable[lit&3]
}

func (lit jsonLiteral) Len() int {
	return len(literalTable[lit&3])
}

func (lit jsonLiteral) EqualBytes(data []byte, headOffset int) bool {
	var offset jsonLiteral
	if headOffset > 0 {
		offset = firstByteSkippedOffset
	}

	return string(data) == literalTable[(lit+offset)%7]
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

	startIdx := iter.head
	endIdx := startIdx + comparerLen

	// quick check if we have enough data buffered
	if iter.tail >= endIdx {
		if lit.EqualBytes(iter.buf[startIdx:endIdx], litOffset) {
			iter.head = endIdx
			return
		}

		iter.ReportError("readLiteral", fmt.Sprintf("expected %s", lit.String()))
		return
	}

	comparer := lit.String()
	comparer = comparer[litOffset:]
	comparerLen = len(comparer) // eliminate the compiler bounds check inside the loop

	for i := 0; i < comparerLen; i++ {
		if iter.readByte() != comparer[i] {
			iter.ReportError("readLiteral", fmt.Sprintf("expected %s", lit.String()))
			return
		}
	}
}
