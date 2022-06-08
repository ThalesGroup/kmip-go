package kmip20

import (
	"context"

	"github.com/gemalto/kmip-go"
	"github.com/gemalto/kmip-go/kmip14"
)

// 6.1.40 Revoke

// Table 269

type RevocationReasonStruct struct {
	RevocationReasonCode kmip14.RevocationReasonCode
}

type RevokeRequestPayload struct {
	UniqueIdentifier         UniqueIdentifierValue
	RevocationReason         RevocationReasonStruct
	CompromiseOccurrenceDate []byte
}

// Table 270

type RevokeResponsePayload struct {
	UniqueIdentifier string
}

type RevokeHandler struct {
	Revoke func(ctx context.Context, payload *RevokeRequestPayload) (*RevokeResponsePayload, error)
}

func (h *RevokeHandler) HandleItem(ctx context.Context, req *kmip.Request) (*kmip.ResponseBatchItem, error) {
	var payload RevokeRequestPayload
	err := req.DecodePayload(&payload)
	if err != nil {
		return nil, err
	}

	respPayload, err := h.Revoke(ctx, &payload)
	if err != nil {
		return nil, err
	}

	return &kmip.ResponseBatchItem{
		ResponsePayload: respPayload,
	}, nil
}
