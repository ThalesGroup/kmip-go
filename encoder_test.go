package kmip

import (
	"testing"
	"bytes"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
	"fmt"
	"time"
	"math/big"
	"github.com/ansel1/merry"
)

func parseBigInt(s string) *big.Int {
	i := &big.Int{}
	_, ok := i.SetString(s, 10)
	if !ok {
		panic(merry.Errorf("can't parse as big int: %v", s))
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
		v: CredentialTypeAttestation,
		exp: "42 00 01 | 05 | 00 00 00 04 | 00 00 00 03 00 00 00 00",
	},
}

func TestEncoder_EncodeValue(t *testing.T) {
	for _, test := range knownGoodSamples {
		t.Run(fmt.Sprintf("%T:%v", test.v, test.v), func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			e := NewTTLVEncoder(buf)
			err := e.EncodeValue(TagActivationDate, test.v)
			require.NoError(t, err)

			exp := hex2bytes(test.exp)

			assert.Equal(t, len(exp), buf.Len())
			if len(exp) > 0 {
				assert.Equal(t, exp, buf.Bytes())
			}
		})
	}

	t.Run("nil", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		e := NewTTLVEncoder(buf)
		err := e.EncodeValue(TagActivationDate, nil)
		require.NoError(t, err)

		require.Empty(t, buf.Bytes())
	})

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

func TestTTLVEncoder_Encode(t *testing.T) {
	b, err := MarshalTTLV(MarhalableStruct{})
	require.NoError(t, err)
	fmt.Println(TTLV2(b))
}

func TestEncoder_encode(t *testing.T) {

	type testCase struct {
		name     string
		tag      Tag
		v        interface{}
		expected []interface{}
	}

	tests := []testCase{
		// byte strings
		{
			v:        []byte{0x01, 0x02, 0x03},
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: []byte{0x01, 0x02, 0x03}}},
		},
		// text strings
		{
			v: "red",
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: "red"}},
		},
		// date time
		{
			v: parseTime("Friday, March 14, 2008, 11:56:40 UTC"),
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: parseTime("Friday, March 14, 2008, 11:56:40 UTC")}},
		},
		// big int
		{
			v: parseBigInt("1234567890000000000000000000"),
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: parseBigInt("1234567890000000000000000000")}},
		},
		// duration
		{
			v: time.Second * 10,
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: time.Second * 10}},
		},
		// boolean
		{
			v: true,
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: true}},
		},
		// enum
		{
			name:"enum",
			v:CredentialTypeAttestation,
			expected: []interface{}{TaggedValue{Tag: TagCancellationResult, Value: EnumLiteral{IntValue:0x03}}},
		},
	}

	// test cases for all the int base types
	for _, v := range []interface{}{int8(5), uint8(5), int16(5), uint16(5), int(5), int32(5), uint32(5), byte(5), rune(5)} {
		tests = append(tests, testCase{
				v:v,
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

			if tc.tag == 0 {
				tc.tag = TagCancellationResult
			}

			err := enc.encode(tc.tag, tc.v)
			require.NoError(t, err)
			enc.flush()

			require.Equal(t, tc.expected, m.writtenValues)
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