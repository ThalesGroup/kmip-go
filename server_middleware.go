package kmip

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/gemalto/flume"
	"github.com/google/uuid"
	"io"
	"sync"
	"time"
)

type ResponseWriter interface {
	io.Writer
}

type ProtocolHandler interface {
	ServeKMIP(ctx context.Context, req *Request, resp ResponseWriter)
}

type MessageHandler interface {
	HandleMessage(ctx context.Context, req *Request, resp *Response)
}

type ItemHandler interface {
	HandleItem(ctx context.Context, req *Request) (item *ResponseBatchItem, err error)
}

type ProtocolHandlerFunc func(context.Context, *Request, ResponseWriter)

func (f ProtocolHandlerFunc) ServeKMIP(ctx context.Context, r *Request, w ResponseWriter) {
	f(ctx, r, w)
}

type MessageHandlerFunc func(context.Context, *Request, *Response)

func (f MessageHandlerFunc) HandleMessage(ctx context.Context, req *Request, resp *Response) {
	f(ctx, req, resp)
}

type ItemHandlerFunc func(context.Context, *Request) (*ResponseBatchItem, error)

func (f ItemHandlerFunc) HandleItem(ctx context.Context, req *Request) (item *ResponseBatchItem, err error) {
	return f(ctx, req)
}

var DefaultProtocolHandler = &StandardProtocolHandler{
	MessageHandler: DefaultOperationMux,
	ProtocolVersion: ProtocolVersion{
		ProtocolVersionMajor: 1,
		ProtocolVersionMinor: 4,
	},
}

var DefaultOperationMux = &OperationMux{}

type StandardProtocolHandler struct {
	ProtocolVersion ProtocolVersion
	MessageHandler  MessageHandler
	LogTraffic      bool
}

func (h *StandardProtocolHandler) parseMessage(ctx context.Context, req *Request) error {
	ttlv := req.TTLV
	if err := ttlv.Valid(); err != nil {
		return merry.Prepend(err, "invalid ttlv")
	}

	if ttlv.Tag() != TagRequestMessage {
		return merry.Errorf("invalid tag: expected RequestMessage, was %s", ttlv.Tag().String())
	}

	var message RequestMessage
	err := Unmarshal(ttlv, &message)
	if err != nil {
		return merry.Prepend(err, "failed to parse message")
	}

	req.Message = &message

	return nil
}

var responsePool = sync.Pool{}

type Response struct {
	ResponseMessage
	buf bytes.Buffer
	enc *Encoder
}

func newResponse() *Response {
	v := responsePool.Get()
	if v != nil {
		r := v.(*Response)
		r.reset()
		return r
	}
	r := Response{}
	r.enc = NewEncoder(&r.buf)
	return &r
}

func releaseResponse(r *Response) {
	responsePool.Put(r)
}

func (r *Response) reset() {
	r.BatchItem = nil
	r.ResponseMessage = ResponseMessage{}
	r.buf.Reset()
}

func (r *Response) Bytes() []byte {
	r.buf.Reset()
	err := r.enc.Encode(&r.ResponseMessage)
	if err != nil {
		panic(err)
	}

	return r.buf.Bytes()
}

func (r *Response) errorResponse(reason ResultReason, msg string) {
	r.BatchItem = []ResponseBatchItem{
		{
			ResultStatus:  ResultStatusOperationFailed,
			ResultReason:  reason,
			ResultMessage: msg,
		},
	}
}

func (h *StandardProtocolHandler) handleRequest(ctx context.Context, req *Request, resp *Response) (logger flume.Logger) {
	// create a server correlation value, which is like a unique transaction ID
	scv := uuid.New().String()

	// create a logger for the transaction, seeded with the scv
	logger = flume.FromContext(ctx).With("scv", scv)
	// attach the logger to the context, so it is available to the handling chain
	ctx = flume.WithLogger(ctx, logger)

	// TODO: it's unclear how the full protocol negogiation is supposed to work
	// should server be pinned to a particular version?  Or should we try and negogiate a common version?
	resp.ResponseHeader.ProtocolVersion = h.ProtocolVersion
	resp.ResponseHeader.TimeStamp = time.Now()
	resp.ResponseHeader.BatchCount = len(resp.BatchItem)
	resp.ResponseHeader.ServerCorrelationValue = scv

	if err := h.parseMessage(ctx, req); err != nil {
		resp.errorResponse(ResultReasonInvalidMessage, err.Error())
		return
	}

	ccv := req.Message.RequestHeader.ClientCorrelationValue
	// add the client correlation value to the logging context.  This value uniquely
	// identifies the client, and is supposed to be included in server logs
	logger = logger.With("ccv", ccv)
	ctx = flume.WithLogger(ctx, logger)
	resp.ResponseHeader.ClientCorrelationValue = req.Message.RequestHeader.ClientCorrelationValue

	clientMajorVersion := req.Message.RequestHeader.ProtocolVersion.ProtocolVersionMajor
	if clientMajorVersion != h.ProtocolVersion.ProtocolVersionMajor {
		resp.errorResponse(ResultReasonInvalidMessage,
			fmt.Sprintf("mismatched protocol versions, client: %d, server: %d", clientMajorVersion, h.ProtocolVersion.ProtocolVersionMajor))
		return
	}

	// set a flag hinting to handlers that extra fields should not be tolerated when
	// unmarshaling payloads.  According to spec, if server and client protocol version
	// minor versions match, then extra fields should cause an error.  Not sure how to enforce
	// this in this higher level handler, since we (the protocol/message handlers) don't unmarshal the payload.
	// That's done by a particular item handler.
	req.DisallowExtraValues = req.Message.RequestHeader.ProtocolVersion.ProtocolVersionMinor == h.ProtocolVersion.ProtocolVersionMinor

	h.MessageHandler.HandleMessage(ctx, req, resp)

	respTTLV := resp.Bytes()

	if req.Message.RequestHeader.MaximumResponseSize > 0 && len(respTTLV) > req.Message.RequestHeader.MaximumResponseSize {
		// new error resp
		resp.errorResponse(ResultReasonResponseTooLarge, "")
		respTTLV = resp.Bytes()
	}

	return

}

func (h *StandardProtocolHandler) ServeKMIP(ctx context.Context, req *Request, writer ResponseWriter) {

	// we precreate the response object and pass it down to handlers, because due
	// the guidance in the spec on the Maximum Response Size, it will be necessary
	// for handlers to recalculate the response size after each batch item, which
	// requires re-encoding the entire response. Seems inefficient.
	resp := newResponse()
	logger := h.handleRequest(ctx, req, resp)

	var err error
	if h.LogTraffic {
		ttlv := resp.Bytes()

		logger.Debug("traffic log", "request", req.TTLV.String(), "response", TTLV(ttlv).String())
		_, err = writer.Write(ttlv)
	} else {
		_, err = resp.buf.WriteTo(writer)
	}
	if err != nil {
		panic(err)
	}

	releaseResponse(resp)

	//// create a server correlation value, which is like a unique transaction ID
	//scv := uuid.New().String()
	//
	//// create a logger for the transaction, seeded with the scv
	//logger := flume.FromContext(ctx).With("scv", scv)
	//// attach the logger to the context, so it is available to the handling chain
	//ctx = flume.WithLogger(ctx, logger)
	//
	//// we precreate the response object and pass it down to handlers, because due
	//// the guidance in the spec on the Maximum Response Size, it will be necessary
	//// for handlers to recalculate the response size after each batch item, which
	//// requires re-encoding the entire response. Seems inefficient.
	//resp := newResponse()
	//// TODO: it's unclear how the full protocol negogiation is supposed to work
	//// should server be pinned to a particular version?  Or should we try and negogiate a common version?
	//resp.ResponseHeader.ProtocolVersion = h.ProtocolVersion
	//resp.ResponseHeader.TimeStamp = time.Now()
	//resp.ResponseHeader.BatchCount = len(resp.BatchItem)
	//resp.ResponseHeader.ServerCorrelationValue = scv
	//
	//if err := h.parseMessage(ctx, req); err != nil {
	//	resp.errorResponse(ResultReasonInvalidMessage, err.Error())
	//	resp.mustWriteTo(writer)
	//	return
	//}
	//
	//ccv := req.Message.RequestHeader.ClientCorrelationValue
	//// add the client correlation value to the logging context.  This value uniquely
	//// identifies the client, and is supposed to be included in server logs
	//ctx = flume.WithLogger(ctx, flume.FromContext(ctx).With("ccv", ccv))
	//resp.ResponseHeader.ClientCorrelationValue = req.Message.RequestHeader.ClientCorrelationValue
	//
	//clientMajorVersion := req.Message.RequestHeader.ProtocolVersion.ProtocolVersionMajor
	//if clientMajorVersion != h.ProtocolVersion.ProtocolVersionMajor {
	//	resp.errorResponse(ResultReasonInvalidMessage,
	//		fmt.Sprintf("mismatched protocol versions, client: %d, server: %d", clientMajorVersion, h.ProtocolVersion.ProtocolVersionMajor))
	//	resp.mustWriteTo(writer)
	//	return
	//}
	//
	//h.MessageHandler.HandleMessage(ctx, req, resp)
	//
	//respTTLV := resp.Bytes()
	//
	//if req.Message.RequestHeader.MaximumResponseSize > 0 && len(respTTLV) > req.Message.RequestHeader.MaximumResponseSize {
	//	// new error resp
	//	resp.errorResponse(ResultReasonResponseTooLarge, "")
	//	respTTLV = resp.Bytes()
	//}
	//
	//resp.mustWriteTo(writer)
	//
	//releaseResponse(resp)
}

func (r *ResponseMessage) addFailure(reason ResultReason, msg string) {
	if msg == "" {
		msg = reason.String()
	}
	r.BatchItem = append(r.BatchItem, ResponseBatchItem{
		ResultStatus:  ResultStatusOperationFailed,
		ResultReason:  reason,
		ResultMessage: msg,
	})
}

type OperationMux struct {
	mu           sync.RWMutex
	handlers     map[Operation]ItemHandler
	ErrorHandler ErrorHandler
}

type ErrorHandler interface {
	HandleError(err error) *ResponseBatchItem
}

type ErrorHandlerFunc func(err error) *ResponseBatchItem

func (f ErrorHandlerFunc) HandleError(err error) *ResponseBatchItem {
	return f(err)
}

var DefaultErrorHandler = ErrorHandlerFunc(func(err error) *ResponseBatchItem {
	reason := GetResultReason(err)
	if reason == ResultReason(0) {
		// error not handled
		return nil
	}

	// prefer user message, but fall back on message
	msg := merry.UserMessage(err)
	if msg == "" {
		msg = merry.Message(err)
	}
	return newFailedResponseBatchItem(reason, msg)
})

func newFailedResponseBatchItem(reason ResultReason, msg string) *ResponseBatchItem {
	return &ResponseBatchItem{
		ResultStatus:  ResultStatusOperationFailed,
		ResultReason:  reason,
		ResultMessage: msg,
	}
}

func (m *OperationMux) bi(ctx context.Context, req *Request, reqItem *RequestBatchItem) *ResponseBatchItem {

	req.CurrentItem = reqItem
	h := m.handlerForOp(reqItem.Operation)
	if h == nil {
		return newFailedResponseBatchItem(ResultReasonOperationNotSupported, "")
	}

	resp, err := h.HandleItem(ctx, req)
	if err != nil {
		eh := m.ErrorHandler
		if eh == nil {
			eh = DefaultErrorHandler
		}
		resp = eh.HandleError(err)
		if resp == nil {
			// errors which don't convert just panic
			panic(err)
		}
	}

	return resp
}

func (m *OperationMux) HandleMessage(ctx context.Context, req *Request, resp *Response) {
	for i := range req.Message.BatchItem {
		reqItem := &req.Message.BatchItem[i]
		respItem := m.bi(ctx, req, reqItem)
		respItem.Operation = reqItem.Operation
		respItem.UniqueBatchItemID = reqItem.UniqueBatchItemID
		resp.BatchItem = append(resp.BatchItem, *respItem)
	}
}

func (m *OperationMux) Handle(op Operation, handler ItemHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.handlers == nil {
		m.handlers = map[Operation]ItemHandler{}
	}

	m.handlers[op] = handler
}

func (m *OperationMux) handlerForOp(op Operation) ItemHandler {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.handlers[op]
}

func (m *OperationMux) missingHandler(ctx context.Context, req *Request, resp *ResponseMessage) error {
	resp.addFailure(ResultReasonOperationNotSupported, "")
	return nil
}
