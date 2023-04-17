package jsoniter

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

type RawString struct {
	buf        []byte
	isRaw      bool
	hasEscapes bool
}

func (r *RawString) IsNil() bool {
	return r.buf == nil
}

// Realize turns a direct view buffer into a copy.
func (r *RawString) Realize() {
	if r.isRaw {
		bufCpy := make([]byte, len(r.buf))
		copy(bufCpy, r.buf)
		r.buf = bufCpy
		r.isRaw = false
	}
}

// String decodes escape sequences and returns the string.
func (r *RawString) String() string {
	if r.buf == nil {
		return ""
	}

	if !r.hasEscapes {
		return string(r.buf[:len(r.buf)-1])
	}

	res, err := unescapeString(r.buf[:len(r.buf)-1])
	if err != nil {
		// the string should have been already checked, so this should never happen
		panic(err)
	}
	return res
}

// Bytes returns a buffer and true if this is a direct view into the iterator,
// or false if the buffer is a copy.
// Note that a direct view buffer is only valid until the next read
// from the iterator. Use Realize before reading further from the iterator
// to preserve the contents.
func (r *RawString) Bytes() ([]byte, bool) {
	raw := r.buf
	if len(raw) > 0 {
		raw = raw[:len(raw)-1]
	}
	return raw, r.isRaw
}

// ContainsEscapes returns true if the string contains escape sequences.
func (r *RawString) ContainsEscapes() bool {
	return r.hasEscapes
}

func unescapeString(buf []byte) (string, error) {
	sb := &strings.Builder{}

	// using a uint here to avoid bounds checks in the loop
	copyStart := uint(0)
	for i := uint(0); i < uint(len(buf)); i++ {
		c := buf[i]
		if c == '\\' {
			sb.Write(buf[copyStart:i])
			n, err := unescapedSequence(sb, buf[i+1:])
			if err != nil {
				return "", err
			}
			i += uint(n)
			copyStart = i + 1
		}
	}
	sb.Write(buf[copyStart:])

	return sb.String(), nil
}

var errInvalidEscape = errors.New(`invalid escape char after \`)

func unescapedSequence(sb *strings.Builder, data []byte) (int, error) {
	r, n := unescapeRune(data)
	if n == 0 {
		return 0, errInvalidEscape
	}

	if !utf16.IsSurrogate(r) {
		sb.WriteRune(r)
		return n, nil
	}

	if len(data) <= n+2 {
		sb.WriteRune(r)
		return n, nil
	}
	data = data[n:]
	if data[0] != '\\' {
		sb.WriteRune(r)
		return n, nil
	}
	data = data[1:]
	r2, n2 := unescapeRune(data)
	if n2 == 0 {
		return 0, errInvalidEscape
	}

	if r2 < utf8.RuneSelf {
		sb.WriteRune(r)
		sb.WriteRune(r2)
		return n + n2 + 1, nil
	}

	res := utf16.DecodeRune(r, r2)
	if res == unicode.ReplacementChar {
		sb.WriteRune(r)
		sb.WriteRune(r2)
		return n + n2 + 1, nil
	}

	sb.WriteRune(res)
	return n + n2 + 1, nil
}

func unescapeRune(data []byte) (rune, int) {
	if len(data) < 1 {
		return 0, 0
	}

	switch c := data[0]; c {
	case 'u':
		if len(data) < 5 {
			return 0, 0
		}
		u4 := u4bufFromBytes(data[1:5])
		r := u4.Parse()
		if r < 0 {
			return 0, 0
		}
		return r, 5
	case '"':
		return '"', 1
	case '\\':
		return '\\', 1
	case '/':
		return '/', 1
	case 'b':
		return '\b', 1
	case 'f':
		return '\f', 1
	case 'n':
		return '\n', 1
	case 'r':
		return '\r', 1
	case 't':
		return '\t', 1
	default:
		return 0, 0
	}
}
