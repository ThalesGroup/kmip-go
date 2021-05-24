package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	_ "github.com/gemalto/kmip-go/kmip14"
	_ "github.com/gemalto/kmip-go/kmip20"
	"github.com/gemalto/kmip-go/ttlv"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const FormatJSON = "json"
const FormatXML = "xml"
const FormatHex = "hex"

func main() {

	flag.Usage = func() {
		s := `ppkmip - kmip pretty printer

Usage:  ppkmip [options] [input] 

Pretty prints KMIP.  Can read KMIP in hex, json, or xml formats, 
and print it out in pretty-printed json, xml, text, raw hex, or
pretty printed hex.
		
The input argument should be a string.  If not present, input will
be read from standard in.

When reading hex input, any non-hex characters, such as whitespace or
embedded formatting characters, will be ignored.  The 'hexpretty'
output format embeds such characters, but because they are ignored,
'hexpretty' output is still valid 'hex' input.
		
The default output format is "text", which is optimized for human readability,
but not for machine parsing.  It can't be used as input.
		
The json and xml input/output formats are compliant with the KMIP spec, and
should be compatible with other KMIP tooling.
		
Examples:
		
    ppkmip 420069010000002042006a0200000004000000010000000042006b02000000040000000000000000
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
	} else if inArg := flag.Arg(0); inArg != "" {
		buf.WriteString(inArg)
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
			inFormat = FormatJSON
		case '<':
			inFormat = FormatXML
		default:
			inFormat = FormatHex
		}
	}

	outFormat = strings.ToLower(outFormat)
	if outFormat == "" {
		outFormat = "text"
	}

	var count int

	switch strings.ToLower(inFormat) {
	case FormatJSON:
		var raw ttlv.TTLV
		decoder := json.NewDecoder(buf)
		for {
			err := decoder.Decode(&raw)
			switch {
			case errors.Is(err, io.EOF):
				return
			case err == nil:
			default:
				fail("error parsing JSON", err)
			}
			printTTLV(outFormat, raw, count)
			count++
		}

	case FormatXML:
		var raw ttlv.TTLV
		decoder := xml.NewDecoder(buf)
		for {
			err := decoder.Decode(&raw)
			switch {
			case errors.Is(err, io.EOF):
				return
			case err == nil:
			default:
				fail("error parsing XML", err)
			}
			printTTLV(outFormat, raw, count)
			count++
		}
	case FormatHex:
		raw := ttlv.TTLV(ttlv.Hex2bytes(buf.String()))
		for len(raw) > 0 {
			printTTLV(outFormat, raw, count)
			count++
			raw = raw.Next()
		}
	default:
		fail("invalid input format: "+inFormat, nil)
	}
}

func printTTLV(outFormat string, raw ttlv.TTLV, count int) {
	if count > 0 {
		fmt.Println("")
	}
	switch outFormat {
	case "text":
		if err := ttlv.Print(os.Stdout, "", "  ", raw); err != nil {
			fail("error printing", err)
		}
	case "json":
		s, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			fail("error printing JSON", err)
		}
		fmt.Print(string(s))
	case "xml":
		s, err := xml.MarshalIndent(raw, "", "  ")
		if err != nil {
			fail("error printing XML", err)
		}
		fmt.Print(string(s))
	case "hex":
		fmt.Print(hex.EncodeToString(raw))
	case "prettyhex":
		if err := ttlv.PrintPrettyHex(os.Stdout, "", "  ", raw); err != nil {
			fail("error printing", err)
		}
	}
}

func fail(msg string, err error) {
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, msg+":", err)
	} else {
		_, _ = fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}
