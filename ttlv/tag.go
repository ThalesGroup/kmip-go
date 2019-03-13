package ttlv

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/gemalto/kmip-go/internal/kmiputil"
	"strings"
)

func RegisterTag(tag Tag, name string) {
	_TagValueToFullNameMap[tag] = name
	name = kmiputil.NormalizeName(name)
	_TagNameToValueMap[name] = tag
	_TagValueToNameMap[tag] = name
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

// implementation of 5.4.1.1 and 5.5.1.1
func NormalizeName(s string) string {
	return kmiputil.NormalizeName(s)
}

// Tag
// 9.1.3.1
type Tag uint32

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

var _TagValueToFullNameMap = map[Tag]string{}
var _TagValueToNameMap = map[Tag]string{}
var _TagNameToValueMap = map[string]Tag{}
