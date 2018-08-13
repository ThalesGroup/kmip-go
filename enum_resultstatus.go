// Code generated by "kmipenums "; DO NOT EDIT.

package kmip

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// Result Status Enumeration
// 9.1.3.2.28
type ResultStatus uint32

const (
	ResultStatusOperationFailed  ResultStatus = 0x00000001
	ResultStatusOperationPending ResultStatus = 0x00000002
	ResultStatusOperationUndone  ResultStatus = 0x00000003
	ResultStatusSuccess          ResultStatus = 0x00000000
)

var _ResultStatusNameToValueMap = map[string]ResultStatus{
	"OperationFailed":  ResultStatusOperationFailed,
	"OperationPending": ResultStatusOperationPending,
	"OperationUndone":  ResultStatusOperationUndone,
	"Success":          ResultStatusSuccess,
}

var _ResultStatusValueToNameMap = map[ResultStatus]string{
	ResultStatusOperationFailed:  "OperationFailed",
	ResultStatusOperationPending: "OperationPending",
	ResultStatusOperationUndone:  "OperationUndone",
	ResultStatusSuccess:          "Success",
}

func (r ResultStatus) String() string {
	if s, ok := _ResultStatusValueToNameMap[r]; ok {
		return s
	}
	return fmt.Sprintf("%#08x", r)
}

func ParseResultStatus(s string) (ResultStatus, error) {
	if strings.HasPrefix(s, "0x") && len(s) == 10 {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return 0, err
		}
		return ResultStatus(binary.BigEndian.Uint32(b)), nil
	}
	if v, ok := _ResultStatusNameToValueMap[s]; ok {
		return v, nil
	} else {
		var v ResultStatus
		return v, fmt.Errorf("%s is not a valid ResultStatus", s)
	}
}

func (r ResultStatus) MarshalText() (text []byte, err error) {
	return []byte(r.String()), nil
}

func (r *ResultStatus) UnmarshalText(text []byte) (err error) {
	*r, err = ParseResultStatus(string(text))
	return
}

func (r ResultStatus) MarshalTTLVEnum() uint32 {
	return uint32(r)
}
