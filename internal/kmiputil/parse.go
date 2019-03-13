package kmiputil

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ansel1/merry"
	"strconv"
	"strings"
)

var ErrInvalidHexString = errors.New("invalid hex string")

// ParseInt32 parses an integer value from a string.  The string
// may be a number, or a hex string, prefixed with "0x".
func ParseInt32(s string) (int32, error) {
	if strings.HasPrefix(s, "0x") {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return 0, merry.Here(ErrInvalidHexString).WithCause(err)
		}
		if len(b) > 4 {
			return 0, merry.Here(ErrInvalidHexString).Append("must be max 4 bytes (8 hex characters)")
		}
		if len(b) < 4 {
			b = append(make([]byte, 4-len(b)), b...)
		}
		if len(b) != 4 {
			panic(fmt.Sprintf("len of b should have been 4, was %v", len(b)))
		}
		return int32(binary.BigEndian.Uint32(b)), nil
	}
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(i), nil
}
