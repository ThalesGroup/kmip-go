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

type MarshalerEnum interface {
	MarshalTTLVEnum() uint32
}

type EnumInt uint32

func (i EnumInt) MarshalTTLVEnum() uint32 {
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

func (e EnumLiteral) MarshalTTLVEnum() uint32 {
	return e.IntValue
}
