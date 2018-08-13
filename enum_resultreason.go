// Code generated by "kmipenums "; DO NOT EDIT.

package kmip

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// Result Reason Enumeration
// 9.1.3.2.29
type ResultReason uint32

const (
	ResultReasonApplicationNamespaceNotSupported ResultReason = 0x0000000f
	ResultReasonAttestationFailed                ResultReason = 0x00000015
	ResultReasonAttestationRequired              ResultReason = 0x00000014
	ResultReasonAuthenticationNotSuccessful      ResultReason = 0x00000003
	ResultReasonCryptographicFailure             ResultReason = 0x0000000a
	ResultReasonEncodingOptionError              ResultReason = 0x00000012
	ResultReasonFeatureNotSupported              ResultReason = 0x00000008
	ResultReasonGeneralFailure                   ResultReason = 0x00000100
	ResultReasonIllegalOperation                 ResultReason = 0x0000000b
	ResultReasonIndexOutOfBounds                 ResultReason = 0x0000000e
	ResultReasonInvalidField                     ResultReason = 0x00000007
	ResultReasonInvalidMessage                   ResultReason = 0x00000004
	ResultReasonItemNotFound                     ResultReason = 0x00000001
	ResultReasonKeyCompressionTypeNotSupported   ResultReason = 0x00000011
	ResultReasonKeyFormatTypeNotSupported        ResultReason = 0x00000010
	ResultReasonKeyValueNotPresent               ResultReason = 0x00000013
	ResultReasonMissingData                      ResultReason = 0x00000006
	ResultReasonNotExtractable                   ResultReason = 0x00000017
	ResultReasonObjectAlreadyExists              ResultReason = 0x00000018
	ResultReasonObjectArchived                   ResultReason = 0x0000000d
	ResultReasonOperationCanceledByRequester     ResultReason = 0x00000009
	ResultReasonOperationNotSupported            ResultReason = 0x00000005
	ResultReasonPermissionDenied                 ResultReason = 0x0000000c
	ResultReasonResponseTooLarge                 ResultReason = 0x00000002
	ResultReasonSensitive                        ResultReason = 0x00000016
)

var _ResultReasonNameToValueMap = map[string]ResultReason{
	"ApplicationNamespaceNotSupported": ResultReasonApplicationNamespaceNotSupported,
	"AttestationFailed":                ResultReasonAttestationFailed,
	"AttestationRequired":              ResultReasonAttestationRequired,
	"AuthenticationNotSuccessful":      ResultReasonAuthenticationNotSuccessful,
	"CryptographicFailure":             ResultReasonCryptographicFailure,
	"EncodingOptionError":              ResultReasonEncodingOptionError,
	"FeatureNotSupported":              ResultReasonFeatureNotSupported,
	"GeneralFailure":                   ResultReasonGeneralFailure,
	"IllegalOperation":                 ResultReasonIllegalOperation,
	"IndexOutOfBounds":                 ResultReasonIndexOutOfBounds,
	"InvalidField":                     ResultReasonInvalidField,
	"InvalidMessage":                   ResultReasonInvalidMessage,
	"ItemNotFound":                     ResultReasonItemNotFound,
	"KeyCompressionTypeNotSupported":   ResultReasonKeyCompressionTypeNotSupported,
	"KeyFormatTypeNotSupported":        ResultReasonKeyFormatTypeNotSupported,
	"KeyValueNotPresent":               ResultReasonKeyValueNotPresent,
	"MissingData":                      ResultReasonMissingData,
	"NotExtractable":                   ResultReasonNotExtractable,
	"ObjectAlreadyExists":              ResultReasonObjectAlreadyExists,
	"ObjectArchived":                   ResultReasonObjectArchived,
	"OperationCanceledByRequester":     ResultReasonOperationCanceledByRequester,
	"OperationNotSupported":            ResultReasonOperationNotSupported,
	"PermissionDenied":                 ResultReasonPermissionDenied,
	"ResponseTooLarge":                 ResultReasonResponseTooLarge,
	"Sensitive":                        ResultReasonSensitive,
}

var _ResultReasonValueToNameMap = map[ResultReason]string{
	ResultReasonApplicationNamespaceNotSupported: "ApplicationNamespaceNotSupported",
	ResultReasonAttestationFailed:                "AttestationFailed",
	ResultReasonAttestationRequired:              "AttestationRequired",
	ResultReasonAuthenticationNotSuccessful:      "AuthenticationNotSuccessful",
	ResultReasonCryptographicFailure:             "CryptographicFailure",
	ResultReasonEncodingOptionError:              "EncodingOptionError",
	ResultReasonFeatureNotSupported:              "FeatureNotSupported",
	ResultReasonGeneralFailure:                   "GeneralFailure",
	ResultReasonIllegalOperation:                 "IllegalOperation",
	ResultReasonIndexOutOfBounds:                 "IndexOutOfBounds",
	ResultReasonInvalidField:                     "InvalidField",
	ResultReasonInvalidMessage:                   "InvalidMessage",
	ResultReasonItemNotFound:                     "ItemNotFound",
	ResultReasonKeyCompressionTypeNotSupported:   "KeyCompressionTypeNotSupported",
	ResultReasonKeyFormatTypeNotSupported:        "KeyFormatTypeNotSupported",
	ResultReasonKeyValueNotPresent:               "KeyValueNotPresent",
	ResultReasonMissingData:                      "MissingData",
	ResultReasonNotExtractable:                   "NotExtractable",
	ResultReasonObjectAlreadyExists:              "ObjectAlreadyExists",
	ResultReasonObjectArchived:                   "ObjectArchived",
	ResultReasonOperationCanceledByRequester:     "OperationCanceledByRequester",
	ResultReasonOperationNotSupported:            "OperationNotSupported",
	ResultReasonPermissionDenied:                 "PermissionDenied",
	ResultReasonResponseTooLarge:                 "ResponseTooLarge",
	ResultReasonSensitive:                        "Sensitive",
}

func (r ResultReason) String() string {
	if s, ok := _ResultReasonValueToNameMap[r]; ok {
		return s
	}
	return fmt.Sprintf("%#08x", r)
}

func ParseResultReason(s string) (ResultReason, error) {
	if strings.HasPrefix(s, "0x") && len(s) == 10 {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return 0, err
		}
		return ResultReason(binary.BigEndian.Uint32(b)), nil
	}
	if v, ok := _ResultReasonNameToValueMap[s]; ok {
		return v, nil
	} else {
		var v ResultReason
		return v, fmt.Errorf("%s is not a valid ResultReason", s)
	}
}

func (r ResultReason) MarshalText() (text []byte, err error) {
	return []byte(r.String()), nil
}

func (r *ResultReason) UnmarshalText(text []byte) (err error) {
	*r, err = ParseResultReason(string(text))
	return
}

func (r ResultReason) MarshalTTLVEnum() uint32 {
	return uint32(r)
}
