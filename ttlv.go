package kmip

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/ansel1/merry"
	"io"
	"math/big"
	"time"
)

const lenTag = 3
const lenLen = 4
const lenInt = 4
const lenDateTime = 8
const lenInterval = 4
const lenEnumeration = 4
const lenLongInt = 8
const lenBool = 8
const lenHeader = lenTag + 1 + lenLen // tag + type + len

type TTLV []byte

func (t *TTLV) UnmarshalTTLV(ttlv TTLV, disallowUnknownFields bool) error {
	if ttlv == nil {
		*t = nil
		return nil
	}

	l := len(ttlv)
	if len(*t) < l {
		*t = make([]byte, l)
	} else {
		*t = (*t)[:l]
	}

	copy(*t, ttlv)
	return nil
}

func (t TTLV) Tag() Tag {
	// don't panic if header is truncated
	if len(t) < 3 {
		return Tag(0)
	}
	return Tag(uint32(t[2]) | uint32(t[1])<<8 | uint32(t[0])<<16)
}

func (t TTLV) Type() Type {
	// don't panic if header is truncated
	if len(t) < 4 {
		return Type(0)
	}
	return Type(t[3])
}

func (t TTLV) Len() int {
	// don't panic if header is truncated
	if len(t) < lenHeader {
		return 0
	}
	return int(binary.BigEndian.Uint32(t[4:8]))
}

func (t TTLV) FullLen() int {
	switch t.Type() {
	case TypeInterval, TypeDateTime, TypeBoolean, TypeEnumeration, TypeLongInteger, TypeInteger:
		return lenHeader + 8
	case TypeByteString, TypeTextString:
		l := t.Len() + lenHeader
		if m := l % 8; m > 0 {
			return l + (8 - m)
		}
		return l
	case TypeBigInteger, TypeStructure:
		return t.Len() + lenHeader
	}
	panic(fmt.Sprintf("invalid type: %x", byte(t.Type())))
}

func (t TTLV) ValueRaw() []byte {
	// don't panic if the value is truncated
	l := t.Len()
	if l == 0 {
		return nil
	}
	if len(t) < lenHeader+l {
		return t[lenHeader:]
	}
	return t[lenHeader : lenHeader+l]
}

func (t TTLV) Value() interface{} {
	switch t.Type() {
	case TypeInterval:
		return t.ValueInterval()
	case TypeDateTime:
		return t.ValueDateTime()
	case TypeByteString:
		return t.ValueByteString()
	case TypeTextString:
		return t.ValueTextString()
	case TypeBoolean:
		return t.ValueBoolean()
	case TypeEnumeration:
		return t.ValueEnumeration()
	case TypeBigInteger:
		return t.ValueBigInteger()
	case TypeLongInteger:
		return t.ValueLongInteger()
	case TypeInteger:
		return t.ValueInteger()
	case TypeStructure:
		return t.ValueStructure()
	}
	panic(fmt.Sprintf("invalid type: %x", byte(t.Type())))
}

func (t TTLV) ValueInteger() int {
	return int(binary.BigEndian.Uint32(t.ValueRaw()))
}

func (t TTLV) ValueLongInteger() int64 {
	return int64(binary.BigEndian.Uint64(t.ValueRaw()))
}

func (t TTLV) ValueBigInteger() *big.Int {
	i := new(big.Int)
	unmarshalBigInt(i, unpadBigInt(t.ValueRaw()))
	return i
}

func (t TTLV) ValueEnumeration() uint32 {
	return binary.BigEndian.Uint32(t.ValueRaw())
}

func (t TTLV) ValueBoolean() bool {
	return t.ValueRaw()[7] != 0
}

func (t TTLV) ValueTextString() string {
	// conveniently, KMIP strings are UTF-8 encoded, as are
	// golang strings
	return string(t.ValueRaw())
}

func (t TTLV) ValueByteString() []byte {
	return t.ValueRaw()
}

func (t TTLV) ValueDateTime() time.Time {
	i := t.ValueLongInteger()
	return time.Unix(i, 0).UTC()
}

func (t TTLV) ValueInterval() time.Duration {
	return time.Duration(binary.BigEndian.Uint32(t.ValueRaw())) * time.Second
}

func (t TTLV) ValueStructure() TTLV {
	return t.ValueRaw()
}

func (t TTLV) Valid() error {
	if err := t.ValidHeader(); err != nil {
		return err
	}

	if len(t) < t.FullLen() {
		return ErrValueTruncated
	}

	if t.Type() == TypeStructure {
		inner := t.ValueStructure()
		for {
			if len(inner) <= 0 {
				break
			}
			if err := inner.Valid(); err != nil {
				return merry.Prepend(err, t.Tag().String())
			}
			inner = inner.Next()
		}
	}

	return nil
}

func (t TTLV) validTag() bool {
	switch t[0] {
	case 0x42, 0x54: // valid
		return true
	}
	return false
}

func (t TTLV) ValidHeader() error {
	if l := len(t); l < lenHeader {
		return ErrHeaderTruncated
	}

	switch t.Type() {
	case TypeStructure, TypeTextString, TypeByteString:
		// any length is valid
	case TypeInteger, TypeEnumeration, TypeInterval:
		if t.Len() != lenInt {
			return ErrInvalidLen
		}
	case TypeLongInteger, TypeBoolean, TypeDateTime:
		if t.Len() != lenLongInt {
			return ErrInvalidLen
		}
	case TypeBigInteger:
		if (t.Len() % 8) != 0 {
			return ErrInvalidLen
		}
	default:
		return ErrInvalidType
	}
	if !t.validTag() {
		return ErrInvalidTag
	}
	return nil

}

func (t TTLV) Next() TTLV {
	if t.Valid() != nil {
		return nil
	}
	n := t[t.FullLen():]
	if len(n) == 0 {
		return nil
	}
	return n
}

func (t TTLV) String() string {
	buf := bytes.NewBuffer(nil)
	Print(buf, "", t)
	return buf.String()
}

func Print(w io.Writer, indent string, t TTLV) (err error) {

	tag := t.Tag()
	typ := t.Type()
	l := t.Len()

	fmt.Fprintf(w, "%s%v (%s/%d):", indent, tag, typ.String(), l)

	if err = t.Valid(); err != nil {
		fmt.Fprintf(w, " (%s)", err.Error())
		switch err {
		case ErrHeaderTruncated:
			// print the err, and as much of the truncated header as we have
			fmt.Fprintf(w, " %#x", []byte(t))
			return
		case ErrInvalidLen, ErrValueTruncated:
			// Something is wrong with the value.  Print the error, and the value
			fmt.Fprintf(w, " %#x", t.ValueRaw())
			return
		}
	}

	switch typ {
	case TypeByteString:
		fmt.Fprintf(w, " %#x", t.ValueByteString())
	case TypeStructure:
		indent += "  "
		s := t.ValueStructure()
		for s != nil {
			fmt.Fprint(w, "\n")
			if err = Print(w, indent, s); err != nil {
				// an error means we've hit invalid bytes in the stream
				// there are no markers to pick back up again, so we have to give up
				return
			}
			s = s.Next()
		}
	default:
		fmt.Fprintf(w, " %v", t.Value())
	}
	return
}

type Reader struct {
	r io.Reader
}

func (r *Reader) Read() (TTLV, error) {
	// TODO: if a full value can't be read, this just errors, but it should probably try to keep reading
	// but then, do I need timeouts or something?

	// TODO: re-use buffers
	// pre-allocate buffer large enough to hold most base types
	buf := bytes.NewBuffer(make([]byte, lenHeader+8))
	n, err := r.r.Read(buf.Bytes()[:lenHeader])
	switch err {
	case nil, io.EOF:
	default:
		return nil, merry.Prepend(err, "reading encBuf")
	}

	if n != lenHeader {
		return nil, merry.New("not enough bytes for a full encBuf")
	}

	t := TTLV(buf.Bytes()[:lenHeader])
	if err := t.ValidHeader(); err != nil {
		return t, err
	}
	l := t.FullLen()

	switch {
	case l == 0:
		// TODO: not really clear from the spec whether zero-length TextString and ByteString values are allowed
		return t, err
	case err == io.EOF:
		return t, merry.Errorf("empty value, expecting %d bytes", l)
	}

	buf.Grow(l)

	n, err = r.r.Read(buf.Bytes()[lenHeader:l])
	switch err {
	case nil, io.EOF:
	default:
		return nil, merry.Prepend(err, "reading value")
	}

	if n+lenHeader != l {
		return nil, merry.New("value truncated")
	}
	return TTLV(buf.Bytes()[:l]), err
}

var one = big.NewInt(1)

func unpadBigInt(data []byte) []byte {
	if len(data) < 2 {
		return data
	}

	i := 0
	for ; (i + 1) < len(data); i++ {
		switch {
		// first two cases keep looping, skipping pad bytes
		// pad bytes are all the same bit as
		// the first bit of the next byte
		case data[i] == 0xFF && data[i+1]&0x80 > 1:
		case data[i] == 0x00 && data[i+1]&0x80 == 0:
		default:
			// we've hit a byte that doesn't match the pad pattern
			return data[i:]
		}
	}
	// we've reached the last byte
	return data[i:]
}

// unmarshalBigInt sets the value of n to the big-endian two's complement
// value stored in the given data. If data[0]&80 != 0, the number
// is negative. If data is empty, the result will be 0.
func unmarshalBigInt(n *big.Int, data []byte) {
	n.SetBytes(data)
	if len(data) > 0 && data[0]&0x80 > 0 {
		// first byte is 1, so number is negative.
		// left shifting 1 by the length in bits of the data
		// then subtracting the value from that gives us the
		// twos complement.
		// e.g. if the value is 111111111, then 1 << 8 gives us
		//                     1000000000
		// and 100000000 - 11111111 = 00000001
		n.Sub(n, new(big.Int).Lsh(one, uint(len(data))*8))
	}
}
