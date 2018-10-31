package compliance

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/gemalto/flume"
	"github.com/gemalto/flume/flumetest"
	"github.com/stretchr/testify/require"
	"gitlab.protectv.local/regan/kmip.git"
	"gitlab.protectv.local/regan/kmip.git/mock"
	"gitlab.protectv.local/regan/kmip.git/ttlv"
	"io"
	"net"
	"os"
	"testing"
)

type Client interface {
	Do(req []byte) ([]byte, error)
	io.Closer
}

var c Client

type MockClient struct {
	srv  *kmip.Server
	addr string
}

var log = flume.New("compliance")

func (mc *MockClient) Do(req []byte) ([]byte, error) {

	// assume the req is xml, convert it to TTLV

	var reqTTLV ttlv.TTLV
	err := xml.Unmarshal(req, &reqTTLV)
	if err != nil {
		return nil, merry.Wrap(err)
	}

	conn, err := net.Dial("tcp", mc.addr)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	defer conn.Close()

	log.Debug("writing request", reqTTLV)
	//writer := bufio.NewWriter(conn)
	_, err = conn.Write(reqTTLV)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	//err = writer.Flush()
	//if err != nil {
	//	return nil, merry.Wrap(err)
	//}

	dec := ttlv.NewDecoder(bufio.NewReader(conn))
	respTTLV, err := dec.NextTTLV()
	if err != nil {
		return nil, merry.Wrap(err)
	}

	// convert the TTLV back to xml
	respXML, err := xml.MarshalIndent(ttlv.TTLV(respTTLV), "", "  ")
	if err != nil {
		return nil, merry.Wrap(err)
	}

	return respXML, nil
}

func (mc *MockClient) Close() error {
	if mc.srv != nil {
		return mc.srv.Close()
	}
	return nil
}

func init() {
	flumetest.SetDefaults()

	ms := mock.NewMockServer()
	ms.Handle(ttlv.OperationRegister, &kmip.RegisterHandler{
		RegisterFunc: func(ctx context.Context, payload *kmip.RegisterRequestPayload) (*kmip.RegisterResponsePayload, error) {
			attrs := kmip.TemplateAttribute{}
			attrs.Set2("Rank and File", "red", 0)
			return &kmip.RegisterResponsePayload{
				UniqueIdentifier:  "red",
				TemplateAttribute: attrs,
			}, nil
		},
	})

	mc := MockClient{
		srv: &kmip.Server{
			Handler: &kmip.StandardProtocolHandler{
				ProtocolVersion: kmip.ProtocolVersion{
					ProtocolVersionMajor: 1,
					ProtocolVersionMinor: 4,
				},
				LogTraffic:     true,
				MessageHandler: ms,
			},
		},
	}

	listener, err := net.Listen("tcp", "0.0.0.0:5696")
	if err != nil {
		panic(err)
	}

	mc.addr = listener.Addr().String()
	go mc.srv.Serve(listener)
	c = &mc

}

func TestTC_CREG_1_14(t *testing.T) {
	t.Skip()
	defer flumetest.Start(t)()

	f, err := os.Open("TC-CREG-1-14.xml")
	require.NoError(t, err)
	defer f.Close()

	xmlDec := xml.NewDecoder(bufio.NewReader(f))
	type KMIP struct {
		Content []TTLV `xml:",any"`
	}

	var k KMIP

	err = xmlDec.Decode(&k)
	require.NoError(t, err)

	for i := 0; i < len(k.Content); i++ {
		req := k.Content[i]
		i++
		expectedResp := k.Content[i]

		reqXml, err := xml.MarshalIndent(req, "", "  ")
		require.NoError(t, err)

		fmt.Println(string(reqXml))

		t.Log("sending req:", string(reqXml))

		respXML, err := c.Do(reqXml)
		require.NoError(t, err)

		t.Log("got response:", string(respXML))
		var resp TTLV
		err = xml.Unmarshal(respXML, &resp)
		require.NoError(t, err)

		eq, vars, diff := Compare(&expectedResp, &resp)
		if !eq {
			t.Logf("vars: %#v", vars)
			t.Log("diff:", diff)
			require.Fail(t, "test failed, response did not match expected")
		}
	}
}
