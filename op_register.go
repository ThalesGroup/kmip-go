package kmip

import (
	"context"
	"github.com/ansel1/merry"
)

// 4.3

// Table 169

type RegisterRequestPayload struct {
	ObjectType ObjectType
	TemplateAttribute TemplateAttribute
	Certificate *Certificate
	SymmetricKey *SymmetricKey
	PrivateKey *PrivateKey
	PublicKey *PublicKey
	SplitKey *SplitKey
	Template *Template
	SecretData *SecretData
	OpaqueObject *OpaqueObject
}

// Table 170

type RegisterResponsePayload struct {
	UniqueIdentifier string
	TemplateAttribute TemplateAttribute
}

type RegisterHandler struct {
	SkipValidation bool
	RegisterFunc func(context.Context, *RegisterRequestPayload) (*RegisterResponsePayload, error)
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
			return nil, WithResultReason(merry.UserError("Object Type is not recognized"), ResultReasonInvalidField)
		case ObjectTypeCertificate:
			payloadPresent = payload.Certificate != nil
		case ObjectTypeSymmetricKey:
			payloadPresent = payload.SymmetricKey != nil
		case ObjectTypePrivateKey:
			payloadPresent = payload.PrivateKey != nil
		case ObjectTypePublicKey:
			payloadPresent = payload.PublicKey != nil
		case ObjectTypeSplitKey:
			payloadPresent = payload.SplitKey != nil
		case ObjectTypeTemplate:
			payloadPresent = payload.Template != nil
		case ObjectTypeSecretData:
			payloadPresent = payload.SecretData != nil
		case ObjectTypeOpaqueObject:
			payloadPresent = payload.OpaqueObject != nil
		}
		if !payloadPresent {
			return nil, WithResultReason(merry.UserErrorf("Object Type %s does not match type of cryptographic object provided", payload.ObjectType.String()), ResultReasonInvalidField)
		}
	}

	respPayload, err := h.RegisterFunc(ctx, &payload)
	if err != nil {
		return nil, err
	}

	req.IDPlaceholder = respPayload.TemplateAttribute.GetTag(TagUniqueIdentifier, 0).(string)

	return &ResponseBatchItem{
		ResponsePayload: respPayload,
	}, nil
}
