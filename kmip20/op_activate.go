//nolint:dupl
package kmip20

import (
	"context"

	"github.com/gemalto/kmip-go"
)

// 4.19 Activate

// Table 210

type ActivateRequestPayload struct {
	UniqueIdentifier *UniqueIdentifierValue
}

// Table 211

type ActivateResponsePayload struct {
	UniqueIdentifier string
}

type ActivateHandler struct {
	Activate func(ctx context.Context, payload *ActivateRequestPayload) (*ActivateResponsePayload, error)
}

func (h *ActivateHandler) HandleItem(ctx context.Context, req *kmip.Request) (*kmip.ResponseBatchItem, error) {
	var payload ActivateRequestPayload

	err := req.DecodePayload(&payload)
	if err != nil {
		return nil, err
	}

	respPayload, err := h.Activate(ctx, &payload)
	if err != nil {
		return nil, err
	}

	return &kmip.ResponseBatchItem{
		ResponsePayload: respPayload,
	}, nil
}
