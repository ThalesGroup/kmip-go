package kmip

import (
	"testing"
	"math/big"
	"github.com/stretchr/testify/assert"
	"fmt"
	"encoding/hex"
	"strings"
	"github.com/stretchr/testify/require"
	"time"
	"os"
	"encoding/binary"
	"bytes"
)

var sample= `
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
	r := Reader{r: bytes.NewReader(hex2bytes(sample))}
	ttlv, err := r.Read()
	require.NoError(t, err)
	fmt.Println(ttlv.String())
}

func TestDecoding(t *testing.T) {
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
			bs: "42 00 20 | 03 | 00 00 00 08 | 01 B6 9B 4B A5 74 92 00",
			exp: int64(123456789000000000),
			typ: TypeLongInteger,
		},
		{
			bs: "42 00 20 | 04 | 00 00 00 10 | 00 00 00 00 03 FD 35 EB 6B C2 DF 46 18 08 00 00",
			exp: bi,
			typ: TypeBigInteger,
		},
		{
			bs: "42 00 20 | 05 | 00 00 00 04 | 00 00 00 FF 00 00 00 00",
			exp: uint32(255),
			typ: TypeEnumeration,
		},
		{
			bs: "42 00 20 | 06 | 00 00 00 08 | 00 00 00 00 00 00 00 01",
			exp: true,
			typ: TypeBoolean,
		},
		{
			bs: "42 00 20 | 07 | 00 00 00 0B | 48 65 6C 6C 6F 20 57 6F 72 6C 64 00 00 00 00 00",
			exp: "Hello World",
			typ: TypeTextString,
		},
		{
			bs: "42 00 20 | 08 | 00 00 00 03 | 01 02 03 00 00 00 00 00",
			exp: []byte{0x01, 0x02, 0x03},
			typ: TypeByteString,
		},
		{
			bs: "42 00 20 | 09 | 00 00 00 08 | 00 00 00 00 47 DA 67 F8",
			exp:dt,
			typ: TypeDateTime,
		},
		{
			bs: "42 00 20 | 0A | 00 00 00 04 | 00 0D 2F 00 00 00 00 00",
			exp: 10 * 24 * time.Hour,
			typ: TypeInterval,
		},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {

			b := hex2bytes(tc.bs)
			tt := TTLV(b)
			assert.NoError(t, tt.Valid())
			assert.Equal(t, tc.typ, tt.Type())
			assert.Equal(t, tc.exp, tt.Value())
			fmt.Println(tt.String())
		})
	}

	// structure
	b := hex2bytes("42 00 20 | 01 | 00 00 00 20 | 42 00 04 | 05 | 00 00 00 04 | 00 00 00 FE 00 00 00 00 | 42 00 05 | 02 | 00 00 00 04 | 00 00 00 FF 00 00 00 00")
	tt := TTLV(b)
	assert.NoError(t, tt.Valid())
	assert.Equal(t, TypeStructure, tt.Type())
	exp := hex2bytes("42 00 04 | 05 | 00 00 00 04 | 00 00 00 FE 00 00 00 00 | 42 00 05 | 02 | 00 00 00 04 | 00 00 00 FF 00 00 00 00")
	assert.Equal(t, TTLV(exp), tt.Value())
	print(os.Stdout, "", tt)
	fmt.Println("")
	//fmt.Println(Print(tt))

	for _, test := range knownGoodSamples {
		t.Run(fmt.Sprintf("%T:%v", test.v, test.v), func(t *testing.T) {
			b := hex2bytes(test.exp)
			fmt.Println(b)
			tt := TTLV(b)
			assert.NoError(t, tt.Valid())

			tagBytes := make([]byte, 4)
			fmt.Println("len:", len(tagBytes))
			fmt.Println("cap:", cap(tagBytes))
			copy(tagBytes[1:], b[:3])
			assert.Equal(t, Tag(binary.BigEndian.Uint32(tagBytes)), tt.Tag())

			assert.Equal(t, Type(b[3]), tt.Type())

			assert.Equal(t, int(binary.BigEndian.Uint32(b[4:8])), tt.Len())

			assert.Equal(t, len(b), tt.FullLen())

			assert.Equal(t, test.v, tt.Value())
			fmt.Println(tt.String())
		})
	}
}

// hex2bytes converts hex string to bytes.  Any non-hex characters in the string are stripped first.
// panics on error
func hex2bytes(s string) []byte {
	// strip non hex bytes
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'A' && r <= 'F':
		case r >= 'a' && r <= 'f':
		default:
			return -1 // drop
		}
		return r
	}, s)
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	return b
}

func TestNormalizeNames(t *testing.T) {
	tests := map[string]string {
		"Structure":"Structure",
		"Date-Time":"DateTime",
		"Byte String":"ByteString",
		"Batch Error Continuation Option":"BatchErrorContinuationOption",
		"CRT Coefficient":"CRTCoefficient",
		"J":"J",
		"Private Key Template-Attribute":"PrivateKeyTemplateAttribute",
		"EC Public Key Type X9.62 Compressed Prime":"ECPublicKeyTypeX9_62CompressedPrime",
		"PKCS#8":"PKCS_8",
		"Encrypt then MAC/sign":"EncryptThenMACSign",
		"P-384":"P_384",
		"MD2 with RSA Encryption (PKCS#1 v1.5)":"MD2WithRSAEncryptionPKCS_1V1_5",
		"Num42bers in first word":"Num42bersInFirstWord",
		"Polynomial Sharing GF (2^16)":"PolynomialSharingGF2_16",
		"3DES":"DES3",
	}

	for input, output := range tests {
		t.Run(input, func(t *testing.T) {
			assert.Equal(t, output, NormalizeName(input))
		})
	}
}



