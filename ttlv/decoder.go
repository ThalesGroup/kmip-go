package ttlv

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/ansel1/merry"
	"io"
	"reflect"
)

var ErrUnexpectedValue = errors.New("no field was found to unmarshal value into")

func Unmarshal(b []byte, v interface{}) error {
	return NewDecoder(bytes.NewReader(b)).Decode(v)
}

type Unmarshaler interface {
	UnmarshalTTLV(d *Decoder, ttlv TTLV) error
}

type Decoder struct {
	r                   io.Reader
	bufr                *bufio.Reader
	DisallowExtraValues bool

	currStruct reflect.Type
	currField  string
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r:    r,
		bufr: bufio.NewReader(r),
	}
}

func (dec *Decoder) Reset(r io.Reader) {
	*dec = Decoder{
		r:    r,
		bufr: dec.bufr,
	}
	dec.bufr.Reset(r)
}

func (dec *Decoder) Decode(v interface{}) error {
	return dec.DecodeValue(v, nil)
}

func (dec *Decoder) DecodeValue(v interface{}, ttlv TTLV) error {
	if ttlv == nil {
		var err error
		ttlv, err = dec.NextTTLV()
		if err != nil {
			return err
		}
	}
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return merry.New("non-pointer passed to Decode")
	}
	return dec.unmarshal(val, ttlv)
}

func (dec *Decoder) unmarshal(val reflect.Value, ttlv TTLV) error {
	// Load value from interface, but only if the result will be
	// usefully addressable.
	if val.Kind() == reflect.Interface && !val.IsNil() {
		e := val.Elem()
		if e.Kind() == reflect.Ptr && !e.IsNil() {
			val = e
		}
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			val.Set(reflect.New(val.Type().Elem()))
		}
		val = val.Elem()
	}

	if val.Type().Implements(unmarshalerType) {
		return val.Interface().(Unmarshaler).UnmarshalTTLV(dec, ttlv)
	}

	if val.CanAddr() {
		valAddr := val.Addr()
		if valAddr.CanInterface() && valAddr.Type().Implements(unmarshalerType) {
			return valAddr.Interface().(Unmarshaler).UnmarshalTTLV(dec, ttlv)
		}
	}

	switch val.Kind() {
	case reflect.Interface:
		// set blank interface equal to the raw TTLV
		fullLen := ttlv.FullLen()
		ttlv2 := make(TTLV, fullLen)
		copy(ttlv2, ttlv[:fullLen])
		val.Set(reflect.ValueOf(ttlv2))
		return nil
	case reflect.Slice:
		typ := val.Type()
		if typ.Elem().Kind() == reflect.Uint8 {
			// []byte
			break
		}

		// Slice of element values.
		// Grow slice.
		n := val.Len()
		val.Set(reflect.Append(val, reflect.Zero(val.Type().Elem())))

		// Recur to read element into slice.
		if err := dec.unmarshal(val.Index(n), ttlv); err != nil {
			val.SetLen(n)
			return err
		}
		return nil
	}

	typeMismatchErr := func() error {
		e := &UnmarshalerError{
			Struct: dec.currStruct,
			Field:  dec.currField,
			Tag:    ttlv.Tag(),
			Type:   ttlv.Type(),
			Val:    val.Type(),
		}
		err := merry.WrapSkipping(e, 1).WithCause(ErrUnsupportedTypeError)
		return err
		//return err.WithMessagef("can't unmarshal TTLV type into go type")
	}

	switch ttlv.Type() {
	case TypeStructure:
		// stash currStruct
		currStruct := dec.currStruct
		err := dec.unmarshalStructure(ttlv, val)
		// restore currStruct
		dec.currStruct = currStruct
		return err
	case TypeInterval:
		if val.Kind() != reflect.Int64 {
			return typeMismatchErr()
		}
		val.SetInt(int64(ttlv.ValueInterval()))
	case TypeDateTime, TypeDateTimeExtended:
		if val.Type() != timeType {
			return typeMismatchErr()
		}
		val.Set(reflect.ValueOf(ttlv.ValueDateTime()))
	case TypeByteString:
		if val.Kind() != reflect.Slice && val.Type().Elem() != byteType {
			return typeMismatchErr()
		}
		val.SetBytes(ttlv.ValueByteString())
	case TypeTextString:
		if val.Kind() != reflect.String {
			return typeMismatchErr()
		}
		val.SetString(ttlv.ValueTextString())
	case TypeBoolean:
		if val.Kind() != reflect.Bool {
			return typeMismatchErr()
		}
		val.SetBool(ttlv.ValueBoolean())
	case TypeEnumeration:
		switch val.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i := int64(ttlv.ValueEnumeration())
			if val.OverflowInt(i) {
				return dec.newUnmarshalerError(ttlv, val.Type(), ErrIntOverflow)
			}
			val.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			i := uint64(ttlv.ValueEnumeration())
			if val.OverflowUint(i) {
				return dec.newUnmarshalerError(ttlv, val.Type(), ErrIntOverflow)
			}
			val.SetUint(i)
		default:
			return typeMismatchErr()
		}
	case TypeInteger:
		switch val.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			i := int64(ttlv.ValueInteger())
			if val.OverflowInt(i) {
				return dec.newUnmarshalerError(ttlv, val.Type(), ErrIntOverflow)
			}
			val.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			i := uint64(ttlv.ValueInteger())
			if val.OverflowUint(i) {
				return dec.newUnmarshalerError(ttlv, val.Type(), ErrIntOverflow)
			}
			val.SetUint(i)
		default:
			return typeMismatchErr()
		}
	case TypeLongInteger:
		switch val.Kind() {
		case reflect.Int64:
			val.SetInt(ttlv.ValueLongInteger())
		case reflect.Uint64:
			val.SetUint(uint64(ttlv.ValueLongInteger()))
		default:
			return typeMismatchErr()
		}
	case TypeBigInteger:
		if val.Type() != bigIntType {
			return typeMismatchErr()
		}
		val.Set(reflect.ValueOf(*ttlv.ValueBigInteger()))
	default:
		return dec.newUnmarshalerError(ttlv, val.Type(), ErrInvalidType)
	}
	return nil

}

func (dec *Decoder) unmarshalStructure(ttlv TTLV, val reflect.Value) error {

	fields, err := getFieldsInfo(val.Type())
	if err != nil {
		return err
	}

	// push currStruct (caller will pop)
	dec.currStruct = val.Type()
Next:
	for n := ttlv.ValueStructure(); n != nil; n = n.Next() {
		for _, field := range fields {
			if field.tag == n.Tag() {
				// push currField
				currField := dec.currField
				dec.currField = field.name
				err := dec.unmarshal(val.FieldByIndex(field.index), n)
				// restore currField
				dec.currField = currField
				if err != nil {
					return err
				}
				continue Next
			}
		}
		// should only get here if no fields matched the value
		if dec.DisallowExtraValues {
			return dec.newUnmarshalerError(ttlv, val.Type(), ErrUnexpectedValue)
		}
	}
	return nil
}

func (dec *Decoder) NextTTLV() (TTLV, error) {
	// first, read the header
	header, err := dec.bufr.Peek(8)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	if err := TTLV(header).ValidHeader(); err != nil {
		// bad header, abort
		return TTLV(header), merry.Prependf(err, "invalid header: %v", TTLV(header))
	}

	// allocate a buffer large enough for the entire message
	fullLen := TTLV(header).FullLen()
	buf := make([]byte, fullLen)

	var totRead int
	for {
		n, err := dec.bufr.Read(buf[totRead:])
		if err != nil {
			return TTLV(buf), merry.Wrap(err)
		}

		totRead += n
		if totRead >= fullLen {
			// we've read off a single full message
			return TTLV(buf), nil
		}
		// keep reading
	}
}

func (dec *Decoder) newUnmarshalerError(ttlv TTLV, valType reflect.Type, cause error) merry.Error {
	e := &UnmarshalerError{
		Struct: dec.currStruct,
		Field:  dec.currField,
		Tag:    ttlv.Tag(),
		Type:   ttlv.Type(),
		Val:    valType,
	}
	return merry.WrapSkipping(e, 1).WithCause(cause)
}

type UnmarshalerError struct {
	Val    reflect.Type
	Struct reflect.Type
	Field  string
	Tag    Tag
	Type   Type
}

func (e *UnmarshalerError) Error() string {
	msg := "kmip: error unmarshaling " + e.Tag.String() + " with type " + e.Type.String() + " into value of type " + e.Val.Name()
	if e.Struct != nil {
		msg += " in struct field " + e.Struct.Name() + "." + e.Field
	}
	return msg
}
