package kmip

import (
	"context"
	"github.com/ansel1/merry"
	"github.com/gemalto/kmip-go/ttlv"
)

// 4.3

// Table 169

type RegisterRequestPayload struct {
	ObjectType        ttlv.ObjectType
	TemplateAttribute TemplateAttribute
	Certificate       *Certificate
	SymmetricKey      *SymmetricKey
	PrivateKey        *PrivateKey
	PublicKey         *PublicKey
	SplitKey          *SplitKey
	Template          *Template
	SecretData        *SecretData
	OpaqueObject      *OpaqueObject
}

// Table 170

type RegisterResponsePayload struct {
	UniqueIdentifier  string
	TemplateAttribute TemplateAttribute
}

type RegisterHandler struct {
	SkipValidation bool
	RegisterFunc   func(context.Context, *RegisterRequestPayload) (*RegisterResponsePayload, error)
}

func (h *RegisterHandler) HandleItem(ctx context.Context, req *Request) (item *ResponseBatchItem, err error) {
	var payload RegisterRequestPayload
	err = req.DecodePayload(&payload)
	if err != nil {
		return nil, merry.Prepend(err, "decoding request")
	}

	if !h.SkipValidation {
		var payloadPresent bool
		switch payload.ObjectType {
		default:
			return nil, WithResultReason(merry.UserError("Object Type is not recognized"), ttlv.ResultReasonInvalidField)
		case ttlv.ObjectTypeCertificate:
			payloadPresent = payload.Certificate != nil
		case ttlv.ObjectTypeSymmetricKey:
			payloadPresent = payload.SymmetricKey != nil
		case ttlv.ObjectTypePrivateKey:
			payloadPresent = payload.PrivateKey != nil
		case ttlv.ObjectTypePublicKey:
			payloadPresent = payload.PublicKey != nil
		case ttlv.ObjectTypeSplitKey:
			payloadPresent = payload.SplitKey != nil
		case ttlv.ObjectTypeTemplate:
			payloadPresent = payload.Template != nil
		case ttlv.ObjectTypeSecretData:
			payloadPresent = payload.SecretData != nil
		case ttlv.ObjectTypeOpaqueObject:
			payloadPresent = payload.OpaqueObject != nil
		}
		if !payloadPresent {
			return nil, WithResultReason(merry.UserErrorf("Object Type %s does not match type of cryptographic object provided", payload.ObjectType.String()), ttlv.ResultReasonInvalidField)
		}
	}

	respPayload, err := h.RegisterFunc(ctx, &payload)
	if err != nil {
		return nil, err
	}

	req.IDPlaceholder = respPayload.UniqueIdentifier

	return &ResponseBatchItem{
		ResponsePayload: respPayload,
	}, nil
}
