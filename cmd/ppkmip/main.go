package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/gemalto/kmip-go/ttlv"
	"io/ioutil"
	"os"
	"strings"
)

func main() {

	flag.Usage = func() {
		s := `Usage of ppkmip:

Pretty prints KMIP.  Can read KMIP in hex, json, or xml formats, 
and print it out in pretty-printed json, xml, text, raw hex, or
pretty printed hex.

When reading hex input, any non-hex characters, such as whitespace or
embedded formatting characters, will be ignored.  The 'hexpretty'
output format embeds such characters, but because they are ignored,
'hexpretty' output is still valid 'hex' input.
		
The default output format is "text", which is optimized for human readability,
but not for machine parsing.
		
The json and xml input/output formats are compliant with the KMIP spec, and
should be compatible with other KMIP tooling.
		
Example:
		
    echo '420078 | 01 | 00000118 
      420077 | 01 | 00000048 
          420069 | 01 | 00000020 
              42006A | 02 | 00000004 | 0000000100000000
              42006B | 02 | 00000004 | 0000000000000000
          420010 | 06 | 00000008 | 0000000000000001
          42000D | 02 | 00000004 | 0000000200000000
      42000F | 01 | 00000068
          42005C | 05 | 00000004 | 0000000800000000
          420093 | 08 | 00000001 | 3600000000000000
          4200790100000040420008010000003842000A07000000044E616D650000000042000B010000002042005507000000067075626B657900004200540500000004000000010000000042000F010000005042005C05000000040000000E00000000420093080000000137000000000000004200790100000028420008010000002042000A0700000008782D6D796174747242000B07000000057465737432000000' | ppkmip

Output:
		
	ProtocolVersionMajor (Integer/4): 1
`
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), s)
		flag.PrintDefaults()
	}

	var inFormat string
	var outFormat string
	var inFile string
	flag.StringVar(&inFormat, "i", "", "input format: hex|json|xml, defaults to auto detect")
	flag.StringVar(&outFormat, "o", "", "output format: text|hex|prettyhex|json|xml, defaults to text, which is a human-readable but not parseable format")
	flag.StringVar(&inFile, "f", "", "input file name, defaults to stdin")

	flag.Parse()

	buf := bytes.NewBuffer(nil)

	if inFile != "" {
		file, err := ioutil.ReadFile(inFile)
		if err != nil {
			fail("error reading input file", err)
		}
		buf = bytes.NewBuffer(file)
	} else {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			buf.Write(scanner.Bytes())
		}

		if err := scanner.Err(); err != nil {
			fail("error reading standard input", err)
		}
	}

	if inFormat == "" {
		// auto detect input format
		switch buf.Bytes()[0] {
		case '[', '{':
			inFormat = "json"
		case '<':
			inFormat = "xml"
		default:
			inFormat = "hex"
		}
	}

	var raw ttlv.TTLV

	switch strings.ToLower(inFormat) {
	case "json":
		err := json.Unmarshal(buf.Bytes(), &raw)
		if err != nil {
			fail("error parsing JSON", err)
		}

	case "xml":
		err := xml.Unmarshal(buf.Bytes(), &raw)
		if err != nil {
			fail("error parsing XML", err)
		}
	case "hex":
		raw = ttlv.Hex2bytes(buf.String())
	default:
		fail("invalid input format: "+inFormat, nil)
	}

	if outFormat == "" {
		outFormat = "text"
	}

	switch strings.ToLower(outFormat) {
	case "text":
		if err := ttlv.Print(os.Stdout, "", "  ", raw); err != nil {
			fail("error printing", err)
		}
	case "json":
		s, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			fail("error printing JSON", err)
		}
		fmt.Println(string(s))
	case "xml":
		s, err := xml.MarshalIndent(raw, "", "  ")
		if err != nil {
			fail("error printing XML", err)
		}
		fmt.Println(string(s))
	case "hex":
		fmt.Println(hex.EncodeToString(raw))
	case "prettyhex":
		if err := ttlv.PrintPrettyHex(os.Stdout, "", "  ", raw); err != nil {
			fail("error printing", err)
		}
	}

}

func fail(msg string, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, msg+":", err)
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}
