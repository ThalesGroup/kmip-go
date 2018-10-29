package kmip

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"gitlab.protectv.local/regan/kmip.git/internal/kmiputil"
	"strconv"
	"strings"
	"sync"

	"github.com/ansel1/merry"
)

type enumDef struct {
	EnumTypeDef
	isMask bool
}

var enumRegistry = sync.Map{}

func parseOneInteger(tag Tag, s string) (int32, error) {
	if strings.HasPrefix(s, "0x") {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return 0, merry.Here(ErrInvalidHexString).WithCause(err)
		}
		if len(b) != 4 {
			return 0, merry.Here(ErrInvalidHexString).Append("must be 4 bytes (8 hex characters)")
		}
		return int32(binary.BigEndian.Uint32(b)), nil
	}
	i, err := strconv.ParseInt(s, 10, 32)
	if err == nil {
		return int32(i), nil
	}
	if v, ok := enumRegistry.Load(tag); ok {
		if u, ok := v.(enumDef).Parse(s); ok {
			return int32(u), nil
		}
	}
	return 0, merry.New("must be number, hex string, or mask value name")
}

func ParseInteger(tag Tag, s string) (int32, error) {
	if strings.IndexAny(s, "| ") < 0 {
		return parseOneInteger(tag, s)
	}
	// split values, look up each, and recombine
	s = strings.Replace(s, "|", " ", -1)
	parts := strings.Split(s, " ")
	var v int32
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		i, err := parseOneInteger(tag, part)
		if err != nil {
			return 0, err
		}
		v |= i
	}
	return v, nil
}

func ParseEnum(tag Tag, s string) (uint32, error) {
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
	u, err := strconv.ParseUint(s, 10, 32)
	if err == nil {
		// it was a raw number
		return uint32(u), nil
	}
	v, _ := enumRegistry.Load(tag)
	if v != nil {
		if u, ok := v.(enumDef).Parse(s); ok {
			return u, nil
		}
	}
	return 0, merry.New("must be a number, hex string, or enum value name")
}

func EnumToTypedEnum(tag Tag, i uint32) interface{} {
	v, _ := enumRegistry.Load(tag)
	if v != nil {
		if v.(enumDef).Typed != nil {
			return v.(enumDef).Typed(i)
		}
	}
	return i
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
	Parse  func(s string) (uint32, bool)
	String func(v uint32) string
	Typed  func(v uint32) interface{}
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
	return TagNone, merry.Errorf("invalid tag \"%s\"", s)
}

func (t Tag) MarshalText() (text []byte, err error) {
	return []byte(t.String()), nil
}

func (t *Tag) UnmarshalText(text []byte) (err error) {
	*t, err = ParseTag(string(text))
	return
}
