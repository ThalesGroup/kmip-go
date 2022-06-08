package kmip20

import (
	"context"

	"github.com/gemalto/kmip-go"
)

// 6.1.15 Destroy

// Table 193

type DestroyRequestPayload struct {
	UniqueIdentifier UniqueIdentifierValue
}

// Table 194

type DestroyResponsePayload struct {
	UniqueIdentifier string
}

type DestroyHandler struct {
	Destroy func(ctx context.Context, payload *DestroyRequestPayload) (*DestroyResponsePayload, error)
}

func (h *DestroyHandler) HandleItem(ctx context.Context, req *kmip.Request) (*kmip.ResponseBatchItem, error) {
	var payload DestroyRequestPayload

	err := req.DecodePayload(&payload)
	if err != nil {
		return nil, err
	}

	respPayload, err := h.Destroy(ctx, &payload)
	if err != nil {
		return nil, err
	}

	// req.Key = respPayload.Key

	return &kmip.ResponseBatchItem{
		ResponsePayload: respPayload,
	}, nil
}
