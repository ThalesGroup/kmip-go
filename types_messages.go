package kmip

import "time"

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
	MaximumResponseSize          int    `kmip:",omitempty"`
	ClientCorrelationValue       string `kmip:",omitempty"`
	ServerCorrelationValue       string `kmip:",omitempty"`
	AsynchronousIndicator        bool   `kmip:",omitempty"`
	AttestationCapableIndicator  bool   `kmip:",omitempty"`
	AttestationType              []AttestationType
	Authentication               *Authentication
	BatchErrorContinuationOption BatchErrorContinuation `kmip:",omitempty"`
	BatchOrderOption             bool                   `kmip:",omitempty"`
	TimeStamp                    *time.Time
	BatchCount                   int
}

type RequestBatchItem struct {
	Operation         Operation
	UniqueBatchItemID []byte `kmip:",omitempty"`
	RequestPayload    interface{}
	MessageExtension  *MessageExtension `kmip:",omitempty"`
}

type ResponseHeader struct {
	ProtocolVersion        ProtocolVersion
	TimeStamp              time.Time
	Nonce                  *Nonce
	AttestationType        []AttestationType
	ClientCorrelationValue string `kmip:",omitempty"`
	ServerCorrelationValue string `kmip:",omitempty"`
	BatchCount             int
}

type ResponseBatchItem struct {
	Operation                    Operation
	UniqueBatchItemID            []byte `kmip:",omitempty"`
	ResultStatus                 ResultStatus
	ResultReason                 ResultReason `kmip:",omitempty"`
	ResultMessage                string       `kmip:",omitempty"`
	AsynchronousCorrelationValue []byte       `kmip:",omitempty"`
	ResponsePayload              interface{}  `kmip:",omitempty"`
	MessageExtension             *MessageExtension
}
