package ttlv_test

import (
	"testing"

	. "github.com/gemalto/kmip-go/kmip14" //nolint:revive
	. "github.com/gemalto/kmip-go/ttlv"   //nolint:revive
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBitMaskString(t *testing.T) {
	tests := []struct {
		in  CryptographicUsageMask
		out string
	}{
		{
			in:  CryptographicUsageMaskSign,
			out: "Sign",
		},
		{
			in:  CryptographicUsageMask(0x00100000),
			out: "0x00100000",
		},
		{
			in:  CryptographicUsageMaskSign | CryptographicUsageMaskExport,
			out: "Sign|Export",
		},
		{
			in:  CryptographicUsageMaskSign | CryptographicUsageMaskExport | CryptographicUsageMask(0x00100000),
			out: "Sign|Export|0x00100000",
		},
		{
			in:  CryptographicUsageMaskSign | CryptographicUsageMaskExport | CryptographicUsageMask(0x00100000) | CryptographicUsageMask(0x00200000),
			out: "Sign|Export|0x00300000",
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.out, func(t *testing.T) {
			assert.Equal(t, testcase.out, testcase.in.String())
		})
	}
}

func TestParseInteger(t *testing.T) {
	tests := []struct {
		out CryptographicUsageMask
		in  string
	}{
		{
			out: CryptographicUsageMaskSign,
			in:  "Sign",
		},
		{
			out: CryptographicUsageMaskDecrypt,
			in:  "0x00000008",
		},
		{
			out: CryptographicUsageMaskDecrypt,
			in:  "8",
		},
		{
			out: CryptographicUsageMask(0x00100000),
			in:  "0x00100000",
		},
		{
			out: CryptographicUsageMask(0x00100000),
			in:  "1048576",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskExport,
			in:  "Sign|Export",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskExport,
			in:  "Sign Export",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskExport,
			in:  "0x00000001 0x00000040",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskExport,
			in:  "0x00000001|0x00000040",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskExport | CryptographicUsageMask(0x00100000),
			in:  "Sign|Export|0x00100000",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskExport | CryptographicUsageMask(0x00100000) | CryptographicUsageMask(0x00200000),
			in:  "Sign|Export|0x00300000",
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.in, func(t *testing.T) {
			mask, e := DefaultRegistry.ParseInt(TagCryptographicUsageMask, testcase.in)
			require.NoError(t, e)
			assert.Equal(t, int32(testcase.out), mask)
		})
	}
}

func TestNormalizeNames(t *testing.T) {
	tests := map[string]string{
		"Structure":                       "Structure",
		"Date-Time":                       "DateTime",
		"Byte String":                     "ByteString",
		"Batch Error Continuation Option": "BatchErrorContinuationOption",
		"CRT Coefficient":                 "CRTCoefficient",
		"J":                               "J",
		"Private Key Template-Attribute":  "PrivateKeyTemplateAttribute",
		"EC Public Key Type X9.62 Compressed Prime": "ECPublicKeyTypeX9_62CompressedPrime",
		"PKCS#8":                                "PKCS_8",
		"Encrypt then MAC/sign":                 "EncryptThenMACSign",
		"P-384":                                 "P_384",
		"MD2 with RSA Encryption (PKCS#1 v1.5)": "MD2WithRSAEncryptionPKCS_1V1_5",
		"Num42bers in first word":               "Num42bersInFirstWord",
		"Polynomial Sharing GF (2^16)":          "PolynomialSharingGF2_16",
		"3DES":                                  "DES3",
	}

	for input, output := range tests {
		t.Run(input, func(t *testing.T) {
			assert.Equal(t, output, NormalizeName(input))
		})
	}
}
