// Code generated by "kmipenums "; DO NOT EDIT.

package kmip

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// Attestation Type Enumeration
// 9.1.3.2.36
type AttestationType uint32

const (
	AttestationTypeSAMLAssertion      AttestationType = 0x00000003
	AttestationTypeTCGIntegrityReport AttestationType = 0x00000002
	AttestationTypeTPMQuote           AttestationType = 0x00000001
)

var _AttestationTypeNameToValueMap = map[string]AttestationType{
	"SAMLAssertion":      AttestationTypeSAMLAssertion,
	"TCGIntegrityReport": AttestationTypeTCGIntegrityReport,
	"TPMQuote":           AttestationTypeTPMQuote,
}

var _AttestationTypeValueToNameMap = map[AttestationType]string{
	AttestationTypeSAMLAssertion:      "SAMLAssertion",
	AttestationTypeTCGIntegrityReport: "TCGIntegrityReport",
	AttestationTypeTPMQuote:           "TPMQuote",
}

func (a AttestationType) String() string {
	if s, ok := _AttestationTypeValueToNameMap[a]; ok {
		return s
	}
	return fmt.Sprintf("%#08x", a)
}

func ParseAttestationType(s string) (AttestationType, error) {
	if strings.HasPrefix(s, "0x") && len(s) == 10 {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return 0, err
		}
		return AttestationType(binary.BigEndian.Uint32(b)), nil
	}
	if v, ok := _AttestationTypeNameToValueMap[s]; ok {
		return v, nil
	} else {
		var v AttestationType
		return v, fmt.Errorf("%s is not a valid AttestationType", s)
	}
}

func (a AttestationType) MarshalText() (text []byte, err error) {
	return []byte(a.String()), nil
}

func (a *AttestationType) UnmarshalText(text []byte) (err error) {
	*a, err = ParseAttestationType(string(text))
	return
}

func (a AttestationType) MarshalTTLVEnum() uint32 {
	return uint32(a)
}