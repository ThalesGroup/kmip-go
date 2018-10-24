package kmip

import (
	"errors"
	"fmt"
	"github.com/ansel1/merry"
	"math"
	"reflect"
)

func Is(err error, originals ...error) bool {
	return merry.Is(err, originals...)
}

func Details(err error) string {
	return merry.Details(err)
}

type MarshalerError struct {
	Type   reflect.Type
	Struct string
	Field  string
	Tag    Tag
}

func (e *MarshalerError) Error() string {
	msg := "kmip: error marshaling value"
	if e.Type != nil {
		msg += " of type " + e.Type.String()
	}
	if e.Struct != "" {
		msg += " in struct field " + e.Struct + "." + e.Field
	}
	return msg
}

var ErrHeaderTruncated = errors.New("header truncated")
var ErrValueTruncated = errors.New("value truncated")
var ErrInvalidTag = errors.New("invalid tag")
var ErrTagConflict = errors.New("")
var ErrInvalidType = errors.New("invalid KMIP type")
var ErrInvalidLen = errors.New("invalid length")
var ErrNoTag = errors.New("unable to determine tag for field")
var ErrIntOverflow = fmt.Errorf("value exceeds max int value %d", math.MaxInt32)
var ErrLongIntOverflow = fmt.Errorf("value exceeds max long int value %d", math.MaxInt64)
var ErrUnsupportedTypeError = errors.New("marshaling/unmarshaling is not supported for this type")
var ErrUnsupportedEnumTypeError = errors.New("unsupported type for enums, must be string, or int types")
var ErrInvalidHexString = errors.New("invalid hex string")
var ErrUnexpectedValue = errors.New("no field was found to unmarshal value into")

type errKey int

const (
	errorKeyResultReason errKey = iota
)

func init() {
	merry.RegisterDetail("Result Reason", errorKeyResultReason)
}

func WithResultReason(err error, rr ResultReason) error {
	return merry.WithValue(err, errorKeyResultReason, rr)
}

func GetResultReason(err error) ResultReason {
	v := merry.Value(err, errorKeyResultReason)
	switch t := v.(type) {
	case nil:
		return ResultReason(0)
	case ResultReason:
		return t
	default:
		panic(fmt.Sprintf("err result reason attribute's value was wrong type, expected ResultReason, got %T", v))
	}
}
