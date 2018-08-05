package kmip

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

type EnumInt uint32

func (i EnumInt) EnumValue() uint32 {
	return uint32(i)
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
