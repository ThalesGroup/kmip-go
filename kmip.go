//go:generate kmipenums

package kmip

import (
	"bytes"
	"gitlab.protectv.local/regan/kmip.git/internal/kmiputil"
)

func Marshal(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := NewEncoder(buf).Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Unmarshal(b []byte, v interface{}) error {
	return NewDecoder(bytes.NewReader(b)).Decode(v)
}

// implementation of 5.4.1.1 and 5.5.1.1
func NormalizeName(s string) string {
	return kmiputil.NormalizeName(s)
}
