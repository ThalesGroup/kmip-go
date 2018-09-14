package kmip

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"gitlab.protectv.local/regan/kmip.git/internal/kmiputil"
	"strings"
	"sync"

	"github.com/ansel1/merry"
)

type enumDef struct {
	EnumTypeDef
	isMask bool
}

var enumRegistry = sync.Map{}

func ParseEnum(tag Tag, s string) (uint32, error) {
	v, _ := enumRegistry.Load(tag)
	if v != nil {
		return v.(enumDef).Parse(s)
	}
	if strings.HasPrefix(s, "0x") {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return 0, merry.Here(ErrInvalidHexString).WithCause(err)
		}
		if len(b) != 4 {
			return 0, merry.Here(ErrInvalidHexString).Append("must be 4 bytes (8 hex characters)")
		}
		return binary.BigEndian.Uint32(b), nil
	}
	return 0, merry.New("unable to parse enum value")
}

func EnumToString(tag Tag, i uint32) string {
	v, _ := enumRegistry.Load(tag)
	if v != nil {
		s := v.(enumDef).String(i)
		if s != "" {
			return s
		}
	}
	return fmt.Sprintf("%#08x", byte(i))
}

type EnumTypeDef struct {
	Parse  func(s string) (uint32, error)
	String func(v uint32) string
}

func RegisterEnum(tag Tag, def EnumTypeDef) {
	enumRegistry.Store(tag, enumDef{EnumTypeDef: def})
}

func RegisterBitMask(tag Tag, def EnumTypeDef) {
	enumRegistry.Store(tag, enumDef{EnumTypeDef: def, isMask: true})
}

func IsEnumeration(tag Tag) bool {
	v, _ := enumRegistry.Load(tag)
	if v != nil {
		return !v.(enumDef).isMask
	}
	return false
}

func IsBitMask(tag Tag) bool {
	v, _ := enumRegistry.Load(tag)
	if v != nil {
		return v.(enumDef).isMask
	}
	return false
}

func RegisterTag(tag Tag, name string) {
	_TagValueToFullNameMap[tag] = name
	name = kmiputil.NormalizeName(name)
	_TagNameToValueMap[name] = tag
	_TagValueToNameMap[tag] = name
}

func (t Tag) String() string {
	if s, ok := _TagValueToNameMap[t]; ok {
		return s
	}
	return fmt.Sprintf("%#06x", uint32(t))
}

func (t Tag) FullName() string {
	if s, ok := _TagValueToFullNameMap[t]; ok {
		return s
	}
	return fmt.Sprintf("%#06x", uint32(t))
}

// returns TagNone if not found.
// returns error if s is a malformed hex string, or a hex string of incorrect length
func ParseTag(s string) (Tag, error) {
	if strings.HasPrefix(s, "0x") {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return TagNone, merry.Prepend(err, "invalid hex string, should be 0x[a-fA-F0-9][a-fA-F0-9]")
		}
		switch len(b) {
		case 3:
			b = append([]byte{0}, b...)
		case 4:
		default:
			return TagNone, merry.Errorf("invalid byte length for tag, should be 3 bytes: %s", s)
		}

		return Tag(binary.BigEndian.Uint32(b)), nil
	}
	if v, ok := _TagNameToValueMap[s]; ok {
		return v, nil
	}
	return TagNone, nil
}

func (t Tag) MarshalText() (text []byte, err error) {
	return []byte(t.String()), nil
}

func (t *Tag) UnmarshalText(text []byte) (err error) {
	*t, err = ParseTag(string(text))
	return
}

var minStandardTag uint32 = 0x00420000
var maxStandardTag uint32 = 0x00430000
var minCustomTag uint32 = 0x00540000
var maxCustomTag uint32 = 0x00550000

func (t Tag) valid() bool {
	switch {
	case uint32(t) >= minStandardTag && uint32(t) < maxStandardTag:
		return true
	case uint32(t) >= minCustomTag && uint32(t) < maxCustomTag:
		return true
	default:
		return false
	}
}
