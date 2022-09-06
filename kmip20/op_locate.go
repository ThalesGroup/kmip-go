package kmip20

import (
	"context"

	"github.com/gemalto/kmip-go"
)

// 6.1.27 Locate

// Table 229

type LocateRequestPayload struct {
	Attributes interface{}
}

// Table 230

type LocateResponsePayload struct {
	UniqueIdentifier string
}

type LocateHandler struct {
	Locate func(ctx context.Context, payload *LocateRequestPayload) (*LocateResponsePayload, error)
}

func (h *LocateHandler) HandleItem(ctx context.Context, req *kmip.Request) (*kmip.ResponseBatchItem, error) {
	var payload LocateRequestPayload

	err := req.DecodePayload(&payload)
	if err != nil {
		return nil, err
	}

	respPayload, err := h.Locate(ctx, &payload)
	if err != nil {
		return nil, err
	}

	return &kmip.ResponseBatchItem{
		ResponsePayload: respPayload,
	}, nil
}
