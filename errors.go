package kmip

import (
	"errors"
	"fmt"
	"github.com/ansel1/merry"
	"gitlab.protectv.local/regan/kmip.git/ttlv"
)

func Is(err error, originals ...error) bool {
	return merry.Is(err, originals...)
}

func Details(err error) string {
	return merry.Details(err)
}

var ErrInvalidTag = errors.New("invalid tag")

type errKey int

const (
	errorKeyResultReason errKey = iota
)

func init() {
	merry.RegisterDetail("Result Reason", errorKeyResultReason)
}

func WithResultReason(err error, rr ttlv.ResultReason) error {
	return merry.WithValue(err, errorKeyResultReason, rr)
}

func GetResultReason(err error) ttlv.ResultReason {
	v := merry.Value(err, errorKeyResultReason)
	switch t := v.(type) {
	case nil:
		return ttlv.ResultReason(0)
	case ttlv.ResultReason:
		return t
	default:
		panic(fmt.Sprintf("err result reason attribute's value was wrong type, expected ResultReason, got %T", v))
	}
}
