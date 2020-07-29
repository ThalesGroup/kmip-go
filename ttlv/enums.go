package ttlv

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/ansel1/merry"
)

var ErrInvalidHexString = errors.New("invalid hex string")

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

func ParseInteger(tag Tag, s string) (int32, error) {
	if !strings.ContainsAny(s, "| ") {
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

func EnumToTyped(tag Tag, i uint32) interface{} {
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
