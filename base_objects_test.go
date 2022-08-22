package kmip

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"testing"

	"github.com/gemalto/kmip-go/kmip14"
	"github.com/gemalto/kmip-go/ttlv"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clientConn returns a connection to the test kmip server.  Should be closed at end of test.
func clientConn(t *testing.T) *tls.Conn {
	t.Helper()

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

func TestCreateKey(t *testing.T) {
	conn := clientConn(t)
	defer conn.Close()

	biID := uuid.New()

	payload := CreateRequestPayload{
		ObjectType: kmip14.ObjectTypeSymmetricKey,
	}

	payload.TemplateAttribute.Append(kmip14.TagCryptographicAlgorithm, kmip14.CryptographicAlgorithmAES)
	payload.TemplateAttribute.Append(kmip14.TagCryptographicLength, 256)
	payload.TemplateAttribute.Append(kmip14.TagCryptographicUsageMask, kmip14.CryptographicUsageMaskEncrypt|kmip14.CryptographicUsageMaskDecrypt)
	payload.TemplateAttribute.Append(kmip14.TagName, Name{
		NameValue: "Key1",
		NameType:  kmip14.NameTypeUninterpretedTextString,
	})

	msg := RequestMessage{
		RequestHeader: RequestHeader{
			ProtocolVersion: ProtocolVersion{
				ProtocolVersionMajor: 1,
				ProtocolVersionMinor: 4,
			},
			BatchCount: 1,
		},
		BatchItem: []RequestBatchItem{
			{
				UniqueBatchItemID: biID[:],
				Operation:         kmip14.OperationCreate,
				RequestPayload:    &payload,
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
	assert.Equal(t, kmip14.OperationCreate, bi.Operation)
	assert.NotEmpty(t, bi.UniqueBatchItemID)
	assert.Equal(t, kmip14.ResultStatusSuccess, bi.ResultStatus)

	var respPayload CreateResponsePayload
	err = decoder.DecodeValue(&respPayload, bi.ResponsePayload.(ttlv.TTLV))
	require.NoError(t, err)

	assert.Equal(t, kmip14.ObjectTypeSymmetricKey, respPayload.ObjectType)
	assert.NotEmpty(t, respPayload.UniqueIdentifier)
}

func TestCreateKeyPair(t *testing.T) {
	conn := clientConn(t)
	defer conn.Close()

	biID := uuid.New()

	payload := CreateKeyPairRequestPayload{}
	payload.CommonTemplateAttribute = &TemplateAttribute{}
	payload.CommonTemplateAttribute.Append(kmip14.TagCryptographicAlgorithm, kmip14.CryptographicAlgorithmRSA)
	payload.CommonTemplateAttribute.Append(kmip14.TagCryptographicLength, 1024)
	payload.CommonTemplateAttribute.Append(kmip14.TagCryptographicUsageMask, kmip14.CryptographicUsageMaskSign|kmip14.CryptographicUsageMaskVerify)

	msg := RequestMessage{
		RequestHeader: RequestHeader{
			ProtocolVersion: ProtocolVersion{
				ProtocolVersionMajor: 1,
				ProtocolVersionMinor: 4,
			},
			BatchCount: 1,
		},
		BatchItem: []RequestBatchItem{
			{
				UniqueBatchItemID: biID[:],
				Operation:         kmip14.OperationCreateKeyPair,
				RequestPayload:    &payload,
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
	assert.Equal(t, kmip14.OperationCreateKeyPair, bi.Operation)
	assert.NotEmpty(t, bi.UniqueBatchItemID)
	assert.Equal(t, kmip14.ResultStatusSuccess, bi.ResultStatus)

	var respPayload CreateKeyPairResponsePayload
	err = decoder.DecodeValue(&respPayload, bi.ResponsePayload.(ttlv.TTLV))
	require.NoError(t, err)

	assert.NotEmpty(t, respPayload.PrivateKeyUniqueIdentifier)
	assert.NotEmpty(t, respPayload.PublicKeyUniqueIdentifier)
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

	var discVerRespPayload struct {
		ProtocolVersion ProtocolVersion
	}
	err = decoder.DecodeValue(&discVerRespPayload, bi.ResponsePayload.(ttlv.TTLV))
	require.NoError(t, err)
	assert.Equal(t, ProtocolVersion{
		ProtocolVersionMajor: 1,
		ProtocolVersionMinor: 2,
	}, discVerRespPayload.ProtocolVersion)
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

func TestGetResponsePayload_unmarshal(t *testing.T) {
	uniqueIdentifier := uuid.NewString()
	cryptographicLength := 256
	keyMaterial := RandomBytes(32)

	tests := []struct {
		name   string
		input  ttlv.Value
		expect GetResponsePayload
	}{
		{
			name: "SymmetricKey",

			input: s(kmip14.TagResponsePayload,
				v(kmip14.TagObjectType, kmip14.ObjectTypeSymmetricKey),
				v(kmip14.TagUniqueIdentifier, uniqueIdentifier),
				s(kmip14.TagSymmetricKey,
					s(kmip14.TagKeyBlock,
						v(kmip14.TagKeyFormatType, kmip14.KeyFormatTypeRaw),
						s(kmip14.TagKeyValue,
							v(kmip14.TagKeyMaterial, keyMaterial),
						),
						v(kmip14.TagCryptographicLength, cryptographicLength),
						v(kmip14.TagCryptographicAlgorithm, kmip14.CryptographicAlgorithmAES),
					),
				),
			),

			expect: GetResponsePayload{
				ObjectType:       kmip14.ObjectTypeSymmetricKey,
				UniqueIdentifier: uniqueIdentifier,
				SymmetricKey: &SymmetricKey{
					KeyBlock: KeyBlock{
						KeyFormatType: kmip14.KeyFormatTypeRaw,
						KeyValue: &KeyValue{
							KeyMaterial: keyMaterial,
						},
						CryptographicLength:    cryptographicLength,
						CryptographicAlgorithm: kmip14.CryptographicAlgorithmAES,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inputTTLV, err := ttlv.Marshal(test.input)
			require.NoError(t, err)

			var actualGRP GetResponsePayload
			err = ttlv.Unmarshal(inputTTLV, &actualGRP)
			require.NoError(t, err)

			require.Equal(t, test.expect, actualGRP)
		})
	}
}

func RandomBytes(numBytes int) []byte {
	randomBytes := make([]byte, numBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(err)
	}
	return randomBytes
}
