package kmip20

import (
	"context"

	"github.com/gemalto/kmip-go"
	"github.com/gemalto/kmip-go/kmip14"
)

// GetRequestPayload ////////////////////////////////////////
//
type GetRequestPayload struct {
	UniqueIdentifier *UniqueIdentifierValue
}

// GetResponsePayload
type GetResponsePayload struct {
	ObjectType       kmip14.ObjectType
	UniqueIdentifier string
	Key              kmip.SymmetricKey
}

type GetHandler struct {
	Get func(ctx context.Context, payload *GetRequestPayload) (*GetResponsePayload, error)
}

func (h *GetHandler) HandleItem(ctx context.Context, req *kmip.Request) (*kmip.ResponseBatchItem, error) {
	var payload GetRequestPayload
	err := req.DecodePayload(&payload)
	if err != nil {
		return nil, err
	}

	respPayload, err := h.Get(ctx, &payload)
	if err != nil {
		return nil, err
	}

	// req.Key = respPayload.Key
	req.IDPlaceholder = respPayload.UniqueIdentifier

	return &kmip.ResponseBatchItem{
		ResponsePayload: respPayload,
	}, nil
}
