package kmip

type Marshaler interface {
	MarshalTTLV(e *Encoder, tag Tag) error
}

type Unmarshaler interface {
	UnmarshalTTLV(d *Decoder, ttlv TTLV) error
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

type TaggedValue struct {
	Tag   Tag
	Value interface{}
}

func (t TaggedValue) MarshalTTLV(e *Encoder, tag Tag) error {
	// if tag is set, override the suggested tag
	if t.Tag != 0 {
		tag = t.Tag
	}

	return e.EncodeValue(tag, t.Value)
}

type EnumValuer interface {
	EnumValue() uint32
}

type EnumInt uint32

func (i EnumInt) EnumValue() uint32 {
	return uint32(i)
}

type Authentication struct {
	Credential []Credential
}

type Nonce struct {
	NonceID    []byte
	NonceValue []byte
}

type ProtocolVersion struct {
	ProtocolVersionMajor int
	ProtocolVersionMinor int
}

type MessageExtension struct {
	VendorIdentification string
	CriticalityIndicator bool
	VendorExtension      interface{}
}

type Attributes struct {
	Attributes []Attribute
}
