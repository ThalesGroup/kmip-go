package ttlv

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ansel1/merry"
	"io"
	"math"
	"math/big"
	"reflect"
	"strings"
	"time"
)

type DateTimeExtended struct {
	time.Time
}

func (t *DateTimeExtended) UnmarshalTTLV(d *Decoder, ttlv TTLV) error {
	if len(ttlv) == 0 {
		return nil
	}

	if t == nil {
		*t = DateTimeExtended{}
	}

	err := d.DecodeValue(&t.Time, ttlv)
	if err != nil {
		return err
	}
	return nil
}

func (t DateTimeExtended) MarshalTTLV(e *Encoder, tag Tag) error {
	e.EncodeDateTimeExtended(tag, t.Time)
	return nil
}

const kmipStructTag = "kmip"

var ErrIntOverflow = fmt.Errorf("value exceeds max int value %d", math.MaxInt32)
var ErrLongIntOverflow = fmt.Errorf("value exceeds max long int value %d", math.MaxInt64)
var ErrUnsupportedEnumTypeError = errors.New("unsupported type for enums, must be string, or int types")
var ErrUnsupportedTypeError = errors.New("marshaling/unmarshaling is not supported for this type")
var ErrNoTag = errors.New("unable to determine tag for field")
var ErrTagConflict = errors.New("")

func Marshal(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := NewEncoder(buf).Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type Marshaler interface {
	MarshalTTLV(e *Encoder, tag Tag) error
}

type EnumValue uint32

type TaggedValue struct {
	Tag   Tag
	Value interface{}
}

func (t TaggedValue) MarshalTTLV(e *Encoder, tag Tag) error {
	// if tag is set, override the suggested tag
	if t.Tag != TagNone {
		tag = t.Tag
	}

	return e.EncodeValue(tag, t.Value)
}

type Structure struct {
	Tag    Tag
	Values []interface{}
}

func (s Structure) MarshalTTLV(e *Encoder, tag Tag) error {
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

type Encoder struct {
	encodeDepth int
	w           io.Writer
	encBuf      encBuf

	// these fields store where the encoder is when marshaling a nested struct.  its
	// used to construct error messages.
	currStruct string
	currField  string
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (e *Encoder) Encode(v interface{}) error {
	return e.EncodeValue(TagNone, v)
}

func (e *Encoder) EncodeValue(tag Tag, v interface{}) error {
	err := e.encode(tag, reflect.ValueOf(v), 0)
	if err != nil {
		return err
	}
	return e.Flush()
}

func (e *Encoder) EncodeStructure(tag Tag, f func(e *Encoder) error) error {
	e.encodeDepth++
	i := e.encBuf.begin(tag, TypeStructure)
	err := f(e)
	e.encBuf.end(i)
	e.encodeDepth--
	return err
}

func (e *Encoder) EncodeEnumeration(tag Tag, v uint32) {
	e.encBuf.encodeEnum(tag, v)
}

func (e *Encoder) EncodeInt(tag Tag, v int32) {
	e.encBuf.encodeInt(tag, v)
}

func (e *Encoder) EncodeLongInt(tag Tag, v int64) {
	e.encBuf.encodeLongInt(tag, v)
}

func (e *Encoder) EncodeInterval(tag Tag, v time.Duration) {
	e.encBuf.encodeInterval(tag, v)
}

func (e *Encoder) EncodeDateTime(tag Tag, v time.Time) {
	e.encBuf.encodeDateTime(tag, v)
}

func (e *Encoder) EncodeDateTimeExtended(tag Tag, v time.Time) {
	e.encBuf.encodeDateTimeExtended(tag, v)
}

func (e *Encoder) EncodeBigInt(tag Tag, v *big.Int) {
	e.encBuf.encodeBigInt(tag, v)
}

func (e *Encoder) EncodeBool(tag Tag, v bool) {
	e.encBuf.encodeBool(tag, v)
}

func (e *Encoder) EncodeTextString(tag Tag, v string) {
	e.encBuf.encodeTextString(tag, v)
}

func (e *Encoder) EncodeByteString(tag Tag, v []byte) {
	e.encBuf.encodeByteString(tag, v)
}

func (e *Encoder) Flush() error {
	if e.encodeDepth > 0 {
		return nil
	}
	_, err := e.encBuf.WriteTo(e.w)
	e.encBuf.Reset()
	return err
}

type MarshalerError struct {
	Type   reflect.Type
	Struct string
	Field  string
	Tag    Tag
}

func (e *MarshalerError) Error() string {
	msg := "kmip: error marshaling value"
	if e.Type != nil {
		msg += " of type " + e.Type.String()
	}
	if e.Struct != "" {
		msg += " in struct field " + e.Struct + "." + e.Field
	}
	return msg
}

func (e *Encoder) marshalingError(tag Tag, t reflect.Type, cause error) merry.Error {
	err := &MarshalerError{
		Type:   t,
		Struct: e.currStruct,
		Field:  e.currField,
		Tag:    tag,
	}
	return merry.WrapSkipping(err, 1).WithCause(cause)
}

func (e *Encoder) encodeInt32(tag Tag, i int32) {
	if IsEnumeration(tag) {
		e.encBuf.encodeEnum(tag, uint32(i))
		return
	}
	e.encBuf.encodeInt(tag, i)
}

func (e *Encoder) encodeInt64(tag Tag, i int64) {
	if IsEnumeration(tag) {
		e.encBuf.encodeEnum(tag, uint32(i))
		return
	}
	e.encBuf.encodeLongInt(tag, i)
}

var byteType = reflect.TypeOf(byte(0))
var marshalerType = reflect.TypeOf((*Marshaler)(nil)).Elem()
var unmarshalerType = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
var timeType = reflect.TypeOf((*time.Time)(nil)).Elem()
var bigIntPtrType = reflect.TypeOf((*big.Int)(nil))
var bigIntType = bigIntPtrType.Elem()
var durationType = reflect.TypeOf(time.Nanosecond)
var ttlvType = reflect.TypeOf((*TTLV)(nil)).Elem()
var enumValueType = reflect.TypeOf((*EnumValue)(nil)).Elem()

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
	case reflect.Func, reflect.Slice, reflect.Map, reflect.Chan, reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return invalidValue
		}
	}
	return v
}

var zeroBigInt = big.Int{}

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

	switch v.Type() {
	case timeType:
		return v.Interface().(time.Time).IsZero()
	case bigIntType:
		i := v.Interface().(big.Int)
		return zeroBigInt.Cmp(&i) == 0
	}
	return false
}

func (e *Encoder) encodeReflectEnum(tag Tag, v reflect.Value) error {
	switch v.Kind() {
	case reflect.String:
		// TODO: if there is a one-to-one relationship between an enum and a tag, we could have
		// a registry allowing us to translate named enum values to encodings.  For now, string values
		// can only be encoded as an enum if they are hex strings starting with 0x
		s := v.String()
		if !strings.HasPrefix(s, "0x") {
			return e.marshalingError(tag, v.Type(), ErrInvalidHexString).Append("string enum values must be hex strings starting with 0x")
		}
		s = s[2:]
		if len(s) != 8 {
			return e.marshalingError(tag, v.Type(), ErrInvalidHexString).Appendf("invalid length, must be 8 (4 bytes), got %d", len(s))
		}
		b, err := hex.DecodeString(s)
		if err != nil {
			return e.marshalingError(tag, v.Type(), merry.WithCause(ErrInvalidHexString, err))
		}

		u := binary.BigEndian.Uint32(b)
		e.encBuf.encodeEnum(tag, u)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i := v.Uint()
		if i > math.MaxUint32 {
			return e.marshalingError(tag, v.Type(), ErrIntOverflow)
		}
		e.encBuf.encodeEnum(tag, uint32(i))
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := v.Int()
		if i > math.MaxUint32 {
			return e.marshalingError(tag, v.Type(), ErrIntOverflow)
		}
		e.encBuf.encodeEnum(tag, uint32(i))
		return nil
	default:
		return e.marshalingError(tag, v.Type(), ErrUnsupportedEnumTypeError)
	}
}

func (e *Encoder) encode(tag Tag, v reflect.Value, flags fieldFlags) error {

	// if pointer or interface
	v = indirect(v)
	if !v.IsValid() {
		return nil
	}

	typ := v.Type()

	if typ == ttlvType {
		// fast path: if the value is TTLV, we write it directly to the output buffer
		_, err := e.encBuf.Write(v.Bytes())
		return err
	}

	// resolve the tag, choosing the first of these which isn't TagNone:
	// 1. the tag required the type
	// 2. the requested tag arg
	// 3. the tag inferred from the type
	typeInfo, err := getTypeInfo(typ)
	if err != nil {
		return err
	}
	if typeInfo.tagRequired || tag == TagNone {
		tag = typeInfo.tag
	}

	// check for Marshaler
	switch {
	case typ.Implements(marshalerType):
		if flags&fOmitEmpty != 0 && isEmptyValue(v) {
			return nil
		}
		return v.Interface().(Marshaler).MarshalTTLV(e, tag)
	case v.CanAddr():
		pv := v.Addr()
		pvtyp := pv.Type()
		switch {
		case pvtyp.Implements(marshalerType):
			if flags&fOmitEmpty != 0 && isEmptyValue(v) {
				return nil
			}
			return pv.Interface().(Marshaler).MarshalTTLV(e, tag)
		}
	}

	// If the type doesn't implement Marshaler, then validate the value is a supported kind
	switch v.Kind() {
	case reflect.Chan, reflect.Map, reflect.Func, reflect.Ptr, reflect.UnsafePointer, reflect.Uintptr, reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Interface:
		return e.marshalingError(tag, v.Type(), ErrUnsupportedTypeError)
	}

	// skip if value is empty and tags include omitempty
	if flags&fOmitEmpty != 0 && isEmptyValue(v) {
		return nil
	}

	// handle the enum flag
	if flags&fEnum != 0 {
		return e.encodeReflectEnum(tag, v)
	}

	switch typ {
	case enumValueType:
		e.encBuf.encodeEnum(tag, uint32(v.Uint()))
		return nil
	case timeType:
		e.encBuf.encodeDateTime(tag, v.Interface().(time.Time))
		return nil
	case bigIntType:
		bi := v.Interface().(big.Int)
		e.encBuf.encodeBigInt(tag, &bi)
		return nil
	case bigIntPtrType:
		e.encBuf.encodeBigInt(tag, v.Interface().(*big.Int))
		return nil
	case durationType:
		e.encBuf.encodeInterval(tag, time.Duration(v.Int()))
		return nil
	}

	switch typ.Kind() {
	case reflect.Struct:
		// push current struct onto stack
		currStruct := e.currStruct
		e.currStruct = typ.Name()

		fields, err := getFieldsInfo(typ)
		if err != nil {
			return err
		}
		err = e.EncodeStructure(tag, func(e *Encoder) error {
			for _, field := range fields {
				fv := v.FieldByIndex(field.index)

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
				//     func (*Wheel) MarshalTTLV(...)
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

				// push the currField
				currField := e.currField
				e.currField = field.name
				err := e.encode(field.tag, fv, field.flags)
				// pop the currField
				e.currField = currField
				if err != nil {
					return err
				}
			}
			return nil
		})
		// pop current struct
		e.currStruct = currStruct
		return err
	case reflect.String:
		e.encBuf.encodeTextString(tag, v.String())
	case reflect.Slice:
		switch typ.Elem() {
		case byteType:
			// special case, encode as a ByteString
			e.encBuf.encodeByteString(tag, v.Bytes())
			return nil
		}
		fallthrough
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			// turn off the omit empty flag.  applies at the field level,
			// not to each member of the slice
			err := e.encode(tag, v.Index(i), flags&^fOmitEmpty)
			if err != nil {
				return err
			}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		i := v.Int()
		if i > math.MaxInt32 {
			return merry.Here(ErrIntOverflow).Prepend(tag.String())
		}
		e.encodeInt32(tag, int32(i))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		u := v.Uint()
		if u > math.MaxInt32 {
			return merry.Here(ErrIntOverflow).Prepend(tag.String())
		}
		e.encodeInt32(tag, int32(u))
	case reflect.Uint64:
		u := v.Uint()
		if u > math.MaxInt64 {
			return merry.Here(ErrLongIntOverflow).Prepend(tag.String())
		}
		e.encodeInt64(tag, int64(u))
	case reflect.Int64:
		e.encodeInt64(tag, int64(v.Int()))
	case reflect.Bool:
		e.encBuf.encodeBool(tag, v.Bool())
	default:
		// all kinds should have been handled by now
		panic(errors.New("should never get here"))
	}
	return nil

}

// encBuf encodes basic KMIP types into TTLV
type encBuf struct {
	bytes.Buffer
}

func (h *encBuf) begin(tag Tag, typ Type) int {
	_ = h.WriteByte(byte(tag >> 16))
	_ = h.WriteByte(byte(tag >> 8))
	_ = h.WriteByte(byte(tag))
	_ = h.WriteByte(byte(typ))
	_, _ = h.Write(zeros[:4])
	return h.Len()
}

func (h *encBuf) end(i int) {
	n := h.Len() - i
	if m := n % 8; m > 0 {
		_, _ = h.Write(zeros[:8-m])
	}
	binary.BigEndian.PutUint32(h.Bytes()[i-4:], uint32(n))
}

func (h *encBuf) writeLongIntVal(tag Tag, typ Type, i int64) {
	s := h.begin(tag, typ)
	ll := h.Len()
	_, _ = h.Write(zeros[:8])
	binary.BigEndian.PutUint64(h.Bytes()[ll:], uint64(i))
	h.end(s)
}

func (h *encBuf) writeIntVal(tag Tag, typ Type, val uint32) {
	s := h.begin(tag, typ)
	ll := h.Len()
	_, _ = h.Write(zeros[:4])
	binary.BigEndian.PutUint32(h.Bytes()[ll:], val)
	h.end(s)
}

var ones = [8]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
var zeros = [8]byte{}

func (h *encBuf) encodeBigInt(tag Tag, i *big.Int) {
	if i == nil {
		return
	}

	ii := h.begin(tag, TypeBigInteger)

	switch i.Sign() {
	case 0:
		_, _ = h.Write(zeros[:8])
	case 1:
		b := i.Bytes()
		l := len(b)
		// if n is positive, but the first bit is a 1, it will look like
		// a negative in 2's complement, so prepend zeroes in front
		if b[0]&0x80 > 0 {
			_ = h.WriteByte(byte(0))
			l++
		}
		// pad front with zeros to multiple of 8
		if m := l % 8; m > 0 {
			_, _ = h.Write(zeros[:8-m])
		}
		_, _ = h.Write(b)
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
			_, _ = h.Write(ones[:8-m])
		}
		_, _ = h.Write(b)
	}
	h.end(ii)
}

func (h *encBuf) encodeInt(tag Tag, i int32) {
	h.writeIntVal(tag, TypeInteger, uint32(i))
}

func (h *encBuf) encodeBool(tag Tag, b bool) {
	if b {
		h.writeLongIntVal(tag, TypeBoolean, 1)
	} else {
		h.writeLongIntVal(tag, TypeBoolean, 0)
	}
}

func (h *encBuf) encodeLongInt(tag Tag, i int64) {
	h.writeLongIntVal(tag, TypeLongInteger, i)
}

func (h *encBuf) encodeDateTime(tag Tag, t time.Time) {
	h.writeLongIntVal(tag, TypeDateTime, t.Unix())
}

func (h *encBuf) encodeDateTimeExtended(tag Tag, t time.Time) {
	// take unix seconds, times a million, to get microseconds, then
	// add nanoseconds remainder/1000
	//
	// this gives us a larger ranger of possible values than just t.UnixNano() / 1000.
	// see UnixNano() docs for its limits.
	//
	// this is limited to max(int64) *microseconds* from epoch, rather than
	// max(int64) nanoseconds like UnixNano().
	m := (t.Unix() * 1000000) + int64(t.Nanosecond()/1000)
	h.writeLongIntVal(tag, TypeDateTimeExtended, m)
}

func (h *encBuf) encodeInterval(tag Tag, d time.Duration) {
	h.writeIntVal(tag, TypeInterval, uint32(d/time.Second))
}

func (h *encBuf) encodeEnum(tag Tag, i uint32) {
	h.writeIntVal(tag, TypeEnumeration, i)
}

func (h *encBuf) encodeTextString(tag Tag, s string) {
	i := h.begin(tag, TypeTextString)
	_, _ = h.WriteString(s)
	h.end(i)
}

func (h *encBuf) encodeByteString(tag Tag, b []byte) {
	if b == nil {
		return
	}
	i := h.begin(tag, TypeByteString)
	_, _ = h.Write(b)
	h.end(i)
}

func getTypeInfo(typ reflect.Type) (ti typeInfo, err error) {
	// figure out whether this type has a required or suggested kmip tag
	// TODO: required tags support, from a subfield like xml.Name
	ti.tag, _ = ParseTag(typ.Name())
	ti.typ = typ
	return
}

var errSkip = errors.New("skip")

func getFieldInfo(typ reflect.Type, sf reflect.StructField) (fi fieldInfo, err error) {

	// skip anonymous and unexported fields
	if sf.Anonymous || /*unexported:*/ sf.PkgPath != "" {
		err = errSkip
		return
	}

	// handle field tags
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
				fi.tag, err = ParseTag(value)
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

	// extract type info for the field.  The KMIP tag
	// for this field is derived from either the field name,
	// the field tags, or the field type.
	fi.ti, err = getTypeInfo(sf.Type)
	if err != nil {
		return
	}

	// order of precedence for field tag:
	// 1. explicit field tag (which must match the type's tag if required)
	// 2. field name
	// 3. field type

	// if the field type requires a tag, which doesn't match the tag
	// encoded in the field tag, throw an error
	if fi.ti.tagRequired && fi.tag != TagNone && fi.ti.tag != fi.tag {
		err := &MarshalerError{
			Type:   sf.Type,
			Struct: typ.Name(),
			Field:  sf.Name,
		}
		return fi, merry.WithCause(err, ErrTagConflict).Appendf(`field tag "%s" conflicts type's tag "%s"`, fi.tag, fi.ti.tag)
	}

	if fi.tag == TagNone {
		// try resolving the tag from the field name, but this is not required.
		// will fall back on trying to extract the tag from the value if this
		// fails
		fi.tag, _ = ParseTag(sf.Name)
	}

	if fi.tag == TagNone {
		fi.tag = fi.ti.tag
	}

	if fi.tag == TagNone {
		err := &MarshalerError{
			Type:   sf.Type,
			Struct: typ.Name(),
			Field:  sf.Name,
		}
		return fi, merry.WithCause(err, ErrNoTag)
	}

	fi.name = sf.Name
	fi.structType = typ
	fi.index = sf.Index
	fi.slice = sf.Type.Kind() == reflect.Slice
	return
}

func getFieldsInfo(typ reflect.Type) (fields []fieldInfo, err error) {

	for i := 0; i < typ.NumField(); i++ {
		fi, err := getFieldInfo(typ, typ.Field(i))
		switch err {
		case errSkip:
		case nil:
			fields = append(fields, fi)
		default:
			return nil, err
		}
	}

	// verify that multiple fields don't have the same tag
	names := map[Tag]string{}
	for _, f := range fields {
		if fname, ok := names[f.tag]; ok {
			err := &MarshalerError{
				Type:   f.ti.typ,
				Struct: typ.Name(),
				Field:  f.name,
				Tag:    f.tag,
			}
			return fields, merry.WithCause(err, ErrTagConflict).Appendf("field resolves to the same tag (%s) as other field (%s)", f.tag, fname)
		}
		names[f.tag] = f.name
	}

	return fields, nil
}

type typeInfo struct {
	typ         reflect.Type
	tag         Tag
	tagRequired bool
}

type fieldFlags int

const (
	fOmitEmpty fieldFlags = 1 << iota
	fEnum
)

type fieldInfo struct {
	structType reflect.Type
	name       string
	tag        Tag
	index      []int
	flags      fieldFlags
	enum       bool
	omitEmpty  bool
	slice      bool
	ti         typeInfo
}
