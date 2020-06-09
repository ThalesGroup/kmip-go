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

	var inFormat string
	var outFormat string
	var inFile string
	flag.StringVar(&inFormat, "i", "", "input format: hex|json|xml, defaults to auto detect")
	flag.StringVar(&outFormat, "o", "", "output format: text|hex|json|xml, defaults to text")
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
		if err := ttlv.Print(os.Stdout, "", raw); err != nil {
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
