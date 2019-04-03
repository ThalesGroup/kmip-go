package kmip

import (
	"github.com/gemalto/kmip-go/ttlv"
	"time"
)

// 7.1

type RequestMessage struct {
	RequestHeader RequestHeader
	BatchItem     []RequestBatchItem
}

type ResponseMessage struct {
	ResponseHeader ResponseHeader
	BatchItem      []ResponseBatchItem
}

// 7.2

type RequestHeader struct {
	ProtocolVersion              ProtocolVersion
	MaximumResponseSize          int    `ttlv:",omitempty"`
	ClientCorrelationValue       string `ttlv:",omitempty"`
	ServerCorrelationValue       string `ttlv:",omitempty"`
	AsynchronousIndicator        bool   `ttlv:",omitempty"`
	AttestationCapableIndicator  bool   `ttlv:",omitempty"`
	AttestationType              []ttlv.AttestationType
	Authentication               *Authentication
	BatchErrorContinuationOption ttlv.BatchErrorContinuationOption `ttlv:",omitempty"`
	BatchOrderOption             bool                              `ttlv:",omitempty"`
	TimeStamp                    *time.Time
	BatchCount                   int
}

type RequestBatchItem struct {
	Operation         ttlv.Operation
	UniqueBatchItemID []byte `ttlv:",omitempty"`
	RequestPayload    interface{}
	MessageExtension  *MessageExtension `ttlv:",omitempty"`
}

type ResponseHeader struct {
	ProtocolVersion        ProtocolVersion
	TimeStamp              time.Time
	Nonce                  *Nonce
	AttestationType        []ttlv.AttestationType
	ClientCorrelationValue string `ttlv:",omitempty"`
	ServerCorrelationValue string `ttlv:",omitempty"`
	BatchCount             int
}

type ResponseBatchItem struct {
	Operation                    ttlv.Operation `ttlv:",omitempty"`
	UniqueBatchItemID            []byte         `ttlv:",omitempty"`
	ResultStatus                 ttlv.ResultStatus
	ResultReason                 ttlv.ResultReason `ttlv:",omitempty"`
	ResultMessage                string            `ttlv:",omitempty"`
	AsynchronousCorrelationValue []byte            `ttlv:",omitempty"`
	ResponsePayload              interface{}       `ttlv:",omitempty"`
	MessageExtension             *MessageExtension
}
