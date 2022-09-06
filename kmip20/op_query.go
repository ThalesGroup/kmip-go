package kmip20

import (
	"context"

	"github.com/gemalto/kmip-go"
	"github.com/gemalto/kmip-go/kmip14"
)

// 7.3 Capability Information
// The Capability Information base object is a structure that contains details of the supported capabilities.
type CapabilityInformation struct {
	StreamingCapability     bool                      // Required: No
	AsynchronousCapability  bool                      // Required: No
	AttestationCapability   bool                      // Required: No
	BatchUndoCapability     bool                      // Required: No
	BatchContinueCapability bool                      // Required: No
	UnwrapMode              kmip14.UnwrapMode         // Required: No
	DestroyAction           kmip14.DestroyAction      // Required: No
	ShreddingAlgorithm      kmip14.ShreddingAlgorithm // Required: No
	RNGMode                 kmip14.RNGMode            // Required: No
	QuantumSafeCapability   bool                      // Required: No
}

// 7.7 Defaults Information
// The Defaults Information is a structure used in Query responses for values that servers will use if clients omit them from factory
// operations requests.
type DefaultsInformation struct {
	ObjectDefaults ObjectDefaults // Required: Yes
}

// 7.9 Extension Information
// An Extension Information object is a structure describing Objects with Item Tag values in the Extensions range. The Extension Name
// is a Text String that is used to name the Object. The Extension Tag is the Item Tag Value of the Object. The Extension Type is
// the Item Type Value of the Object.
type ExtensionInformation struct {
	ExtensionName               string // Required: Yes
	ExtensionTag                int    // Required: No
	ExtensionType               int    // Required: No
	ExtensionEnumeration        int    // Required: No
	ExtensionAttribute          bool   // Required: No
	ExtensionParentStructureTag int    // Required: No
	ExtensionDescription        string // Required: No
}

// 7.18 Object Defaults
// The Object Defaults is a structure that details the values that the server will use if the client omits them on factory methods for
// objects. The structure list the Attributes and  their values by Object Type enumeration.
type ObjectDefaults struct {
	ObjectType kmip14.ObjectType // Required: Yes
	Attributes kmip.Attributes   // Required: Yes
}

// 7.30 RNG Parameters
// The RNG Parameters base object is a structure that contains a mandatory RNG Algorithm and a set of OPTIONAL fields that describe a
// Random Number Generator. Specific fields pertain only to certain types of RNGs. The RNG Algorithm SHALL be specified and if the
// algorithm implemented is unknown or the implementation does not want to provide the specific details of the RNG Algorithm then the
// Unspecified enumeration SHALL be used. If the cryptographic building blocks used within the RNG are known they MAY be specified in
// combination of the remaining fields within the RNG Parameters structure.
type RNGParameters struct {
	RNGAlgorithm           kmip14.RNGAlgorithm           // Required: Yes
	CryptographicAlgorithm kmip14.CryptographicAlgorithm // Required: No
	CryptographicLength    int                           // Required: No
	HashingAlgorithm       kmip14.HashingAlgorithm       // Required: No
	DRBGAlgorithm          kmip14.DRBGAlgorithm          // Required: No
	RecommendedCurve       kmip14.RecommendedCurve       // Required: No
	FIPS186Variation       kmip14.FIPS186Variation       // Required: No
	PredictionResistance   bool                          // Required: No
}

// 7.31 Server Information
// The Server Information  base object is a structure that contains a set of OPTIONAL fields that describe server information.
// Where a server supports returning information in a vendor-specific field for which there is an equivalent field within the structure,
// the server SHALL provide the standardized version of the field.
type ServerInformation struct {
	ServerName                   string   // Required: No
	ServerSerialNumber           string   // Required: No
	ServerVersion                string   // Required: No
	ServerLoad                   string   // Required: No
	ProductName                  string   // Required: No
	BuildLevel                   string   // Required: No
	BuildDate                    string   // Required: No
	ClusterInfo                  string   // Required: No
	AlternativeFailoverEndpoints []string // Required: No
	VendorSpecific               []string // Required: No
}

// 6.1.37 Query

// Table 259

type QueryRequestPayload struct {
	QueryFunction QueryFunction
}

// Table 260

type QueryResponsePayload struct {
	Operation                []kmip14.Operation
	ObjectType               []ObjectType
	VendorIdentification     string
	ServerInformation        []ServerInformation
	ApplicationNamespace     []string
	ExtensionInformation     []ExtensionInformation
	AttestationType          kmip14.AttestationType
	RNGParameters            []RNGParameters
	ProfileInformation       []ProfileName
	ValidationInformation    []kmip14.ValidationAuthorityType
	CapabilityInformation    []CapabilityInformation
	ClientRegistrationMethod kmip14.ClientRegistrationMethod
	DefaultsInformation      *DefaultsInformation
	ProtectionStorageMasks   []ProtectionStorageMask
}

type QueryHandler struct {
	Query func(ctx context.Context, payload *QueryRequestPayload) (*QueryResponsePayload, error)
}

func (h *QueryHandler) HandleItem(ctx context.Context, req *kmip.Request) (*kmip.ResponseBatchItem, error) {
	var payload QueryRequestPayload

	err := req.DecodePayload(&payload)
	if err != nil {
		return nil, err
	}

	respPayload, err := h.Query(ctx, &payload)
	if err != nil {
		return nil, err
	}

	return &kmip.ResponseBatchItem{
		ResponsePayload: respPayload,
	}, nil
}
