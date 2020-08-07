package kmip_test

import (
	"bufio"
	"fmt"
	"github.com/gemalto/kmip-go"
	"github.com/gemalto/kmip-go/kmip14"
	"github.com/gemalto/kmip-go/ttlv"
	"github.com/google/uuid"
	"net"
	"time"
)

func Example_client() {

	conn, err := net.DialTimeout("tcp", "localhost:5696", 3*time.Second)
	if err != nil {
		panic(err)
	}

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
				Operation:         kmip14.OperationDiscoverVersions,
				RequestPayload: kmip.DiscoverVersionsRequestPayload{
					ProtocolVersion: []kmip.ProtocolVersion{
						{ProtocolVersionMajor: 1, ProtocolVersionMinor: 2},
					},
				},
			},
		},
	}

	req, err := ttlv.Marshal(msg)
	if err != nil {
		panic(err)
	}

	fmt.Println(req)

	_, err = conn.Write(req)
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 5000)
	_, err = bufio.NewReader(conn).Read(buf)
	if err != nil {
		panic(err)
	}

	resp := ttlv.TTLV(buf)
	fmt.Println(resp)

}

func ExampleServer() {
	listener, err := net.Listen("tcp", "0.0.0.0:5696")
	if err != nil {
		panic(err)
	}

	kmip.DefaultProtocolHandler.LogTraffic = true

	kmip.DefaultOperationMux.Handle(kmip14.OperationDiscoverVersions, &kmip.DiscoverVersionsHandler{
		SupportedVersions: []kmip.ProtocolVersion{
			{
				ProtocolVersionMajor: 1,
				ProtocolVersionMinor: 4,
			},
			{
				ProtocolVersionMajor: 1,
				ProtocolVersionMinor: 3,
			},
			{
				ProtocolVersionMajor: 1,
				ProtocolVersionMinor: 2,
			},
		},
	})
	srv := kmip.Server{}
	panic(srv.Serve(listener))

}
