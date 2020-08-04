package ttlv

import (
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
)

func Example_json() {
	input := `{"tag":"KeyFormatType","type":"Enumeration","value":"X_509"}`
	var output TTLV

	_ = json.Unmarshal([]byte(input), &output)
	fmt.Println(output)

	b, _ := json.Marshal(output)
	fmt.Println(string(b))

	// Output:
	// KeyFormatType (Enumeration/4): X_509
	// {"tag":"KeyFormatType","type":"Enumeration","value":"X_509"}
}

func Example_xml() {
	input := `<Operation type="Enumeration" value="Activate"></Operation>`
	var output TTLV

	_ = xml.Unmarshal([]byte(input), &output)
	fmt.Println(output)

	b, _ := xml.Marshal(output)
	fmt.Println(string(b))

	// Output:
	// Operation (Enumeration/4): Activate
	// <Operation type="Enumeration" value="Activate"></Operation>
}

func ExamplePrintPrettyHex() {
	b, _ := hex.DecodeString("420069010000002042006a0200000004000000010000000042006b02000000040000000000000000")
	_ = PrintPrettyHex(os.Stdout, "", "  ", b)

	// Output:
	// 420069 | 01 | 00000020
	//   42006a | 02 | 00000004 | 0000000100000000
	//   42006b | 02 | 00000004 | 0000000000000000
}

func ExamplePrint() {
	b, _ := hex.DecodeString("420069010000002042006a0200000004000000010000000042006b02000000040000000000000000")
	_ = Print(os.Stdout, "", "  ", b)

	// Output:
	// ProtocolVersion (Structure/32):
	//   ProtocolVersionMajor (Integer/4): 1
	//   ProtocolVersionMinor (Integer/4): 0
}
