package kmip

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNormalizeNames(t *testing.T) {
	tests := map[string]string{
		"Structure":                       "Structure",
		"Date-Time":                       "DateTime",
		"Byte String":                     "ByteString",
		"Batch Error Continuation Option": "BatchErrorContinuationOption",
		"CRT Coefficient":                 "CRTCoefficient",
		"J":                               "J",
		"Private Key Template-Attribute":            "PrivateKeyTemplateAttribute",
		"EC Public Key Type X9.62 Compressed Prime": "ECPublicKeyTypeX9_62CompressedPrime",
		"PKCS#8":                "PKCS_8",
		"Encrypt then MAC/sign": "EncryptThenMACSign",
		"P-384":                 "P_384",
		"MD2 with RSA Encryption (PKCS#1 v1.5)": "MD2WithRSAEncryptionPKCS_1V1_5",
		"Num42bers in first word":               "Num42bersInFirstWord",
		"Polynomial Sharing GF (2^16)":          "PolynomialSharingGF2_16",
		"3DES": "DES3",
	}

	for input, output := range tests {
		t.Run(input, func(t *testing.T) {
			assert.Equal(t, output, NormalizeName(input))
		})
	}
}
