package kmip

import (
	"bytes"
	"github.com/ansel1/merry"
	"gitlab.protectv.local/regan/kmip.git/internal/kmiputil"
	"reflect"
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
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return merry.New("non-pointer passed to Unmarshal")
	}
	return unmarshal(val, TTLV(b))
}

// implementation of 5.4.1.1 and 5.5.1.1
func NormalizeName(s string) string {
	return kmiputil.NormalizeName(s)
}
