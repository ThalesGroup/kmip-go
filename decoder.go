package kmip

import (
	"github.com/ansel1/merry"
	"reflect"
)

func Unmarshal(b []byte, v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return merry.New("non-pointer passed to Unmarshal")
	}
	return unmarshal(val, TTLV(b))
}

func unmarshal(val reflect.Value, ttlv TTLV) error {

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

	switch val.Kind() {
	case reflect.Interface:
		// TODO: For now, simply ignore the field. In the near
		//       future we may choose to unmarshal the start
		//       element on it, if not nil.
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
		if err := unmarshal(val.Index(n), ttlv); err != nil {
			val.SetLen(n)
			return err
		}
		return nil
	}

	typeMismatchErr := func() error {
		err := tagErrorSkipping(ErrUnsupportedTypeError, ttlv.Tag(), val, 1)
		return err.Appendf("can't unmarshal %s into %s (%s)", ttlv.Type(), val.Type().String(), val.Kind().String())
	}

	switch ttlv.Type() {
	case TypeStructure:
		return unmarshalStructure(ttlv, val)
	case TypeInterval:
		if val.Kind() != reflect.Int64 {
			return typeMismatchErr()
		}
		val.SetInt(int64(ttlv.ValueInterval()))
	case TypeDateTime:
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
				return tagError(ErrIntOverflow, ttlv.Tag(), val)
			}
			val.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			i := uint64(ttlv.ValueInteger())
			if val.OverflowUint(i) {
				return tagError(ErrIntOverflow, ttlv.Tag(), val)
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
				return tagError(ErrIntOverflow, ttlv.Tag(), val)
			}
			val.SetInt(i)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			i := uint64(ttlv.ValueInteger())
			if val.OverflowUint(i) {
				return tagError(ErrIntOverflow, ttlv.Tag(), val)
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
		return tagError(ErrInvalidType, ttlv.Tag(), val).Append(ttlv.Type().String())
	}
	return nil

}

func unmarshalStructure(ttlv TTLV, val reflect.Value) error {

	if ttlv.Type() != TypeStructure {
		return tagError(ErrInvalidType, ttlv.Tag(), val).Append("kmip structure values must unmarshal into a struct")
	}

	ti, err := getTypeInfo(val.Type())
	if err != nil {
		return err
	}

	// keep track of which fields we've already matched up with a TTLV
	// values.  Since TTLV values can appear more than once, we want to
	// match them up with subsequent fields.
	matched := make([]bool, len(ti.fields))

Next:
	for n := ttlv.ValueStructure(); n != nil; n = n.Next() {
		for i, field := range ti.fields {
			if field.tag == n.Tag() && !matched[i] {
				err := unmarshal(val.FieldByIndex(field.index), n)
				if err != nil {
					return err
				}
				if !field.slice {
					matched[i] = true
				}
				continue Next
			}
		}
	}
	return nil
}
