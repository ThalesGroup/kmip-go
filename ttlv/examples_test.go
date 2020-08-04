package ttlv

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
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
