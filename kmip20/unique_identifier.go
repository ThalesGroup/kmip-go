package kmip20

import (
	"github.com/ansel1/merry"
	"github.com/gemalto/kmip-go/ttlv"
)

type UniqueIdentifierValue struct {
	Text  string
	Enum  UniqueIdentifier
	Index int32
}

func (u *UniqueIdentifierValue) UnmarshalTTLV(d *ttlv.Decoder, v ttlv.TTLV) error {
	if len(v) == 0 {
		return nil
	}

	if u == nil {
		*u = UniqueIdentifierValue{}
	}

	switch v.Type() {
	case ttlv.TypeTextString:
		u.Text = v.String()
	case ttlv.TypeEnumeration:
		u.Enum = UniqueIdentifier(v.ValueEnumeration())
	case ttlv.TypeInteger:
		u.Index = v.ValueInteger()
	default:
		return merry.Errorf("invalid type for UniqueIdentifier: %s", v.Type().String())
	}

	return nil
}

func (u UniqueIdentifierValue) MarshalTTLV(e *ttlv.Encoder, tag ttlv.Tag) error {
	switch {
	case u.Text != "":
		e.EncodeTextString(tag, u.Text)
	case u.Enum != 0:
		e.EncodeEnumeration(tag, uint32(u.Enum))
	case u.Index != 0:
		e.EncodeInteger(tag, u.Index)
	}

	return nil
}
