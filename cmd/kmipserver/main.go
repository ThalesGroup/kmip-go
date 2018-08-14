package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"crypto/x509"
	"gitlab.protectv.local/regan/kmip.git"
	"bufio"
	"io"
	"context"
	"github.com/k0kubun/pp"
)

func main() {
	cert, err := tls.LoadX509KeyPair("/Users/russellegan/Downloads/cryptsoft/kmipc_server-1.9.2a/bin/server.pem", "/Users/russellegan/Downloads/cryptsoft/kmipc_server-1.9.2a/bin/server.pem")
	if err != nil {
		panic(err)
	}

	conf := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	listener, err := tls.Listen("tcp", "0.0.0.0:5696", conf)
	if err != nil {
		panic(err)
	}

	fmt.Println("server: listening")

	srv := kmip.Server{
		Handler:kmip.HandlerFunc(func(ctx context.Context, req *kmip.Request, resp *kmip.ResponseMessage) error {
			fmt.Println("got: ", pp.Sprint(req))
			resp.ResponseHeader.ProtocolVersion.ProtocolVersionMajor = 1
			resp.ResponseHeader.ProtocolVersion.ProtocolVersionMinor = 0
			resp.ResponseHeader.BatchCount = 1
			resp.BatchItem = []kmip.ResponseBatchItem{
				{
					Operation:    kmip.OperationDiscoverVersions,
					ResultStatus: kmip.ResultStatusSuccess,
					ResponsePayload: kmip.DiscoverVersionsResponsePayload{
						ProtocolVersion: []kmip.ProtocolVersion{
							{
								ProtocolVersionMajor: 1,
								ProtocolVersionMinor: 0,
							},
						},
					},
				},
			}
			return nil
		}),
	}

	panic(srv.Serve(listener))

	//for {
	//	conn, err := listener.Accept()
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	fmt.Printf("server: accepted from %s\n", conn.RemoteAddr())
	//
	//	go handleClient(conn)
	//}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	tlscon, ok := conn.(*tls.Conn)
	if ok {
		fmt.Print("ok=true")
		state := tlscon.ConnectionState()
		for _, v := range state.PeerCertificates {
			fmt.Print(x509.MarshalPKIXPublicKey(v.PublicKey))
		}
	}

	for {
		fmt.Print("server: conn: waiting")

		bufr := bufio.NewReader(conn)
		ttlv, err := readTTLV(bufr)
		if err != nil {
			fmt.Printf("server: conn: read: %s", err)
			if len(ttlv) > 0 {
				fmt.Println(ttlv)
			}
			if err == io.EOF {
				fmt.Println("server: conn: client disconnected")
				return
			}
			panic(err)
		}

		fmt.Println("server: conn: read", ttlv)

		fmt.Println("writing response")
		err = writeTTLV(conn)
		if err != nil {
			panic(err)
		}

	}
	fmt.Println("server: conn: closed")
}

func readTTLV(bufr *bufio.Reader) (kmip.TTLV, error) {

	// first, read the header
	header, err := bufr.Peek(8)
	if err != nil {
		return nil, err
	}

	if err := kmip.TTLV(header).ValidHeader(); err != nil {
		// bad header, abort
		return kmip.TTLV(header), err
	}

	// allocate a buffer large enough for the entire message
	fullLen := kmip.TTLV(header).FullLen()
	buf := make([]byte, fullLen)

	var totRead int
	for {
		n, err := bufr.Read(buf[totRead:])
		if err != nil {
			return kmip.TTLV(buf), err
		}

		totRead += n
		if totRead >= fullLen {
			// we've read off a single full message
			return kmip.TTLV(buf), nil
		}
		// keep reading
	}

}

func writeTTLV(w io.Writer) error {
	ttlv, err := kmip.Marshal(kmip.ResponseMessage{
		ResponseHeader:kmip.ResponseHeader{
			ProtocolVersion:kmip.ProtocolVersion{
				ProtocolVersionMajor:1,
				ProtocolVersionMinor:0,
			},
			BatchCount:1,
		},
		BatchItem:[]kmip.ResponseBatchItem{
			{
				Operation:kmip.OperationDiscoverVersions,
				ResultStatus:kmip.ResultStatusSuccess,
				ResponsePayload:kmip.DiscoverVersionsResponsePayload{
					ProtocolVersion:[]kmip.ProtocolVersion{
						{
							ProtocolVersionMajor:1,
							ProtocolVersionMinor:0,
						},
					},
				},
			},
		},
	})

	if err != nil {
		return err
	}

	fmt.Println("logging", kmip.TTLV(ttlv))

	_, err = w.Write(ttlv)
	if err != nil {
		return err
	}

	return nil
}
