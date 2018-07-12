package kmip

import (
	"io"
	"encoding/binary"
	"github.com/ansel1/merry"
	"math/big"
	"time"
	"bytes"
	"reflect"
	)

// TODO: I'm not crazy about this approach to enums, but I don't have anything better
// Enum values must implement this interface to get correctly encoded as KMIP enum values.
// I don't currently have any other way to distinguish enum values from plain integers.
// All the other base KMIP types map pretty well to base golang types, but this is the
// exception.
//
// EnumValues must have an int value to be correctly encoded to TTLV format, which is why this
// interface only focuses on the int value.  If the encoder requires the int value, and it is
// 0, an error will be thrown.  Enum values also have canonical string values,
// and the xml and json formats allow them to be used instead of the int values.  If the enum value
// implements encoding.TextMarshaler, then the encoder will call that to obtain the string value.
// If the enum value implements encoding.TextUnmarshaler, the decoder will use that the unmarshal
// the string value.  If the decoder is trying to decode the string value, but the value its
// unmarshaling into doesn't implement encoding.TextUnmarshaler, an error will be thrown.
//
// TODO: still more to define here about the behavior when the string value is a hex string
//
// Generally, the encoder and decoder will do their best to adapt to whichever form of the value
// is available and allowed by the situation, otherwise it will throw an error.

type EnumValuer interface {
	EnumValue() uint32
}

type EnumLiteral struct {
	IntValue uint32
	StringValue string
}

func (e *EnumLiteral) UnmarshalText(text []byte) error {
	if e == nil {
		*e = EnumLiteral{}
	}
	e.StringValue = string(text)
	return nil
}

func (e *EnumLiteral) MarshalText() (text []byte, err error) {
	return []byte(e.StringValue), nil
}



func (e EnumLiteral) EnumValue() uint32 {
	return e.IntValue
}

type Structure struct {
	Tag Tag
	Values []interface{}
}

func (s Structure) MarshalTaggedValue(e *Encoder, tag Tag) error {
	if s.Tag != 0 {
		tag = s.Tag
	}

	return e.EncodeStructure(tag, func(encoder *Encoder) error {
		for _, v := range s.Values {
			err := encoder.Encode(v)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

type TaggedValue struct {
	Tag Tag
	Value interface{}
}

func (t TaggedValue) MarshalTaggedValue(e *Encoder, tag Tag) error {
	// if tag is set, override the suggested tag
	if t.Tag != 0 {
		tag = t.Tag
	}

	return e.EncodeValue(tag, t.Value)
}
//
//type Structure struct {
//	tag Tag
//	values []interface{}
//}
//
//func (s *Structure) MarshalTaggedValue(e Encoder, tag Tag) error {
//	// if tag is set, override the suggested tag
//	if s.tag != 0 {
//		tag = s.tag
//	}
//
//	defer e.EncodeStructure(tag)()
//	for _, value := range s.values {
//		err := e.Encode(value)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}

func MarshalTTLV(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := NewTTLVEncoder(buf).Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}



type Encoder struct {
	w io.Writer
	format formatter
}

func NewTTLVEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

type Marshaler interface{
	MarshalTaggedValue(e *Encoder, tag Tag) error
}

func (e *Encoder) EncodeValue(tag Tag, v interface{}) error {
	err := e.encode(tag, v)
	if err != nil {
		return err
	}
	return e.flush()
}

func (e *Encoder) Encode(v interface{}) error {
	err := e.encode(Tag(0), v)
	if err != nil {
		return err
	}
	return e.flush()

}

func (e *Encoder) flush() error {
	if e.format != nil {
		_, err := e.format.WriteTo(e.w)
		return err
	}
	return nil
}

func (e *Encoder) encode(tag Tag, v interface{}) error {
	if e.format == nil {
		e.format = newEncBuf()
	}
	switch t := v.(type) {
	case nil:
		return nil
	case Marshaler:
		err := t.MarshalTaggedValue(e, tag)
		if err != nil {
			return err
		}
	case int:
		e.format.EncodeInt(tag, int32(t))
	case int8:
		e.format.EncodeInt(tag, int32(t))
	case uint8:
		e.format.EncodeInt(tag, int32(t))
	case int16:
		e.format.EncodeInt(tag, int32(t))
	case uint16:
		e.format.EncodeInt(tag, int32(t))
	case int32:
		e.format.EncodeInt(tag, t)
	case uint32:
		e.format.EncodeInt(tag, int32(t))
	case bool:
		e.format.EncodeBool(tag, t)
	case int64:
		e.format.EncodeLongInt(tag, t)
	case uint64:
		e.format.EncodeLongInt(tag, int64(t))
	case time.Time:
		e.format.EncodeDateTime(tag, t)
	case time.Duration:
		e.format.EncodeInterval(tag, t)
	case big.Int:
		e.format.EncodeBigInt(tag, &t)
	case *big.Int:
		e.format.EncodeBigInt(tag, t)
	case string:
		e.format.EncodeTextString(tag, t)
	case []byte:
		e.format.EncodeByteString(tag, t)
	case EnumValuer:
		e.format.EncodeEnum(tag, t.EnumValue())
	case []interface{}:
		// TODO: this should be in the reflect-based encoder, to handle any type of slice
		for _, v := range t {
			err := e.EncodeValue(tag, v)
			if err != nil {
				return err
			}
		}
	default:
		return merry.Errorf("can't encode type: %T", v)
	}
	return nil
}

func (e *Encoder) EncodeStructure(tag Tag, f func(e *Encoder) error) error {
	if e.format == nil {
		e.format = newEncBuf()
	}
	parentFormat := e.format
	defer func() {
		e.format = parentFormat
	}()
	var err error
	e.format.EncodeStructure(tag, func(childFormat formatter) {
		// don't flush to buffer while building the body of the struct
		e.format = noWriteFormat{childFormat}
		err = f(e)
	})
	if err != nil {
		return err
	}
	return e.flush()
}

// encBuf is a scratch space for the encoder.  Must be at least 16 long, to hold
// 8 byte header + up to 8 byte values
type encBuf struct {
	bytes.Buffer
	// enough to hold an entire TTLV for most base types
	scratch [16]byte
}

func (h *encBuf) EncodeStructure(tag Tag, f func(formatter)) {
	h.encodeHeader(tag, TypeStructure, 0)
	i := h.Len()
	h.Write(h.scratch[:8])
	f(h)
	binary.BigEndian.PutUint32(h.scratch[:4], uint32(h.Len()-lenHeader-i))
	copy(h.Bytes()[i+4:], h.scratch[:4])
}

func newEncBuf() *encBuf {
	return &encBuf{}
}

func (h *encBuf) encodeHeader(tag Tag, p Type, l uint32) {
	h.scratch[0] = byte(tag >> 16)
	h.scratch[1] = byte(tag >> 8)
	h.scratch[2] = byte(tag)
	h.scratch[3] = byte(p)
	binary.BigEndian.PutUint32(h.scratch[4:8], l)
}

var ones = [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
var zeros = [8]byte{}

func (h *encBuf) EncodeBigInt(tag Tag, i *big.Int) {
	start := h.Len()
	// write out 8 bytes of random, allocating the space where
	// the header will be written later
	h.Write(h.scratch[:8])

	switch i.Sign() {
	case 0:
		h.Write(zeros[:8])
	case 1:
		b := i.Bytes()
		l := len(b)
		// if n is positive, but the first bit is a 1, it will look like
		// a negative in 2's complement, so prepend zeroes in front
		if b[0]&0x80 > 0 {
			h.WriteByte(byte(0))
			l++
		}
		// pad front with zeros to multiple of 8
		if m := l % 8; m > 0 {
			h.Write(zeros[:8-m])
		}
		h.Write(b)
	case -1:
		length := uint(i.BitLen()/8+1) * 8
		j := new(big.Int).Lsh(one, length)
		b := j.Add(i, j).Bytes()
		// When the most significant bit is on a byte
		// boundary, we can get some extra significant
		// bits, so strip them off when that happens.
		if len(b) >= 2 && b[0] == 0xff && b[1]&0x80 != 0 {
			b = b[1:]
		}
		l := len(b)
		// pad front with ones to multiple of 8
		if m := l % 8; m > 0 {
			h.Write(ones[:8-m])
		}
		h.Write(b)
	}
	// now calculate the length and encode the header
	h.encodeHeader(tag, TypeBigInteger, uint32(h.Len()-lenHeader-start))
	// write the header in the 8 bytes we allocated above
	copy(h.Bytes()[start:], h.scratch[:8])
}

func (h *encBuf) EncodeInt(tag Tag, i int32) {
	h.encodeHeader(tag, TypeInteger, lenInt)
	h.encodeIntVal(i)
	h.Write(h.scratch[:16])
}

func (h *encBuf) encodeIntVal(i int32) {
	binary.BigEndian.PutUint32(h.scratch[8:12], uint32(i))
	// pad extra bytes
	for i := 12; i < 16; i++ {
		h.scratch[i] = 0
	}
}

func (h *encBuf) EncodeBool(tag Tag, b bool) {
	h.encodeHeader(tag, TypeBoolean, lenBool)
	if b {
		h.encodeLongIntVal(1)
	} else {
		h.encodeLongIntVal(0)
	}
	h.Write(h.scratch[:16])
}

func (h *encBuf) EncodeLongInt(tag Tag, i int64) {
	h.encodeHeader(tag, TypeLongInteger, lenLongInt)
	h.encodeLongIntVal(i)
	h.Write(h.scratch[:16])
}

func (h *encBuf) encodeLongIntVal(i int64) {
	binary.BigEndian.PutUint64(h.scratch[8:], uint64(i))
}

func (h *encBuf) EncodeDateTime(tag Tag, t time.Time) {
	h.encodeHeader(tag, TypeDateTime, lenDateTime)
	h.encodeLongIntVal(t.Unix())
	h.Write(h.scratch[:16])
}

func (h *encBuf) EncodeInterval(tag Tag, d time.Duration) {
	h.encodeHeader(tag, TypeInterval, lenInterval)
	h.encodeIntVal(int32(d/time.Second))
	h.Write(h.scratch[:16])
}

func (h *encBuf) EncodeEnum(tag Tag, i uint32) {
	h.encodeHeader(tag, TypeEnumeration, lenEnumeration)
	h.encodeIntVal(int32(i))
	h.Write(h.scratch[:16])
}

func (h *encBuf) EncodeTextString(tag Tag, s string) {
	start := h.Len()
	// write out 8 bytes of random, allocating the space where
	// the header will be written later
	h.Write(h.scratch[:8])

	n, _ := h.WriteString(s)
	if m := n % 8; m > 0 {
		h.Write(zeros[:8-m])
	}
	h.encodeHeader(tag, TypeTextString, uint32(n))
	copy(h.Bytes()[start:], h.scratch[:8])
}

func (h *encBuf) EncodeByteString(tag Tag, b []byte) {
	start := h.Len()
	// write out 8 bytes of random, allocating the space where
	// the header will be written later
	h.Write(h.scratch[:8])

	n, _ := h.Write(b)
	if m := n % 8; m > 0 {
		h.Write(zeros[:8-m])
	}
	h.encodeHeader(tag, TypeByteString, uint32(n))
	copy(h.Bytes()[start:], h.scratch[:8])
}

type fieldInfo struct {
	index []int
	typ reflect.Type
	sf reflect.StructField
	fieldTag string
	fieldName string
	tagName string
	omitEmpty bool
}

/*func GetFieldInfo(v interface{}) ([]fieldInfo,error) {
	fields := map[string]fieldInfo{}
	rv := reflect.ValueOf(v)
	t := rv.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.Anonymous {
			// not handling anon fields at this time
			continue
		}
		if !isExported(&sf) {
			// skip unexported fields
			continue
		}

		fieldTag := sf.Tag.Get("kmip")
		if fieldTag == "-" {
			// skip
			continue
		}

		fi := fieldInfo{
			index: sf.Index,
			typ:sf.Type,
			fieldName:NormalizeName(sf.Name),
			fieldTag: fieldTag,
		}

		if fi.fieldTag != "" {
			opts := strings.Split(fi.fieldTag, ",")
			if len(opts) > 0 {
				fi.tagName = NormalizeName(opts[0])
			}
			for _, opt := range opts[1:] {
				switch opt {
				case "omitempty":
					fi.omitEmpty = true
				}
			}
		}

		// resolve a valid KMIP tag
		if fi.tagName != "" {
			kmipTag, err := ParseTag(fi.tagName)
			if err != nil {
				return nil, err
			}

			// if the kmip tag was in the field tag, this takes precedence over a prior
			// field where the tag was inferred from the field name
		}



	}
}*/

func isExported(sf *reflect.StructField) bool {
	return sf.PkgPath == ""
}