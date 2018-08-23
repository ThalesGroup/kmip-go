package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"github.com/google/uuid"
	"gitlab.protectv.local/regan/kmip.git"
)

func main() {

	client()
}

func client() {
	//resp, out, err := requester.Receive(requester.Get(), requester.URL("http://52.86.120.81"),
	//	requester.Client(httpclient.SkipVerify(true), httpclient.NoRedirects()))
	//
	//http.Get()
	//
	//if err != nil {
	//	panic(err)
	//}
	//
	//fmt.Println(resp.StatusCode)
	//fmt.Println(string(out))

	cert, err := tls.LoadX509KeyPair("/Users/russellegan/Downloads/cryptsoft/kmipc_server-1.9.2a/bin/client.pem", "/Users/russellegan/Downloads/cryptsoft/kmipc_server-1.9.2a/bin/client.pem")
	if err != nil {
		panic(err)
	}

	conf := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	conn, err := tls.Dial("tcp", "localhost:5696", conf)

	//conn, err := net.DialTimeout("tcp","localhost:5696", 3 * time.Second)
	if err != nil {
		panic(err)
	}

	fmt.Println("connected")

	biID := uuid.New()

	msg := kmip.RequestMessage{
		RequestHeader: kmip.RequestHeader{
			ProtocolVersion: kmip.ProtocolVersion{
				ProtocolVersionMajor: 1,
				ProtocolVersionMinor: 0,
			},
			BatchCount:             1,
			ClientCorrelationValue: uuid.New().String(),
		},
		BatchItem: []kmip.RequestBatchItem{
			{
				UniqueBatchItemID: biID[:],
				Operation:         kmip.OperationDiscoverVersions,
				RequestPayload: kmip.DiscoverVersionsRequestPayload{
					ProtocolVersion: []kmip.ProtocolVersion{
						{ProtocolVersionMajor: 1, ProtocolVersionMinor: 2},
					},
				},
			},
		},
	}

	mmsg, err := kmip.Marshal(msg)
	if err != nil {
		panic(err)
	}

	fmt.Println("== REQUEST ==")
	fmt.Println("")
	fmt.Println(kmip.TTLV(mmsg))
	fmt.Println("")

	n, err := conn.Write(mmsg)
	if err != nil {
		panic(err)
	}

	fmt.Println("wrote", n, "bytes")

	buf := make([]byte, 5000)
	n, err = bufio.NewReader(conn).Read(buf)
	if err != nil {
		panic(err)
	}

	fmt.Println("read", n, "bytes")

	ttlv := kmip.TTLV(buf)

	fmt.Println("")
	fmt.Println("== RESPONSE ==")
	if ttlv.Valid() == nil {

		fmt.Println("")
		fmt.Println(ttlv)
	} else {
		fmt.Println("response is invalid:")
		fmt.Println(kmip.Details(err))
	}
}
