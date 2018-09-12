package kmip

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/ansel1/merry"
	"github.com/go-errors/errors"
	"io"
	"math"
	"math/big"
	"reflect"
	"strings"
	"time"
)

const kmipStructTag = "kmip"

type Encoder struct {
	structDepth int
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
	if e.structDepth > 0 {
		return nil
	}
	_, err := e.encBuf.WriteTo(e.w)
	e.encBuf.Reset()
	return err
}

func (e *Encoder) encode(tag Tag, v interface{}) error {

	switch t := v.(type) {
	case nil:
		return nil
	case Marshaler:
		return t.MarshalTTLV(e, tag)
	}

	// if no tag is specified, we need to use the reflect path to see if we can infer it
	if !tag.valid() {
		return e.encodeReflectValue(tag, reflect.ValueOf(v), 0)
	}

	// try non-reflection encoding first
	err := e.encodeInterfaceValue(tag, v)

	// fallback on reflection encoding
	if err == errNoEncoder {
		err = e.encodeReflectValue(tag, reflect.ValueOf(v), 0)
	}
	return err
}

var errNoEncoder = errors.New("no non-reflect encoders")

func (e *Encoder) newMarshalingError(tag Tag, t reflect.Type, cause error) merry.Error {
	err := &MarshalerError{
		Type:   t,
		Struct: e.currStruct,
		Field:  e.currField,
		Tag:    tag,
	}
	return merry.WrapSkipping(err, 1).WithCause(cause)
}

func (e *Encoder) encodeInterfaceValue(tag Tag, v interface{}) error {
	// these are fast path encoders, which avoid reflect
	// in as many cases as possible.
	//
	// This doesn't provide much performance improvement
	// when encoding fields of a structure by reflection, but
	// for Marshaler implementations, it can mean avoiding
	// reflection altogether, which does provide a good boost

	switch t := v.(type) {
	case EnumValuer:
		e.encBuf.encodeEnum(tag, t.EnumValue())
	case TTLV:
		// raw TTLV value
		e.encBuf.Write(t)
	case int:
		if t > math.MaxInt32 {
			return e.newMarshalingError(tag, intType, ErrIntOverflow)
		}
		e.encBuf.encodeInt(tag, int32(t))
	case int8:
		e.encBuf.encodeInt(tag, int32(t))
	case int16:
		e.encBuf.encodeInt(tag, int32(t))
	case int32:
		e.encBuf.encodeInt(tag, t)
	case uint:
		if t > math.MaxInt32 {
			return e.newMarshalingError(tag, uintType, ErrIntOverflow)
		}
		e.encBuf.encodeInt(tag, int32(t))
	case uint8:
		e.encBuf.encodeInt(tag, int32(t))
	case uint16:
		e.encBuf.encodeInt(tag, int32(t))
	case uint32:
		if t > math.MaxInt32 {
			return e.newMarshalingError(tag, uint32Type, ErrIntOverflow)
		}
		e.encBuf.encodeInt(tag, int32(t))
	case bool:
		e.encBuf.encodeBool(tag, t)
	case int64:
		e.encBuf.encodeLongInt(tag, t)
	case uint64:
		if t > math.MaxInt64 {
			return e.newMarshalingError(tag, uint64Type, ErrLongIntOverflow)
		}
		e.encBuf.encodeLongInt(tag, int64(t))
	case time.Time:
		e.encBuf.encodeDateTime(tag, t)
	case time.Duration:
		e.encBuf.encodeInterval(tag, t)
	case big.Int:
		e.encBuf.encodeBigInt(tag, &t)
	case *big.Int:
		e.encBuf.encodeBigInt(tag, t)
	case string:
		e.encBuf.encodeTextString(tag, t)
	case []byte:
		e.encBuf.encodeByteString(tag, t)

	case []interface{}:
		for _, v := range t {
			err := e.EncodeValue(tag, v)
			if err != nil {
				return err
			}
		}
	case uintptr, float32, float64, complex64, complex128:
		return e.newMarshalingError(tag, reflect.TypeOf(v), ErrUnsupportedTypeError)
	default:
		return errNoEncoder
	}
	return nil
}

var byteType = reflect.TypeOf(byte(0))
var marshalerType = reflect.TypeOf((*Marshaler)(nil)).Elem()
var unmarshalerType = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
var timeType = reflect.TypeOf((*time.Time)(nil)).Elem()
var intType = reflect.TypeOf((*int)(nil)).Elem()
var uintType = reflect.TypeOf((*uint)(nil)).Elem()
var uint32Type = reflect.TypeOf((*uint32)(nil)).Elem()
var uint64Type = reflect.TypeOf((*uint64)(nil)).Elem()
var bigIntPtrType = reflect.TypeOf((*big.Int)(nil))
var bigIntType = bigIntPtrType.Elem()
var durationType = reflect.TypeOf(time.Nanosecond)
var marshalerEnumType = reflect.TypeOf((*EnumValuer)(nil)).Elem()
var ttlvType = reflect.TypeOf((*TTLV)(nil)).Elem()

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
			return e.newMarshalingError(tag, v.Type(), ErrInvalidHexString).Append("string enum values must be hex strings starting with 0x")
		}
		s = s[2:]
		if len(s) != 8 {
			return e.newMarshalingError(tag, v.Type(), ErrInvalidHexString).Appendf("invalid length, must be 8 (4 bytes), got %d", len(s))
		}
		b, err := hex.DecodeString(s)
		if err != nil {
			return e.newMarshalingError(tag, v.Type(), merry.WithCause(ErrInvalidHexString, err))
		}

		u := binary.BigEndian.Uint32(b)
		e.encBuf.encodeEnum(tag, u)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i := v.Uint()
		if i > math.MaxUint32 {
			return e.newMarshalingError(tag, v.Type(), ErrIntOverflow)
		}
		e.encBuf.encodeEnum(tag, uint32(i))
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := v.Int()
		if i > math.MaxUint32 {
			return e.newMarshalingError(tag, v.Type(), ErrIntOverflow)
		}
		e.encBuf.encodeEnum(tag, uint32(i))
		return nil
	default:
		return e.newMarshalingError(tag, v.Type(), ErrUnsupportedEnumTypeError)
	}
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
	case typ == ttlvType:
		if flags&fOmitEmpty != 0 && isEmptyValue(v) {
			return nil
		}
		e.encBuf.Write(v.Bytes())
	case typ.Implements(marshalerType):
		if flags&fOmitEmpty != 0 && isEmptyValue(v) {
			return nil
		}
		return v.Interface().(Marshaler).MarshalTTLV(e, tag)
	case typ.Implements(marshalerEnumType):
		if flags&fOmitEmpty != 0 && isEmptyValue(v) {
			return nil
		}
		e.encBuf.encodeEnum(tag, v.Interface().(EnumValuer).EnumValue())
		return nil
	case v.CanAddr():
		pv := v.Addr()
		pvtyp := pv.Type()
		switch {
		case pvtyp.Implements(marshalerType):
			if flags&fOmitEmpty != 0 && isEmptyValue(v) {
				return nil
			}
			return pv.Interface().(Marshaler).MarshalTTLV(e, tag)
		case pvtyp.Implements(marshalerEnumType):
			if flags&fOmitEmpty != 0 && isEmptyValue(v) {
				return nil
			}
			e.encBuf.encodeEnum(tag, pv.Interface().(EnumValuer).EnumValue())
			return nil
		}
	}

	switch v.Kind() {
	case reflect.Chan, reflect.Map, reflect.Func, reflect.Ptr, reflect.UnsafePointer, reflect.Uintptr, reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Interface:
		return e.newMarshalingError(tag, v.Type(), ErrUnsupportedTypeError)
	}

	// skip if value is empty and tags include omitempty
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

	if !tag.valid() {
		// error, no value tag to use
		return e.newMarshalingError(tag, typ, ErrInvalidTag).Append(tag.String())
	}

	if flags&fEnum != 0 {
		return e.encodeReflectEnum(tag, v)
	}

	switch typ {
	case timeType, bigIntType, bigIntPtrType, durationType:
		// these are some special types which are handled by the non-reflect path
		return e.encodeInterfaceValue(tag, v.Interface())
	}

	switch typ.Kind() {
	case reflect.Struct:
		// push current struct onto stack
		currStruct := e.currStruct
		e.currStruct = typ.Name()
		err := e.EncodeStructure(tag, func(e *Encoder) error {
			for _, field := range typeInfo.fields {
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
				err := e.encodeReflectValue(field.tag, fv, field.flags)
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
		e.encBuf.encodeInt(tag, int32(i))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		u := v.Uint()
		if u > math.MaxInt32 {
			return merry.Here(ErrIntOverflow).Prepend(tag.String())
		}
		e.encBuf.encodeInt(tag, int32(u))
	case reflect.Uint64:
		u := v.Uint()
		if u > math.MaxInt64 {
			return merry.Here(ErrLongIntOverflow).Prepend(tag.String())
		}
		e.encBuf.encodeLongInt(tag, int64(u))
	case reflect.Int64:
		e.encBuf.encodeLongInt(tag, int64(v.Int()))
	case reflect.Bool:
		e.encBuf.encodeBool(tag, v.Bool())
	default:
		// all kinds should have been handled by now
		panic(errors.New("should never get here"))
	}
	return nil

}

func (e *Encoder) EncodeStructure(tag Tag, f func(e *Encoder) error) error {
	if !tag.valid() {
		return merry.Here(ErrInvalidTag).Append(tag.String())
	}

	e.structDepth++
	i := e.encBuf.startStruct(tag)
	err := f(e)
	e.encBuf.endStruct(i)
	e.structDepth--
	if err != nil {
		return err
	}
	return e.flush()
}

// encBuf encodes basic KMIP types into TTLV
type encBuf struct {
	bytes.Buffer
	// enough to hold an entire TTLV for most base types
	scratch [16]byte
}

func (h *encBuf) startStruct(tag Tag) int {
	h.encodeHeader(tag, TypeStructure, 0)
	i := h.Len()
	h.Write(h.scratch[:8])
	return i
}

func (h *encBuf) endStruct(i int) {
	binary.BigEndian.PutUint32(h.scratch[:4], uint32(h.Len()-lenHeader-i))
	copy(h.Bytes()[i+4:], h.scratch[:4])
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

func (h *encBuf) encodeBigInt(tag Tag, i *big.Int) {
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

func (h *encBuf) encodeInt(tag Tag, i int32) {
	if IsEnumeration(tag) {
		h.encodeEnum(tag, uint32(i))
		return
	}
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

func (h *encBuf) encodeBool(tag Tag, b bool) {
	h.encodeHeader(tag, TypeBoolean, lenBool)
	if b {
		h.encodeLongIntVal(1)
	} else {
		h.encodeLongIntVal(0)
	}
	h.Write(h.scratch[:16])
}

func (h *encBuf) encodeLongInt(tag Tag, i int64) {
	if IsEnumeration(tag) {
		h.encodeEnum(tag, uint32(i))
		return
	}
	h.encodeHeader(tag, TypeLongInteger, lenLongInt)
	h.encodeLongIntVal(i)
	h.Write(h.scratch[:16])
}

func (h *encBuf) encodeLongIntVal(i int64) {
	binary.BigEndian.PutUint64(h.scratch[8:], uint64(i))
}

func (h *encBuf) encodeDateTime(tag Tag, t time.Time) {
	h.encodeHeader(tag, TypeDateTime, lenDateTime)
	h.encodeLongIntVal(t.Unix())
	h.Write(h.scratch[:16])
}

func (h *encBuf) encodeInterval(tag Tag, d time.Duration) {
	h.encodeHeader(tag, TypeInterval, lenInterval)
	h.encodeIntVal(int32(d / time.Second))
	h.Write(h.scratch[:16])
}

func (h *encBuf) encodeEnum(tag Tag, i uint32) {
	h.encodeHeader(tag, TypeEnumeration, lenEnumeration)
	binary.BigEndian.PutUint32(h.scratch[8:12], i)
	// pad extra bytes
	for i := 12; i < 16; i++ {
		h.scratch[i] = 0
	}
	h.Write(h.scratch[:16])
}

func (h *encBuf) encodeTextString(tag Tag, s string) {
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

func (h *encBuf) encodeByteString(tag Tag, b []byte) {
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
	ti.tag, _ = ParseTag(typ.Name())
	ti.typ = typ
	ti.name = typ.Name()

	if typ.Kind() == reflect.Struct {
		ti.fields, err = getFieldsInfo(typ)
	}
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
				if !fi.tag.valid() {
					var err error = &MarshalerError{
						Type:   typ,
						Struct: typ.Name(),
						Field:  sf.Name,
					}
					err = merry.WithCause(err, ErrInvalidTag).Append(fi.tag.String())
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
			Field:  fi.name,
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
			Field:  fi.name,
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
	name        string
	fields      []fieldInfo
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
