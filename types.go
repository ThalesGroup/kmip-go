package kmip

type Marshaler interface {
	MarshalTTLV(e *Encoder, tag Tag) error
}

type Unmarshaler interface {
	UnmarshalTTLV(ttlv TTLV) error
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

type Authentication struct {
	Credential []Credential
}

type Credential struct {
	CredentialType  CredentialType
	CredentialValue interface{}
}

// TODO: add an unmarshal impl to Credential to handle decoding the right kind
// of credential based on the credential type value
// TODO: add general support in decoder for decoding into an interface{}, setting it to TTLV

type UsernameAndPasswordCredentialValue struct {
	Username string
	Password string `kmip:",omitempty"`
}

type DeviceCredentialValue struct {
	DeviceSerialNumber string `kmip:",omitempty"`
	Password           string `kmip:",omitempty"`
	DeviceIdentifier   string `kmip:",omitempty"`
	NetworkIdentifier  string `kmip:",omitempty"`
	MachineIdentifier  string `kmip:",omitempty"`
	MediaIdentifier    string `kmip:",omitempty"`
}

type AttestationCredentialValue struct {
	Nonce                  Nonce
	AttestationType        AttestationType
	AttestationMeasurement []byte `kmip:",omitempty"`
	AttestationAssertion   []byte `kmip:",omitempty"`
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
