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
but not for machine parsing.  It can't be used as input.
		
The json and xml input/output formats are compliant with the KMIP spec, and
should be compatible with other KMIP tooling.
		
Example:
		
    echo "420069010000002042006a0200000004000000010000000042006b02000000040000000000000000" | ppkmip

Output (in 'text' format):
        
    ProtocolVersion (Structure/32):
      ProtocolVersionMajor (Integer/4): 1
      ProtocolVersionMinor (Integer/4): 0
        
hex format:
        
    420069010000002042006a0200000004000000010000000042006b02000000040000000000000000
        
prettyhex format:
        
    420069 | 01 | 00000020
      42006a | 02 | 00000004 | 0000000100000000
      42006b | 02 | 00000004 | 0000000000000000
        
json format:
        
    {
      "tag": "ProtocolVersion",
      "value": [
        {
          "tag": "ProtocolVersionMajor",
          "type": "Integer",
          "value": 1
        },
        {
          "tag": "ProtocolVersionMinor",
          "type": "Integer",
          "value": 0
        }
      ]
    }
        
xml format:
        
    <ProtocolVersion>
      <ProtocolVersionMajor type="Integer" value="1"></ProtocolVersionMajor>
      <ProtocolVersionMinor type="Integer" value="0"></ProtocolVersionMinor>
    </ProtocolVersion>
`
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), s)
		flag.PrintDefaults()
	}

	var inFormat string
	var outFormat string
	var inFile string
	flag.StringVar(&inFormat, "i", "", "input format: hex|json|xml, defaults to auto detect")
	flag.StringVar(&outFormat, "o", "", "output format: text|hex|prettyhex|json|xml, defaults to text")
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
