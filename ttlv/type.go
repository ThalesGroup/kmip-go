package ttlv

import (
	"encoding/hex"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/gemalto/kmip-go/internal/kmiputil"
	"strings"
)

func RegisterType(typ Type, name string) {
	name = kmiputil.NormalizeName(name)
	_TypeNameToValueMap[name] = typ
	_TypeValueToNameMap[typ] = name
}

func ParseType(s string) (Type, error) {
	if strings.HasPrefix(s, "0x") && len(s) == 4 {
		b, err := hex.DecodeString(s[2:])
		return Type(b[0]), err
	}
	if v, ok := _TypeNameToValueMap[s]; ok {
		return v, nil
	} else {
		return v, merry.Errorf("invalid type \"%s\"", s)
	}
}

// 2 and 9.1.1.2

type Type byte

const (
	TypeStructure   Type = 0x01
	TypeInteger     Type = 0x02
	TypeLongInteger Type = 0x03
	TypeBigInteger  Type = 0x04
	TypeEnumeration Type = 0x05
	TypeBoolean     Type = 0x06
	TypeTextString  Type = 0x07
	TypeByteString  Type = 0x08
	TypeDateTime    Type = 0x09
	TypeInterval    Type = 0x0A
)

var _TypeNameToValueMap = map[string]Type{
	"BigInteger":  TypeBigInteger,
	"Boolean":     TypeBoolean,
	"ByteString":  TypeByteString,
	"DateTime":    TypeDateTime,
	"Enumeration": TypeEnumeration,
	"Integer":     TypeInteger,
	"Interval":    TypeInterval,
	"LongInteger": TypeLongInteger,
	"Structure":   TypeStructure,
	"TextString":  TypeTextString,
}

var _TypeValueToNameMap = map[Type]string{
	TypeBigInteger:  "BigInteger",
	TypeBoolean:     "Boolean",
	TypeByteString:  "ByteString",
	TypeDateTime:    "DateTime",
	TypeEnumeration: "Enumeration",
	TypeInteger:     "Integer",
	TypeInterval:    "Interval",
	TypeLongInteger: "LongInteger",
	TypeStructure:   "Structure",
	TypeTextString:  "TextString",
}

func (t Type) String() string {
	if s, ok := _TypeValueToNameMap[t]; ok {
		return s
	}
	return fmt.Sprintf("%#02x", byte(t))
}

func (t Type) MarshalText() (text []byte, err error) {
	return []byte(t.String()), nil
}

func (t *Type) UnmarshalText(text []byte) (err error) {
	*t, err = ParseType(string(text))
	return
}
