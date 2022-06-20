package jsoniter

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

	iter := Iterator{
		buf:  r.buf,
		tail: len(r.buf),
	}
	res := iter.readStringInner()
	if iter.Error != nil {
		// should never happen
		panic(iter.Error)
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
