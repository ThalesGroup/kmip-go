package ttlv

func RegisterTypes(r *Registry) {
	var m = map[string]Type{
		"BigInteger":       TypeBigInteger,
		"Boolean":          TypeBoolean,
		"ByteString":       TypeByteString,
		"DateTime":         TypeDateTime,
		"Enumeration":      TypeEnumeration,
		"Integer":          TypeInteger,
		"Interval":         TypeInterval,
		"LongInteger":      TypeLongInteger,
		"Structure":        TypeStructure,
		"TextString":       TypeTextString,
		"DateTimeExtended": TypeDateTimeExtended,
	}

	for name, v := range m {
		r.RegisterType(v, name)
	}
}

// Type describes the type of a KMIP TTLV.
// 2 and 9.1.1.2
type Type byte

const (
	TypeStructure        Type = 0x01
	TypeInteger          Type = 0x02
	TypeLongInteger      Type = 0x03
	TypeBigInteger       Type = 0x04
	TypeEnumeration      Type = 0x05
	TypeBoolean          Type = 0x06
	TypeTextString       Type = 0x07
	TypeByteString       Type = 0x08
	TypeDateTime         Type = 0x09
	TypeInterval         Type = 0x0A
	TypeDateTimeExtended Type = 0x0B
)

// String returns the canonical name of the type.  If the type
// name isn't registered, it returns the hex value of the type,
// e.g. "0x01" (TypeStructure).  The value of String() is suitable
// for use in the JSON or XML encoding of TTLV.
func (t Type) String() string {
	return DefaultRegistry.FormatType(t)
}

func (t Type) MarshalText() (text []byte, err error) {
	return []byte(t.String()), nil
}

func (t *Type) UnmarshalText(text []byte) (err error) {
	*t, err = DefaultRegistry.ParseType(string(text))
	return
}
