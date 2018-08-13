// Code generated by "kmipenums "; DO NOT EDIT.

package kmip

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// Batch Error Continuation Enumeration
// 9.1.3.2.30
type BatchErrorContinuation uint32

const (
	BatchErrorContinuationContinue BatchErrorContinuation = 0x00000001
	BatchErrorContinuationStop     BatchErrorContinuation = 0x00000002
	BatchErrorContinuationUndo     BatchErrorContinuation = 0x00000003
)

var _BatchErrorContinuationNameToValueMap = map[string]BatchErrorContinuation{
	"Continue": BatchErrorContinuationContinue,
	"Stop":     BatchErrorContinuationStop,
	"Undo":     BatchErrorContinuationUndo,
}

var _BatchErrorContinuationValueToNameMap = map[BatchErrorContinuation]string{
	BatchErrorContinuationContinue: "Continue",
	BatchErrorContinuationStop:     "Stop",
	BatchErrorContinuationUndo:     "Undo",
}

func (b BatchErrorContinuation) String() string {
	if s, ok := _BatchErrorContinuationValueToNameMap[b]; ok {
		return s
	}
	return fmt.Sprintf("%#08x", b)
}

func ParseBatchErrorContinuation(s string) (BatchErrorContinuation, error) {
	if strings.HasPrefix(s, "0x") && len(s) == 10 {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return 0, err
		}
		return BatchErrorContinuation(binary.BigEndian.Uint32(b)), nil
	}
	if v, ok := _BatchErrorContinuationNameToValueMap[s]; ok {
		return v, nil
	} else {
		var v BatchErrorContinuation
		return v, fmt.Errorf("%s is not a valid BatchErrorContinuation", s)
	}
}

func (b BatchErrorContinuation) MarshalText() (text []byte, err error) {
	return []byte(b.String()), nil
}

func (b *BatchErrorContinuation) UnmarshalText(text []byte) (err error) {
	*b, err = ParseBatchErrorContinuation(string(text))
	return
}

func (b BatchErrorContinuation) MarshalTTLVEnum() uint32 {
	return uint32(b)
}
