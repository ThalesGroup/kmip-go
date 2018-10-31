package main

import (
	"crypto/tls"
	"fmt"
	"github.com/gemalto/flume"
	"gitlab.protectv.local/regan/kmip.git"
	"gitlab.protectv.local/regan/kmip.git/ttlv"
)

func main() {
	flume.Configure(flume.Config{
		Development:  true,
		DefaultLevel: flume.DebugLevel,
	})

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

	kmip.DefaultProtocolHandler.LogTraffic = true

	kmip.DefaultOperationMux.Handle(ttlv.OperationDiscoverVersions, &kmip.DiscoverVersionsHandler{
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

	//kmip.DefaultHandler.MessageHandler = kmip.HandlerFunc(func(ctx context.Context, req *kmip.Request, resp *kmip.ResponseMessage) error {
	//	fmt.Println("got: ", pp.Sprint(req))
	//	resp.ResponseHeader.ProtocolVersion.ProtocolVersionMajor = 1
	//	resp.ResponseHeader.ProtocolVersion.ProtocolVersionMinor = 0
	//	resp.ResponseHeader.BatchCount = 1
	//	resp.BatchItem = []kmip.ResponseBatchItem{
	//		{
	//			Operation:    kmip.OperationDiscoverVersions,
	//			ResultStatus: kmip.ResultStatusSuccess,
	//			ResponsePayload: kmip.DiscoverVersionsResponsePayload{
	//				ProtocolVersion: []kmip.ProtocolVersion{
	//					{
	//						ProtocolVersionMajor: 1,
	//						ProtocolVersionMinor: 0,
	//					},
	//				},
	//			},
	//		},
	//	}
	//	return nil
	//})

	srv := kmip.Server{}

	panic(srv.Serve(listener))

}
