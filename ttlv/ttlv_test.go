package ttlv

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"math/big"
	"strconv"
	"testing"
	"time"
)

var sample = `
420078 | 01 | 00000118 
	420077 | 01 | 00000048 
		420069 | 01 | 00000020 
			42006A | 02 | 00000004 | 0000000100000000
			42006B | 02 | 00000004 | 0000000000000000
		420010 | 06 | 00000008 | 0000000000000001
		42000D | 02 | 00000004 | 0000000200000000
	42000F | 01 | 00000068
		42005C | 05 | 00000004 | 0000000800000000
		420093 | 08 | 00000001 | 3600000000000000
		4200790100000040420008010000003842000A07000000044E616D650000000042000B010000002042005507000000067075626B657900004200540500000004000000010000000042000F010000005042005C05000000040000000E00000000420093080000000137000000000000004200790100000028420008010000002042000A0700000008782D6D796174747242000B07000000057465737432000000`

func TestPrint(t *testing.T) {
	b := Hex2bytes("420069010000002042006a0200000004000000010000000042006b02000000040000000000000000")
	buf := &bytes.Buffer{}
	err := Print(buf, "", "  ", b)
	require.NoError(t, err)
	assert.Equal(t, `ProtocolVersion (Structure/32):
  ProtocolVersionMajor (Integer/4): 1
  ProtocolVersionMinor (Integer/4): 0`, buf.String())

	// Should tolerate invalid ttlv value
	b = Hex2bytes("620069010000002042006a0200000004000000010000000042006b02000000040000000000000000")
	buf.Reset()
	err = Print(buf, "", "  ", b)
	assert.Error(t, err)
	assert.Equal(t, `0x620069 (Structure/32): (invalid tag) 0x42006a0200000004000000010000000042006b02000000040000000000000000`, buf.String())

	// Should tolerate invalid value with valid header
	b = Hex2bytes("42006b0200000004000000000000")
	buf.Reset()
	err = Print(buf, "", "  ", b)
	assert.Error(t, err)
	assert.Equal(t, `ProtocolVersionMinor (Integer/4): (value truncated) 0x00000000`, buf.String())
}

func TestPrintPrettyHex(t *testing.T) {
	b := Hex2bytes("420069010000002042006a0200000004000000010000000042006b02000000040000000000000000")
	buf := &bytes.Buffer{}
	err := PrintPrettyHex(buf, "", "  ", b)
	require.NoError(t, err)
	assert.Equal(t, `420069 | 01 | 00000020
  42006a | 02 | 00000004 | 0000000100000000
  42006b | 02 | 00000004 | 0000000000000000`, buf.String())

	// Should tolerate invalid ttlv value
	b = Hex2bytes("620069010000002042006a0200000004000000010000000042006b02000000040000000000000000")
	buf.Reset()
	err = PrintPrettyHex(buf, "", "  ", b)
	require.NoError(t, err)
	assert.Equal(t, `620069010000002042006a0200000004000000010000000042006b02000000040000000000000000`, buf.String())

	// Should tolerate invalid value with valid header
	b = Hex2bytes("42006b0200000004000000000000")
	buf.Reset()
	err = PrintPrettyHex(buf, "", "  ", b)
	require.NoError(t, err)
	assert.Equal(t, `42006b | 02 | 00000004
000000000000`, buf.String())
}

func TestTTLV(t *testing.T) {
	bi := &big.Int{}
	bi, ok := bi.SetString("1234567890000000000000000000", 10)
	require.True(t, ok)

	dt, err := time.Parse("Monday, January 2, 2006, 15:04:05 MST", "Friday, March 14, 2008, 11:56:40 UTC")
	require.NoError(t, err)

	tests := []struct {
		bs  string
		b   []byte
		exp interface{}
		typ Type
	}{
		{
			bs:  "42 00 20 | 02 | 00 00 00 04 | 00 00 00 08 00 00 00 00",
			exp: int32(8),
			typ: TypeInteger,
		},
		{
			bs:  "42 00 20 | 03 | 00 00 00 08 | 01 B6 9B 4B A5 74 92 00",
			exp: int64(123456789000000000),
			typ: TypeLongInteger,
		},
		{
			bs:  "42 00 20 | 04 | 00 00 00 10 | 00 00 00 00 03 FD 35 EB 6B C2 DF 46 18 08 00 00",
			exp: bi,
			typ: TypeBigInteger,
		},
		{
			bs:  "42 00 20 | 05 | 00 00 00 04 | 00 00 00 FF 00 00 00 00",
			exp: uint32(255),
			typ: TypeEnumeration,
		},
		{
			bs:  "42 00 20 | 06 | 00 00 00 08 | 00 00 00 00 00 00 00 01",
			exp: true,
			typ: TypeBoolean,
		},
		{
			bs:  "42 00 20 | 07 | 00 00 00 0B | 48 65 6C 6C 6F 20 57 6F 72 6C 64 00 00 00 00 00",
			exp: "Hello World",
			typ: TypeTextString,
		},
		{
			bs:  "42 00 20 | 08 | 00 00 00 03 | 01 02 03 00 00 00 00 00",
			exp: []byte{0x01, 0x02, 0x03},
			typ: TypeByteString,
		},
		{
			bs:  "42 00 20 | 09 | 00 00 00 08 | 00 00 00 00 47 DA 67 F8",
			exp: dt,
			typ: TypeDateTime,
		},
		{
			bs:  "42 00 20 | 0A | 00 00 00 04 | 00 0D 2F 00 00 00 00 00",
			exp: 10 * 24 * time.Hour,
			typ: TypeInterval,
		},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {

			b := Hex2bytes(tc.bs)
			tt := TTLV(b)
			assert.NoError(t, tt.Valid())
			assert.Equal(t, tc.typ, tt.Type())
			assert.Equal(t, tc.exp, tt.Value())
		})
	}

	// structure
	b := Hex2bytes("42 00 20 | 01 | 00 00 00 20 | 42 00 04 | 05 | 00 00 00 04 | 00 00 00 FE 00 00 00 00 | 42 00 05 | 02 | 00 00 00 04 | 00 00 00 FF 00 00 00 00")
	tt := TTLV(b)
	assert.NoError(t, tt.Valid())
	assert.Equal(t, TypeStructure, tt.Type())
	exp := Hex2bytes("42 00 04 | 05 | 00 00 00 04 | 00 00 00 FE 00 00 00 00 | 42 00 05 | 02 | 00 00 00 04 | 00 00 00 FF 00 00 00 00")
	assert.Equal(t, TTLV(exp), tt.Value())

	for _, test := range knownGoodSamples {
		name := test.name
		if name == "" {
			name = fmt.Sprintf("%T:%v", test.v, test.v)
		}
		t.Run(name, func(t *testing.T) {
			b := Hex2bytes(test.exp)
			tt := TTLV(b)
			require.NoError(t, tt.Valid())

			tagBytes := make([]byte, 4)
			copy(tagBytes[1:], b[:3])
			assert.Equal(t, Tag(binary.BigEndian.Uint32(tagBytes)), tt.Tag())

			assert.Equal(t, Type(b[3]), tt.Type())

			assert.Equal(t, int(binary.BigEndian.Uint32(b[4:8])), tt.Len())

			assert.Equal(t, len(b), tt.FullLen())

			// allow permitting type conversions, not exact equality
			// also handle special case of non-pointer big.Ints, which
			// will be decoded as *big.Int.

			switch v := test.v.(type) {
			case big.Int:
				if assert.IsType(t, &v, tt.Value()) {
					assert.True(t, tt.Value().(*big.Int).Cmp(&v) == 0)
				}
			case *big.Int:
				if assert.IsType(t, v, tt.Value()) {
					assert.True(t, tt.Value().(*big.Int).Cmp(v) == 0)
				}
			case TTLV:
				assert.Equal(t, v, tt)
			default:
				assert.EqualValues(t, test.v, tt.Value())
			}

		})
	}
}

func TestTTLV_UnmarshalTTLV(t *testing.T) {
	var ttlv TTLV

	require.Nil(t, ttlv)

	buf := bytes.NewBuffer(nil)
	enc := NewEncoder(buf)
	require.NoError(t, enc.EncodeValue(TagComment, "red"))

	err := ttlv.UnmarshalTTLV(nil, TTLV(buf.Bytes()))
	require.NoError(t, err)

	require.NotNil(t, ttlv)
	require.Equal(t, TTLV(buf.Bytes()), ttlv)

	// if ttlv is already allocated and is long enough, allocate
	// into the existing byte slice, rather than allocating a new one
	// (avoid unnecessary allocation for performance)

	ttlv = make(TTLV, buf.Len()+100) // create a TTLV buf a bit larger than necessary
	// copy some marker bytes into the end.  after unmarshaling, the marker bytes should
	// be intact, since they are in the end part of the buffer
	copy(ttlv[buf.Len():], []byte("whitewhale"))
	err = ttlv.UnmarshalTTLV(nil, TTLV(buf.Bytes()))

	require.NoError(t, err)
	require.Equal(t, TTLV(buf.Bytes()), ttlv)
	require.Equal(t, buf.Len()+100, cap(ttlv))
	require.Len(t, ttlv, buf.Len())
	require.EqualValues(t, []byte("whitewhale"), ttlv[buf.Len():buf.Len()+10])

	// if ttlv is not nil, but is not long enough to hold TTLV value,
	// everything still works

	ttlv = make(TTLV, buf.Len()-2)
	err = ttlv.UnmarshalTTLV(nil, TTLV(buf.Bytes()))

	require.NoError(t, err)
	require.Equal(t, TTLV(buf.Bytes()), ttlv)

}

func TestTTLV_UnmarshalJSON_errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		msg   string
	}{
		{
			name:  "invalidtag",
			input: `{"tag":"NotATag","type":"Boolean","value":2}`,
			msg:   "invalid tag: unregistered enum name: NotATag",
		},
		{
			name:  "invalidtype",
			input: `{"tag":"BatchCount","type":"NotAType","value":2}`,
			msg:   "invalid type: unregistered enum name: NotAType",
		},
		{
			name:  "boolinvalidtype",
			input: `{"tag":"BatchCount","type":"Boolean","value":2}`,
			msg:   "BatchCount: invalid Boolean: must be boolean or hex string",
		},
		{
			name:  "boolinvalidhex",
			input: `{"tag":"BatchCount","type":"Boolean","value":"0x0000000000000003"}`,
			msg:   "BatchCount: invalid Boolean: hex string for Boolean value must be either 0x0000000000000001 (true) or 0x0000000000000000 (false)",
		},
		{
			name:  "stringinvalidtype",
			input: `{"tag":"BatchCount","type":"TextString","value":29}`,
			msg:   "BatchCount: invalid TextString: must be string",
		},
		{
			name:  "bytesinvalidtype",
			input: `{"tag":"BatchCount","type":"ByteString","value":29}`,
			msg:   "BatchCount: invalid ByteString: must be hex string",
		},
		{
			name:  "bytesinvalidhex",
			input: `{"tag":"BatchCount","type":"ByteString","value":"0T"}`,
			msg:   "BatchCount: invalid ByteString: encoding/hex: invalid byte: U+0054 'T'",
		},
		{
			name:  "bytesinvalidprefix",
			input: `{"tag":"BatchCount","type":"ByteString","value":"0xFF5601"}`,
			msg:   "BatchCount: invalid ByteString: should not have 0x prefix",
		},
		{
			name:  "intervalinvalidtype",
			input: `{"tag":"BatchCount","type":"Interval","value":true}`,
			msg:   "BatchCount: invalid Interval: must be number or hex string",
		},
		{
			name:  "intervalinvalidhexstring",
			input: `{"tag":"BatchCount","type":"Interval","value":"0000000A"}`,
			msg:   "BatchCount: invalid Interval: hex value must start with 0x",
		},
		{
			name:  "intervalinvalidhex",
			input: `{"tag":"BatchCount","type":"Interval","value":"0x0000000T"}`,
			msg:   "BatchCount: invalid Interval: invalid hex string: encoding/hex: invalid byte: U+0054 'T'",
		},
		{
			name:  "intervalLen",
			input: `{"tag":"BatchCount","type":"Interval","value":"0xA0A0A0A0A0"}`,
			msg:   "BatchCount: invalid Interval: invalid hex string: must be 4 bytes",
		},
		{
			name:  "datetimeinvalidtype",
			input: `{"tag":"BatchCount","type":"DateTime","value":true}`,
			msg:   "BatchCount: invalid DateTime: must be string",
		},
		{
			name:  "datetimeinvalidhex",
			input: `{"tag":"BatchCount","type":"DateTime","value":"0x0H"}`,
			msg:   "BatchCount: invalid DateTime: invalid hex string: encoding/hex: invalid byte: U+0048 'H'",
		},
		{
			name:  "datetimeinvalidlen",
			input: `{"tag":"BatchCount","type":"DateTime","value":"0xA0A0A0A0A0A0A0A0A0"}`,
			msg:   "BatchCount: invalid DateTime: invalid hex string: must be 8 bytes",
		},
		{
			name:  "datetimeinvalidstring",
			input: `{"tag":"BatchCount","type":"DateTime","value":"notadate"}`,
			msg:   "BatchCount: invalid DateTime: must be ISO8601 format: parsing time \"notadate\" as \"2006-01-02T15:04:05.999999999Z07:00\": cannot parse \"notadate\" as \"2006\"",
		},
		{
			name:  "integerinvalidtype",
			input: `{"tag":"BatchCount","type":"Integer","value":true}`,
			msg:   "BatchCount: invalid Integer: must be number, hex string, or mask value name",
		},
		{
			name:  "integerinvalidvalue",
			input: `{"tag":"BatchCount","type":"Integer","value":"0000000A"}`,
			msg:   "BatchCount: invalid Integer: unregistered enum name: 0000000A",
		},
		{
			name:  "integerinvalidhex",
			input: `{"tag":"BatchCount","type":"Integer","value":"0x0000000T"}`,
			msg:   "BatchCount: invalid Integer: invalid hex string: encoding/hex: invalid byte: U+0054 'T'",
		},
		{
			name:  "integerinvalidlen",
			input: `{"tag":"BatchCount","type":"Integer","value":"0xA0A0A0A0A0"}`,
			msg:   "BatchCount: invalid Integer: invalid hex string: must be 4 bytes",
		},
		{
			name:  "longintegerinvalidtype",
			input: `{"tag":"BatchCount","type":"LongInteger","value":true}`,
			msg:   "BatchCount: invalid LongInteger: must be number or hex string",
		},
		{
			name:  "longintegerinvalidhexstring",
			input: `{"tag":"BatchCount","type":"LongInteger","value":"000000000000000A"}`,
			msg:   "BatchCount: invalid LongInteger: hex value must start with 0x",
		},
		{
			name:  "longintegerinvalidhex",
			input: `{"tag":"BatchCount","type":"LongInteger","value":"0x000000000000000T"}`,
			msg:   "BatchCount: invalid LongInteger: invalid hex string: encoding/hex: invalid byte: U+0054 'T'",
		},
		{
			name:  "longintegerinvalidlen",
			input: `{"tag":"BatchCount","type":"LongInteger","value":"0xA0A0A0A0A0A0A0A0A0"}`,
			msg:   "BatchCount: invalid LongInteger: invalid hex string: must be 8 bytes",
		},
		{
			name:  "bigintegerinvalidtype",
			input: `{"tag":"BatchCount","type":"BigInteger","value":true}`,
			msg:   "BatchCount: invalid BigInteger: must be number or hex string",
		},
		{
			name:  "bigintegerinvalidhexstring",
			input: `{"tag":"BatchCount","type":"BigInteger","value":"000000000000000A"}`,
			msg:   "BatchCount: invalid BigInteger: hex value must start with 0x",
		},
		{
			name:  "bigintegerinvalidhex",
			input: `{"tag":"BatchCount","type":"BigInteger","value":"0x000000000000000T"}`,
			msg:   "BatchCount: invalid BigInteger: encoding/hex: invalid byte: U+0054 'T'",
		},
		{
			name:  "bigintegerinvalidlen",
			input: `{"tag":"BatchCount","type":"BigInteger","value":"0x000000000F"}`,
			msg:   "BatchCount: invalid BigInteger: must be multiple of 8 bytes (16 hex characters)",
		},
		{
			name:  "enuminvalidtype",
			input: `{"tag":"ObjectType","type":"Enumeration","value":true}`,
			msg:   "ObjectType: invalid Enumeration: must be number or string",
		},
		{
			name:  "enuminvalidhex",
			input: `{"tag":"ObjectType","type":"Enumeration","value":"0x0000000T"}`,
			msg:   "ObjectType: invalid Enumeration: invalid hex string: encoding/hex: invalid byte: U+0054 'T'",
		},
		{
			name:  "enuminvalidlen",
			input: `{"tag":"ObjectType","type":"Enumeration","value":"0xA0A0A0A0A0"}`,
			msg:   "ObjectType: invalid Enumeration: invalid hex string: must be 4 bytes",
		},
		{
			name:  "enuminvalidname",
			input: `{"tag":"ObjectType","type":"Enumeration","value":"NotAValue"}`,
			msg:   "ObjectType: invalid Enumeration: unregistered enum name: NotAValue",
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			err := json.Unmarshal([]byte(testcase.input), &TTLV{})
			require.EqualError(t, err, testcase.msg)
		})

	}
}

func TestTTLV_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		inputs []string
		exp    interface{}
	}{
		{
			name: "booltrue",
			inputs: []string{
				`{"tag":"BatchCount","type":"Boolean","value":true}`,
				`{"tag":"BatchCount","type":"0x06","value":true}`,
				`{"tag":"0x42000d","type":"Boolean","value":true}`,
				`{"tag":"BatchCount","type":"Boolean","value":"0x0000000000000001"}`,
			},
			exp: Value{Tag: TagBatchCount, Value: true},
		},
		{
			name: "boolfalse",
			inputs: []string{
				`{"tag":"BatchCount","type":"Boolean","value":false}`,
				`{"tag":"BatchCount","type":"Boolean","value":"0x0000000000000000"}`,
			},
			exp: Value{Tag: TagBatchCount, Value: false},
		},
		{
			name: "string",
			inputs: []string{
				`{"tag":"BatchCount","type":"TextString","value":"red"}`,
			},
			exp: Value{Tag: TagBatchCount, Value: "red"},
		},
		{
			name: "stringempty",
			inputs: []string{
				`{"tag":"BatchCount","type":"TextString","value":""}`,
			},
			exp: Value{Tag: TagBatchCount, Value: ""},
		},
		{
			name: "bytes",
			inputs: []string{
				`{"tag":"BatchCount","type":"ByteString","value":"FF5601"}`,
			},
			exp: Value{Tag: TagBatchCount, Value: []byte{0xFF, 0x56, 0x01}},
		},
		{
			name: "bytesempty",
			inputs: []string{
				`{"tag":"BatchCount","type":"ByteString","value":""}`,
			},
			exp: Value{Tag: TagBatchCount, Value: []byte{}},
		},
		{
			name: "interval",
			inputs: []string{
				`{"tag":"BatchCount","type":"Interval","value":10}`,
				`{"tag":"BatchCount","type":"Interval","value":"0x0000000A"}`,
			},
			exp: Value{Tag: TagBatchCount, Value: 10 * time.Second},
		},
		{
			name: "datetime",
			inputs: []string{
				`{"tag":"BatchCount","type":"DateTime","value":"2001-01-01T10:00:00+10:00"}`,
			},
			exp: Value{Tag: TagBatchCount, Value: time.Date(2001, 01, 01, 0, 0, 0, 0, time.FixedZone("UTC", 0))},
		},
		{
			name: "datetimehex",
			inputs: []string{
				`{"tag":"BatchCount","type":"DateTime","value":"0x0000000047DA67F8"}`,
			},
			exp: Value{Tag: TagBatchCount, Value: time.Date(2008, 03, 14, 11, 56, 40, 0, time.FixedZone("UTC", 0))},
		},
		{
			name: "integer",
			inputs: []string{
				`{"tag":"BatchCount","type":"Integer","value":"0x00000005"}`,
				`{"tag":"BatchCount","type":"Integer","value":5}`,
			},
			exp: Value{Tag: TagBatchCount, Value: 5},
		},
		{
			name: "integermask",
			inputs: []string{
				//`{"tag":"CryptographicUsageMask","type":"Integer","value":"0x00000005"}`,
				`{"tag":"CryptographicUsageMask","type":"Integer","value":"Decrypt|Export"}`,
				`{"tag":"CryptographicUsageMask","type":"Integer","value":"Decrypt|0x00000040"}`,
				`{"tag":"CryptographicUsageMask","type":"Integer","value":"0x00000048"}`,
			},
			exp: Value{Tag: TagCryptographicUsageMask, Value: CryptographicUsageMaskDecrypt | CryptographicUsageMaskExport},
		},
		{
			name: "longinteger",
			inputs: []string{
				`{"tag":"BatchCount","type":"LongInteger","value":"0x0000000000000005"}`,
				`{"tag":"BatchCount","type":"LongInteger","value":5}`,
			},
			exp: Value{Tag: TagBatchCount, Value: int64(5)},
		},
		{
			name: "biginteger",
			inputs: []string{
				`{"tag":"BatchCount","type":"BigInteger","value":"0x0000000000000005"}`,
				`{"tag":"BatchCount","type":"BigInteger","value":"0x00000000000000000000000000000005"}`,
				`{"tag":"BatchCount","type":"BigInteger","value":5}`,
			},
			exp: Value{Tag: TagBatchCount, Value: big.NewInt(5)},
		},
		{
			name: "enumeration",
			inputs: []string{
				`{"tag":"ObjectType","type":"Enumeration","value":2}`,
				`{"tag":"ObjectType","type":"Enumeration","value":"0x00000002"}`,
				`{"tag":"ObjectType","type":"Enumeration","value":"SymmetricKey"}`,
			},
			exp: Value{Tag: TagObjectType, Value: ObjectTypeSymmetricKey},
		},
		{
			name: "structure",
			inputs: []string{
				`{
					"tag":"BatchCount",
					"value":[
						{"tag":"CryptographicUsageMask", "type":"Integer", "value":"Decrypt|Encrypt"},
						{"tag":"CryptographicAlgorithm", "type":"Enumeration", "value":"Blowfish"},
						{"tag":"ObjectType", "type":"Structure", "value":[
							{"tag":"Operation", "type":"TextString", "value":"red"}
						]}
					]
				}`,
			},
			exp: Value{Tag: TagBatchCount, Value: Values{
				Value{Tag: TagCryptographicUsageMask, Value: CryptographicUsageMaskDecrypt | CryptographicUsageMaskEncrypt},
				Value{Tag: TagCryptographicAlgorithm, Value: CryptographicAlgorithmBlowfish},
				Value{Tag: TagObjectType, Value: Values{
					Value{Tag: TagOperation, Value: "red"},
				}},
			}},
		},
		{
			name: "attributes",
			exp: Value{Tag: TagAttribute, Value: Values{
				Value{Tag: TagAttributeName, Value: "Key Format Type"},
				Value{Tag: TagAttributeValue, Value: KeyFormatTypeX_509},
			}},
			inputs: []string{`{"tag":"Attribute","value":[
				{"tag":"AttributeName","type":"TextString","value":"Key Format Type"},
				{"tag":"AttributeValue","type":"Enumeration","value":"X_509"}
			]}`},
		},
		{
			name: "attributesmask",
			exp: Value{Tag: TagAttribute, Value: Values{
				Value{Tag: TagAttributeName, Value: "Cryptographic Usage Mask"},
				Value{Tag: TagAttributeValue, Value: CryptographicUsageMaskEncrypt},
			}},
			inputs: []string{`{"tag":"Attribute","value":[
				{"tag":"AttributeName","type":"TextString","value":"Cryptographic Usage Mask"},
				{"tag":"AttributeValue","type":"Integer","value":"Encrypt"}
			]}`},
		},
	}
	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			expTTLV, err := Marshal(testcase.exp)
			require.NoError(t, err)

			for _, input := range testcase.inputs {
				t.Log(input)
				var ttlv TTLV
				err = json.Unmarshal([]byte(input), &ttlv)
				require.NoError(t, err)

				assert.Equal(t, TTLV(expTTLV), ttlv)
			}

		})
	}
}

func TestTTLV_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		exp  string
	}{
		{
			in:  Value{Tag: TagBatchCount, Value: 10},
			exp: `{"tag":"BatchCount","type":"Integer","value":10}`,
		},
		{
			in:  Value{Tag: Tag(0x540002), Value: 10},
			exp: `{"tag":"0x540002","type":"Integer","value":10}`,
		},
		{
			in:  Value{Tag: TagBatchCount, Value: `"Red Rover"`},
			exp: `{"tag":"BatchCount","type":"TextString","value":"\"Red Rover\""}`,
		},
		{
			in:  Value{Tag: TagBatchCount, Value: true},
			exp: `{"tag":"BatchCount","type":"Boolean","value":true}`,
		},
		{
			in:  Value{Tag: TagBatchCount, Value: false},
			exp: `{"tag":"BatchCount","type":"Boolean","value":false}`,
		},
		{
			in:  Value{Tag: TagBatchCount, Value: math.MaxInt32},
			exp: `{"tag":"BatchCount","type":"Integer","value":` + strconv.Itoa(math.MaxInt32) + `}`,
		},
		{
			in:  Value{Tag: TagBatchCount, Value: int64(math.MaxInt32) + 1},
			exp: `{"tag":"BatchCount","type":"LongInteger","value":` + strconv.FormatInt(int64(math.MaxInt32)+1, 10) + `}`,
		},
		{
			// test values higher than max json number, should be encoded in hex
			in: Value{Tag: TagBatchCount, Value: int64(1) << 53},
			exp: func() string {
				ttlv, err := Marshal(Value{Tag: TagBatchCount, Value: int64(1) << 53})
				require.NoError(t, err)
				return `{"tag":"BatchCount","type":"LongInteger","value":"0x` + hex.EncodeToString(TTLV(ttlv).ValueRaw()) + `"}`
			}(),
		},
		{
			in:  Value{Tag: TagBatchCount, Value: big.NewInt(10)},
			exp: `{"tag":"BatchCount","type":"BigInteger","value":10}`,
		},
		{
			// test values higher than max json number, should be encoded in hex
			in: Value{Tag: TagBatchCount, Value: big.NewInt(int64(1) << 53)},
			exp: func() string {
				ttlv, err := Marshal(Value{Tag: TagBatchCount, Value: big.NewInt(int64(1) << 53)})
				require.NoError(t, err)
				return `{"tag":"BatchCount","type":"BigInteger","value":"0x` + hex.EncodeToString(TTLV(ttlv).ValueRaw()) + `"}`
			}(),
		},
		{
			in:  Value{Tag: TagBatchCount, Value: WrappingMethodMACSign},
			exp: `{"tag":"BatchCount","type":"Enumeration","value":"0x00000002"}`,
		},
		{
			in:  Value{Tag: TagKeyFormatType, Value: KeyFormatTypeX_509},
			exp: `{"tag":"KeyFormatType","type":"Enumeration","value":"X_509"}`,
		},
		{
			in:  Value{Tag: TagKeyFormatType, Value: EnumValue(0x00050000)},
			exp: `{"tag":"KeyFormatType","type":"Enumeration","value":"0x00050000"}`,
		},
		{
			in: Value{Tag: TagBatchCount, Value: func() time.Time {
				d, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05+04:00")
				require.NoError(t, err)
				return d
			}()},
			exp: `{"tag":"BatchCount","type":"DateTime","value":"2006-01-02T11:04:05Z"}`,
		},
		{
			in:  Value{Tag: TagKeyFormatType, Value: 10 * time.Second},
			exp: `{"tag":"KeyFormatType","type":"Interval","value":10}`,
		},
		{
			in: Value{Tag: TagKeyFormatType, Value: Values{
				Value{Tag: TagBatchCount, Value: 10},
				Value{Tag: Tag(0x540002), Value: 10},
				Value{Tag: TagBatchItem, Value: true},
			}},
			exp: `{"tag":"KeyFormatType","value":[
				{"tag":"BatchCount","type":"Integer","value":10},
				{"tag":"0x540002","type":"Integer","value":10},
				{"tag":"BatchItem","type":"Boolean","value":true}
			]}`,
		},
		{
			in: Value{Tag: TagAttribute, Value: Values{
				Value{Tag: TagAttributeName, Value: "Key Format Type"},
				Value{Tag: TagAttributeValue, Value: KeyFormatTypeX_509},
			}},
			exp: `{"tag":"Attribute","value":[
				{"tag":"AttributeName","type":"TextString","value":"Key Format Type"},
				{"tag":"AttributeValue","type":"Enumeration","value":"X_509"}
			]}`,
		},
		{
			in: Value{Tag: TagAttribute, Value: Values{
				Value{Tag: TagAttributeName, Value: "Key Format Type"},
				Value{Tag: TagAttributeValue, Value: "X_509"},
			}},
			exp: `{"tag":"Attribute","value":[
				{"tag":"AttributeName","type":"TextString","value":"Key Format Type"},
				{"tag":"AttributeValue","type":"TextString","value":"X_509"}
			]}`,
		},
		{
			name: "attributesmask",
			in: Value{Tag: TagAttribute, Value: Values{
				Value{Tag: TagAttributeName, Value: "Cryptographic Usage Mask"},
				Value{Tag: TagAttributeValue, Value: CryptographicUsageMaskExport},
			}},
			exp: `{"tag":"Attribute","value":[
				{"tag":"AttributeName","type":"TextString","value":"Cryptographic Usage Mask"},
				{"tag":"AttributeValue","type":"Integer","value":"Export"}
			]}`,
		},
		{
			in: Value{Tag: TagAttribute, Value: Values{
				Value{Tag: TagAttributeName, Value: "Key Format Type"},
				Value{Tag: TagAttributeValue, Value: EnumValue(0x00000300)},
			}},
			exp: `{"tag":"Attribute","value":[
				{"tag":"AttributeName","type":"TextString","value":"Key Format Type"},
				{"tag":"AttributeValue","type":"Enumeration","value":"0x00000300"}
			]}`,
		},
		{
			in:  Value{Tag: TagCryptographicUsageMask, Value: CryptographicUsageMaskCRLSign},
			exp: `{"tag":"CryptographicUsageMask","type":"Integer","value":"CRLSign"}`,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			b, err := Marshal(testcase.in)
			require.NoError(t, err)
			ttlv := TTLV(b)
			j, err := json.Marshal(ttlv)
			require.NoError(t, err)
			require.JSONEq(t, testcase.exp, string(j))
		})
	}
}

func TestTTLV_MarshalXML(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		exp  string
	}{
		{
			name: "integer",
			in:   Value{Tag: TagBatchCount, Value: 10},
			exp:  `<BatchCount type="Integer" value="10"></BatchCount>`,
		},
		{
			name: "unknowntag",
			in:   Value{Tag: 0x54FFFF, Value: 10},
			exp:  `<TTLV tag="0x54ffff" type="Integer" value="10"></TTLV>`,
		},
		{
			name: "booltrue",
			in:   Value{Tag: TagBatchCount, Value: true},
			exp:  `<BatchCount type="Boolean" value="true"></BatchCount>`,
		},
		{
			name: "boolfalse",
			in:   Value{Tag: TagBatchCount, Value: false},
			exp:  `<BatchCount type="Boolean" value="false"></BatchCount>`,
		},
		{
			name: "longinteger",
			in:   Value{Tag: TagBatchCount, Value: int64(6)},
			exp:  `<BatchCount type="LongInteger" value="6"></BatchCount>`,
		},
		{
			name: "biginteger",
			in:   Value{Tag: TagBatchCount, Value: big.NewInt(6)},
			exp:  `<BatchCount type="BigInteger" value="0000000000000006"></BatchCount>`,
		},
		{
			name: "bitmask",
			in:   Value{Tag: TagCryptographicUsageMask, Value: CryptographicUsageMaskExport | CryptographicUsageMaskSign},
			exp:  `<CryptographicUsageMask type="Integer" value="Sign Export"></CryptographicUsageMask>`,
		},
		{
			name: "enumeration",
			in:   Value{Tag: TagOperation, Value: OperationActivate},
			exp:  `<Operation type="Enumeration" value="Activate"></Operation>`,
		},
		{
			name: "enumerationext",
			in:   Value{Tag: TagOperation, Value: 0x0000002c},
			exp:  `<Operation type="Enumeration" value="0x0000002c"></Operation>`,
		},
		{
			name: "textstring",
			in:   Value{Tag: TagBatchCount, Value: "red"},
			exp:  `<BatchCount type="TextString" value="red"></BatchCount>`,
		},
		{
			name: "textstringempty",
			in:   Value{Tag: TagBatchCount, Value: ""},
			exp:  `<BatchCount type="TextString"></BatchCount>`,
		},
		{
			name: "bytestring",
			in:   Value{Tag: TagBatchCount, Value: []byte{0x01, 0x02, 0x03}},
			exp:  `<BatchCount type="ByteString" value="010203"></BatchCount>`,
		},
		{
			name: "datetime",
			in:   Value{Tag: TagBatchCount, Value: time.Date(2001, 01, 01, 0, 0, 0, 0, time.FixedZone("UTC", 0))},
			exp:  `<BatchCount type="DateTime" value="2001-01-01T00:00:00Z"></BatchCount>`,
		},
		{
			name: "interval",
			in:   Value{Tag: TagBatchCount, Value: 10 * time.Second},
			exp:  `<BatchCount type="Interval" value="10"></BatchCount>`,
		},
		{
			name: "structure",
			in: Value{Tag: TagKeyFormatType, Value: Values{
				Value{Tag: TagBatchCount, Value: 10},
				Value{Tag: Tag(0x540002), Value: 10},
				Value{Tag: TagBatchItem, Value: true},
			}},
			exp: `<KeyFormatType><BatchCount type="Integer" value="10"></BatchCount><TTLV tag="0x540002" type="Integer" value="10"></TTLV><BatchItem type="Boolean" value="true"></BatchItem></KeyFormatType>`,
		},
		{
			name: "attributes",
			in: Value{Tag: TagAttribute, Value: Values{
				Value{Tag: TagAttributeName, Value: "Key Format Type"},
				Value{Tag: TagAttributeValue, Value: KeyFormatTypeX_509},
			}},
			exp: `<Attribute><AttributeName type="TextString" value="Key Format Type"></AttributeName><AttributeValue type="Enumeration" value="X_509"></AttributeValue></Attribute>`,
		},
		{
			name: "attributesmask",
			in: Value{Tag: TagAttribute, Value: Values{
				Value{Tag: TagAttributeName, Value: "Cryptographic Usage Mask"},
				Value{Tag: TagAttributeValue, Value: CryptographicUsageMaskExport},
			}},
			exp: `<Attribute><AttributeName type="TextString" value="Cryptographic Usage Mask"></AttributeName><AttributeValue type="Integer" value="Export"></AttributeValue></Attribute>`,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			b, err := Marshal(testcase.in)
			require.NoError(t, err)
			ttlv := TTLV(b)
			j, err := xml.Marshal(ttlv)
			require.NoError(t, err)
			require.Equal(t, testcase.exp, string(j))
		})
	}
}

func TestTTLV_UnmarshalXML(t *testing.T) {
	tests := []struct {
		name   string
		inputs []string
		exp    interface{}
	}{
		{
			name: "booltrue",
			inputs: []string{
				`<BatchCount type="Boolean" value="true"/>`,
				`<BatchCount type="Boolean" value="1"/>`,
				`<BatchCount tag="BatchCount" type="Boolean" value="true"/>`,
				`<TTLV tag="BatchCount" type="Boolean" value="true"/>`,
				`<BatchCount type="0x06" value="true"/>`,
				`<TTLV tag="0x42000d" type="Boolean" value="true"/>`,
			},
			exp: Value{Tag: TagBatchCount, Value: true},
		},
		{
			name: "boolfalse",
			inputs: []string{
				`<BatchCount type="Boolean" value="false"/>`,
				`<BatchCount type="Boolean" value="0"/>`,
			},
			exp: Value{Tag: TagBatchCount, Value: false},
		},
		{
			name: "string",
			inputs: []string{
				`<BatchCount type="TextString" value="red"/>`,
			},
			exp: Value{Tag: TagBatchCount, Value: "red"},
		},
		{
			name: "stringempty",
			inputs: []string{
				`<BatchCount type="TextString"/>`,
			},
			exp: Value{Tag: TagBatchCount, Value: ""},
		},
		{
			name: "bytes",
			inputs: []string{
				`<BatchCount type="ByteString" value="FF5601"/>`,
			},
			exp: Value{Tag: TagBatchCount, Value: []byte{0xFF, 0x56, 0x01}},
		},
		{
			name: "bytesempty",
			inputs: []string{
				`<BatchCount type="ByteString" value=""/>`,
			},
			exp: Value{Tag: TagBatchCount, Value: []byte{}},
		},
		{
			name: "interval",
			inputs: []string{
				`<BatchCount type="Interval" value="10"/>`,
			},
			exp: Value{Tag: TagBatchCount, Value: 10 * time.Second},
		},
		{
			name: "datetime",
			inputs: []string{
				`<BatchCount type="DateTime" value="2001-01-01T10:00:00+10:00"/>`,
			},
			exp: Value{Tag: TagBatchCount, Value: time.Date(2001, 01, 01, 0, 0, 0, 0, time.FixedZone("UTC", 0))},
		},
		{
			name: "integer",
			inputs: []string{
				`<BatchCount type="Integer" value="5"/>`,
			},
			exp: Value{Tag: TagBatchCount, Value: 5},
		},
		{
			name: "integermask",
			inputs: []string{
				//`{"tag":"CryptographicUsageMask","type":"Integer","value":"0x00000005"}`,
				`<CryptographicUsageMask type="Integer" value="0x00000048"/>`,
				`<CryptographicUsageMask type="Integer" value="72"/>`,
				`<CryptographicUsageMask type="Integer" value="Decrypt Export"/>`,
				`<CryptographicUsageMask type="Integer" value="Decrypt 0x00000040"/>`,
			},
			exp: Value{Tag: TagCryptographicUsageMask, Value: CryptographicUsageMaskDecrypt | CryptographicUsageMaskExport},
		},
		{
			name: "longinteger",
			inputs: []string{
				`<BatchCount type="LongInteger" value="5"/>`,
			},
			exp: Value{Tag: TagBatchCount, Value: int64(5)},
		},
		{
			name: "biginteger",
			inputs: []string{
				`<BatchCount type="BigInteger" value="00000000000000000000000000000005"/>`,
			},
			exp: Value{Tag: TagBatchCount, Value: big.NewInt(5)},
		},
		{
			name: "enumeration",
			inputs: []string{
				`<ObjectType type="Enumeration" value="0x00000002"/>`,
				`<ObjectType type="Enumeration" value="SymmetricKey"/>`,
			},
			exp: Value{Tag: TagObjectType, Value: ObjectTypeSymmetricKey},
		},
		{
			name: "structure",
			inputs: []string{
				`<BatchCount>
						<CryptographicUsageMask type="Integer" value="Decrypt|Encrypt"/>
						<CryptographicAlgorithm type="Enumeration" value="Blowfish"/>
						<ObjectType>
							<Operation type="TextString" value="red"/>
						</ObjectType>
				</BatchCount>`,
			},
			exp: Value{Tag: TagBatchCount, Value: Values{
				Value{Tag: TagCryptographicUsageMask, Value: CryptographicUsageMaskDecrypt | CryptographicUsageMaskEncrypt},
				Value{Tag: TagCryptographicAlgorithm, Value: CryptographicAlgorithmBlowfish},
				Value{Tag: TagObjectType, Value: Values{
					Value{Tag: TagOperation, Value: "red"},
				}},
			}},
		},
		{
			name: "attributes",
			exp: Value{Tag: TagAttribute, Value: Values{
				Value{Tag: TagAttributeName, Value: "Key Format Type"},
				Value{Tag: TagAttributeValue, Value: KeyFormatTypeX_509},
			}},
			inputs: []string{`<Attribute>
				<AttributeName type="TextString" value="Key Format Type"/>
				<AttributeValue type="Enumeration" value="X_509"/>
			</Attribute>`},
		},
		{
			name: "attributesmask",
			exp: Value{Tag: TagAttribute, Value: Values{
				Value{Tag: TagAttributeName, Value: "Cryptographic Usage Mask"},
				Value{Tag: TagAttributeValue, Value: CryptographicUsageMaskEncrypt},
			}},
			inputs: []string{`<Attribute>
				<AttributeName type="TextString" value="Cryptographic Usage Mask"/>
				<AttributeValue type="Integer" value="Encrypt"/>
			</Attribute>`},
		},
	}
	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			expTTLV, err := Marshal(testcase.exp)
			require.NoError(t, err)

			for _, input := range testcase.inputs {
				t.Log(input)
				var ttlv TTLV
				err = xml.Unmarshal([]byte(input), &ttlv)
				require.NoError(t, err)

				assert.Equal(t, TTLV(expTTLV), ttlv)
			}

		})
	}
}

func TestTTLV_UnmarshalXML_errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		msg   string
	}{
		{
			name:  "invalidtag",
			input: `<Elephant type="Boolean" value="true"/>`,
			msg:   "invalid tag: unregistered enum name: Elephant",
		},
		{
			name:  "invalidtype",
			input: `<BatchCount type="Car" value="true"/>`,
			msg:   "invalid type: unregistered enum name: Car",
		},
		{
			name:  "boolinvalid",
			input: `<BatchCount type="Boolean" value="frank"/>`,
			msg:   "BatchCount: invalid Boolean: must be 0, 1, true, or false: strconv.ParseBool: parsing \"frank\": invalid syntax",
		},
		{
			name:  "bytesinvalidhex",
			input: `<BatchCount type="ByteString" value="0T"/>`,
			msg:   "BatchCount: invalid ByteString: encoding/hex: invalid byte: U+0054 'T'",
		},
		{
			name:  "bytesinvalidprefix",
			input: `<BatchCount type="ByteString" value="0xFF5601"/>`,
			msg:   "BatchCount: invalid ByteString: should not have 0x prefix",
		},
		{
			name:  "intervalinvalid",
			input: `<BatchCount type="Interval" value="red"/>`,
			msg:   "BatchCount: invalid Interval: must be a number: strconv.ParseUint: parsing \"red\": invalid syntax",
		},
		{
			name:  "datetimeinvalid",
			input: `<BatchCount type="DateTime" value="notadate"/>`,
			msg:   "BatchCount: invalid DateTime: must be ISO8601 format: parsing time \"notadate\" as \"2006-01-02T15:04:05.999999999Z07:00\": cannot parse \"notadate\" as \"2006\"",
		},
		{
			name:  "integerinvalidvalue",
			input: `<BatchCount type="Integer" value="red"/>`,
			msg:   "BatchCount: invalid Integer: unregistered enum name: red",
		},
		{
			name:  "integerinvalidhex",
			input: `<BatchCount type="Integer" value="0x0000000T"/>`,
			msg:   "BatchCount: invalid Integer: invalid hex string: encoding/hex: invalid byte: U+0054 'T'",
		},
		{
			name:  "integerinvalidlen",
			input: `<BatchCount type="Integer" value="0xFFFFFFFFFF"/>`,
			msg:   "BatchCount: invalid Integer: invalid hex string: must be 4 bytes",
		},
		{
			name:  "longintegerinvalid",
			input: `<BatchCount type="LongInteger" value="red"/>`,
			msg:   "BatchCount: invalid LongInteger: must be number: strconv.ParseInt: parsing \"red\": invalid syntax",
		},
		{
			name:  "bigintegerinvalid",
			input: `<BatchCount type="BigInteger" value="red"/>`,
			msg:   "BatchCount: invalid BigInteger: encoding/hex: invalid byte: U+0072 'r'",
		},
		{
			name:  "bigintegerinvalidlen",
			input: `<BatchCount type="BigInteger" value="000000000F"/>`,
			msg:   "BatchCount: invalid BigInteger: must be multiple of 8 bytes",
		},
		{
			name:  "bigintegerinvalidprefix",
			input: `<BatchCount type="BigInteger" value="0x0000000F"/>`,
			msg:   "BatchCount: invalid BigInteger: should not have 0x prefix",
		},
		{
			name:  "enuminvalidhex",
			input: `<ObjectType type="Enumeration" value="0x0000000T"/>`,
			msg:   "ObjectType: invalid Enumeration: invalid hex string: encoding/hex: invalid byte: U+0054 'T'",
		},
		{
			name:  "enuminvalidlen",
			input: `<ObjectType type="Enumeration" value="0x2222222222"/>`,
			msg:   "ObjectType: invalid Enumeration: invalid hex string: must be 4 bytes",
		},
		{
			name:  "enuminvalidname",
			input: `<ObjectType type="Enumeration" value="NotAValue"/>`,
			msg:   "ObjectType: invalid Enumeration: unregistered enum name: NotAValue",
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			err := xml.Unmarshal([]byte(testcase.input), &TTLV{})
			require.EqualError(t, err, testcase.msg)
		})

	}
}
