package ttlv

import (
	"bytes"
	"encoding/hex"
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

			got, err = Marshal(Value{Tag: Tag(0x420001), Value: sample.v})

			require.NoError(t, err)
			assert.Equal(t, exp, got)
		})
	}
}

func TestMarshal_tagPrecedence(t *testing.T) {
	// test precedence order for picking the tag to marshal to

	// first: the name of the type (only applies to values which
	// where did not come from a field)
	// next: name of struct field

	// for values not originating in a field, infer tag from the type name
	type Name struct {

		// for fields:

		// infer from field name
		Comment string

		// infer from field tag (higher precedent than name)
		AttributeValue string `ttlv:"BatchCount"`

		// infer from dynamic value if the TTLVTag subfield (higher precedent than the name or tag)
		ArchiveDate struct {
			TTLVTag Tag
			Comment string
		} `ttlv:"AttributeValue"`

		// highest precedent: the tag on the TTLVTag subfield
		AttributeName struct {
			TTLVTag Tag `ttlv:"PSource"`
			Comment string
		}

		// If this last option is specified, it must agree with the field tag
		MaskGenerator struct {
			TTLVTag Tag `ttlv:"Description"`
			Comment string
		} `ttlv:"Description"`
	}

	n := Name{
		Comment:        "red",
		AttributeValue: "blue",
	}

	// dynamic value of the TTLVTag field should override the field name
	n.ArchiveDate.TTLVTag = TagNameType
	n.ArchiveDate.Comment = "yellow"

	// this dynamic value of TTLVTag field will be ignored because there is
	// an explicit tag on the on it
	n.AttributeName.TTLVTag = TagObjectGroup
	n.AttributeName.Comment = "orange"
	n.MaskGenerator.Comment = "black"

	b, err := Marshal(n)
	require.NoError(t, err)

	var v Value
	err = Unmarshal(b, &v)
	require.NoError(t, err)

	assert.Equal(t, Value{
		Tag: TagName,
		Value: Values{
			{TagComment, "red"},
			// field name should be overridden by the explicit tag
			{TagBatchCount, "blue"},
			// tag should come from the value of TTLVTag
			{TagNameType, Values{
				{TagComment, "yellow"},
			}},
			// tag should come from the tag on TTLVTag field
			{TagPSource, Values{
				{TagComment, "orange"},
			}},
			{TagDescription, Values{
				{TagComment, "black"},
			}},
		},
	}, v)
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
				Attribute complex128 `ttlv:",omitempty"`
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
		Value          interface{} `ttlv:"AttributeValue"`
		AttributeIndex int         `ttlv:",omitempty"`
	}
	type cert struct {
		CertificateIdentifier string
		CertificateIssuer     struct {
			CertificateIssuerAlternativeName string
			CertificateIssuerC               *string
			CertificateIssuerEmail           uint32 `ttlv:",enum"`
			CN                               string `ttlv:"CertificateIssuerCN,enum"`
			CertificateIssuerUID             uint32 `ttlv:",omitempty,enum"`
			DC                               uint32 `ttlv:"CertificateIssuerDC,omitempty,enum"`
			Len                              int    `ttlv:"CertificateLength,omitempty,enum"`
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
			expected: Value{Tag: TagCancellationResult, Value: []byte{0x01, 0x02, 0x03}},
		},
		{
			name:     "bytesliceptr",
			v:        func() *[]byte { b := []byte{0x01, 0x02, 0x03}; return &b }(),
			expected: Value{Tag: TagCancellationResult, Value: []byte{0x01, 0x02, 0x03}},
		},
		// text strings
		{
			v:        "red",
			expected: Value{Tag: TagCancellationResult, Value: "red"},
		},
		{
			name:     "array",
			v:        [1]string{"red"},
			expected: Value{Tag: TagCancellationResult, Value: "red"},
		},
		{
			name:     "strptr",
			v:        func() *string { s := "red"; return &s }(),
			expected: Value{Tag: TagCancellationResult, Value: "red"},
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
			expected: Value{Tag: TagCancellationResult, Value: parseTime("2008-03-14T11:56:40Z")},
		},
		// big int ptr
		{
			v:        parseBigInt("1234567890000000000000000000"),
			expected: Value{Tag: TagCancellationResult, Value: parseBigInt("1234567890000000000000000000")},
		},
		// big int
		{
			v:        func() interface{} { return *(parseBigInt("1234567890000000000000000000")) }(),
			expected: Value{Tag: TagCancellationResult, Value: parseBigInt("1234567890000000000000000000")},
		},
		// duration
		{
			v:        time.Second * 10,
			expected: Value{Tag: TagCancellationResult, Value: time.Second * 10},
		},
		// boolean
		{
			v:        true,
			expected: Value{Tag: TagCancellationResult, Value: true},
		},
		// enum value
		{
			name:     "enum",
			v:        CredentialTypeAttestation,
			expected: Value{Tag: TagCancellationResult, Value: EnumValue(0x03)},
		},
		// slice
		{
			name: "slice",
			v:    []interface{}{5, 6, 7},
			expected: []interface{}{
				Value{Tag: TagCancellationResult, Value: int32(5)},
				Value{Tag: TagCancellationResult, Value: int32(6)},
				Value{Tag: TagCancellationResult, Value: int32(7)},
			},
		},
		{
			v: []string{"red", "green", "blue"},
			expected: []interface{}{
				Value{Tag: TagCancellationResult, Value: "red"},
				Value{Tag: TagCancellationResult, Value: "green"},
				Value{Tag: TagCancellationResult, Value: "blue"},
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
			expected: Value{Tag: TagArchiveDate, Value: int32(5)},
		},
		{
			name:     "namedtype",
			v:        AttributeValue("blue"),
			expected: Value{Tag: TagCancellationResult, Value: "blue"},
		},
		{
			name:         "namedtypenotag",
			noDefaultTag: true,
			v:            AttributeValue("blue"),
			expected:     Value{Tag: TagAttributeValue, Value: "blue"},
		},
		// struct
		{
			name: "struct",
			v:    struct{ AttributeName string }{"red"},
			expected: Value{
				Tag: TagCancellationResult,
				Value: Values{
					Value{Tag: TagAttributeName, Value: "red"},
				},
			},
		},
		{
			name: "structtag",
			v: struct {
				AttributeName string `ttlv:"Attribute"`
			}{"red"},
			expected: Value{
				Tag: TagCancellationResult,
				Value: Values{
					Value{Tag: TagAttribute, Value: "red"},
				},
			},
		},
		{
			name: "structptr",
			v:    &Attribute{"red"},
			expected: Value{
				Tag: TagCancellationResult,
				Value: Values{
					Value{Tag: TagAttributeValue, Value: "red"},
				},
			},
		},
		{
			name: "structtaghex",
			v: struct {
				AttributeName string `ttlv:"0x42000b"`
			}{"red"},
			expected: Value{
				Tag: TagCancellationResult,
				Value: Values{
					Value{Tag: TagAttributeValue, Value: "red"},
				},
			},
		},
		{
			name: "structtagskip",
			v: struct {
				AttributeName  string `ttlv:"-"`
				AttributeValue string
			}{"red", "green"},
			expected: Value{
				Tag: TagCancellationResult,
				Value: Values{
					Value{Tag: TagAttributeValue, Value: "green"},
				},
			},
		},
		{
			name: "skipstructanonfield",
			v: struct {
				AttributeName string
				Attribute
			}{"red", Attribute{"green"}},
			expected: Value{
				Tag: TagCancellationResult,
				Value: Values{
					Value{Tag: TagAttributeName, Value: "red"},
				},
			},
		},
		{
			name: "skipnonexportedfields",
			v: struct {
				AttributeName  string
				attributeValue string
			}{"red", "green"},
			expected: Value{
				Tag: TagCancellationResult,
				Value: Values{
					Value{Tag: TagAttributeName, Value: "red"},
				},
			},
		},
		{
			name: "dateTimeTypes",
			v: struct {
				A time.Time        `ttlv:"CertificateIssuerCN"`
				B time.Time        `ttlv:"CertificateIssuerDC,dateTimeExtended"`
				C DateTimeExtended `ttlv:"AttributeName"`
			}{
				parseTime("2008-03-14T11:56:40.123456Z"),
				parseTime("2008-03-14T11:56:40.123456Z"),
				DateTimeExtended{parseTime("2008-03-14T11:56:40.123456Z")},
			},
			expected: Value{
				Tag: TagCancellationResult,
				Value: Values{
					Value{Tag: TagCertificateIssuerCN, Value: parseTime("2008-03-14T11:56:40Z")},
					Value{Tag: TagCertificateIssuerDC, Value: DateTimeExtended{parseTime("2008-03-14T11:56:40.123456Z")}},
					Value{Tag: TagAttributeName, Value: DateTimeExtended{parseTime("2008-03-14T11:56:40.123456Z")}},
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
			expected: Value{
				Tag: TagCancellationResult,
				Value: Values{
					Value{Tag: TagAttribute, Value: int32(5)},
					Value{Tag: TagAttributeName, Value: int32(5)},
					Value{Tag: TagAttributeValue, Value: int32(5)},
					Value{Tag: TagArchiveDate, Value: int32(5)},
					Value{Tag: TagCancellationResult, Value: int32(5)},
					Value{Tag: TagCustomAttribute, Value: int32(5)},
					Value{Tag: TagAttributeIndex, Value: int32(5)},
					Value{Tag: TagCertificate, Value: int32(5)},
				},
			},
		},
		{
			name: "nilmarshalerfields",
			v:    &MarshalableFields{},
			expected: Value{
				Tag: TagCancellationResult,
				Value: Values{
					Value{Tag: TagAttributeValue, Value: int32(5)},
					Value{Tag: TagCustomAttribute, Value: int32(5)},
				},
			},
		},
		{
			name:     "invalidnamedtypemarshaler",
			v:        Marshalablefloat32(7),
			expected: Value{Tag: TagCancellationResult, Value: int32(5)},
		},
		{
			name: "tagfromfieldname",
			v: struct {
				AttributeValue string
			}{"red"},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttributeValue, Value: "red"},
			},
			},
		},
		{
			name: "tagfromfieldtag",
			v: struct {
				Color string `ttlv:"ArchiveDate"`
			}{"red"},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagArchiveDate, Value: "red"},
			},
			},
		},
		{
			name: "fieldtagoverridesfieldname",
			v: struct {
				AttributeValue string `ttlv:"ArchiveDate"`
			}{"red"},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagArchiveDate, Value: "red"},
			},
			},
		},
		{
			name: "fieldtagoverridestype",
			v: struct {
				Color AttributeValue `ttlv:"ArchiveDate"`
			}{"red"},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagArchiveDate, Value: "red"},
			},
			},
		},
		{
			name: "fieldnameoverridestype",
			v: struct {
				ArchiveDate AttributeValue
			}{"red"},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagArchiveDate, Value: "red"},
			},
			},
		},
		{
			name: "omitempty",
			v: struct {
				Attribute      string
				AttributeValue string `ttlv:",omitempty"`
				ArchiveDate    string `ttlv:",omitempty"`
			}{
				AttributeValue: "blue",
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: ""},
				Value{Tag: TagAttributeValue, Value: "blue"},
			}},
		},
		{
			name: "omitemptydatetime",
			v: struct {
				Attribute      time.Time
				AttributeValue time.Time `ttlv:",omitempty"`
				ArchiveDate    time.Time `ttlv:",omitempty"`
			}{
				AttributeValue: parseTime("2008-03-14T11:56:40Z"),
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: time.Time{}},
				Value{Tag: TagAttributeValue, Value: parseTime("2008-03-14T11:56:40Z")},
			}},
		},
		{
			name: "omitemptydatetimeptr",
			v: struct {
				Attribute      *time.Time
				AttributeValue *time.Time `ttlv:",omitempty"`
				ArchiveDate    *time.Time `ttlv:",omitempty"`
			}{
				Attribute:      &time.Time{},
				AttributeValue: func() *time.Time { t := parseTime("2008-03-14T11:56:40Z"); return &t }(),
				ArchiveDate:    &time.Time{},
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: time.Time{}},
				Value{Tag: TagAttributeValue, Value: parseTime("2008-03-14T11:56:40Z")},
			}},
		},
		{
			name: "omitemptybigint",
			v: struct {
				Attribute      big.Int
				AttributeValue big.Int `ttlv:",omitempty"`
				ArchiveDate    big.Int `ttlv:",omitempty"`
			}{
				AttributeValue: *parseBigInt("1"),
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: big.Int{}},
				Value{Tag: TagAttributeValue, Value: parseBigInt("1")},
			}},
		},
		{
			name: "omitemptybigintptr",
			v: struct {
				Attribute      *big.Int
				AttributeValue *big.Int `ttlv:",omitempty"`
				ArchiveDate    *big.Int `ttlv:",omitempty"`
			}{
				Attribute:      parseBigInt("0"),
				AttributeValue: parseBigInt("1"),
				ArchiveDate:    parseBigInt("0"),
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: big.Int{}},
				Value{Tag: TagAttributeValue, Value: parseBigInt("1")},
			}},
		},
		{
			name: "omitemptyint",
			v: struct {
				Attribute      int
				AttributeValue int `ttlv:",omitempty"`
				ArchiveDate    int `ttlv:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(0)},
				Value{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyint8",
			v: struct {
				Attribute      int8
				AttributeValue int8 `ttlv:",omitempty"`
				ArchiveDate    int8 `ttlv:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(0)},
				Value{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyint16",
			v: struct {
				Attribute      int16
				AttributeValue int16 `ttlv:",omitempty"`
				ArchiveDate    int16 `ttlv:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(0)},
				Value{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyint32",
			v: struct {
				Attribute      int32
				AttributeValue int32 `ttlv:",omitempty"`
				ArchiveDate    int32 `ttlv:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(0)},
				Value{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyint64",
			v: struct {
				Attribute      int64
				AttributeValue int64 `ttlv:",omitempty"`
				ArchiveDate    int64 `ttlv:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int64(0)},
				Value{Tag: TagAttributeValue, Value: int64(6)},
			}},
		},
		{
			name: "omitemptyuint",
			v: struct {
				Attribute      uint
				AttributeValue uint `ttlv:",omitempty"`
				ArchiveDate    uint `ttlv:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(0)},
				Value{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyuint8",
			v: struct {
				Attribute      uint8
				AttributeValue uint8 `ttlv:",omitempty"`
				ArchiveDate    uint8 `ttlv:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(0)},
				Value{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyuint16",
			v: struct {
				Attribute      uint16
				AttributeValue uint16 `ttlv:",omitempty"`
				ArchiveDate    uint16 `ttlv:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(0)},
				Value{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyuint32",
			v: struct {
				Attribute      uint32
				AttributeValue uint32 `ttlv:",omitempty"`
				ArchiveDate    uint32 `ttlv:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(0)},
				Value{Tag: TagAttributeValue, Value: int32(6)},
			}},
		},
		{
			name: "omitemptyuint64",
			v: struct {
				Attribute      uint64
				AttributeValue uint64 `ttlv:",omitempty"`
				ArchiveDate    uint64 `ttlv:",omitempty"`
			}{
				AttributeValue: 6,
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int64(0)},
				Value{Tag: TagAttributeValue, Value: int64(6)},
			}},
		},
		{
			name: "omitemptybool",
			v: struct {
				Attribute      bool
				AttributeValue bool `ttlv:",omitempty"`
				ArchiveDate    bool `ttlv:",omitempty"`
			}{
				AttributeValue: true,
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: false},
				Value{Tag: TagAttributeValue, Value: true},
			}},
		},
		{
			name: "omitemptyfloat32",
			v: struct {
				Attribute      Marshalablefloat32
				AttributeValue Marshalablefloat32 `ttlv:",omitempty"`
				ArchiveDate    Marshalablefloat32 `ttlv:",omitempty"`
			}{
				AttributeValue: Marshalablefloat32(6),
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(5)},
				Value{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptyfloat32ptr",
			v: &struct {
				Attribute      Marshalablefloat32Ptr
				AttributeValue Marshalablefloat32Ptr `ttlv:",omitempty"`
				ArchiveDate    Marshalablefloat32Ptr `ttlv:",omitempty"`
			}{
				AttributeValue: Marshalablefloat32Ptr(7),
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(5)},
				Value{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptyfloat64",
			v: &struct {
				Attribute      Marshalablefloat64
				AttributeValue Marshalablefloat64 `ttlv:",omitempty"`
				ArchiveDate    Marshalablefloat64 `ttlv:",omitempty"`
			}{
				AttributeValue: Marshalablefloat64(7),
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(5)},
				Value{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptyfloat64ptr",
			v: &struct {
				Attribute      Marshalablefloat64Ptr
				AttributeValue Marshalablefloat64Ptr `ttlv:",omitempty"`
				ArchiveDate    Marshalablefloat64Ptr `ttlv:",omitempty"`
			}{
				AttributeValue: Marshalablefloat64Ptr(7),
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(5)},
				Value{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptymap",
			v: &struct {
				AttributeIndex MarshalableMap
				Attribute      MarshalableMap
				AttributeValue MarshalableMap `ttlv:",omitempty"`
				ArchiveDate    MarshalableMap `ttlv:",omitempty"`
			}{
				Attribute:      MarshalableMap{},
				AttributeValue: MarshalableMap{"color": "red"},
				ArchiveDate:    MarshalableMap{},
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(5)},
				Value{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptymapptr",
			v: &struct {
				AttributeIndex MarshalableMapPtr
				Attribute      MarshalableMapPtr
				AttributeValue MarshalableMapPtr `ttlv:",omitempty"`
				ArchiveDate    MarshalableMapPtr `ttlv:",omitempty"`
			}{
				Attribute:      MarshalableMapPtr{},
				AttributeValue: MarshalableMapPtr{"color": "red"},
				ArchiveDate:    MarshalableMapPtr{},
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(5)},
				Value{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptymarshalableslice",
			v: &struct {
				AttributeIndex MarshalableSlice
				Attribute      MarshalableSlice
				AttributeValue MarshalableSlice `ttlv:",omitempty"`
				ArchiveDate    MarshalableSlice `ttlv:",omitempty"`
			}{
				Attribute:      MarshalableSlice{},
				AttributeValue: MarshalableSlice{"color"},
				ArchiveDate:    MarshalableSlice{},
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(5)},
				Value{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "omitemptymarshalablesliceptr",
			v: &struct {
				AttributeIndex MarshalableSlicePtr
				Attribute      MarshalableSlicePtr
				AttributeValue MarshalableSlicePtr `ttlv:",omitempty"`
				ArchiveDate    MarshalableSlicePtr `ttlv:",omitempty"`
			}{
				Attribute:      MarshalableSlicePtr{},
				AttributeValue: MarshalableSlicePtr{"color"},
				ArchiveDate:    MarshalableSlicePtr{},
			},
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: int32(5)},
				Value{Tag: TagAttributeValue, Value: int32(5)},
			}},
		},
		{
			name: "enumtag",
			v: struct {
				Comment string `ttlv:",enum"`
				Int     int    `ttlv:"CommonTemplateAttribute,enum"`
				Int8    int8   `ttlv:"CompromiseDate,enum"`
				Int16   int16  `ttlv:"CompromiseOccurrenceDate,enum"`
				Int32   int32  `ttlv:"ContactInformation,enum"`
				Int64   int64  `ttlv:"CorrelationValue,enum"`
				Uint    uint   `ttlv:"CounterLength,enum"`
				Uint8   uint8  `ttlv:"Credential,enum"`
				Uint16  uint16 `ttlv:"CredentialType,enum"`
				Uint32  uint32 `ttlv:"CredentialValue,enum"`
				Uint64  uint64 `ttlv:"CriticalityIndicator,enum"`
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
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagComment, Value: EnumValue(1)},
				Value{Tag: TagCommonTemplateAttribute, Value: EnumValue(2)},
				Value{Tag: TagCompromiseDate, Value: EnumValue(3)},
				Value{Tag: TagCompromiseOccurrenceDate, Value: EnumValue(4)},
				Value{Tag: TagContactInformation, Value: EnumValue(5)},
				Value{Tag: TagCorrelationValue, Value: EnumValue(6)},
				Value{Tag: TagCounterLength, Value: EnumValue(7)},
				Value{Tag: TagCredential, Value: EnumValue(8)},
				Value{Tag: TagCredentialType, Value: EnumValue(9)},
				Value{Tag: TagCredentialValue, Value: EnumValue(10)},
				Value{Tag: TagCriticalityIndicator, Value: EnumValue(11)},
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
			expected: Value{Tag: TagCancellationResult, Value: Values{
				Value{Tag: TagAttribute, Value: Values{
					Value{Tag: TagAttributeName, Value: "color"},
					Value{Tag: TagAttributeValue, Value: "red"},
				}},
				Value{Tag: TagAttribute, Value: Values{
					Value{Tag: TagAttributeName, Value: "size"},
					Value{Tag: TagAttributeValue, Value: int32(5)},
					Value{Tag: TagAttributeIndex, Value: int32(1)},
				}},
				Value{Tag: TagCertificate, Value: Values{
					Value{Tag: TagCertificateIdentifier, Value: "blue"},
					Value{Tag: TagCertificateIssuer, Value: Values{
						Value{Tag: TagCertificateIssuerAlternativeName, Value: "rick"},
						Value{Tag: TagCertificateIssuerC, Value: "bob"},
						Value{Tag: TagCertificateIssuerEmail, Value: EnumValue(0)},
						Value{Tag: TagCertificateIssuerCN, Value: EnumValue(2)},
						Value{Tag: TagCertificateIssuerUID, Value: EnumValue(3)},
						Value{Tag: TagCertificateLength, Value: EnumValue(10)},
					}},
				}},
				Value{Tag: TagBlockCipherMode, Value: int32(5)},
			}},
		},
		{
			name:     "intenum",
			v:        Value{Tag: TagWrappingMethod, Value: int(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "int8enum",
			v:        Value{Tag: TagWrappingMethod, Value: int8(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "int16enum",
			v:        Value{Tag: TagWrappingMethod, Value: int16(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "int32enum",
			v:        Value{Tag: TagWrappingMethod, Value: int32(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "int64enum",
			v:        Value{Tag: TagWrappingMethod, Value: int64(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "uintenum",
			v:        Value{Tag: TagWrappingMethod, Value: uint(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "uint8enum",
			v:        Value{Tag: TagWrappingMethod, Value: uint8(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "uint16enum",
			v:        Value{Tag: TagWrappingMethod, Value: uint16(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "uint32enum",
			v:        Value{Tag: TagWrappingMethod, Value: uint32(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
		{
			name:     "uint64enum",
			v:        Value{Tag: TagWrappingMethod, Value: uint64(WrappingMethodMACSign)},
			expected: TTLV(hex2bytes("42009e | 05 | 00 00 00 04 | 00000002 00000000")),
		},
	}

	// test cases for all the int base types
	for _, v := range []interface{}{int8(5), uint(5), uint8(5), int16(5), uint16(5), int(5), int32(5), uint32(5), byte(5), rune(5)} {
		tests = append(tests, testCase{
			v:        v,
			expected: []interface{}{Value{Tag: TagCancellationResult, Value: int32(5)}},
		})
	}

	// test cases for all long int base types
	for _, v := range []interface{}{int64(5), uint64(5)} {
		tests = append(tests, testCase{
			v:        v,
			expected: []interface{}{Value{Tag: TagCancellationResult, Value: int64(5)}},
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
			expected: Value{
				Tag: TagCancellationResult,
				Value: Values{
					Value{
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
			expected: Value{
				Tag: TagCancellationResult,
				Value: Values{
					Value{
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

func TestTaggedValue_UnmarshalTTLV(t *testing.T) {
	var tv Value

	b := hex2bytes("42000d02000000040000000500000000")

	err := Unmarshal(b, &tv)
	require.NoError(t, err)

	assert.Equal(t, Value{Tag: TagBatchCount, Value: 5}, tv)

	s := Value{Tag: TagAttributeValue, Value: Values{
		Value{Tag: TagNameType, Value: "red"},
		Value{Tag: TagAttributeValue, Value: "blue"},
	}}

	b, err = Marshal(s)
	require.NoError(t, err)

	t.Log(TTLV(b))

	err = Unmarshal(b, &tv)
	require.NoError(t, err)

	assert.Equal(t, s, tv)

}

func TestTaggedValue_MarshalTTLV(t *testing.T) {
	tv := Value{}

	b, err := Marshal(&tv)
	require.NoError(t, err)

	assert.Empty(t, b)

	tv.Value = 5

	_, err = Marshal(&tv)
	require.Error(t, err)

	tv.Tag = TagBatchCount
	b, err = Marshal(&tv)
	require.NoError(t, err)

	ttlv := TTLV(b)

	assert.Equal(t, TagBatchCount, ttlv.Tag())
	assert.Equal(t, TypeInteger, ttlv.Type())
	assert.Equal(t, 5, ttlv.ValueInteger())

	buf := bytes.NewBuffer(nil)
	enc := NewEncoder(buf)
	err = enc.EncodeValue(TagAttributeValue, tv)
	require.NoError(t, err)

	ttlv = TTLV(buf.Bytes())
	assert.Equal(t, TagBatchCount, ttlv.Tag())
	assert.Equal(t, TypeInteger, ttlv.Type())
	assert.Equal(t, 5, ttlv.ValueInteger())

	fmt.Println(hex.EncodeToString(buf.Bytes()))

	buf.Reset()
	tv.Tag = TagNone

	err = enc.EncodeValue(TagAttributeValue, tv)
	require.NoError(t, err)

	ttlv = TTLV(buf.Bytes())
	assert.Equal(t, TagAttributeValue, ttlv.Tag())
	assert.Equal(t, TypeInteger, ttlv.Type())
	assert.Equal(t, 5, ttlv.ValueInteger())

	tv.Value = Values{
		{Tag: TagComment, Value: "red"},
	}

	b, err = Marshal(tv)
	require.NoError(t, err)

	ttlv = TTLV(b)

	assert.Equal(t, TypeStructure, ttlv.Type())

	ttlv2 := ttlv.ValueStructure()
	assert.Equal(t, TypeTextString, ttlv2.Type())
	assert.Equal(t, TagComment, ttlv2.Tag())
	assert.Equal(t, "red", ttlv2.ValueTextString())

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
		_ = enc.encode(TagNone, rv, nil)
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
