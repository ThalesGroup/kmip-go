package kmip

import (
	"io"
	"encoding/binary"
	"github.com/ansel1/merry"
	"math/big"
	"time"
	"bytes"
	"reflect"
	"strings"
	"github.com/go-errors/errors"
	"math"
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

const kmipStructTag = "kmip"

type EnumValuer interface {
	EnumValue() uint32
}

type EnumLiteral struct {
	IntValue    uint32
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
	Tag    Tag
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
	Tag   Tag
	Value interface{}
}

func (t TaggedValue) MarshalTaggedValue(e *Encoder, tag Tag) error {
	// if tag is set, override the suggested tag
	if t.Tag != 0 {
		tag = t.Tag
	}

	return e.EncodeValue(tag, t.Value)
}

func MarshalTTLV(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := NewTTLVEncoder(buf).Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type Encoder struct {
	w      io.Writer
	format formatter
}

func NewTTLVEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

type Marshaler interface {
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
	err := e.encode(TagNone, v)
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

	// try non-reflection encoding first
	err := e.encodeInterfaceValue(tag, v)
	if err == errNoEncoder {
		err = e.encodeReflectValue(tag, reflect.ValueOf(v), 0)
	}
	return err
}

var errNoEncoder = errors.New("no non-reflect encoders")

func (e *Encoder) encodeInterfaceValue(tag Tag, v interface{}) error {
	// these are fast path encoders, which avoid reflect
	// in as many cases as possible.
	//
	// This doesn't provide much performance improvement
	// when encoding fields of a structure by reflection, but
	// for Marshaler implementations, it can mean avoiding
	// reflection altogether, which does provide a good boost
	switch t := v.(type) {
	case nil:
		return nil
	case Marshaler:
		return t.MarshalTaggedValue(e, tag)
	case EnumValuer:
		e.format.EncodeEnum(tag, t)
	case int:
		if t > math.MaxInt32 {
			return tagError(ErrIntOverflow, tag, v)
		}
		e.format.EncodeInt(tag, int32(t))
	case int8:
		e.format.EncodeInt(tag, int32(t))
	case int16:
		e.format.EncodeInt(tag, int32(t))
	case int32:
		e.format.EncodeInt(tag, t)
	case uint:
		if t > math.MaxInt32 {
			return tagError(ErrIntOverflow, tag, v)
		}
		e.format.EncodeInt(tag, int32(t))
	case uint8:
		e.format.EncodeInt(tag, int32(t))
	case uint16:
		e.format.EncodeInt(tag, int32(t))
	case uint32:
		if t > math.MaxInt32 {
			return tagError(ErrIntOverflow, tag, v)
		}
		e.format.EncodeInt(tag, int32(t))
	case bool:
		e.format.EncodeBool(tag, t)
	case int64:
		e.format.EncodeLongInt(tag, t)
	case uint64:
		if t > math.MaxInt64 {
			return tagError(ErrLongIntOverflow, tag, v)
		}
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

	case []interface{}:
		for _, v := range t {
			err := e.EncodeValue(tag, v)
			if err != nil {
				return err
			}
		}
	case uintptr, float32, float64, complex64, complex128:
		return tagError(ErrUnsupportedTypeError, tag, v).Appendf("%T", v)
	default:
		return errNoEncoder
	}
	return nil
}

var byteType = reflect.TypeOf(byte(0))
var marshalerType = reflect.TypeOf((*Marshaler)(nil)).Elem()
var timeType = reflect.TypeOf((*time.Time)(nil)).Elem()
var bigIntPtrType = reflect.TypeOf((*big.Int)(nil))
var bigIntType = bigIntPtrType.Elem()
var durationType = reflect.TypeOf(time.Nanosecond)
var enumValuerType = reflect.TypeOf((*EnumValuer)(nil)).Elem()

var invalidValue = reflect.Value{}
// indirect dives into interfaces values, and one level deep into pointers
// returns an invalid value if the resolved value is nil or invalid
func indirect(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if !v.IsValid() {
		return v
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Func, reflect.Slice,reflect.Map,reflect.Chan,reflect.Ptr,reflect.Interface:
		if v.IsNil() {
			return invalidValue
		}
	}
	return v
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func (e *Encoder) encodeReflectValue(tag Tag, v reflect.Value, flags fieldFlags) error {

	v = indirect(v)
	if !v.IsValid() {
		return nil
	}

	typ := v.Type()

	// check for implementations of Marshaler and EnumValuer
	// need to check for empty values in each branch: if a type implements
	// Marshaler, but is empty && omitempty, it should be skipped.  But
	// if a type doesn't implement Marshaler, then I want it to hit
	// the filter on Kind() first, to return unsupported type errors, before
	// checking for empty.  In other words, if the type doesn't implement
	// marshaler, I want to error on invalid types *before* doing the isEmpty logic.
	switch {
	case typ.Implements(marshalerType):
		if flags&fOmitEmpty != 0 && isEmptyValue(v) {
			return nil
		}
		return v.Interface().(Marshaler).MarshalTaggedValue(e, tag)
	case typ.Implements(enumValuerType):
		if flags&fOmitEmpty != 0 && isEmptyValue(v) {
			return nil
		}
		e.format.EncodeEnum(tag, v.Interface().(EnumValuer))
		return nil
	case v.CanAddr():
		pv := v.Addr()
		pvtyp := pv.Type()
		switch {
		case pvtyp.Implements(marshalerType):
			if flags&fOmitEmpty != 0 && isEmptyValue(v) {
				return nil
			}
			return pv.Interface().(Marshaler).MarshalTaggedValue(e, tag)
		case pvtyp.Implements(enumValuerType):
			if flags&fOmitEmpty != 0 && isEmptyValue(v) {
				return nil
			}
			e.format.EncodeEnum(tag, pv.Interface().(EnumValuer))
			return nil
		}
	}

	switch v.Kind() {
	case reflect.Chan, reflect.Map, reflect.Func, reflect.Ptr, reflect.UnsafePointer, reflect.Uintptr, reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Interface:
			return tagError(ErrUnsupportedTypeError, tag, v).Appendf("%s", v.Type().String())
	}

	if flags&fOmitEmpty != 0 && isEmptyValue(v) {
		return nil
	}

	typeInfo, err := getTypeInfo(typ)
	if err != nil {
		return err
	}
	if typeInfo.tagRequired || tag == TagNone {
		tag = typeInfo.tag
	}

	if tag == TagNone {
		// error, no value tag to use
		return tagError(ErrNoTag, TagNone, v)
	}

	switch typ {
	case timeType, bigIntType, bigIntPtrType, durationType:
		// these are some special types which are handled by the non-reflect path
		return e.encodeInterfaceValue(tag, v.Interface())
	}

	// TODO: basic types
	switch typ.Kind() {
	case reflect.Struct:
		return e.EncodeStructure(tag, func(e *Encoder) error {
			for _, field := range typeInfo.fields {
				fv := v.FieldByIndex(field.index)
				// TODO: check for omitempty

				// note: we're staying in reflection world here instead of
				// converting back to an interface{} value and going through
				// the non-reflection path again.  Calling Interface()
				// on the reflect value would make a potentially addressable value
				// into an unaddressable value, reducing the chances we can coerce
				// the value into a Marshalable.
				//
				// tl;dr
				// Consider a type which implements Marshaler with
				// a pointer receiver, and a struct with a non-pointer field of that type:
				//
				//     type Wheel struct{}
				//     func (*Wheel) MarshalTaggedValue(...)
				//
				//     type Car struct{
				//         Wheel Wheel
				//     }
				//
				// When traversing the Car struct, should the encoder invoke Wheel's
				// Marshaler method, or not?  Technically, the type `Wheel`
				// doesn't implement the Marshaler interface.  Only the type `*Wheel`
				// implements it.  However, the other encoders in the SDK, like JSON
				// and XML, will try, if possible, to get a pointer to field values like this, in
				// order to invoke the Marshaler interface anyway.
				//
				// Encoders can only get a pointer to field values if the field
				// value is `addressable`.  Addressability is explained in the docs for reflect.Value#CanAddr().
				// Using reflection to turn a reflect.Value() back into an interface{}
				// can make a potentially addressable value (like the field of an addressible struct)
				// into an unaddressable value (reflect.Value#Interface{} always returns an unaddressable
				// copy).
				err := e.encodeReflectValue(field.tag, fv, field.flags)
				if err != nil {
					// prepend the field name on the error context path
					errC := GetErrorContext(err)
					if errC != nil {
						errC.Path = append(errC.Path, "")
						copy(errC.Path[1:], errC.Path)
						errC.Path[0] = field.name
					}
					return err
				}
			}
			return nil
		})

	case reflect.String:
		e.format.EncodeTextString(tag, v.String())
	case reflect.Slice:
		switch typ.Elem() {
		case byteType:
			// special case, encode as a ByteString
			e.format.EncodeByteString(tag, v.Bytes())
			return nil
		}
		fallthrough
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			// turn off the omit empty flag.  applies at the field level,
			// not to each member of the slice
			err := e.encodeReflectValue(tag, v.Index(i), flags&^fOmitEmpty)
			if err != nil {
				return err
			}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		i := v.Int()
		if i > math.MaxInt32 {
			return merry.Here(ErrIntOverflow).Prepend(tag.String())
		}
		e.format.EncodeInt(tag, int32(i))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		u := v.Uint()
		if u > math.MaxInt32 {
			return merry.Here(ErrIntOverflow).Prepend(tag.String())
		}
		e.format.EncodeInt(tag, int32(u))
	case reflect.Uint64:
		u := v.Uint()
		if u > math.MaxInt64 {
			return merry.Here(ErrLongIntOverflow).Prepend(tag.String())
		}
		e.format.EncodeLongInt(tag, int64(u))
	case reflect.Int64:
		e.format.EncodeLongInt(tag, int64(v.Int()))
	case reflect.Bool:
		e.format.EncodeBool(tag, v.Bool())
	default:
		// all kinds should have been handled by now
		panic(errors.New("should never get here"))
	}
	// TODO: arrays
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
	h.encodeIntVal(int32(d / time.Second))
	h.Write(h.scratch[:16])
}

func (h *encBuf) EncodeEnum(tag Tag, i EnumValuer) {
	h.encodeHeader(tag, TypeEnumeration, lenEnumeration)
	h.encodeIntVal(int32(i.EnumValue()))
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

func getTypeInfo(typ reflect.Type) (ti typeInfo, err error) {
	// figure out whether this type has a required or suggested kmip tag
	// TODO: required tags support, from a subfield like xml.Name
	ti.tag, err = parseTag(typ.Name(), false)
	if err != nil {
		return
	}

	if typ.Kind() == reflect.Struct {
		ti.fields, err = getFieldsInfo(typ)
	}
	return
}

var errSkip = errors.New("skip")

func getFieldInfo(sf reflect.StructField) (fi fieldInfo, err error) {

	if sf.Anonymous || /*unexported:*/ sf.PkgPath != "" {
		err = errSkip
		return
	}

	parts := strings.Split(sf.Tag.Get(kmipStructTag), ",")
	for i, value := range parts {
		if i == 0 {
			switch value {
			case "-":
				// skip
				err = errSkip
				return
			case "":
			default:
				fi.tag, err = parseTag(value, true)
				if err != nil {
					return
				}
			}
		} else {
			switch value {
			case "enum":
				fi.flags = fi.flags | fEnum
			case "omitempty":
				fi.flags = fi.flags | fOmitEmpty
			}
		}
	}

	if fi.tag == TagNone {
		// try resolving the tag from the field, but this is not required.
		// will fall back on trying to extract the tag from the value if this
		// fails
		fi.tag, err = parseTag(sf.Name, false)
		if err != nil {
			return
		}
	}

	fi.name = sf.Name
	fi.index = sf.Index
	return
}

func getFieldsInfo(typ reflect.Type) (fields []fieldInfo, err error) {
	// TODO: error fields of unsupported types, like maps
	// TODO: error on fields with no candidate tag

	for i := 0; i < typ.NumField(); i++ {
		fi, err := getFieldInfo(typ.Field(i))
		switch err {
		case errSkip:
		case nil:
			fields = append(fields, fi)
		default:
			return nil, err
		}
	}
	return fields, nil
}

func parseTag(tagStr string, required bool) (Tag, error) {
	// parse tag will handle raw hex values, like 0x, or registered
	// canonical tag names.  ignore errors
	t, err := ParseTag(tagStr)
	if Is(err, ErrTagNotRegistered) && required {
		return t, err
	}
	return t, nil
}

type typeInfo struct {
	tag         Tag
	tagRequired bool
	fields      []fieldInfo
}

type fieldFlags int

const (
	fOmitEmpty fieldFlags = 1 << iota
	fEnum
)

type fieldInfo struct {
	name string
	tag       Tag
	index     []int
	flags fieldFlags
	enum      bool
	omitEmpty bool
}
