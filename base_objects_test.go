package kmip

import (
	"bufio"
	"crypto/tls"
	"github.com/gemalto/kmip-go/kmip14"
	"github.com/gemalto/kmip-go/ttlv"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// clientConn returns a connection to the test kmip server.  Should be closed at end of test.
func clientConn(t *testing.T) *tls.Conn {
	cert, err := tls.LoadX509KeyPair("./pykmip-server/server.cert", "./pykmip-server/server.key")
	require.NoError(t, err)

	// the containerized pykmip we're using requires a very specific cipher suite, which isn't
	// enabled by go by default.
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		},
	}

	conn, err := tls.Dial("tcp", "127.0.0.1:5696", tlsConfig)
	require.NoError(t, err)

	return conn
}

func TestRequest(t *testing.T) {
	conn := clientConn(t)
	defer conn.Close()

	biID := uuid.New()

	msg := RequestMessage{
		RequestHeader: RequestHeader{
			ProtocolVersion: ProtocolVersion{
				ProtocolVersionMajor: 1,
				ProtocolVersionMinor: 2,
			},
			BatchCount: 1,
		},
		BatchItem: []RequestBatchItem{
			{
				UniqueBatchItemID: biID[:],
				Operation:         kmip14.OperationDiscoverVersions,
				RequestPayload: DiscoverVersionsRequestPayload{
					ProtocolVersion: []ProtocolVersion{
						{ProtocolVersionMajor: 1, ProtocolVersionMinor: 2},
					},
				},
			},
		},
	}

	req, err := ttlv.Marshal(msg)
	require.NoError(t, err)

	t.Log(req)

	_, err = conn.Write(req)
	require.NoError(t, err)

	decoder := ttlv.NewDecoder(bufio.NewReader(conn))
	resp, err := decoder.NextTTLV()
	require.NoError(t, err)

	t.Log(resp)

	var respMsg ResponseMessage
	err = decoder.DecodeValue(&respMsg, resp)
	require.NoError(t, err)

	assert.Equal(t, 1, respMsg.ResponseHeader.BatchCount)
	assert.Len(t, respMsg.BatchItem, 1)
	bi := respMsg.BatchItem[0]
	assert.Equal(t, kmip14.OperationDiscoverVersions, bi.Operation)
	assert.NotEmpty(t, bi.UniqueBatchItemID)
	assert.Equal(t, kmip14.ResultStatusSuccess, bi.ResultStatus)

	var protVer ProtocolVersion
	err = decoder.DecodeValue(&protVer, bi.ResponsePayload.(ttlv.TTLV))
	require.NoError(t, err)
	assert.Equal(t, ProtocolVersion{
		ProtocolVersionMajor: 1,
		ProtocolVersionMinor: 2,
	}, protVer)
}

func TestTemplateAttribute_marshal(t *testing.T) {
	tests := []struct {
		name     string
		in       TemplateAttribute
		inF      func() TemplateAttribute
		expected ttlv.Value
	}{
		{
			name: "basic",
			in: TemplateAttribute{
				Name: []Name{
					{
						NameValue: "first",
						NameType:  kmip14.NameTypeUninterpretedTextString,
					},
					{
						NameValue: "this is a uri",
						NameType:  kmip14.NameTypeURI,
					},
				},
				Attribute: []Attribute{
					{
						AttributeName:  kmip14.TagAlwaysSensitive.CanonicalName(),
						AttributeIndex: 5,
						AttributeValue: true,
					},
				},
			},
			expected: s(kmip14.TagTemplateAttribute,
				s(kmip14.TagName,
					v(kmip14.TagNameValue, "first"),
					v(kmip14.TagNameType, kmip14.NameTypeUninterpretedTextString),
				),
				s(kmip14.TagName,
					v(kmip14.TagNameValue, "this is a uri"),
					v(kmip14.TagNameType, kmip14.NameTypeURI),
				),
				s(kmip14.TagAttribute,
					v(kmip14.TagAttributeName, kmip14.TagAlwaysSensitive.CanonicalName()),
					v(kmip14.TagAttributeIndex, 5),
					v(kmip14.TagAttributeValue, true),
				),
			),
		},
		{
			name: "noname",
			in: TemplateAttribute{Attribute: []Attribute{
				{
					AttributeName:  kmip14.TagAlwaysSensitive.CanonicalName(),
					AttributeIndex: 5,
					AttributeValue: true,
				},
			}},
			expected: s(kmip14.TagTemplateAttribute,
				s(kmip14.TagAttribute,
					v(kmip14.TagAttributeName, kmip14.TagAlwaysSensitive.CanonicalName()),
					v(kmip14.TagAttributeIndex, 5),
					v(kmip14.TagAttributeValue, true),
				),
			),
		},
		{
			name: "noattribute",
			in: TemplateAttribute{
				Name: []Name{
					{
						NameValue: "first",
						NameType:  kmip14.NameTypeUninterpretedTextString,
					},
				},
			},
			expected: s(kmip14.TagTemplateAttribute,
				s(kmip14.TagName,
					v(kmip14.TagNameValue, "first"),
					v(kmip14.TagNameType, kmip14.NameTypeUninterpretedTextString),
				),
			),
		},
		{
			name: "omitzeroindex",
			in: TemplateAttribute{
				Attribute: []Attribute{
					{
						AttributeName:  kmip14.TagAlwaysSensitive.CanonicalName(),
						AttributeValue: true,
					},
				},
			},
			expected: s(kmip14.TagTemplateAttribute,
				s(kmip14.TagAttribute,
					v(kmip14.TagAttributeName, kmip14.TagAlwaysSensitive.CanonicalName()),
					v(kmip14.TagAttributeValue, true),
				),
			),
		},
		{
			name: "use canonical names",
			inF: func() TemplateAttribute {
				var ta TemplateAttribute
				ta.Append(kmip14.TagCryptographicAlgorithm, ttlv.EnumValue(kmip14.CryptographicAlgorithmBlowfish))
				return ta
			},
			expected: s(kmip14.TagTemplateAttribute,
				s(kmip14.TagAttribute,
					v(kmip14.TagAttributeName, "Cryptographic Algorithm"),
					v(kmip14.TagAttributeValue, ttlv.EnumValue(kmip14.CryptographicAlgorithmBlowfish)),
				),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			in := test.in
			if test.inF != nil {
				in = test.inF()
			}

			out, err := ttlv.Marshal(&in)
			require.NoError(t, err)

			expected, err := ttlv.Marshal(test.expected)
			require.NoError(t, err)

			require.Equal(t, out, expected)

			var ta TemplateAttribute
			err = ttlv.Unmarshal(expected, &ta)
			require.NoError(t, err)

			require.Equal(t, in, ta)
		})
	}
}

func v(tag ttlv.Tag, val interface{}) ttlv.Value {
	return ttlv.NewValue(tag, val)
}

func s(tag ttlv.Tag, vals ...ttlv.Value) ttlv.Value {
	return ttlv.NewStruct(tag, vals...)
}
