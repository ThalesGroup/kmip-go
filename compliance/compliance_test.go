package compliance

import (
	"bufio"
	"encoding/xml"
	"github.com/ansel1/merry"
	"github.com/gemalto/flume"
	"github.com/gemalto/flume/flumetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.protectv.local/regan/kmip.git"
	"gitlab.protectv.local/regan/kmip.git/mock"
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
	srv *kmip.Server
	addr string
}

var log = flume.New("compliance")

func (mc *MockClient) Do(req []byte) ([]byte, error) {

	// assume the req is xml, convert it to TTLV

	var reqTTLV kmip.TTLV
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

	dec := kmip.NewDecoder(bufio.NewReader(conn))
	respTTLV, err :=dec.NextTTLV()
	if err != nil {
		return nil, merry.Wrap(err)
	}

	// convert the TTLV back to xml
	respXML, err := xml.MarshalIndent(kmip.TTLV(respTTLV), "", "  ")
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

	mc := MockClient{
		srv: &kmip.Server{
			Handler:&kmip.StandardProtocolHandler{
				ProtocolVersion:kmip.ProtocolVersion{
					ProtocolVersionMinor:1,
					ProtocolVersionMajor:4,
				},
				LogTraffic:true,
				MessageHandler:mock.NewMockServer(),
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

		t.Log("sending req:", string(reqXml))

		respXML, err := c.Do(reqXml)
		require.NoError(t, err)

		t.Log("got response:", string(respXML))
		var resp TTLV
		err = xml.Unmarshal(respXML, &resp)
		require.NoError(t, err)

		eq, vars, diff := Compare(&expectedResp, &resp)
		if !assert.True(t, eq) {
			t.Logf("vars: %#v", vars)
			t.Log("diff:", diff)
		}
	}
}
