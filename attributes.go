package kmip

// 3

// Cryptographic Parameters 3.6 Table 65
//
// The Cryptographic Parameters attribute is a structure (see Table 65) that contains a set of OPTIONAL
// fields that describe certain cryptographic parameters to be used when performing cryptographic operations
// using the object. Specific fields MAY pertain only to certain types of Managed Cryptographic Objects. The
// Cryptographic Parameters attribute of a Certificate object identifies the cryptographic parameters of the
// public key contained within the Certificate.
//
// The Cryptographic Algorithm is also used to specify the parameters for cryptographic operations. For operations
// involving digital signatures, either the Digital Signature Algorithm can be specified or the Cryptographic
// Algorithm and Hashing Algorithm combination can be specified.
//
// Random IV can be used to request that the KMIP server generate an appropriate IV for a
// cryptographic operation that uses an IV. The generated Random IV is returned in the response
// to the cryptographic operation.
//
// IV Length is the length of the Initialization Vector in bits. This parameter SHALL be provided when the
// specified Block Cipher Mode supports variable IV lengths such as CTR or GCM.
//
// Tag Length is the length of the authentication tag in bytes. This parameter SHALL be provided when the
// Block Cipher Mode is GCM or CCM.
//
// The IV used with counter modes of operation (e.g., CTR and GCM) cannot repeat for a given cryptographic key.
// To prevent an IV/key reuse, the IV is often constructed of three parts: a fixed field, an invocation field,
// and a counter as described in [SP800-38A] and [SP800-38D]. The Fixed Field Length is the length of the fixed
// field portion of the IV in bits. The Invocation Field Length is the length of the invocation field portion of
// the IV in bits. The Counter Length is the length of the counter portion of the IV in bits.
//
// Initial Counter Value is the starting counter value for CTR mode (for [RFC3686] it is 1).
type CryptographicParameters struct {
	BlockCipherMode           BlockCipherMode           `kmip:",omitempty"`
	PaddingMethod             PaddingMethod             `kmip:",omitempty"`
	HashingAlgorithm          HashingAlgorithm          `kmip:",omitempty"`
	KeyRoleType               KeyRoleType               `kmip:",omitempty"`
	DigitalSignatureAlgorithm DigitalSignatureAlgorithm `kmip:",omitempty"`
	CryptographicAlgorithm    CryptographicAlgorithm    `kmip:",omitempty"`
	RandomIV                  bool                      `kmip:",omitempty"`
	IVLength                  int                       `kmip:",omitempty"`
	TagLength                 int                       `kmip:",omitempty"`
	FixedFieldLength          int                       `kmip:",omitempty"`
	InvocationFieldLength     int                       `kmip:",omitempty"`
	CounterLength             int                       `kmip:",omitempty"`
	InitialCounterValue       int                       `kmip:",omitempty"`
	SaltLength                int                       `kmip:",omitempty"`
	MaskGenerator             MaskGenerator             `kmip:",omitempty" default:"1"` // defaults to MGF1
	MaskHashingAlgorithm      HashingAlgorithm          `kmip:",omitempty" default:"4"` // defaults to SHA-1
	PSource                   []byte                    `kmip:",omitempty"`
	TrailerField              int                       `kmip:",omitempty"`
}
