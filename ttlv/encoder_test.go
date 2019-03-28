package ttlv

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"
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
		v:   parseTime("2008-03-14T11:56:40Z"),
		exp: `42 00 01 | 09 | 00 00 00 08 | 00 00 00 00 47 DA 67 F8`,
	},
	{
		exp: "42 00 01 | 0A | 00 00 00 04 | 00 0D 2F 00 00 00 00 00",
		v:   10 * 24 * time.Hour,
	},
	{
		v:   DateTimeExtended{parseTime("2017-11-20T5:20:40.345567Z")},
		exp: "42 00 01 | 0B | 00 00 00 08 | 00 05 5E 63 3F 4D 1F DF",
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
		name: "bigintzero",
		exp:  "42 00 01 | 04 | 00 00 00 08 | 00 00 00 00 00 00 00 00",
		v:    parseBigInt("0"),
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
	{
		v:   func() TTLV { return TTLV(hex2bytes("42 00 01 | 06 | 00 00 00 08 | 00 00 00 00 00 00 00 01")) }(),
		exp: "42 00 01 | 06 | 00 00 00 08 | 00 00 00 00 00 00 00 01",
	},
}

type MarshalerStruct struct{}

func (MarshalerStruct) MarshalTTLV(e *Encoder, tag Tag) error {
	return e.EncodeStructure(TagBatchCount, func(e *Encoder) error {
		e.EncodeInt(TagActivationDate, 4)
		e.EncodeInt(TagAlternativeName, 5)
		e.EncodeInt(TagBatchCount, 3)
		err := e.EncodeStructure(TagArchiveDate, func(e *Encoder) error {
			return e.EncodeValue(TagBatchCount, 3)
		})
		if err != nil {
			return err
		}
		e.EncodeTextString(TagCancellationResult, "blue")
		return e.EncodeStructure(TagAuthenticatedEncryptionAdditionalData, func(e *Encoder) error {
			return e.EncodeStructure(TagMaskGenerator, func(e *Encoder) error {
				return e.EncodeValue(TagBatchCount, 3)
			})

		})
	})

}

type MarshalerFunc func(e *Encoder, tag Tag) error

func (f MarshalerFunc) MarshalTTLV(e *Encoder, tag Tag) error {
	// This makes it safe to call a nil MarshalerFunc
	if f == nil {
		return nil
	}
	return f(e, tag)
}

type ptrMarshaler struct{}

func (*ptrMarshaler) MarshalTTLV(e *Encoder, tag Tag) error {
	return e.EncodeValue(tag, 5)
}

type nonptrMarshaler struct{}

func (nonptrMarshaler) MarshalTTLV(e *Encoder, tag Tag) error {
	return e.EncodeValue(tag, 5)
}

func TestMarshal(t *testing.T) {
	for _, sample := range knownGoodSamples {
		tname := sample.name
		if tname == "" {
			tname = fmt.Sprintf("%T", sample.v)
		}
		t.Run(tname, func(t *testing.T) {
			exp := hex2bytes(sample.exp)

			var got []byte
			var err error

			got, err = Marshal(TaggedValue{Tag: Tag(0x420001), Value: sample.v})

			require.NoError(t, err)
			assert.Equal(t, exp, got)
		})
	}
}

func TestEncoder_Encode(t *testing.T) {
	_, err := Marshal(MarshalerStruct{})
	require.NoError(t, err)
}

func TestEncoder_EncodeValue_errors(t *testing.T) {
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
			v: struct {
				CustomAttribute struct {
					AttributeValue complex128
				}
			}{},
			expErr: ErrUnsupportedTypeError,
		},
		{
			name: "unsupportedtypeignoresomitempty",
			v: struct {
				Attribute complex128 `kmip:",omitempty"`
			}{},
			expErr: ErrUnsupportedTypeError,
		},
	}
	enc := NewEncoder(bytes.NewBuffer(nil))
	for _, test := range tests {
		testName := test.name
		if testName == "" {
			testName = fmt.Sprintf("%T", test.v)
		}
		t.Run(testName, func(t *testing.T) {
			err := enc.EncodeValue(TagCancellationResult, test.v)
			require.Error(t, err)
			t.Log(Details(err))
			require.True(t, Is(err, test.expErr), Details(err))
		})
	}
}

type Marshalablefloat32 float32

func (Marshalablefloat32) MarshalTTLV(e *Encoder, tag Tag) error {
	return e.EncodeValue(tag, 5)
}

type Marshalablefloat32Ptr float32

func (*Marshalablefloat32Ptr) MarshalTTLV(e *Encoder, tag Tag) error {
	return e.EncodeValue(tag, 5)
}

type Marshalablefloat64 float64

func (Marshalablefloat64) MarshalTTLV(e *Encoder, tag Tag) error {
	return e.EncodeValue(tag, 5)
}

type Marshalablefloat64Ptr float64

func (*Marshalablefloat64Ptr) MarshalTTLV(e *Encoder, tag Tag) error {
	return e.EncodeValue(tag, 5)
}

type MarshalableMap map[string]string

func (MarshalableMap) MarshalTTLV(e *Encoder, tag Tag) error {
	return e.EncodeValue(tag, 5)
}

type MarshalableMapPtr map[string]string

func (*MarshalableMapPtr) MarshalTTLV(e *Encoder, tag Tag) error {
	return e.EncodeValue(tag, 5)
}

type MarshalableSlice []string

func (MarshalableSlice) MarshalTTLV(e *Encoder, tag Tag) error {
	return e.EncodeValue(tag, 5)
}

type MarshalableSlicePtr []string

func (*MarshalableSlicePtr) MarshalTTLV(e *Encoder, tag Tag) error {
	return e.EncodeValue(tag, 5)
}

func TestEncoder_EncodeValue(t *testing.T) {

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

	type attr struct {
		AttributeName  string
		Value          interface{} `kmip:"AttributeValue"`
		AttributeIndex int         `kmip:",omitempty"`
	}
	type cert struct {
		CertificateIdentifier string
		CertificateIssuer     struct {
			CertificateIssuerAlternativeName string
			CertificateIssuerC               *string
			CertificateIssuerEmail           uint32 `kmip:",enum"`
			CN                               string `kmip:"CertificateIssuerCN,enum"`
			CertificateIssuerUID             uint32 `kmip:",omitempty,enum"`
			DC                               uint32 `kmip:"CertificateIssuerDC,omitempty,enum"`
			Len                              int    `kmip:"CertificateLength,omitempty,enum"`
		}
	}
	type Complex struct {
		Attribute       []attr
		Certificate     *cert
		BlockCipherMode nonptrMarshaler
	}

	type testCase struct {
		name         string
		tag          Tag
		noDefaultTag bool
		v            interface{}
		expected     interface{}
	}

	tests := []testCase{
		// byte strings
		{
			name:     "byteslice",
			v:        []byte{0x01, 0x02, 0x03},
			expected: TaggedValue{Tag: TagCancellationResult, Value: []byte{0x01, 0x02, 0x03}},
		},
		{
			name:     "bytesliceptr",
			v:        func() *[]byte { b := []byte{0x01, 0x02, 0x03}; return &b }(),
			expected: TaggedValue{Tag: TagCancellationResult, Value: []byte{0x01, 0x02, 0x03}},
		},
		// text strings
		{
			v:        "red",
			expected: TaggedValue{Tag: TagCancellationResult, Value: "red"},
		},
		{
			name:     "array",
			v:        [1]string{"red"},
			expected: TaggedValue{Tag: TagCancellationResult, Value: "red"},
		},
		{
			name:     "strptr",
			v:        func() *string { s := "red"; return &s }(),
			expected: TaggedValue{Tag: TagCancellationResult, Value: "red"},
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
			v:        parseTime("2008-03-14T11:56:40Z"),
			expected: TaggedValue{Tag: TagCancellationResult, Value: parseTime("2008-03-14T11:56:40Z")},
		},
		// big int ptr
		{
			v:        parseBigInt("1234567890000000000000000000"),
			expected: TaggedValue{Tag: TagCancellationResult, Value: parseBigInt("1234567890000000000000000000")},
		},
		// big int
		{
			v:        func() interface{} { return *(parseBigInt("1234567890000000000000000000")) }(),
			expected: TaggedValue{Tag: TagCancellationResult, Value: parseBigInt("1234567890000000000000000000")},
		},
		// duration
		{
			v:        time.Second * 10,
			expected: TaggedValue{Tag: TagCancellationResult, Value: time.Second * 10},
		},
		// boolean
		{
			v:        true,
			expected: TaggedValue{Tag: TagCancellationResult, Value: true},
		},
		// enum value
		{
			name:     "enum",
			v:        CredentialTypeAttestation,
			expected: TaggedValue{Tag: TagCancellationResult, Value: EnumValue(0x03)},
		},
		// slice
		{
			name: "slice",
			v:    []interface{}{5, 6, 7},
			expected: []interface{}{
				TaggedValue{Tag: TagCancellationResult, Value: int32(5)},
				TaggedValue{Tag: TagCancellationResult, Value: int32(6)},
				TaggedValue{Tag: TagCancellationResult, Value: int32(7)},
			},
		},
		{
			v: []string{"red", "green", "blue"},
			expected: []interface{}{
				TaggedValue{Tag: TagCancellationResult, Value: "red"},
				TaggedValue{Tag: TagCancellationResult, Value: "green"},
				TaggedValue{Tag: TagCancellationResult, Value: "blue"},
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
			expected: TaggedValue{Tag: TagArchiveDate, Value: int32(5)},
		},
		{
			name:     "namedtype",
			v:        AttributeValue("blue"),
			expected: TaggedValue{Tag: TagCancellationResult, Value: "blue"},
		},
		{
			name:         "namedtypenotag",
			noDefaultTag: true,
			v:            AttributeValue("blue"),
			expected:     TaggedValue{Tag: TagAttributeValue, Value: "blue"},
		},
		// struct
		{
			name: "struct",
			v:    struct{ AttributeName string }{"red"},
			expected: Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeName, Value: "red"},
				},
			},
		},
		{
			name: "structtag",
			v: struct {
				AttributeName string `kmip:"Attribute"`
			}{"red"},
			expected: Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttribute, Value: "red"},
				},
			},
		},
		{
			name: "structptr",
			v:    &Attribute{"red"},
			expected: Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeValue, Value: "red"},
				},
			},
		},
		{
			name: "structtaghex",
			v: struct {
				AttributeName string `kmip:"0x42000b"`
			}{"red"},
			expected: Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeValue, Value: "red"},
				},
			},
		},
		{
			name: "structtagskip",
			v: struct {
				AttributeName  string `kmip:"-"`
				AttributeValue string
			}{"red", "green"},
			expected: Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeValue, Value: "green"},
				},
			},
		},
		{
			name: "skipstructanonfield",
			v: struct {
				AttributeName string
				Attribute
			}{"red", Attribute{"green"}},
			expected: Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeName, Value: "red"},
				},
			},
		},
		{
			name: "skipnonexportedfields",
			v: struct {
				AttributeName  string
				attributeValue string
			}{"red", "green"},
			expected: Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeName, Value: "red"},
				},
			},
		},
		{
			name: "dateTimeTypes",
			v: struct {
				A time.Time        `kmip:"CertificateIssuerCN"`
				B time.Time        `kmip:"CertificateIssuerDC,dateTimeExtended"`
				C DateTimeExtended `kmip:"AttributeName"`
			}{
				parseTime("2008-03-14T11:56:40.123456Z"),
				parseTime("2008-03-14T11:56:40.123456Z"),
				DateTimeExtended{parseTime("2008-03-14T11:56:40.123456Z")},
			},
			expected: Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagCertificateIssuerCN, Value: parseTime("2008-03-14T11:56:40Z")},
					TaggedValue{Tag: TagCertificateIssuerDC, Value: DateTimeExtended{parseTime("2008-03-14T11:56:40.123456Z")}},
					TaggedValue{Tag: TagAttributeName, Value: DateTimeExtended{parseTime("2008-03-14T11:56:40.123456Z")}},
				},
			},
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
			expected: Structure{
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
			},
		},
		{
			name: "nilmarshalerfields",
			v:    &MarshalableFields{},
			expected: Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{Tag: TagAttributeValue, Value: int32(5)},
					TaggedValue{Tag: TagCustomAttribute, Value: int32(5)},
				},
			},
		},
		{
			name:     "invalidnamedtypemarshaler",
			v:        Marshalablefloat32(7),
			expected: TaggedValue{Tag: TagCancellationResult, Value: int32(5)},
		},
		{
			name: "tagfromtype",
			v: struct {
				Color AttributeValue
			}{"red"},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttributeValue, Value: "red"},
			},
			},
		},
		{
			name: "tagfromfieldname",
			v: struct {
				AttributeValue string
			}{"red"},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttributeValue, Value: "red"},
			},
			},
		},
		{
			name: "tagfromfieldtag",
			v: struct {
				Color string `kmip:"ArchiveDate"`
			}{"red"},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagArchiveDate, Value: "red"},
			},
			},
		},
		{
			name: "fieldtagoverridesfieldname",
			v: struct {
				AttributeValue string `kmip:"ArchiveDate"`
			}{"red"},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagArchiveDate, Value: "red"},
			},
			},
		},
		{
			name: "fieldtagoverridestype",
			v: struct {
				Color AttributeValue `kmip:"ArchiveDate"`
			}{"red"},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagArchiveDate, Value: "red"},
			},
			},
		},
		{
			name: "fieldnameoverridestype",
			v: struct {
				ArchiveDate AttributeValue
			}{"red"},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagArchiveDate, Value: "red"},
			},
			},
		},
		{
			name: "omitempty",
			v: struct {
				Attribute      string
				AttributeValue string `kmip:",omitempty"`
				ArchiveDate    string `kmip:",omitempty"`
			}{
				AttributeValue: "blue",
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: ""},
				TaggedValue{Tag: TagAttributeValue, Value: "blue"},
			}},
		},
		{
			name: "omitemptydatetime",
			v: struct {
				Attribute      time.Time
				AttributeValue time.Time `kmip:",omitempty"`
				ArchiveDate    time.Time `kmip:",omitempty"`
			}{
				AttributeValue: parseTime("2008-03-14T11:56:40Z"),
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: time.Time{}},
				TaggedValue{Tag: TagAttributeValue, Value: parseTime("2008-03-14T11:56:40Z")},
			}},
		},
		{
			name: "omitemptydatetimeptr",
			v: struct {
				Attribute      *time.Time
				AttributeValue *time.Time `kmip:",omitempty"`
				ArchiveDate    *time.Time `kmip:",omitempty"`
			}{
				Attribute:      &time.Time{},
				AttributeValue: func() *time.Time { t := parseTime("2008-03-14T11:56:40Z"); return &t }(),
				ArchiveDate:    &time.Time{},
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: time.Time{}},
				TaggedValue{Tag: TagAttributeValue, Value: parseTime("2008-03-14T11:56:40Z")},
			}},
		},
		{
			name: "omitemptybigint",
			v: struct {
				Attribute      big.Int
				AttributeValue big.Int `kmip:",omitempty"`
				ArchiveDate    big.Int `kmip:",omitempty"`
			}{
				AttributeValue: *parseBigInt("1"),
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: big.Int{}},
				TaggedValue{Tag: TagAttributeValue, Value: parseBigInt("1")},
			}},
		},
		{
			name: "omitemptybigintptr",
			v: struct {
				Attribute      *big.Int
				AttributeValue *big.Int `kmip:",omitempty"`
				ArchiveDate    *big.Int `kmip:",omitempty"`
			}{
				Attribute:      parseBigInt("0"),
				AttributeValue: parseBigInt("1"),
				ArchiveDate:    parseBigInt("0"),
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: big.Int{}},
				TaggedValue{Tag: TagAttributeValue, Value: parseBigInt("1")},
			}},
		},
		{
			name: "omitemptyint",
			v: struct {
				Attribute      int
				AttributeValue int `kmip:",omitempty"`
				ArchiveDate    int `kmip:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(0)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyint8",
			v: struct {
				Attribute      int8
				AttributeValue int8 `kmip:",omitempty"`
				ArchiveDate    int8 `kmip:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(0)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyint16",
			v: struct {
				Attribute      int16
				AttributeValue int16 `kmip:",omitempty"`
				ArchiveDate    int16 `kmip:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(0)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyint32",
			v: struct {
				Attribute      int32
				AttributeValue int32 `kmip:",omitempty"`
				ArchiveDate    int32 `kmip:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(0)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyint64",
			v: struct {
				Attribute      int64
				AttributeValue int64 `kmip:",omitempty"`
				ArchiveDate    int64 `kmip:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int64(0)},
				TaggedValue{Tag: TagAttributeValue, Value: int64(6)},
			}},
		},
		{
			name: "omitemptyuint",
			v: struct {
				Attribute      uint
				AttributeValue uint `kmip:",omitempty"`
				ArchiveDate    uint `kmip:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(0)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyuint8",
			v: struct {
				Attribute      uint8
				AttributeValue uint8 `kmip:",omitempty"`
				ArchiveDate    uint8 `kmip:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(0)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyuint16",
			v: struct {
				Attribute      uint16
				AttributeValue uint16 `kmip:",omitempty"`
				ArchiveDate    uint16 `kmip:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(0)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyuint32",
			v: struct {
				Attribute      uint32
				AttributeValue uint32 `kmip:",omitempty"`
				ArchiveDate    uint32 `kmip:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(0)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyuint64",
			v: struct {
				Attribute      uint64
				AttributeValue uint64 `kmip:",omitempty"`
				ArchiveDate    uint64 `kmip:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int64(0)},
				TaggedValue{Tag: TagAttributeValue, Value: int64(6)},
			}},
		},
		{
			name: "omitemptybool",
			v: struct {
				Attribute      bool
				AttributeValue bool `kmip:",omitempty"`
				ArchiveDate    bool `kmip:",omitempty"`
			}{
				AttributeValue: true,
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: false},
				TaggedValue{Tag: TagAttributeValue, Value: true},
			}},
		},
		{
			name: "omitemptyfloat32",
			v: struct {
				Attribute      Marshalablefloat32
				AttributeValue Marshalablefloat32 `kmip:",omitempty"`
				ArchiveDate    Marshalablefloat32 `kmip:",omitempty"`
			}{
				AttributeValue: Marshalablefloat32(6),
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(5)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptyfloat32ptr",
			v: &struct {
				Attribute      Marshalablefloat32Ptr
				AttributeValue Marshalablefloat32Ptr `kmip:",omitempty"`
				ArchiveDate    Marshalablefloat32Ptr `kmip:",omitempty"`
			}{
				AttributeValue: Marshalablefloat32Ptr(7),
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(5)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptyfloat64",
			v: &struct {
				Attribute      Marshalablefloat64
				AttributeValue Marshalablefloat64 `kmip:",omitempty"`
				ArchiveDate    Marshalablefloat64 `kmip:",omitempty"`
			}{
				AttributeValue: Marshalablefloat64(7),
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(5)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptyfloat64ptr",
			v: &struct {
				Attribute      Marshalablefloat64Ptr
				AttributeValue Marshalablefloat64Ptr `kmip:",omitempty"`
				ArchiveDate    Marshalablefloat64Ptr `kmip:",omitempty"`
			}{
				AttributeValue: Marshalablefloat64Ptr(7),
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(5)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptymap",
			v: &struct {
				AttributeIndex MarshalableMap
				Attribute      MarshalableMap
				AttributeValue MarshalableMap `kmip:",omitempty"`
				ArchiveDate    MarshalableMap `kmip:",omitempty"`
			}{
				Attribute:      MarshalableMap{},
				AttributeValue: MarshalableMap{"color": "red"},
				ArchiveDate:    MarshalableMap{},
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(5)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptymapptr",
			v: &struct {
				AttributeIndex MarshalableMapPtr
				Attribute      MarshalableMapPtr
				AttributeValue MarshalableMapPtr `kmip:",omitempty"`
				ArchiveDate    MarshalableMapPtr `kmip:",omitempty"`
			}{
				Attribute:      MarshalableMapPtr{},
				AttributeValue: MarshalableMapPtr{"color": "red"},
				ArchiveDate:    MarshalableMapPtr{},
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(5)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptymarshalableslice",
			v: &struct {
				AttributeIndex MarshalableSlice
				Attribute      MarshalableSlice
				AttributeValue MarshalableSlice `kmip:",omitempty"`
				ArchiveDate    MarshalableSlice `kmip:",omitempty"`
			}{
				Attribute:      MarshalableSlice{},
				AttributeValue: MarshalableSlice{"color"},
				ArchiveDate:    MarshalableSlice{},
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(5)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptymarshalablesliceptr",
			v: &struct {
				AttributeIndex MarshalableSlicePtr
				Attribute      MarshalableSlicePtr
				AttributeValue MarshalableSlicePtr `kmip:",omitempty"`
				ArchiveDate    MarshalableSlicePtr `kmip:",omitempty"`
			}{
				Attribute:      MarshalableSlicePtr{},
				AttributeValue: MarshalableSlicePtr{"color"},
				ArchiveDate:    MarshalableSlicePtr{},
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagAttribute, Value: int32(5)},
				TaggedValue{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "enumtag",
			v: struct {
				Comment string `kmip:",enum"`
				Int     int    `kmip:"CommonTemplateAttribute,enum"`
				Int8    int8   `kmip:"CompromiseDate,enum"`
				Int16   int16  `kmip:"CompromiseOccurrenceDate,enum"`
				Int32   int32  `kmip:"ContactInformation,enum"`
				Int64   int64  `kmip:"CorrelationValue,enum"`
				Uint    uint   `kmip:"CounterLength,enum"`
				Uint8   uint8  `kmip:"Credential,enum"`
				Uint16  uint16 `kmip:"CredentialType,enum"`
				Uint32  uint32 `kmip:"CredentialValue,enum"`
				Uint64  uint64 `kmip:"CriticalityIndicator,enum"`
			}{
				Comment: "0x00000001",
				Int:     2,
				Int8:    3,
				Int16:   4,
				Int32:   5,
				Int64:   6,
				Uint:    7,
				Uint8:   8,
				Uint16:  9,
				Uint32:  10,
				Uint64:  11,
			},
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				TaggedValue{Tag: TagComment, Value: EnumValue(1)},
				TaggedValue{Tag: TagCommonTemplateAttribute, Value: EnumValue(2)},
				TaggedValue{Tag: TagCompromiseDate, Value: EnumValue(3)},
				TaggedValue{Tag: TagCompromiseOccurrenceDate, Value: EnumValue(4)},
				TaggedValue{Tag: TagContactInformation, Value: EnumValue(5)},
				TaggedValue{Tag: TagCorrelationValue, Value: EnumValue(6)},
				TaggedValue{Tag: TagCounterLength, Value: EnumValue(7)},
				TaggedValue{Tag: TagCredential, Value: EnumValue(8)},
				TaggedValue{Tag: TagCredentialType, Value: EnumValue(9)},
				TaggedValue{Tag: TagCredentialValue, Value: EnumValue(10)},
				TaggedValue{Tag: TagCriticalityIndicator, Value: EnumValue(11)},
			}},
		},
		{
			v: func() interface{} {
				c := Complex{
					Attribute: []attr{
						{
							AttributeName: "color",
							Value:         "red",
						},
						{
							AttributeName:  "size",
							Value:          5,
							AttributeIndex: 1,
						},
					},
					Certificate: &cert{
						CertificateIdentifier: "blue",
					},
				}
				c.Certificate.CertificateIssuer.Len = 4
				c.Certificate.CertificateIssuer.CertificateIssuerAlternativeName = "rick"
				s := "bob"
				c.Certificate.CertificateIssuer.CertificateIssuerC = &s
				c.Certificate.CertificateIssuer.CertificateIssuerEmail = 0
				c.Certificate.CertificateIssuer.CN = "0x00000002"
				c.Certificate.CertificateIssuer.CertificateIssuerUID = 3
				c.Certificate.CertificateIssuer.DC = 0
				c.Certificate.CertificateIssuer.Len = 10

				return c
			}(),
			expected: Structure{Tag: TagCancellationResult, Values: []interface{}{
				Structure{Tag: TagAttribute, Values: []interface{}{
					TaggedValue{Tag: TagAttributeName, Value: "color"},
					TaggedValue{Tag: TagAttributeValue, Value: "red"},
				}},
				Structure{Tag: TagAttribute, Values: []interface{}{
					TaggedValue{Tag: TagAttributeName, Value: "size"},
					TaggedValue{Tag: TagAttributeValue, Value: int32(5)},
					TaggedValue{Tag: TagAttributeIndex, Value: int32(1)},
				}},
				Structure{Tag: TagCertificate, Values: []interface{}{
					TaggedValue{Tag: TagCertificateIdentifier, Value: "blue"},
					Structure{Tag: TagCertificateIssuer, Values: []interface{}{
						TaggedValue{Tag: TagCertificateIssuerAlternativeName, Value: "rick"},
						TaggedValue{Tag: TagCertificateIssuerC, Value: "bob"},
						TaggedValue{Tag: TagCertificateIssuerEmail, Value: EnumValue(0)},
						TaggedValue{Tag: TagCertificateIssuerCN, Value: EnumValue(2)},
						TaggedValue{Tag: TagCertificateIssuerUID, Value: EnumValue(3)},
						TaggedValue{Tag: TagCertificateLength, Value: EnumValue(10)},
					}},
				}},
				TaggedValue{Tag: TagBlockCipherMode, Value: int32(5)},
			}},
		},
		{
			name:     "intenum",
			v:        TaggedValue{Tag: TagWrappingMethod, Value: int(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "int8enum",
			v:        TaggedValue{Tag: TagWrappingMethod, Value: int8(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "int16enum",
			v:        TaggedValue{Tag: TagWrappingMethod, Value: int16(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "int32enum",
			v:        TaggedValue{Tag: TagWrappingMethod, Value: int32(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "int64enum",
			v:        TaggedValue{Tag: TagWrappingMethod, Value: int64(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "uintenum",
			v:        TaggedValue{Tag: TagWrappingMethod, Value: uint(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "uint8enum",
			v:        TaggedValue{Tag: TagWrappingMethod, Value: uint8(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "uint16enum",
			v:        TaggedValue{Tag: TagWrappingMethod, Value: uint16(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "uint32enum",
			v:        TaggedValue{Tag: TagWrappingMethod, Value: uint32(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "uint64enum",
			v:        TaggedValue{Tag: TagWrappingMethod, Value: uint64(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
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
			buf := bytes.NewBuffer(nil)
			enc := NewEncoder(buf)

			tag := tc.tag
			if tc.tag == TagNone && !tc.noDefaultTag {
				tag = TagCancellationResult
			}

			err := enc.EncodeValue(tag, tc.v)

			require.NoError(t, err, Details(err))

			buf2 := bytes.NewBuffer(nil)
			enc2 := NewEncoder(buf2)
			err = enc2.EncodeValue(tag, tc.expected)
			require.NoError(t, err, Details(err))

			require.Equal(t, TTLV(buf2.Bytes()), TTLV(buf.Bytes()))
		})

	}

}

func TestEncoder_EncodeStructure(t *testing.T) {

	type testCase struct {
		name     string
		f        func(*Encoder) error
		expected interface{}
	}

	cases := []testCase{
		{
			name: "Encode Bool",
			f: func(e *Encoder) error {
				e.EncodeBool(TagActivationDate, true)
				return nil
			},
			expected: Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{
						Tag:   TagActivationDate,
						Value: true,
					},
				},
			},
		},
		{
			name: "Encode Value",
			f: func(e *Encoder) error {
				return e.EncodeValue(TagActivationDate, true)
			},
			expected: Structure{
				Tag: TagCancellationResult,
				Values: []interface{}{
					TaggedValue{
						Tag:   TagActivationDate,
						Value: true,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			enc := NewEncoder(buf)
			require.NoError(t, enc.EncodeStructure(TagCancellationResult, tc.f))

			require.NoError(t, enc.Flush())

			buf2 := bytes.NewBuffer(nil)
			enc2 := NewEncoder(buf2)
			require.NoError(t, enc2.EncodeValue(TagNone, tc.expected))

			require.Equal(t, TTLV(buf2.Bytes()), TTLV(buf.Bytes()))

		})
	}

}

func parseTime(s string) time.Time {
	v, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(err)
	}
	return v
}

func BenchmarkEncodeSlice(b *testing.B) {
	enc := NewEncoder(ioutil.Discard)

	type Attribute struct {
		AttributeValue string
	}

	v := Attribute{"red"}

	rv := reflect.ValueOf(v)

	for i := 0; i < b.N; i++ {
		_ = enc.encode(TagNone, rv, 0)
	}
}

type BatchItem struct {
	BatchCount            int
	Attribute             *BatchItem
	AttributeValue        string
	WrappingMethod        WrappingMethod
	CancellationResult    *BatchItem
	Certificate           int64
	CertificateIdentifier *big.Int
	CertificateIssuer     time.Time
	CertificateRequest    []interface{}
	CompromiseDate        []byte
	Credential            bool
	D                     time.Duration
}

func (b *BatchItem) MarshalTTLV(e *Encoder, tag Tag) error {
	return e.EncodeStructure(TagBatchItem, func(e *Encoder) error {
		e.EncodeInt(TagBatchCount, int32(b.BatchCount))
		if b.Attribute != nil {
			if err := b.Attribute.MarshalTTLV(e, TagAttribute); err != nil {
				return err
			}
		}
		e.EncodeTextString(TagAttributeValue, b.AttributeValue)
		e.EncodeEnumeration(TagWrappingMethod, uint32(b.WrappingMethod))
		if b.CancellationResult != nil {
			if err := b.CancellationResult.MarshalTTLV(e, TagCancellationResult); err != nil {
				return err
			}
		}
		e.EncodeLongInt(TagCertificate, b.Certificate)
		e.EncodeBigInt(TagCertificateIdentifier, b.CertificateIdentifier)
		e.EncodeDateTime(TagCertificateIssuer, b.CertificateIssuer)
		for _, v := range b.CertificateRequest {
			if err := e.EncodeValue(TagCertificateRequest, v); err != nil {
				return err
			}
		}
		e.EncodeByteString(TagCompromiseDate, b.CompromiseDate)
		e.EncodeBool(TagCredential, b.Credential)
		e.EncodeInterval(TagD, b.D)

		return nil
	})
}

func BenchmarkMarshal_struct(b *testing.B) {
	s := BatchItem{
		BatchCount:            10,
		AttributeValue:        "red",
		WrappingMethod:        WrappingMethodEncrypt,
		Certificate:           90,
		CertificateIdentifier: big.NewInt(200),
		CertificateIssuer:     time.Now(),
		CompromiseDate:        []byte("asdfasdfasdfas"),
		Credential:            true,
		D:                     time.Minute,
	}
	// make a couple clones
	s1, s2, s3, s4, s5, s6, s7, s8, s9 := s, s, s, s, s, s, s, s, s

	v := &s
	v.Attribute = &s1
	v.CancellationResult = &s2
	v.Attribute.Attribute = &s3
	v.CancellationResult.CancellationResult = &s4
	v.Attribute.Attribute.Attribute = &s5
	v.CancellationResult.CancellationResult.CancellationResult = &s6
	//v.CertificateRequest = append(v.CertificateRequest, s7, s8, s9)
	v.CertificateRequest = append(v.CertificateRequest, &s7, &s8, &s9)

	_, e := Marshal(v)
	require.NoError(b, e)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Marshal(v)
	}
}

func BenchmarkEncoder_Encode_interval(b *testing.B) {
	enc := NewEncoder(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		_ = enc.EncodeValue(TagCertificateRequest, time.Minute)
	}
}

func BenchmarkEncoder_EncodeByteString(b *testing.B) {
	enc := NewEncoder(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		enc.EncodeTextString(TagCertificateIssuer, "al;kjsaflksjdflakjsdfl;aksjdflaksjdflaksjdfl;ksjd")
		require.NoError(b, enc.Flush())

	}
}

func BenchmarkEncoder_EncodeInt(b *testing.B) {
	enc := NewEncoder(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		enc.EncodeInt(TagCertificateIssuer, 8)
		require.NoError(b, enc.Flush())

	}
}
