package ttlv

import "fmt"

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
