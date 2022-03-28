package kmip20

import (
	"testing"

	"github.com/gemalto/kmip-go/kmip14"
	"github.com/gemalto/kmip-go/ttlv"
	"github.com/stretchr/testify/require"
)

func TestCreateRequestPayload(t *testing.T) {
	type createReqAttrs struct {
		CryptographicAlgorithm CryptographicAlgorithm
		CryptographicLength    int
		CryptographicUsageMask kmip14.CryptographicUsageMask
	}

	tests := []struct {
		name     string
		in       CreateRequestPayload
		expected ttlv.Value
	}{
		{
			name: "structforattrs",
			in: CreateRequestPayload{
				ObjectType: ObjectTypeSymmetricKey,
				Attributes: createReqAttrs{
					CryptographicAlgorithm: CryptographicAlgorithmARIA,
					CryptographicLength:    56,
					CryptographicUsageMask: kmip14.CryptographicUsageMaskEncrypt | kmip14.CryptographicUsageMaskDecrypt,
				},
			},
			expected: s(kmip14.TagRequestPayload,
				v(kmip14.TagObjectType, ObjectTypeSymmetricKey),
				s(TagAttributes,
					v(kmip14.TagCryptographicAlgorithm, CryptographicAlgorithmARIA),
					v(kmip14.TagCryptographicLength, 56),
					v(kmip14.TagCryptographicUsageMask, kmip14.CryptographicUsageMaskEncrypt|kmip14.CryptographicUsageMaskDecrypt),
				),
			),
		},
		{
			name: "valuesforattrs",
			in: CreateRequestPayload{
				ObjectType: ObjectTypeSymmetricKey,
				Attributes: ttlv.NewStruct(ttlv.TagNone,
					v(kmip14.TagCryptographicAlgorithm, CryptographicAlgorithmARIA),
					v(kmip14.TagCryptographicLength, 56),
					v(kmip14.TagCryptographicUsageMask, kmip14.CryptographicUsageMaskEncrypt|kmip14.CryptographicUsageMaskDecrypt),
				),
			},
			expected: s(kmip14.TagRequestPayload,
				v(kmip14.TagObjectType, ObjectTypeSymmetricKey),
				s(TagAttributes,
					v(kmip14.TagCryptographicAlgorithm, CryptographicAlgorithmARIA),
					v(kmip14.TagCryptographicLength, 56),
					v(kmip14.TagCryptographicUsageMask, kmip14.CryptographicUsageMaskEncrypt|kmip14.CryptographicUsageMaskDecrypt),
				),
			),
		},
		{
			name: "attributesstruct",
			in: CreateRequestPayload{
				ObjectType: ObjectTypeSymmetricKey,
				Attributes: Attributes{
					Values: ttlv.Values{
						v(kmip14.TagCryptographicAlgorithm, CryptographicAlgorithmARIA),
						v(kmip14.TagCryptographicLength, 56),
						v(kmip14.TagCryptographicUsageMask, kmip14.CryptographicUsageMaskEncrypt|kmip14.CryptographicUsageMaskDecrypt),
					},
				},
			},
			expected: s(kmip14.TagRequestPayload,
				v(kmip14.TagObjectType, ObjectTypeSymmetricKey),
				s(TagAttributes,
					v(kmip14.TagCryptographicAlgorithm, CryptographicAlgorithmARIA),
					v(kmip14.TagCryptographicLength, 56),
					v(kmip14.TagCryptographicUsageMask, kmip14.CryptographicUsageMaskEncrypt|kmip14.CryptographicUsageMaskDecrypt),
				),
			),
		},
		{
			name: "omitempty",
			in: CreateRequestPayload{
				ObjectType: ObjectTypeCertificate,
			},
			expected: s(kmip14.TagRequestPayload,
				v(kmip14.TagObjectType, ObjectTypeCertificate),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			in := test.in

			out, err := ttlv.Marshal(&in)
			require.NoError(t, err)

			expected, err := ttlv.Marshal(test.expected)
			require.NoError(t, err)

			require.Equal(t, expected, out)

			// test roundtrip by unmarshaling back into an instance of the struct,
			// then marshaling again.  Should produce the same output.
			var p CreateRequestPayload
			err = ttlv.Unmarshal(expected, &p)
			require.NoError(t, err)

			out2, err := ttlv.Marshal(&p)
			require.NoError(t, err)

			require.Equal(t, out, out2)
		})
	}
}

func v(tag ttlv.Tag, val interface{}) ttlv.Value {
	return ttlv.NewValue(tag, val)
}

func s(tag ttlv.Tag, vals ...ttlv.Value) ttlv.Value {
	return ttlv.NewStruct(tag, vals...)
}
