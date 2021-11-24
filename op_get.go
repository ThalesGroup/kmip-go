package kmip

import (
	"context"
	"github.com/gemalto/kmip-go/kmip14"
)

// GetRequestPayload ////////////////////////////////////////
//
type GetRequestPayload struct {
	UniqueIdentifier  string
}

// GetResponsePayload 
type GetResponsePayload struct {
	ObjectType        kmip14.ObjectType
	UniqueIdentifier  string
	Key               string
}

type GetHandler struct {
	Get func(ctx context.Context, payload *GetRequestPayload) (*GetResponsePayload, error)
}

func (h *GetHandler) HandleItem(ctx context.Context, req *Request) (*ResponseBatchItem, error) {
	var payload GetRequestPayload
	err := req.DecodePayload(&payload)
	if err != nil {
		return nil, err
	}

																respPayload, err := h.Get(ctx, &payload)
	if err != nil {
		return nil, err
	}

	//req.Key = respPayload.Key

	return &ResponseBatchItem{
		ResponsePayload: respPayload,
	}, nil
}
