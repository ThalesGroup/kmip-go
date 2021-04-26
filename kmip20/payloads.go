package kmip20

import "github.com/gemalto/kmip-go/ttlv"

type Attributes struct {
	Values ttlv.Values
}

type CreateRequestPayload struct {
	TTLVTag                struct{} `ttlv:"RequestPayload"`
	ObjectType             ObjectType
	Attributes             interface{}
	ProtectionStorageMasks ProtectionStorageMask `ttlv:",omitempty"`
}

type CreateResponsePayload struct {
	ObjectType       ObjectType
	UniqueIdentifier string
}

type CreateKeyPairRequestPayload struct {
	CommonAttributes              interface{}
	PrivateKeyAttributes          interface{}
	PublicKeyAttributes           interface{}
	CommonProtectionStorageMasks  ProtectionStorageMask `ttlv:",omitempty"`
	PrivateProtectionStorageMasks ProtectionStorageMask `ttlv:",omitempty"`
	PublicProtectionStorageMasks  ProtectionStorageMask `ttlv:",omitempty"`
}

type CreateKeyPairResponsePayload struct {
	PrivateKeyUniqueIdentifier string
	PublicKeyUniqueIdentifier  string
}
