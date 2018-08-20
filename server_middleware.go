package kmip

import (
	"context"
	"fmt"
	"github.com/google/uuid"
)

type Middleware func(next Handler) Handler

func Wrap(h Handler, mw ...Middleware) Handler {
	for i := len(mw); i > 0; i-- {
		h = mw[i-1](h)
	}
	return h
}

type RawMiddleware func(next RawHandler) RawHandler

func WrapRaw(h RawHandler, mw ...RawMiddleware) RawHandler {
	for i := len(mw); i > 0; i-- {
		h = mw[i-1](h)
	}
	return h
}

type RawHandler interface {
	HandleRaw(ctx context.Context, req *Request) TTLV
}

var DefaultHandler = &ProtocolHandler{}

type ProtocolHandler struct {
	ProtocolVersion ProtocolVersion
	Handler         Handler
}

// TODO: replace with instance pooling
func (h *ProtocolHandler) newResponseMessage() *ResponseMessage {
	return &ResponseMessage{
		ResponseHeader: ResponseHeader{
			ProtocolVersion:        h.ProtocolVersion,
			ServerCorrelationValue: uuid.New().String(),
		},
	}
}

func (h *ProtocolHandler) HandleRaw(ctx context.Context, req *Request) TTLV {

	resp := h.newResponseMessage()

	ttlv := req.TTLV
	if err := ttlv.Valid(); err != nil {
		resp.addFailure(ResultReasonInvalidMessage, "invalid ttlv: "+err.Error())
		return mustEncode(resp)
	}

	if ttlv.Tag() != TagRequestMessage {
		resp.addFailure(ResultReasonInvalidMessage, fmt.Sprintf("invalid tag: expected RequestMessage, was %s", ttlv.Tag().String()))
		return mustEncode(resp)
	}

	// TODO: replace with instance pooling
	var reqm RequestMessage
	req.RequestMessage = &reqm

	err := Unmarshal(ttlv, &reqm)
	if err != nil {
		resp.addFailure(ResultReasonInvalidMessage, "failed to parse message: "+err.Error())
		return mustEncode(resp)
	}

	// TODO: test for protocol mismatch case
	if reqm.RequestHeader.ProtocolVersion.ProtocolVersionMajor != h.ProtocolVersion.ProtocolVersionMajor {
		resp.addFailure(ResultReasonInvalidMessage, "mismatched protocol versions")
		return mustEncode(resp)
	}

	// TODO: what do to if Handler is nil?
	err = h.Handler.Handle(ctx, req, resp)
	if err != nil {
		// TODO: have error types which translate into error responses.  Panic on other types of errors.
		panic(err)
	}

	respTTLV := mustEncode(resp)

	if req.RequestMessage.RequestHeader.MaximumResponseSize > 0 && len(respTTLV) > req.RequestMessage.RequestHeader.MaximumResponseSize {
		// new error resp
		resp = h.newResponseMessage()
		resp.addFailure(ResultReasonResponseTooLarge, "")
		return mustEncode(resp)
	}

	return respTTLV
}

func mustEncode(v interface{}) TTLV {
	ttlv, err := Marshal(v)
	if err != nil {
		panic(err)
	}
	return ttlv
}

func (r *ResponseMessage) addBatchItem(b *ResponseBatchItem) {
	r.BatchItem = append(r.BatchItem, *b)
	r.ResponseHeader.BatchCount++
}

func (r *ResponseMessage) addFailure(reason ResultReason, msg string) {
	r.addBatchItem(&ResponseBatchItem{
		ResultStatus:  ResultStatusOperationFailed,
		ResultReason:  reason,
		ResultMessage: msg,
	})
}
