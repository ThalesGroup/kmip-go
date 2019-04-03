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

const structFieldTag = "ttlv"

var ErrIntOverflow = fmt.Errorf("value exceeds max int value %d", math.MaxInt32)
var ErrLongIntOverflow = fmt.Errorf("value exceeds max long int value %d", math.MaxInt64)
var ErrUnsupportedEnumTypeError = errors.New("unsupported type for enums, must be string, or int types")
var ErrUnsupportedTypeError = errors.New("marshaling/unmarshaling is not supported for this type")
var ErrNoTag = errors.New("unable to determine tag for field")
var ErrTagConflict = errors.New("tag conflict")

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

func (v EnumValue) MarshalTTLV(e *Encoder, tag Tag) error {
	e.EncodeEnumeration(tag, uint32(v))
	return nil
}

// Value is a go-typed mapping for a TTLV value.  It holds a tag, and the value in
// the form of a native go type.
//
// Value supports marshaling and unmarshaling, allowing a mapping between encoded TTLV
// bytes and native go types.
//
// TTLV Structure types are mapped to the Values go type.  When marshaling, if the Value
// field is set to a Values{}, the resulting TTLV will be TypeStructure.  When unmarshaling
// a TTLV with TypeStructure, the Value field will contain a Values{}.
type Value struct {
	Tag   Tag
	Value interface{}
}

// UnmarshalTTLV implements Unmarshaler
func (t *Value) UnmarshalTTLV(d *Decoder, ttlv TTLV) error {
	t.Tag = ttlv.Tag()
	switch ttlv.Type() {
	case TypeStructure:
		var v Values

		ttlv = ttlv.ValueStructure()
		for ttlv.Valid() == nil {
			err := d.DecodeValue(&v, ttlv)
			if err != nil {
				return err
			}
			ttlv = ttlv.Next()
		}

		t.Value = v
	default:
		t.Value = ttlv.Value()
	}
	return nil
}

// MarshalTTLV implements Marshaler
func (t Value) MarshalTTLV(e *Encoder, tag Tag) error {
	// if tag is set, override the suggested tag
	if t.Tag != TagNone {
		tag = t.Tag
	}

	if tvs, ok := t.Value.(Values); ok {
		return e.EncodeStructure(tag, func(e *Encoder) error {
			for _, v := range tvs {
				if err := e.Encode(v); err != nil {
					return err
				}
			}
			return nil
		})
	}

	return e.EncodeValue(tag, t.Value)
}

// Values is a slice of Value objects.  It represents the body of a TTLV with a type of Structure.
type Values []Value

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
	err := e.encode(tag, reflect.ValueOf(v), nil)
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
var tagType = reflect.TypeOf(Tag(0))

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

func (e *Encoder) encode(tag Tag, v reflect.Value, fi *fieldInfo) error {

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

	typeInfo, err := getTypeInfo(typ)
	if err != nil {
		return err
	}
	if tag == TagNone {
		tag = tagForMarshal(v, typeInfo, fi)
	}

	var flags fieldFlags
	if fi != nil {
		flags = fi.flags
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

	// recurse to handle slices of values
	switch v.Kind() {
	case reflect.Slice:
		if typ.Elem() == byteType {
			// special case, encode as a ByteString, handled below
			break
		}
		fallthrough
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			// turn off the omit empty flag.  applies at the field level,
			// not to each member of the slice
			// TODO: is this true?
			var fi2 *fieldInfo
			if fi != nil {
				fi2 = &(*fi)
				fi2.flags = fi2.flags &^ fOmitEmpty
			}
			err := e.encode(tag, v.Index(i), fi2)
			if err != nil {
				return err
			}
		}
		return nil
	}

	if tag == TagNone {
		return e.marshalingError(tag, v.Type(), ErrNoTag)
	}

	// handle the enum flag
	if flags&fEnum != 0 {
		return e.encodeReflectEnum(tag, v)
	}

	switch typ {
	case timeType:
		if flags&fDateTimeExtended != 0 {
			e.encBuf.encodeDateTimeExtended(tag, v.Interface().(time.Time))
		} else {
			e.encBuf.encodeDateTime(tag, v.Interface().(time.Time))
		}
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

		err = e.EncodeStructure(tag, func(e *Encoder) error {
			for _, field := range typeInfo.valueFields {
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
				err := e.encode(TagNone, fv, &field)
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
		// special case, encode as a ByteString
		// all slices which aren't []byte should have been handled above
		// the call to v.Bytes() will panic if this assumption is wrong
		e.encBuf.encodeByteString(tag, v.Bytes())
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

func tagForMarshal(v reflect.Value, ti typeInfo, fi *fieldInfo) Tag {
	// the tag on the TTLVTag field
	if ti.tagField != nil && ti.tagField.explicitTag != TagNone {
		return ti.tagField.explicitTag
	}

	// the value of the TTLVTag field of type Tag
	if v.IsValid() && ti.tagField != nil && ti.tagField.ti.typ == tagType {
		tag := v.FieldByIndex(ti.tagField.index).Interface().(Tag)
		if tag != TagNone {
			return tag
		}
	}

	// if value is in a struct field, infer the tag from the field
	// else infer from the value's type name
	if fi != nil {
		return fi.tag
	} else {
		return ti.inferredTag
	}
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
	ti.inferredTag, _ = ParseTag(typ.Name())
	ti.typ = typ
	err = ti.getFieldsInfo()
	return ti, err
}

var errSkip = errors.New("skip")

func getFieldInfo(typ reflect.Type, sf reflect.StructField) (fi fieldInfo, err error) {

	// skip anonymous and unexported fields
	if sf.Anonymous || /*unexported:*/ sf.PkgPath != "" {
		err = errSkip
		return
	}

	fi.name = sf.Name
	fi.structType = typ
	fi.index = sf.Index

	var anyField bool

	// handle field tags
	parts := strings.Split(sf.Tag.Get(structFieldTag), ",")
	for i, value := range parts {
		if i == 0 {
			switch value {
			case "-":
				// skip
				err = errSkip
				return
			case "":
			default:
				fi.explicitTag, err = ParseTag(value)
				if err != nil {
					return
				}
			}
		} else {
			switch value {
			case "enum":
				fi.flags |= fEnum
			case "omitempty":
				fi.flags |= fOmitEmpty
			case "dateTimeExtended":
				fi.flags |= fDateTimeExtended
			case "any":
				anyField = true
				fi.flags |= fAny
			}
		}
	}

	if anyField && fi.explicitTag != TagNone {
		return fi, merry.Here(ErrTagConflict).Appendf(`field %s.%s may not specify a TTLV tag and the "any" flag`, fi.structType.Name(), fi.name)
	}

	// extract type info for the field.  The KMIP tag
	// for this field is derived from either the field name,
	// the field tags, or the field type.
	fi.ti, err = getTypeInfo(sf.Type)
	if err != nil {
		return
	}

	if fi.ti.tagField != nil && fi.ti.tagField.explicitTag != TagNone {
		fi.tag = fi.ti.tagField.explicitTag
		if fi.explicitTag != TagNone && fi.explicitTag != fi.tag {
			// if there was a tag on the struct field containing this value, it must
			// agree with the value's intrinsic tag
			return fi, merry.Here(ErrTagConflict).Appendf(`TTLV tag "%s" in tag of %s.%s conflicts with TTLV tag "%s" in %s.%s`, fi.explicitTag, fi.structType.Name(), fi.name, fi.ti.tagField.explicitTag, fi.ti.typ.Name(), fi.ti.tagField.name)
		}
	}

	// pre-calculate the tag for this field.  This intentional duplicates
	// some of tagForMarshaling().  The value is primarily used in unmarshaling
	// where the dynamic value of the field is not needed.
	if fi.tag == TagNone {
		fi.tag = fi.explicitTag
	}
	if fi.tag == TagNone {
		fi.tag, _ = ParseTag(fi.name)
	}

	return

}

func (ti *typeInfo) getFieldsInfo() error {

	if ti.typ.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < ti.typ.NumField(); i++ {
		fi, err := getFieldInfo(ti.typ, ti.typ.Field(i))
		switch {
		case err == errSkip:
			// skip
		case err != nil:
			return err
		case fi.name == "TTLVTag":
			ti.tagField = &fi
		default:
			ti.valueFields = append(ti.valueFields, fi)
		}
	}

	// verify that multiple fields don't have the same tag
	names := map[Tag]string{}
	for _, f := range ti.valueFields {
		if f.flags&fAny != 0 {
			// ignore any fields
			continue
		}
		tag := f.tag
		if tag != TagNone {
			if fname, ok := names[tag]; ok {
				return merry.Here(ErrTagConflict).Appendf("field resolves to the same tag (%s) as other field (%s)", tag, fname)
			}
			names[tag] = f.name
		}
	}

	return nil
}

type typeInfo struct {
	typ         reflect.Type
	inferredTag Tag
	tagField    *fieldInfo
	valueFields []fieldInfo
}

type fieldFlags int

const (
	fOmitEmpty fieldFlags = 1 << iota
	fEnum
	fDateTimeExtended
	fAny
)

type fieldInfo struct {
	structType       reflect.Type
	explicitTag, tag Tag
	name             string
	index            []int
	flags            fieldFlags
	ti               typeInfo
}
