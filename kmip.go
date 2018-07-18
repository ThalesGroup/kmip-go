package kmip

import (
	"encoding/binary"
	"math/big"
	"time"
	"bytes"
	"io"
	"github.com/ansel1/merry"
	"fmt"
	"strings"
	"regexp"
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

type TTLV2 []byte

func (t TTLV2) Tag() Tag {
	// don't panic if header is truncated
	if len(t) < 3 {
		return Tag(0)
	}
	return Tag(uint32(t[2]) | uint32(t[1])<<8 | uint32(t[0])<<16)
}

func (t TTLV2) Type() Type {
	// don't panic if header is truncated
	if len(t) < 4 {
		return Type(0)
	}
	return Type(t[3])
}

func (t TTLV2) Len() int {
	// don't panic if header is truncated
	if len(t) < lenHeader {
		return 0
	}
	return int(binary.BigEndian.Uint32(t[4:8]))
}

func (t TTLV2) FullLen() int {
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

func (t TTLV2) ValueRaw() []byte {
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

func (t TTLV2) Value() interface{} {
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

func (t TTLV2) ValueInteger() int {
	return int(binary.BigEndian.Uint32(t.ValueRaw()))
}

func (t TTLV2) ValueLongInteger() int64 {
	return int64(binary.BigEndian.Uint64(t.ValueRaw()))
}

func (t TTLV2) ValueBigInteger() *big.Int {
	i := new(big.Int)
	unmarshalBigInt(i, unpadBigInt(t.ValueRaw()))
	return i
}

func (t TTLV2) ValueEnumeration() uint32 {
	return binary.BigEndian.Uint32(t.ValueRaw())
}

func (t TTLV2) ValueBoolean() bool {
	return t.ValueRaw()[7] != 0
}

func (t TTLV2) ValueTextString() string {
	// conveniently, KMIP strings are UTF-8 encoded, as are
	// golang strings
	return string(t.ValueRaw())
}

func (t TTLV2) ValueByteString() []byte {
	return t.ValueRaw()
}

func (t TTLV2) ValueDateTime() time.Time {
	i := t.ValueLongInteger()
	return time.Unix(i, 0).UTC()
}

func (t TTLV2) ValueInterval() time.Duration {
	return time.Duration(binary.BigEndian.Uint32(t.ValueRaw())) * time.Second
}

func (t TTLV2) ValueStructure() TTLV2 {
	return t.ValueRaw()
}



func (t TTLV2) Valid() error {
	if err := t.ValidHeader(); err != nil {
		return err
	}

	if len(t) < t.FullLen() {
		return ErrValueTruncated
	}

	return nil
}

func (t TTLV2) ValidHeader() error {
	if l := len(t); l < lenHeader {
		return ErrHeaderTruncated
	}
	switch t[0] {
	case 0x42, 0x54: // valid
	default:
		return ErrInvalidTag
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
	return nil

}

func (t TTLV2) Next() TTLV2 {
	if t.Valid() != nil {
		return nil
	}
	n := t[t.FullLen():]
	if len(n) == 0 {
		return nil
	}
	return n
}

func (t TTLV2) String() string {
	buf := bytes.NewBuffer(nil)
	Print(buf, "", t)
	return buf.String()
}

func Print(w io.Writer, indent string, t TTLV2) (err error){
	err = t.ValidHeader()

	if err == ErrHeaderTruncated {
		// print the err, and as much of the truncated header as we have
		fmt.Fprintf(w, "%s%s: %#x", indent, err.Error(), []byte(t))
		return
	}

	tag := t.Tag()
	typ := t.Type()
	l := t.Len()
	fmt.Fprintf(w, "%s%v (%s/%d):", indent, tag, typ.String(), l)

	if err != nil {
		// Something was wrong with the header.  Print the err and return
		fmt.Fprintf(w, " %v", err)
		return
	}

	if err = t.Valid(); err != nil {
		// Something is wrong with the value.  Print the error, and the value
		fmt.Fprintf(w, "%s%s: %#x", indent, err.Error(), t.ValueRaw())
		return
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


func (r *Reader) Read() (TTLV2, error) {
	// TODO: if a full value can't be read, this just errors, but it should probably try to keep reading
	// but then, do I need timeouts or something?

	// TODO: re-use buffers
	// pre-allocate buffer large enough to hold most base types
	buf := bytes.NewBuffer(make([]byte, lenHeader + 8))
	n, err := r.r.Read(buf.Bytes()[:lenHeader])
	switch err {
	case nil, io.EOF:
	default:
		return nil, merry.Prepend(err, "reading encBuf")
	}

	if n != lenHeader {
		return nil, merry.New("not enough bytes for a full encBuf")
	}

	t := TTLV2(buf.Bytes()[:lenHeader])
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

	if n + lenHeader != l {
		return nil, merry.New("value truncated")
	}
	return TTLV2(buf.Bytes()[:l]), err
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

// implementation of 5.4.1.1 and 5.5.1.1
func NormalizeName(s string) string {
	// 1. Replace round brackets ‘(‘, ‘)’ with spaces
	s = regexp.MustCompile(`[()]`).ReplaceAllString(s, " ")

	// 2. If a non-word char (not alpha, digit or underscore) is followed by a letter (either upper or lower case) then a lower case letter, replace the non-word char with space
	s = regexp.MustCompile(`(\W)([a-zA-Z][a-z])`).ReplaceAllString(s, " $2")

	words := strings.Split(s, " ")

	for i, w := range words {
		// 3. Replace remaining non-word chars (except whitespace) with underscore.
		w = regexp.MustCompile(`\W`).ReplaceAllString(w, "_")

		if i == 0 {
			// 4. If the first word begins with a digit, move all digits at start of first word to end of first word
			w = regexp.MustCompile(`^([\d]+)(.*)`).ReplaceAllString(w, `$2$1`)
		}

		// 5. Capitalize the first letter of each word
		words[i] = strings.Title(w)
	}

	// 6. Concatenate all words with spaces removed
	return strings.Join(words, "")

}