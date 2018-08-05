package kmip

import (
	"io"
	"math/big"
	"time"
)

type formatter interface {
	io.WriterTo
	EncodeStructure(tag Tag, f func(formatter))
	EncodeByteString(tag Tag, b []byte)
	EncodeTextString(tag Tag, s string)
	EncodeEnum(tag Tag, i EnumValuer)
	EncodeInterval(tag Tag, d time.Duration)
	EncodeDateTime(tag Tag, t time.Time)
	EncodeLongInt(tag Tag, i int64)
	EncodeBool(tag Tag, b bool)
	EncodeInt(tag Tag, i int32)
	EncodeBigInt(tag Tag, i *big.Int)
}

type noWriteFormat struct {
	formatter
}

func (noWriteFormat) WriteTo(w io.Writer) (n int64, err error) {
	// don't flush
	return 0, nil
}

type memFormat struct {
	writtenValues  []interface{}
	bufferedValues []interface{}
}

func (m *memFormat) clear() {
	m.writtenValues = nil
	m.bufferedValues = nil
}

func (m *memFormat) EncodeStructure(tag Tag, f func(formatter)) {
	inner := memFormat{}
	f(&inner)
	inner.WriteTo(nil)
	m.bufferedValues = append(m.bufferedValues, Structure{Tag: tag, Values: inner.writtenValues})
}

func (m *memFormat) bufferValue(tag Tag, v interface{}) {
	m.bufferedValues = append(m.bufferedValues, TaggedValue{Tag: tag, Value: v})
}

func (m *memFormat) EncodeByteString(tag Tag, b []byte) {
	m.bufferValue(tag, b)
}

func (m *memFormat) EncodeTextString(tag Tag, s string) {
	m.bufferValue(tag, s)
}

func (m *memFormat) EncodeEnum(tag Tag, i EnumValuer) {
	m.bufferValue(tag, EnumLiteral{IntValue: i.EnumValue()})
}

func (m *memFormat) EncodeInterval(tag Tag, d time.Duration) {
	m.bufferValue(tag, d)
}

func (m *memFormat) EncodeDateTime(tag Tag, t time.Time) {
	m.bufferValue(tag, t)
}

func (m *memFormat) EncodeLongInt(tag Tag, i int64) {
	m.bufferValue(tag, i)
}

func (m *memFormat) EncodeBool(tag Tag, b bool) {
	m.bufferValue(tag, b)
}

func (m *memFormat) EncodeInt(tag Tag, i int32) {
	m.bufferValue(tag, i)
}

func (m *memFormat) EncodeBigInt(tag Tag, i *big.Int) {
	m.bufferValue(tag, i)
}

func (m *memFormat) WriteTo(w io.Writer) (n int64, err error) {
	l := len(m.bufferedValues)
	if l > 0 {
		m.writtenValues = append(m.writtenValues, m.bufferedValues...)
		m.bufferedValues = m.bufferedValues[:0]
	}

	return int64(l), nil
}
