//go:generate kmipgen -o kmip_2_0_additions_generated.go -i kmip_2_0_additions.json -p kmip20

package kmip20

import (
	"bytes"
	"encoding/xml"
	"github.com/gemalto/kmip-go/ttlv"
	"io"
	"io/ioutil"
	"strings"
)

type value struct {
	Any ttlv.TTLV `xml:",any"`
}

type attrs struct {
	Any []ttlv.TTLV `xml:",any"`
}

type kmipBlock struct {
	Name  xml.Name `xml:"KMIP"`
	Attrs attrs    `xml:"attr"`
	Value value    `xml:"value"`
}

// nolint:dupl,gochecknoinits
func init() {
	// register new 2.0 values
	// KMIP 2.0 introduces a tag named "Attribute Reference", whose value is the enumeration of all Tags
	ttlv.RegisterEnum(ttlv.Tag(0x42013B), ttlv.EnumTypeDef{
		Parse: func(s string) (u uint32, b bool) {
			tag, e := ttlv.ParseTag(s)
			return uint32(tag), e == nil
		},
		String: func(v uint32) string {
			s := ttlv.Tag(v).String()
			// if s is a hex string, it will only be 6 characters (3 bytes)
			// because that's how long the hex representation of tags is supposed
			// to be if they are used in fields of JSON/XML which are designated to
			// hold tags.
			// But in 2.0, if the value is being written in a field which holds
			// integers/enum values, then the hex string should have 8 characters
			if strings.HasPrefix(s, "0x") && len(s) > 10 {
				return "0x" + strings.Repeat("0", 10-len(s)) + s[2:]
			}
			return s
		},
		Typed: func(v uint32) interface{} {
			return ttlv.Tag(v)
		},
	})

	// KMIP 2.0 has made the value of the Extension Type tag an enumeration of all type values
	ttlv.RegisterEnum(ttlv.TagExtensionType, ttlv.EnumTypeDef{
		Parse: func(s string) (u uint32, b bool) {
			typ, err := ttlv.ParseType(s)
			return uint32(typ), err == nil
		},
		String: func(v uint32) string {
			s := ttlv.Type(v).String()
			// if s is a hex string, it will only be 6 characters (3 bytes)
			// because that's how long the hex representation of tags is supposed
			// to be if they are used in fields of JSON/XML which are designated to
			// hold tags.
			// But in 2.0, if the value is being written in a field which holds
			// integers/enum values, then the hex string should have 8 characters
			if strings.HasPrefix(s, "0x") && len(s) > 10 {
				return "0x" + strings.Repeat("0", 10-len(s)) + s[2:]
			}
			return s
		},
		Typed: func(v uint32) interface{} {
			return ttlv.Type(v)
		},
	})

	// Register all the new values for existing enums
	ttlv.RegisterCredentialType(ttlv.CredentialType(0x00000004), "One Time Password")
	ttlv.RegisterCredentialType(ttlv.CredentialType(0x00000005), "Hashed Password")
	ttlv.RegisterCredentialType(ttlv.CredentialType(0x00000006), "Ticket")

	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x00000029), "ARIA")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x0000002A), "SEED")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x0000002B), "SM2")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x0000002C), "SM3")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x0000002D), "SM4")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x0000002E), "GOST R 34.10-2012")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x0000002F), "GOST R 34.11-2012")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x00000030), "GOST R 34.13-2015")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x00000031), "GOST 28147-89")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x00000032), "XMSS")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x00000033), "SPHINCS-256")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x00000034), "McEliece")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x00000035), "McEliece-6960119")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x00000036), "McEliece-8192128")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x00000037), "Ed25519")
	ttlv.RegisterCryptographicAlgorithm(ttlv.CryptographicAlgorithm(0x00000038), "Ed448")

	ttlv.RegisterDerivationMethod(ttlv.DerivationMethod(0x00000009), "AWS Signature Version 4")
	ttlv.RegisterDerivationMethod(ttlv.DerivationMethod(0x0000000A), "HKDF")

	ttlv.RegisterLinkType(ttlv.LinkType(0x0000010E), "Wrapping Key Link")

	ttlv.RegisterObjectType(ttlv.ObjectType(0x0000000A), "Certificate Request")

	ttlv.RegisterOperation(ttlv.Operation(0x0000002C), "Log")
	ttlv.RegisterOperation(ttlv.Operation(0x0000002D), "Login")
	ttlv.RegisterOperation(ttlv.Operation(0x0000002E), "Logout")
	ttlv.RegisterOperation(ttlv.Operation(0x0000002F), "Delegated Login")
	ttlv.RegisterOperation(ttlv.Operation(0x00000030), "Adjust Attribute")
	ttlv.RegisterOperation(ttlv.Operation(0x00000031), "Set Attribute")
	ttlv.RegisterOperation(ttlv.Operation(0x00000032), "Set Endpoint Role")
	ttlv.RegisterOperation(ttlv.Operation(0x00000033), "PKCS#11")
	ttlv.RegisterOperation(ttlv.Operation(0x00000034), "Interop")
	ttlv.RegisterOperation(ttlv.Operation(0x00000035), "Re-Provision")

	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000104), "Complete Server Basic")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000105), "Complete Server TLS v1.2")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000106), "Tape Library Client")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000107), "Tape Library Server")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000108), "Symmetric Key Lifecycle Client")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000109), "Symmetric Key Lifecycle Server")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000010A), "Asymmetric Key Lifecycle Client")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000010B), "Asymmetric Key Lifecycle Server")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000010C), "Basic Cryptographic Client")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000010D), "Basic Cryptographic Server")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000010E), "Advanced Cryptographic Client")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000010F), "Advanced Cryptographic Server")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000110), "RNG Cryptographic Client ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000111), "RNG Cryptographic Server ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000112), "Basic Symmetric Key Foundry Client ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000113), "Intermediate Symmetric Key Foundry Client ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000114), "Advanced Symmetric Key Foundry Client ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000115), "Symmetric Key Foundry Server")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000116), "Opaque Managed Object Store Client ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000117), "Opaque Managed Object Store Server ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000118), "Suite B minLOS_128 Client ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000119), "Suite B minLOS_128 Server ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000011A), "Suite B minLOS_192 Client ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000011B), "Suite B minLOS_192 Server ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000011C), "Storage Array with Self Encrypting Drive Client")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000011D), "Storage Array with Self Encrypting Drive Server")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000011E), "HTTPS Client ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000011F), "HTTPS Server ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000120), "JSON Client ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000121), "JSON Server ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000122), "XML Client ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000123), "XML Server ")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000124), "AES XTS Client")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000125), "AES XTS Server")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000126), "Quantum Safe Client")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000127), "Quantum Safe Server")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000128), "PKCS#11 Client")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x00000129), "PKCS#11 Server")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000012A), "Baseline Client")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000012B), "Baseline Server")
	ttlv.RegisterProfileName(ttlv.ProfileName(0x0000012C), "Complete Server")

	ttlv.RegisterQueryFunction(ttlv.QueryFunction(0x0000000D), "Query Defaults Information")
	ttlv.RegisterQueryFunction(ttlv.QueryFunction(0x0000000E), "Query Storage Protection Masks")

	ttlv.RegisterRecommendedCurve(ttlv.RecommendedCurve(0x00000045), "CURVE25519")
	ttlv.RegisterRecommendedCurve(ttlv.RecommendedCurve(0x00000046), "CURVE448")

	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000019), "Invalid Ticket")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000001A), "Usage Limit Exceeded")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000001B), "Numeric Range")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000001C), "Invalid Data Type")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000001D), "Read Only Attribute")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000001E), "Multi Valued Attribute")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000001F), "Unsupported Attribute")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000020), "Attribute Instance Not Found")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000021), "Attribute Not Found")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000022), "Attribute Read Only")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000023), "Attribute Single Valued")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000024), "Bad Cryptographic Parameters")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000025), "Bad Password")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000026), "Codec Error")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000028), "Illegal Object Type")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000029), "Incompatible Cryptographic Usage Mask")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000002A), "Internal Server Error")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000002B), "Invalid Asynchronous Correlation Value")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000002C), "Invalid Attribute")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000002D), "Invalid Attribute Value")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000002E), "Invalid Correlation Value")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000002F), "Invalid CSR")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000030), "Invalid Object Type")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000032), "Key Wrap Type Not Supported")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000034), "Missing Initialization Vector")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000035), "Non Unique Name Attribute")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000036), "Object Destroyed")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000037), "Object Not Found")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000039), "Not Authorized")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000003A), "Server Limit Exceeded")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000003B), "Unknown Enumeration")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000003C), "Unknown Message Extension")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000003D), "Unknown Tag")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000003E), "Unsupported Cryptographic Parameters")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x0000003F), "Unsupported Protocol Version")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000040), "Wrapping Object Archived")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000041), "Wrapping Object Destroyed")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000042), "Wrapping Object Not Found")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000043), "Wrong Key Lifecycle State")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000044), "Protection Storage Unavailable")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000045), "PKCS#11 Codec Error ")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000046), "PKCS#11 Invalid Function ")
	ttlv.RegisterResultReason(ttlv.ResultReason(0x00000047), "PKCS#11 Invalid Interface")

}

func Parse() error {

	b, err := ioutil.ReadFile("objects-kmip20.xml")
	if err != nil {
		return err
	}

	decoder := xml.NewDecoder(bytes.NewReader(b))

	for {
		var block kmipBlock
		err = decoder.Decode(&block)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		//fmt.Println("Value:")
		//fmt.Println(block.Value.Any.String())
		//fmt.Println()
		//fmt.Printf("Attrs (len %d):\n", len(block.Attrs.Any))
		//for _, v := range block.Attrs.Any {
		//	fmt.Println(v.String())
		//}
	}

	return nil
}
