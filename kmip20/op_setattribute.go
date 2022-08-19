package kmip20

import (
	"context"

	"github.com/gemalto/kmip-go"
)

// 6.1.47 Set Attribute

// Table 296

type SetAttributeRequestPayload struct {
	UniqueIdentifier *UniqueIdentifierValue
	AttributeName    string
	AttributeValue   string
}

// Table 297

type SetAttributeResponsePayload struct {
	UniqueIdentifier string
	AttributeName    string
	AttributeValue   string
}

type SetAttributeHandler struct {
	SetAttribute func(ctx context.Context, payload *SetAttributeRequestPayload) (*SetAttributeResponsePayload, error)
}

func (h *SetAttributeHandler) HandleItem(ctx context.Context, req *kmip.Request) (*kmip.ResponseBatchItem, error) {
	var payload SetAttributeRequestPayload
	err := req.DecodePayload(&payload)
	if err != nil {
		return nil, err
	}

	respPayload, err := h.SetAttribute(ctx, &payload)
	if err != nil {
		return nil, err
	}

	return &kmip.ResponseBatchItem{
		ResponsePayload: respPayload,
	}, nil
}
