package kmip

import (
	"github.com/ansel1/merry"
	"fmt"
	"math"
	"reflect"
	"errors"
	"strings"
	)

func Is(err error, originals ...error) bool {
	return merry.Is(err, originals...)
}

func appendField(b *strings.Builder, name, val string) {
	if val == "" {
		return
	}
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	if name != "" {
		b.WriteString(name)
		b.WriteString(": ")
	}
	b.WriteString(val)
}

func Details(err error) string {
	if err == nil {
		return ""
	}

	//var parts []string
	//if s := merry.Message(err); s != "" {
	//	parts = append(parts, "Error Message: " +s)
	//}
	//if s := merry.UserMessage(err); s != "" {
	//	parts = append(parts, "User Message: " + s)
	//}
	//if s := GetErrorContext(err).String(); s != "" {
	//	parts = append(parts, s)
	//}
	//
	//if s := merry.Stacktrace(err); s != "" {
	//	parts = append(parts, "")
	//	parts = append(parts, s)
	//}
	//
	//return strings.Join(parts, "\n")

	b := strings.Builder{}

	appendField(&b, "Error Message", merry.Message(err))
	appendField(&b, "User Message", merry.UserMessage(err))
	appendField(&b, "", GetErrorContext(err).String())

	s := merry.Stacktrace(err)
	if s != "" {
		b.WriteString("\n\n")
		b.WriteString(s)
	}

	return b.String()
}

var ErrHeaderTruncated= errors.New("header truncated")
var ErrValueTruncated= errors.New("value truncated")
var ErrInvalidTag= errors.New("invalid tag")
var ErrInvalidType= errors.New("invalid type")
var ErrInvalidLen= errors.New("invalid length")
var ErrNoTag= errors.New("no tag")
var ErrTagNotRegistered= errors.New("tag name not registered")
var ErrIntOverflow= fmt.Errorf("value exceeds max int value %d", math.MaxInt32)
var ErrLongIntOverflow= fmt.Errorf("value exceeds max long int value %d", math.MaxInt64)
var ErrUnsupportedTypeError = errors.New("unsupported type")
var ErrUnsupportedEnumTypeError = errors.New("unsupported type for enums, must be string, or int types")
var ErrInvalidHexString = errors.New("invalid hex string")

type errKey int
const (
	errorCtx errKey = iota
)

type ErrorContext struct {
	Tag Tag
	Value interface{}
	Path []string
}

func (ctx *ErrorContext) String() string {
	if ctx == nil {
		return ""
	}

	b := strings.Builder{}
	if ctx.Tag != TagNone {
		appendField(&b, "Tag", ctx.Tag.String())
	}
	var rt reflect.Type
	switch t := ctx.Value.(type) {
	case nil:
	case reflect.Value:
		rt = t.Type()
	case reflect.Type:
		rt = t
	default:
		rt = reflect.TypeOf(ctx.Value)
	}

	if rt != nil {
		appendField(&b, "Type", rt.String())
		appendField(&b, "Kind", rt.Kind().String())
	}

	appendField(&b, "Path", strings.Join(ctx.Path, "."))

	return b.String()
}

func WithErrorContext(err error, ctx ErrorContext) error {
	return merry.WithValue(err, errorCtx, &ctx)
}

func GetErrorContext(err error) *ErrorContext {
	v := merry.Value(err, errorCtx)
	if v == nil {
		return nil
	}
	return v.(*ErrorContext)
}

func tagError(err error, tag Tag, v interface{}) merry.Error {
	var merr merry.Error

	switch m := err.(type) {
	case nil:
		return nil
	case merry.Error:
		// optimization: only capture the stack once, since its expensive
		merr = m.WithStackSkipping(1)
	default:
		merr = merry.WrapSkipping(err, 1)
	}

	if tag != TagNone || v != nil {
		merr = merr.WithValue(errorCtx, &ErrorContext{Tag: tag, Value: v})
	}

	if tag != TagNone {
		merr = merr.Prepend(tag.String())
	}

	return merr
}