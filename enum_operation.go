// Code generated by "kmipenums "; DO NOT EDIT.

package kmip

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// Operation Enumeration
// 9.1.3.2.27
type Operation uint32

const (
	OperationActivate           Operation = 0x00000012
	OperationAddAttribute       Operation = 0x0000000d
	OperationArchive            Operation = 0x00000015
	OperationCancel             Operation = 0x00000019
	OperationCertify            Operation = 0x00000006
	OperationCheck              Operation = 0x00000009
	OperationCreate             Operation = 0x00000001
	OperationCreateKeyPair      Operation = 0x00000002
	OperationCreateSplitKey     Operation = 0x00000028
	OperationDecrypt            Operation = 0x00000020
	OperationDeleteAttribute    Operation = 0x0000000f
	OperationDeriveKey          Operation = 0x00000005
	OperationDestroy            Operation = 0x00000014
	OperationDiscoverVersions   Operation = 0x0000001e
	OperationEncrypt            Operation = 0x0000001f
	OperationExport             Operation = 0x0000002b
	OperationGet                Operation = 0x0000000a
	OperationGetAttributeList   Operation = 0x0000000c
	OperationGetAttributes      Operation = 0x0000000b
	OperationGetUsageAllocation Operation = 0x00000011
	OperationHash               Operation = 0x00000027
	OperationImport             Operation = 0x0000002a
	OperationJoinSplitKey       Operation = 0x00000029
	OperationLocate             Operation = 0x00000008
	OperationMAC                Operation = 0x00000023
	OperationMACVerify          Operation = 0x00000024
	OperationModifyAttribute    Operation = 0x0000000e
	OperationNotify             Operation = 0x0000001b
	OperationObtainLease        Operation = 0x00000010
	OperationPoll               Operation = 0x0000001a
	OperationPut                Operation = 0x0000001c
	OperationQuery              Operation = 0x00000018
	OperationRNGRetrieve        Operation = 0x00000025
	OperationRNGSeed            Operation = 0x00000026
	OperationReCertify          Operation = 0x00000007
	OperationReKey              Operation = 0x00000004
	OperationReKeyKeyPair       Operation = 0x0000001d
	OperationRecover            Operation = 0x00000016
	OperationRegister           Operation = 0x00000003
	OperationRevoke             Operation = 0x00000013
	OperationSign               Operation = 0x00000021
	OperationSignatureVerify    Operation = 0x00000022
	OperationValidate           Operation = 0x00000017
)

var _OperationNameToValueMap = map[string]Operation{
	"Activate":           OperationActivate,
	"AddAttribute":       OperationAddAttribute,
	"Archive":            OperationArchive,
	"Cancel":             OperationCancel,
	"Certify":            OperationCertify,
	"Check":              OperationCheck,
	"Create":             OperationCreate,
	"CreateKeyPair":      OperationCreateKeyPair,
	"CreateSplitKey":     OperationCreateSplitKey,
	"Decrypt":            OperationDecrypt,
	"DeleteAttribute":    OperationDeleteAttribute,
	"DeriveKey":          OperationDeriveKey,
	"Destroy":            OperationDestroy,
	"DiscoverVersions":   OperationDiscoverVersions,
	"Encrypt":            OperationEncrypt,
	"Export":             OperationExport,
	"Get":                OperationGet,
	"GetAttributeList":   OperationGetAttributeList,
	"GetAttributes":      OperationGetAttributes,
	"GetUsageAllocation": OperationGetUsageAllocation,
	"Hash":               OperationHash,
	"Import":             OperationImport,
	"JoinSplitKey":       OperationJoinSplitKey,
	"Locate":             OperationLocate,
	"MAC":                OperationMAC,
	"MACVerify":          OperationMACVerify,
	"ModifyAttribute":    OperationModifyAttribute,
	"Notify":             OperationNotify,
	"ObtainLease":        OperationObtainLease,
	"Poll":               OperationPoll,
	"Put":                OperationPut,
	"Query":              OperationQuery,
	"RNGRetrieve":        OperationRNGRetrieve,
	"RNGSeed":            OperationRNGSeed,
	"ReCertify":          OperationReCertify,
	"ReKey":              OperationReKey,
	"ReKeyKeyPair":       OperationReKeyKeyPair,
	"Recover":            OperationRecover,
	"Register":           OperationRegister,
	"Revoke":             OperationRevoke,
	"Sign":               OperationSign,
	"SignatureVerify":    OperationSignatureVerify,
	"Validate":           OperationValidate,
}

var _OperationValueToNameMap = map[Operation]string{
	OperationActivate:           "Activate",
	OperationAddAttribute:       "AddAttribute",
	OperationArchive:            "Archive",
	OperationCancel:             "Cancel",
	OperationCertify:            "Certify",
	OperationCheck:              "Check",
	OperationCreate:             "Create",
	OperationCreateKeyPair:      "CreateKeyPair",
	OperationCreateSplitKey:     "CreateSplitKey",
	OperationDecrypt:            "Decrypt",
	OperationDeleteAttribute:    "DeleteAttribute",
	OperationDeriveKey:          "DeriveKey",
	OperationDestroy:            "Destroy",
	OperationDiscoverVersions:   "DiscoverVersions",
	OperationEncrypt:            "Encrypt",
	OperationExport:             "Export",
	OperationGet:                "Get",
	OperationGetAttributeList:   "GetAttributeList",
	OperationGetAttributes:      "GetAttributes",
	OperationGetUsageAllocation: "GetUsageAllocation",
	OperationHash:               "Hash",
	OperationImport:             "Import",
	OperationJoinSplitKey:       "JoinSplitKey",
	OperationLocate:             "Locate",
	OperationMAC:                "MAC",
	OperationMACVerify:          "MACVerify",
	OperationModifyAttribute:    "ModifyAttribute",
	OperationNotify:             "Notify",
	OperationObtainLease:        "ObtainLease",
	OperationPoll:               "Poll",
	OperationPut:                "Put",
	OperationQuery:              "Query",
	OperationRNGRetrieve:        "RNGRetrieve",
	OperationRNGSeed:            "RNGSeed",
	OperationReCertify:          "ReCertify",
	OperationReKey:              "ReKey",
	OperationReKeyKeyPair:       "ReKeyKeyPair",
	OperationRecover:            "Recover",
	OperationRegister:           "Register",
	OperationRevoke:             "Revoke",
	OperationSign:               "Sign",
	OperationSignatureVerify:    "SignatureVerify",
	OperationValidate:           "Validate",
}

func (o Operation) String() string {
	if s, ok := _OperationValueToNameMap[o]; ok {
		return s
	}
	return fmt.Sprintf("%#08x", o)
}

func ParseOperation(s string) (Operation, error) {
	if strings.HasPrefix(s, "0x") && len(s) == 10 {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return 0, err
		}
		return Operation(binary.BigEndian.Uint32(b)), nil
	}
	if v, ok := _OperationNameToValueMap[s]; ok {
		return v, nil
	} else {
		var v Operation
		return v, fmt.Errorf("%s is not a valid Operation", s)
	}
}

func (o Operation) MarshalText() (text []byte, err error) {
	return []byte(o.String()), nil
}

func (o *Operation) UnmarshalText(text []byte) (err error) {
	*o, err = ParseOperation(string(text))
	return
}

func (o Operation) MarshalTTLVEnum() uint32 {
	return uint32(o)
}
