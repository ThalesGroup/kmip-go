package kmip

import (
	"testing"
	"bytes"
	"github.com/stretchr/testify/require"
	"fmt"
	"time"
	"math/big"
	"io"
	"io/ioutil"
	"reflect"
	"math"
)

func parseBigInt(s string) *big.Int {
	i := &big.Int{}
	_, ok := i.SetString(s, 10)
	if !ok {
		panic(fmt.Errorf("can't parse as big int: %v", s))
	}
	return i
}

var knownGoodSamples = []struct {
	name string
	v    interface{}
	exp  string
}{
	{
		v:   8,
		exp: `42 00 01 | 02 | 00 00 00 04 | 00 00 00 08 00 00 00 00`,
	},
	{
		v:   int8(8),
		exp: `42 00 01 | 02 | 00 00 00 04 | 00 00 00 08 00 00 00 00`,
	},
	{
		v:   uint8(8),
		exp: `42 00 01 | 02 | 00 00 00 04 | 00 00 00 08 00 00 00 00`,
	},
	{
		v:   int16(8),
		exp: `42 00 01 | 02 | 00 00 00 04 | 00 00 00 08 00 00 00 00`,
	},
	{
		v:   uint16(8),
		exp: `42 00 01 | 02 | 00 00 00 04 | 00 00 00 08 00 00 00 00`,
	},
	{
		v:   int32(8),
		exp: `42 00 01 | 02 | 00 00 00 04 | 00 00 00 08 00 00 00 00`,
	},
	{
		v:   uint32(8),
		exp: `42 00 01 | 02 | 00 00 00 04 | 00 00 00 08 00 00 00 00`,
	},
	{
		v:   true,
		exp: `42 00 01 | 06 | 00 00 00 08 | 00 00 00 00 00 00 00 01`,
	},
	{
		v:   false,
		exp: `42 00 01 | 06 | 00 00 00 08 | 00 00 00 00 00 00 00 00`,
	},
	{
		v:   int64(8),
		exp: `42 00 01 | 03 | 00 00 00 08 | 00 00 00 00 00 00 00 08`,
	},
	{
		v:   uint64(8),
		exp: `42 00 01 | 03 | 00 00 00 08 | 00 00 00 00 00 00 00 08`,
	},
	{
		v:   parseTime("Friday, March 14, 2008, 11:56:40 UTC"),
		exp: `42 00 01 | 09 | 00 00 00 08 | 00 00 00 00 47 DA 67 F8`,
	},
	{
		exp: "42 00 01 | 0A | 00 00 00 04 | 00 0D 2F 00 00 00 00 00",
		v:   10 * 24 * time.Hour,
	},
	{
		exp: "42 00 01 | 04 | 00 00 00 10 | 00 00 00 00 03 FD 35 EB 6B C2 DF 46 18 08 00 00",
		v:   parseBigInt("1234567890000000000000000000"),
	},
	{
		// test non-pointer big int.  probably an edge case
		exp: "42 00 01 | 04 | 00 00 00 10 | 00 00 00 00 03 FD 35 EB 6B C2 DF 46 18 08 00 00",
		v: func() interface{} {
			return *(parseBigInt("1234567890000000000000000000"))
		}(),
	},
	{
		exp: "42 00 01 | 04 | 00 00 00 10 | 00 00 00 00 00 00 00 00 FF FF FF FF FF FF FF FF",
		v:   parseBigInt("18446744073709551615"),
	},
	{
		exp: "42 00 01 | 07 | 00 00 00 0B | 48 65 6C 6C 6F 20 57 6F 72 6C 64 00 00 00 00 00",
		v:   "Hello World",
	},
	{
		exp: "42 00 01 | 08 | 00 00 00 03 | 01 02 03 00 00 00 00 00",
		v:   []byte{0x01, 0x02, 0x03},
	},
	{
		// this positive number has a 1 bit on byte boundary.  Need to prepend
		// a 00 so the first significant bit is 0 (for positive).  Adding
		// the 0 byte increases the total length of the encode number from 8
		// to 9 bytes, which means we also need to pad out zeros to 16 bytes, since
		// TTLV requires values to be multiples of 8 bytes
		exp: "42 00 01 | 04 | 00 00 00 10 | 00 00 00 00 00 00 00 00 FF FF FF FF FF FF FF FF",
		v:   parseBigInt("18446744073709551615"),
	},
	{
		exp: "42 00 01 | 04 | 00 00 00 10 | FF FF FF FF FF FF FF FF 00 00 00 00 00 00 00 01",
		v:   parseBigInt("-18446744073709551615"),
	},
	{
		exp: "42 00 01 | 04 | 00 00 00 08 | 00 FF FF FF FF FF FF FF",
		v:   parseBigInt("72057594037927935"),
	},
	{
		exp: "42 00 01 | 04 | 00 00 00 08 | 00 00 00 00 00 00 00 00",
		v:   parseBigInt("0"),
	},
	{
		v:   parseBigInt("-1042342234234123423435647768234"),
		exp: "42 00 01 | 04 | 00 00 00 10 | FF FF FF F2 D8 02 B6 52 7F 99 EE 98 23 99 A9 56",
	},
	{
		v:   parseBigInt("-100"),
		exp: "42 00 01 | 04 | 00 00 00 08 | FF FF FF FF FF FF FF 9C",
	},
	{
		v:   parseBigInt("100"),
		exp: "42 00 01 | 04 | 00 00 00 08 | 00 00 00 00 00 00 00 64",
	},
	{
		v:   parseBigInt("255"),
		exp: "42 00 01 | 04 | 00 00 00 08 | 00 00 00 00 00 00 00 FF",
	},
	{
		v:   parseBigInt("1"),
		exp: "42 00 01 | 04 | 00 00 00 08 | 00 00 00 00 00 00 00 01",
	},
	{
		v:   parseBigInt("-1"),
		exp: "42 00 01 | 04 | 00 00 00 08 | FF FF FF FF FF FF FF FF",
	},
	{
		v:   parseBigInt("-2"),
		exp: "42 00 01 | 04 | 00 00 00 08 | FF FF FF FF FF FF FF FE",
	},
	{
		v:   parseBigInt("-256"),
		exp: "42 00 01 | 04 | 00 00 00 08 | FF FF FF FF FF FF FF 00",
	},
	{
		v:   parseBigInt("-255"),
		exp: "42 00 01 | 04 | 00 00 00 08 | FF FF FF FF FF FF FF 01",
	},
	{
		v:   parseBigInt("-32768"),
		exp: "42 00 01 | 04 | 00 00 00 08 | FF FF FF FF FF FF 80 00",
	},
	{
		v:   parseBigInt("-128"),
		exp: "42 00 01 | 04 | 00 00 00 08 | FF FF FF FF FF FF FF 80",
	},
	{
		v:   CredentialTypeAttestation,
		exp: "42 00 01 | 05 | 00 00 00 04 | 00 00 00 03 00 00 00 00",
	},
}

type MarhalableStruct struct{}

func (MarhalableStruct) MarshalTaggedValue(e *Encoder, tag Tag) error {
	return e.EncodeStructure(TagBatchCount, func(e *Encoder) error {
		e.EncodeValue(TagActivationDate, 4)
		e.EncodeValue(TagAlternativeName, 5)
		e.EncodeValue(TagBatchCount, 3)
		e.EncodeStructure(TagArchiveDate, func(e *Encoder) error {
			return e.EncodeValue(TagBatchCount, 3)
		})
		e.EncodeValue(TagCancellationResult, "blue")
		return e.EncodeStructure(TagAuthenticatedEncryptionAdditionalData, func(e *Encoder) error {
			return e.EncodeStructure(TagMaskGenerator, func(e *Encoder) error {
				return e.EncodeValue(TagBatchCount, 3)
			})

		})
	})

}

type MarshalerFunc func(e *Encoder, tag Tag) error

func (f MarshalerFunc) MarshalTaggedValue(e *Encoder, tag Tag) error {
	// TODO: workaround for encoding a nil value of type MarshalerFunc.  The non reflect
	// path currently has no way to detect whether an interface value that implements Marshaler is actually
	// nil.  This makes it save to call a nil MarshalerFunc
	if f == nil {
		return nil
	}
	return f(e, tag)
}

type ptrMarshaler struct{}

func (*ptrMarshaler) MarshalTaggedValue(e *Encoder, tag Tag) error {
	return e.EncodeValue(tag, 5)
}

type nonptrMarshaler struct{}

func (nonptrMarshaler) MarshalTaggedValue(e *Encoder, tag Tag) error {
	return e.EncodeValue(tag, 5)
}

func TestTTLVEncoder_Encode(t *testing.T) {
	b, err := MarshalTTLV(MarhalableStruct{})
	require.NoError(t, err)
	fmt.Println(TTLV2(b))
}

func fastPathSupported(v interface{}) bool {
	switch v.(type) {
	case nil:
		return true
	case EnumValuer, Marshaler:
		// interfaces
		return true
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, bool, time.Time, time.Duration, big.Int, *big.Int, string, []byte, []interface{}:
		// base types
		return true
	case uintptr, complex64, complex128, float32, float64:
		// types which are not encodeable, but should be detected and rejected in the fast path
		return true
	}
	return false
}

func TestEncoder_encode_unsupported(t *testing.T) {
	type testCase struct {
		name   string
		v      interface{}
		expErr error
	}

	tests := []testCase{
		{
			v:      map[string]string{},
			expErr: ErrUnsupportedTypeError,
		},
		{
			v:      float32(5),
			expErr: ErrUnsupportedTypeError,
		},
		{
			v:      float64(5),
			expErr: ErrUnsupportedTypeError,
		},
		{
			v:      complex64(5),
			expErr: ErrUnsupportedTypeError,
		},
		{
			v:      complex128(5),
			expErr: ErrUnsupportedTypeError,
		},
		{
			v:      uintptr(5),
			expErr: ErrUnsupportedTypeError,
		},
		{
			v:      uint32(math.MaxInt32 + 1),
			expErr: ErrIntOverflow,
		},
		{
			v:      int(math.MaxInt32 + 1),
			expErr: ErrIntOverflow,
		},
		{
			v:      uint(math.MaxInt32 + 1),
			expErr: ErrIntOverflow,
		},
		{
			v:      uint64(math.MaxInt64 + 1),
			expErr: ErrLongIntOverflow,
		},
		{
			v: struct{
				CustomAttribute struct{
					AttributeValue complex128
				}
			}{},
			expErr: ErrUnsupportedTypeError,
		},
	}
	enc := NewTTLVEncoder(bytes.NewBuffer(nil))
	enc.format = newEncBuf()
	for _, test := range tests {
		testName := test.name
		if testName == "" {
			testName = fmt.Sprintf("%T", test.v)
		}
		t.Run(testName, func(t *testing.T) {
			// test both reflect and non-reflect paths
			err := enc.encodeReflectValue(TagCancellationResult, reflect.ValueOf(test.v))
			require.Error(t, err)
			t.Log(Details(err))
			require.True(t, Is(err, test.expErr), Details(err))

			err = enc.encodeInterfaceValue(TagCancellationResult, test.v)
			require.Error(t, err)
			if fastPathSupported(test.v) {
				require.True(t, Is(err, test.expErr), Details(err))
			} else {
				require.True(t, err == errNoEncoder)
			}
		})
	}
}

func TestEncoder_encode(t *testing.T) {

	type AttributeValue string
	type Attribute struct {
		AttributeValue string
	}
	type MarshalableFields struct {
		Attribute          MarshalerFunc
		AttributeName      Marshaler
		AttributeValue     nonptrMarshaler
		ArchiveDate        *nonptrMarshaler
		CancellationResult **nonptrMarshaler
		CustomAttribute    ptrMarshaler
		AttributeIndex     *ptrMarshaler
		Certificate        **ptrMarshaler
	}

	type testCase struct {
		name         string
		tag          Tag
		nodefaulttag bool
		v            interface{}
		expected     []interface{}
	}

	tests := []testCase{
		// byte strings
		{
			name:     "byteslice",
			v:        []byte{0x01, 0x02, 0x03},
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: []byte{0x01, 0x02, 0x03}}},
		},
		{
			name:     "bytesliceptr",
			v:        func() *[]byte { b := []byte{0x01, 0x02, 0x03}; return &b }(),
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: []byte{0x01, 0x02, 0x03}}},
		},
		// text strings
		{
			v:        "red",
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: "red"}},
		},
		{
			name:     "strptr",
			v:        func() *string { s := "red"; return &s }(),
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: "red"}},
		},
		{
			name:     "zeroptr",
			v:        func() *string { var s *string; return s }(),
			expected: nil,
		},
		{
			name:     "ptrtonil",
			v:        func() *string { var s *string; s = nil; return s }(),
			expected: nil,
		},
		{
			name:     "zerointerface",
			v:        func() io.Writer { var i io.Writer; return i }(),
			expected: nil,
		},
		{
			name:     "nilinterface",
			v:        func() io.Writer { var i io.Writer; i = nil; return i }(),
			expected: nil,
		},
		{
			name:     "nilinterfaceptr",
			v:        func() *io.Writer { var i *io.Writer; return i }(),
			expected: nil,
		},
		// date time
		{
			v:        parseTime("Friday, March 14, 2008, 11:56:40 UTC"),
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: parseTime("Friday, March 14, 2008, 11:56:40 UTC")}},
		},
		// big int ptr
		{
			v:        parseBigInt("1234567890000000000000000000"),
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: parseBigInt("1234567890000000000000000000")}},
		},
		// big int
		{
			v:        func() interface{} { return *(parseBigInt("1234567890000000000000000000")) }(),
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: parseBigInt("1234567890000000000000000000")}},
		},
		// duration
		{
			v:        time.Second * 10,
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: time.Second * 10}},
		},
		// boolean
		{
			v:        true,
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: true}},
		},
		// enum value
		{
			name:     "enum",
			v:        CredentialTypeAttestation,
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: EnumLiteral{IntValue: 0x03}}},
		},
		// slice
		{
			v: []interface{}{5, 6, 7},
			expected: []interface{}{
				TaggedValue{Tag: TagCancellationResult, Value: int32(5)},
				TaggedValue{Tag: TagCancellationResult, Value: int32(6)},
				TaggedValue{Tag: TagCancellationResult, Value: int32(7)},
			},
		},
		// nil
		{
			v:        nil,
			expected: nil,
		},
		// marshalable
		{
			name:     "marshalable",
			v:        MarshalerFunc(func(e *Encoder, tag Tag) error { return e.EncodeValue(TagArchiveDate, 5) }),
			expected: []interface{}{TaggedValue{Tag: TagArchiveDate, Value: int32(5)}},
		},
		{
			name:     "namedtype",
			v:        AttributeValue("blue"),
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: "blue"}},
		},
		{
			name:         "namedtypenotag",
			nodefaulttag: true,
			v:            AttributeValue("blue"),
			expected:     []interface{}{TaggedValue{Tag: TagAttributeValue, Value: "blue"}},
		},
		// struct
		{
			name: "struct",
			v:    struct{ AttributeName string }{"red"},
			expected: []interface{}{Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeName, Value: "red"},
				},
			}},
		},
		{
			name: "structtag",
			v:    struct{ AttributeName string `kmip:"Attribute"` }{"red"},
			expected: []interface{}{Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttribute, Value: "red"},
				},
			}},
		},
		{
			name: "structptr",
			v:    &Attribute{"red"},
			expected: []interface{}{Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeValue, Value: "red"},
				},
			}},
		},
		{
			name: "structtaghex",
			v:    struct{ AttributeName string `kmip:"0x42000b"` }{"red"},
			expected: []interface{}{Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeValue, Value: "red"},
				},
			}},
		},
		{
			name: "structtagskip",
			v: struct {
				AttributeName  string `kmip:"-"`
				AttributeValue string
			}{"red", "green"},
			expected: []interface{}{Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeValue, Value: "green"},
				},
			}},
		},
		{
			name: "skipstructanonfield",
			v: struct {
				AttributeName string
				Attribute
			}{"red", Attribute{"green"}},
			expected: []interface{}{Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeName, Value: "red"},
				},
			}},
		},
		{
			name: "skipnonexportedfields",
			v: struct {
				AttributeName  string
				attributeValue string
			}{"red", "green"},
			expected: []interface{}{Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeName, Value: "red"},
				},
			}},
		},
		{
			name: "marshalerfields",
			v: &MarshalableFields{
				Attribute: func(e *Encoder, tag Tag) error {
					return e.EncodeValue(tag, 5)
				},
				AttributeName:      &ptrMarshaler{},
				ArchiveDate:        &nonptrMarshaler{},
				CancellationResult: func() **nonptrMarshaler { p := &nonptrMarshaler{}; return &p }(),
				AttributeIndex:     &ptrMarshaler{},
				Certificate:        func() **ptrMarshaler { p := &ptrMarshaler{}; return &p }(),
			},
			expected: []interface{}{Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttribute, Value: int32(5)},
					TaggedValue{Tag: TagAttributeName, Value: int32(5)},
					TaggedValue{Tag: TagAttributeValue, Value: int32(5)},
					TaggedValue{Tag: TagArchiveDate, Value: int32(5)},
					TaggedValue{Tag: TagCancellationResult, Value: int32(5)},
					TaggedValue{Tag: TagCustomAttribute, Value: int32(5)},
					TaggedValue{Tag: TagAttributeIndex, Value: int32(5)},
					TaggedValue{Tag: TagCertificate, Value: int32(5)},
				},
			}},
		},
		{
			name: "nilmarshalerfields",
			v:    &MarshalableFields{},
			expected: []interface{}{Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeValue, Value: int32(5)},
					TaggedValue{Tag: TagCustomAttribute, Value: int32(5)},
				},
			}},
		},
	}

	// test cases for all the int base types
	for _, v := range []interface{}{int8(5), uint(5), uint8(5), int16(5), uint16(5), int(5), int32(5), uint32(5), byte(5), rune(5)} {
		tests = append(tests, testCase{
			v:        v,
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: int32(5)}},
		})
	}

	// test cases for all long int base types
	for _, v := range []interface{}{int64(5), uint64(5)} {
		tests = append(tests, testCase{
			v:        v,
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: int64(5)}},
		})
	}

	for _, tc := range tests {

		testName := tc.name
		if testName == "" {
			testName = fmt.Sprintf("%T", tc.v)
		}
		t.Run(testName, func(t *testing.T) {
			m := memFormat{}
			enc := NewTTLVEncoder(nil)
			enc.format = &m

			if tc.tag == TagNone && !tc.nodefaulttag {
				tc.tag = TagCancellationResult
			}

			err := enc.encodeReflectValue(tc.tag, reflect.ValueOf(tc.v))
			require.NoError(t, err)
			enc.flush()

			require.Equal(t, tc.expected, m.writtenValues)

			m.clear()
			err = enc.encodeInterfaceValue(tc.tag, tc.v)
			if fastPathSupported(tc.v) {
				require.NoError(t, err)
				enc.flush()

				require.Equal(t, tc.expected, m.writtenValues)
			} else {
				require.True(t, err == errNoEncoder)
			}
		})

	}

}

func parseTime(s string) time.Time {
	v, err := time.Parse("Monday, January 2, 2006, 15:04:05 MST", s)
	if err != nil {
		panic(err)
	}
	return v
}

func BenchmarkEncodeSlice(b *testing.B) {
	enc := NewTTLVEncoder(ioutil.Discard)

	type Attribute struct {
		AttributeValue string
	}

	v := Attribute{"red"}

	rv := reflect.ValueOf(v)

	for i := 0; i < b.N; i++ {
		enc.encodeReflectValue(TagNone, rv)
	}
}
