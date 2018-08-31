package kmip

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/gemalto/flume"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var serverLog = flume.New("kmip_server")

type Server struct {
	Handler ProtocolHandler

	mu         sync.Mutex
	listeners  map[*net.Listener]struct{}
	inShutdown int32 // accessed atomically (non-zero means we're in Shutdown)
}

// ErrServerClosed is returned by the Server's Serve, ServeTLS, ListenAndServe,
// and ListenAndServeTLS methods after a call to Shutdown or Close.
var ErrServerClosed = errors.New("http: Server closed")

// Serve accepts incoming connections on the Listener l, creating a
// new service goroutine for each. The service goroutines read requests and
// then call srv.MessageHandler to reply to them.
//
// Serve always returns a non-nil error and closes l.
// After Shutdown or Close, the returned error is ErrServerClosed.
func (srv *Server) Serve(l net.Listener) error {
	//if fn := testHookServerServe; fn != nil {
	//	fn(srv, l) // call hook with unwrapped listener
	//}

	l = &onceCloseListener{Listener: l}
	defer l.Close()

	if !srv.trackListener(&l, true) {
		return ErrServerClosed
	}
	defer srv.trackListener(&l, false)

	var tempDelay time.Duration     // how long to sleep on accept failure
	baseCtx := context.Background() // base is always background, per Issue 16220
	ctx := baseCtx
	//ctx := context.WithValue(baseCtx, ServerContextKey, srv)
	for {
		rw, e := l.Accept()
		if e != nil {
			if srv.shuttingDown() {
				return ErrServerClosed
			}
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				//srv.logf("http: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		c := &conn{server: srv, rwc: rw}
		//c.setState(c.rwc, StateNew) // before Serve can return
		go c.serve(ctx)
	}
}

// Close immediately closes all active net.Listeners and any
// connections in state StateNew, StateActive, or StateIdle. For a
// graceful shutdown, use Shutdown.
//
// Close does not attempt to close (and does not even know about)
// any hijacked connections, such as WebSockets.
//
// Close returns any error returned from closing the Server's
// underlying Listener(s).
func (srv *Server) Close() error {
	atomic.StoreInt32(&srv.inShutdown, 1)
	srv.mu.Lock()
	defer srv.mu.Unlock()
	//srv.closeDoneChanLocked()
	err := srv.closeListenersLocked()
	//for c := range srv.activeConn {
	//	c.rwc.Close()
	//	delete(srv.activeConn, c)
	//}
	return err
}

// shutdownPollInterval is how often we poll for quiescence
// during Server.Shutdown. This is lower during tests, to
// speed up tests.
// Ideally we could find a solution that doesn't involve polling,
// but which also doesn't have a high runtime cost (and doesn't
// involve any contentious mutexes), but that is left as an
// exercise for the reader.
var shutdownPollInterval = 500 * time.Millisecond

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Shutdown works by first closing all open
// listeners, then closing all idle connections, and then waiting
// indefinitely for connections to return to idle and then shut down.
// If the provided context expires before the shutdown is complete,
// Shutdown returns the context's error, otherwise it returns any
// error returned from closing the Server's underlying Listener(s).
//
// When Shutdown is called, Serve, ListenAndServe, and
// ListenAndServeTLS immediately return ErrServerClosed. Make sure the
// program doesn't exit and waits instead for Shutdown to return.
//
// Shutdown does not attempt to close nor wait for hijacked
// connections such as WebSockets. The caller of Shutdown should
// separately notify such long-lived connections of shutdown and wait
// for them to close, if desired. See RegisterOnShutdown for a way to
// register shutdown notification functions.
//
// Once Shutdown has been called on a server, it may not be reused;
// future calls to methods such as Serve will return ErrServerClosed.
func (srv *Server) Shutdown(ctx context.Context) error {
	atomic.StoreInt32(&srv.inShutdown, 1)

	srv.mu.Lock()
	lnerr := srv.closeListenersLocked()
	//srv.closeDoneChanLocked()
	//for _, f := range srv.onShutdown {
	//	go f()
	//}
	srv.mu.Unlock()

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	return lnerr
	//for {
	//	if srv.closeIdleConns() {
	//		return lnerr
	//	}
	//	select {
	//	case <-ctx.Done():
	//		return ctx.Err()
	//	case <-ticker.C:
	//	}
	//}
}

func (s *Server) closeListenersLocked() error {
	var err error
	for ln := range s.listeners {
		if cerr := (*ln).Close(); cerr != nil && err == nil {
			err = cerr
		}
		delete(s.listeners, ln)
	}
	return err
}

// trackListener adds or removes a net.Listener to the set of tracked
// listeners.
//
// We store a pointer to interface in the map set, in case the
// net.Listener is not comparable. This is safe because we only call
// trackListener via Serve and can track+defer untrack the same
// pointer to local variable there. We never need to compare a
// Listener from another caller.
//
// It reports whether the server is still up (not Shutdown or Closed).
func (s *Server) trackListener(ln *net.Listener, add bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listeners == nil {
		s.listeners = make(map[*net.Listener]struct{})
	}
	if add {
		if s.shuttingDown() {
			return false
		}
		s.listeners[ln] = struct{}{}
	} else {
		delete(s.listeners, ln)
	}
	return true
}

func (s *Server) shuttingDown() bool {
	return atomic.LoadInt32(&s.inShutdown) != 0
}

type conn struct {
	rwc        net.Conn
	remoteAddr string
	localAddr  string
	tlsState   *tls.ConnectionState
	// cancelCtx cancels the connection-level context.
	cancelCtx context.CancelFunc

	// bufr reads from rwc.
	bufr *bufio.Reader
	dec  *Decoder

	server *Server
}

func (c *conn) close() {
	// TODO: http package has a buffered writer on the conn too, which is flushed here
	c.rwc.Close()
}

// Serve a new connection.
func (c *conn) serve(ctx context.Context) {

	ctx = flume.WithLogger(ctx, serverLog)
	ctx, cancelCtx := context.WithCancel(ctx)
	c.cancelCtx = cancelCtx
	c.remoteAddr = c.rwc.RemoteAddr().String()
	c.localAddr = c.rwc.LocalAddr().String()
	//ctx = context.WithValue(ctx, LocalAddrContextKey, c.rwc.LocalAddr())
	defer func() {
		if err := recover(); err != nil {
			// TODO: logging support
			//if err := recover(); err != nil && err != ErrAbortHandler {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			if e, ok := err.(error); ok {
				fmt.Printf("kmip: panic serving %v: %v\n%s", c.remoteAddr, Details(e), buf)
			} else {
				fmt.Printf("kmip: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
			}

			//c.server.logf("http: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
		}
		cancelCtx()
		//if !c.hijacked() {
		c.close()
		//	c.setState(c.rwc, StateClosed)
		//}
	}()

	if tlsConn, ok := c.rwc.(*tls.Conn); ok {
		//if d := c.server.ReadTimeout; d != 0 {
		//	c.rwc.SetReadDeadline(time.Now().Add(d))
		//}
		//if d := c.server.WriteTimeout; d != 0 {
		//	c.rwc.SetWriteDeadline(time.Now().Add(d))
		//}
		if err := tlsConn.Handshake(); err != nil {
			// TODO: logging support
			fmt.Printf("kmip: TLS handshake error from %s: %v", c.rwc.RemoteAddr(), err)
			//c.server.logf("http: TLS handshake error from %s: %v", c.rwc.RemoteAddr(), err)
			return
		}
		c.tlsState = new(tls.ConnectionState)
		*c.tlsState = tlsConn.ConnectionState()
		//if proto := c.tlsState.NegotiatedProtocol; validNPN(proto) {
		//	if fn := c.server.TLSNextProto[proto]; fn != nil {
		//		h := initNPNRequest{tlsConn, serverHandler{c.server}}
		//		fn(c.server, tlsConn, h)
		//	}
		//	return
		//}
	}

	// TODO: do we really need instance pooling here?  We expect KMIP connections to be long lasting
	c.dec = NewDecoder(c.rwc)
	c.bufr = bufio.NewReader(c.rwc)
	//c.bufw = newBufioWriterSize(checkConnErrorWriter{c}, 4<<10)

	for {
		w, err := c.readRequest(ctx)
		//if c.r.remain != c.server.initialReadLimitSize() {
		// If we read any bytes off the wire, we're active.
		//c.setState(c.rwc, StateActive)
		//}
		if err != nil {
			if err == io.EOF {
				fmt.Println("client closed connection")
				return
			}

			// TODO: do something with this error
			panic(err)
			//const errorHeaders= "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"
			//
			//if err == errTooLarge {
			//	// Their HTTP client may or may not be
			//	// able to read this if we're
			//	// responding to them and hanging up
			//	// while they're still writing their
			//	// request. Undefined behavior.
			//	const publicErr= "431 Request Header Fields Too Large"
			//	fmt.Fprintf(c.rwc, "HTTP/1.1 "+publicErr+errorHeaders+publicErr)
			//	c.closeWriteAndWait()
			//	return
			//}
			//if isCommonNetReadError(err) {
			//	return // don't reply
			//}
			//
			//publicErr := "400 Bad Request"
			//if v, ok := err.(badRequestError); ok {
			//	publicErr = publicErr + ": " + string(v)
			//}
			//
			//fmt.Fprintf(c.rwc, "HTTP/1.1 "+publicErr+errorHeaders+publicErr)
			//return
		}

		// Expect 100 Continue support
		//req := w.req
		//if req.expectsContinue() {
		//	if req.ProtoAtLeast(1, 1) && req.ContentLength != 0 {
		//		// Wrap the Body reader with one that replies on the connection
		//		req.Body = &expectContinueReader{readCloser: req.Body, resp: w}
		//	}
		//} else if req.Header.get("Expect") != "" {
		//	w.sendExpectationFailed()
		//	return
		//}

		//c.curReq.Store(w)

		//if requestBodyRemains(req.Body) {
		//	registerOnHitEOF(req.Body, w.conn.r.startBackgroundRead)
		//} else {
		//	w.conn.r.startBackgroundRead()
		//}

		// HTTP cannot have multiple simultaneous active requests.[*]
		// Until the server replies to this request, it can't read another,
		// so we might as well run the handler in this goroutine.
		// [*] Not strictly true: HTTP pipelining. We could let them all process
		// in parallel even if their responses need to be serialized.
		// But we're not going to implement HTTP pipelining because it
		// was never deployed in the wild and the answer is HTTP/2.

		h := c.server.Handler
		if h == nil {
			h = DefaultProtocolHandler
		}

		//var resp ResponseMessage
		//err = c.server.MessageHandler.Handle(ctx, w, &resp)
		// TODO: this cancelCtx() was created at the connection level, not the request level.  Need to
		// figure out how to handle connection vs request timeouts and cancels.
		//cancelCtx()

		// TODO: use recycled buffered writer
		writer := bufio.NewWriter(c.rwc)
		h.ServeKMIP(ctx, w, writer)
		err = writer.Flush()
		if err != nil {
			// TODO: handle error
			panic(err)
		}

		//serverHandler{c.server}.ServeHTTP(w, w.req)
		//w.cancelCtx()
		//if c.hijacked() {
		//	return
		//}
		//w.finishRequest()
		//if !w.shouldReuseConnection() {
		//	if w.requestBodyLimitHit || w.closedRequestBodyEarly() {
		//		c.closeWriteAndWait()
		//	}
		//	return
		//}
		//c.setState(c.rwc, StateIdle)
		//c.curReq.Store((*response)(nil))

		//if !w.conn.server.doKeepAlives() {
		//	// We're in shutdown mode. We might've replied
		//	// to the user without "Connection: close" and
		//	// they might think they can send another
		//	// request, but such is life with HTTP/1.1.
		//	return
		//}
		//
		//if d := c.server.idleTimeout(); d != 0 {
		//	c.rwc.SetReadDeadline(time.Now().Add(d))
		//	if _, err := c.bufr.Peek(4); err != nil {
		//		return
		//	}
		//}
		//c.rwc.SetReadDeadline(time.Time{})
	}
}

// Read next request from connection.
func (c *conn) readRequest(ctx context.Context) (w *Request, err error) {
	//if c.hijacked() {
	//	return nil, ErrHijacked
	//}

	//var (
	//	wholeReqDeadline time.Time // or zero if none
	//	hdrDeadline      time.Time // or zero if none
	//)
	//t0 := time.Now()
	//if d := c.server.readHeaderTimeout(); d != 0 {
	//	hdrDeadline = t0.Add(d)
	//}
	//if d := c.server.ReadTimeout; d != 0 {
	//	wholeReqDeadline = t0.Add(d)
	//}
	//c.rwc.SetReadDeadline(hdrDeadline)
	//if d := c.server.WriteTimeout; d != 0 {
	//	defer func() {
	//		c.rwc.SetWriteDeadline(time.Now().Add(d))
	//	}()
	//}

	//c.r.setReadLimit(c.server.initialReadLimitSize())
	//if c.lastMethod == "POST" {
	// RFC 7230 section 3 tolerance for old buggy clients.
	//peek, _ := c.bufr.Peek(4) // ReadRequest will get err below
	//c.bufr.Discard(numLeadingCRorLF(peek))
	//}
	ttlv, err := c.dec.NextTTLV()
	if err != nil {
		return nil, err
	}
	//if err != nil {
	//if c.r.hitReadLimit() {
	//	return nil, errTooLarge
	//}
	//}

	// TODO: use pooling to recycle requests?
	req := &Request{
		TTLV:       ttlv,
		RemoteAddr: c.remoteAddr,
		LocalAddr:  c.localAddr,
		TLS:        c.tlsState,
	}

	//c.r.setInfiniteReadLimit()

	// Adjust the read deadline if necessary.
	//if !hdrDeadline.Equal(wholeReqDeadline) {
	//	c.rwc.SetReadDeadline(wholeReqDeadline)
	//}

	return req, nil
}

type Request struct {
	TTLV                TTLV
	Message             *RequestMessage
	CurrentItem         *RequestBatchItem
	DisallowExtraValues bool

	TLS        *tls.ConnectionState
	RemoteAddr string
	LocalAddr  string

	IDPlaceholder string

	decoder *Decoder
}

// coerceToTTLV attempts to coerce an interface value to TTLV.
// In most production scenarios, this is intended to be used in
// places where the value is already a TTLV, and just needs to be
// type cast.  If v is not TTLV, it will be marshaled.  This latter
// behavior is slow, so it should be used only in tests.
func coerceToTTLV(v interface{}) (TTLV, error) {
	switch t := v.(type) {
	case nil:
		return nil, nil
	case TTLV:
		return t, nil
	default:
		return Marshal(v)
	}
}

// Unmarshal unmarshals ttlv into structures.  Handlers should prefer this
// method over than their own Decoders or Unmarshal().  This method
// enforces rules about whether extra fields are allowed, and reuses
// buffers for efficiency.
func (r *Request) Unmarshal(ttlv TTLV, into interface{}) error {
	if len(ttlv) == 0 {
		return nil
	}
	r.decoder.Reset(bytes.NewReader(ttlv))
	return r.decoder.Decode(into)
}

func (r *Request) DecodePayload(v interface{}) error {
	if r.CurrentItem == nil {
		return nil
	}
	ttlv, err := coerceToTTLV(r.CurrentItem.RequestPayload)
	if err != nil {
		return err
	}
	return r.Unmarshal(ttlv, v)
}

// onceCloseListener wraps a net.Listener, protecting it from
// multiple Close calls.
type onceCloseListener struct {
	net.Listener
	once     sync.Once
	closeErr error
}

func (oc *onceCloseListener) Close() error {
	oc.once.Do(oc.close)
	return oc.closeErr
}

func (oc *onceCloseListener) close() { oc.closeErr = oc.Listener.Close() }
