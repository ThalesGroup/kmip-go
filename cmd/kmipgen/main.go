package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/gemalto/kmip-go/internal/kmiputil"
	"go/format"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

// Specifications is the struct which the specifications JSON is unmarshaled into.
type Specifications struct {
	// Enums is a collection of enumeration specifications, describing the name of the
	// enumeration value set, each of the values in the set, and the tag(s) using
	// this enumeration set.
	Enums []EnumDef `json:"enums"`
	// Masks is a collection of mask specifications, describing the name
	// of the mask set, the values, and the tag(s) using it.
	Masks []EnumDef `json:"masks"`
	// Tags is a map of names to tag values.  The name should be
	// the full name, with spaces, from the spec.
	// The values may either be JSON numbers, or a JSON string
	// containing a hex encoded number, e.g. "0x42015E"
	Tags    map[string]interface{} `json:"tags"`
	Package string                 `json:"-"`
}

// EnumDef describes a single enum or mask value.
type EnumDef struct {
	// Name of the value.  Names should be the full name
	//	from the spec, including spaces.
	Name string `json:"name" validate:"required"`
	// Comment describing the value set.  Generator will add this to
	// the golang source code comment on type generated for this value set.
	Comment string `json:"comment"`
	// Values is a map of names to enum values.  Names should be the full name
	// from the spec, including spaces.
	// The values may either be JSON numbers, or a JSON string
	// containing a hex encoded number, e.g. "0x42015E"
	Values map[string]interface{} `json:"values"`
	// Tags is a list of tag names using this value set.  Names should be the full name
	//	// from the spec, including spaces.
	Tags []string `json:"tags"`
}

func main() {

	flag.Usage = func() {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Usage of kmipgen:")
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "")
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Generates go code which registers tags, enumeration values, and mask values with kmip-go.")
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Specifications are defined in a JSON file.")
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "")
		flag.PrintDefaults()
	}

	var specs Specifications

	var inputFilename string
	var outputFilename string
	var usage bool

	flag.StringVar(&inputFilename, "i", "", "Input `filename` of specifications.  Required.")
	flag.StringVar(&outputFilename, "o", "", "Output `filename`.  Defaults to standard out.")
	flag.StringVar(&specs.Package, "p", "ttlv", "Go `package` name in generated code.")
	flag.BoolVar(&usage, "h", false, "Show this usage message.")
	flag.Parse()

	if usage {
		flag.Usage()
		os.Exit(0)
	}

	if inputFilename == "" {
		fmt.Println("input file name cannot be empty")
		flag.Usage()
		os.Exit(1)
	}

	inputFile, err := os.Open(inputFilename)
	if err != nil {
		fmt.Println("error opening input file: ", err.Error())
		os.Exit(1)
	}
	defer inputFile.Close()

	err = json.NewDecoder(bufio.NewReader(inputFile)).Decode(&specs)
	if err != nil {
		fmt.Println("error reading input file: ", err.Error())
	}

	var outputWriter *os.File
	outputWriter = os.Stdout

	if outputFilename != "" {
		p, err := filepath.Abs(outputFilename)
		if err != nil {
			panic(err)
		}

		fmt.Println("writing to", p)

		f, err := os.Create(p)
		if err != nil {
			panic(err)
		}

		outputWriter = f

		defer func() {
			err := f.Sync()
			if err != nil {
				fmt.Println("error syncing file: ", err.Error())
			}
			err = f.Close()
			if err != nil {
				fmt.Println("error closing file: ", err.Error())
			}
		}()
	}

	src, err := genCode(&specs)
	if err != nil {
		fmt.Println("error generating code: ", err.Error())
	}

	_, err = outputWriter.WriteString(src)
	if err != nil {
		fmt.Println("error writing to output file", err.Error())
		os.Exit(1)
	}
}

type tagVal struct {
	FullName string
	Name     string
	Value    uint32
}

type enumVal struct {
	Name     string
	Comment  string
	Var      string
	TypeName string
	Vals     []tagVal
	Tags     []string
	BitMask  bool
}

type inputs struct {
	Tags        []tagVal
	Package     string
	Imports     []string
	TTLVPackage string
	Enums       []enumVal
	Masks       []enumVal
}

func parseUint32(v interface{}) (uint32, error) {
	switch n := v.(type) {
	case string:

		if strings.HasPrefix(n, "0x") {
			b, err := hex.DecodeString(n[2:])
			if err != nil {
				return 0, merry.Prepend(err, "invalid hex string")
			}
			if len(b) > 4 {
				return 0, merry.New("invalid hex string: must be max 4 bytes (8 hex characters)")
			}
			if len(b) < 4 {
				b = append(make([]byte, 4-len(b)), b...)
			}
			return binary.BigEndian.Uint32(b), nil
		}

		i, err := strconv.ParseUint(n, 10, 32)
		if err != nil {
			return 0, merry.Prependf(err, "invalid integer value (%v)", n)
		}

		return uint32(i), nil
	case float64:
		return uint32(n), nil
	default:
		return 0, merry.New("value must be a number, or a hex string, like 0x42015E")
	}
}

func prepareInput(s *Specifications) (*inputs, error) {
	in := inputs{
		Package: s.Package,
	}

	// prepare imports
	if s.Package != "ttlv" {
		in.Imports = append(in.Imports, "github.com/gemalto/kmip-go/ttlv")
		in.TTLVPackage = "ttlv."
	}

	// prepare tag inputs
	// normalize all the value names
	for key, value := range s.Tags {

		i, err := parseUint32(value)
		if err != nil {
			return nil, merry.Prependf(err, "invalid tag value (%v)", value)
		}

		val := tagVal{key, kmiputil.NormalizeName(key), i}
		in.Tags = append(in.Tags, val)
	}

	// sort tags by value
	sort.Slice(in.Tags, func(i, j int) bool {
		return in.Tags[i].Value < in.Tags[j].Value
	})

	toEnumVal := func(v EnumDef) (enumVal, error) {
		ev := enumVal{
			Name:     v.Name,
			Comment:  v.Comment,
			TypeName: kmiputil.NormalizeName(v.Name),
		}
		ev.Var = strings.ToLower(string([]rune(ev.TypeName)[:1]))

		// normalize all the value names
		for key, value := range v.Values {
			n := kmiputil.NormalizeName(key)

			i, err := parseUint32(value)
			if err != nil {
				return enumVal{}, merry.Prependf(err, "invalid tag value (%v)", value)
			}

			ev.Vals = append(ev.Vals, tagVal{key, n, i})
		}

		// sort the vals by value order
		sort.Slice(ev.Vals, func(i, j int) bool {
			return ev.Vals[i].Value < ev.Vals[j].Value
		})

		// normalize the tag names
		for _, t := range v.Tags {
			ev.Tags = append(ev.Tags, "Tag"+kmiputil.NormalizeName(t))
		}
		return ev, nil
	}

	// prepare enum and mask values
	for _, v := range s.Enums {
		ev, err := toEnumVal(v)
		if err != nil {
			return nil, merry.Prependf(err, "error parsing enum %v", v.Name)
		}
		in.Enums = append(in.Enums, ev)
	}

	for _, v := range s.Masks {
		ev, err := toEnumVal(v)
		if err != nil {
			return nil, merry.Prependf(err, "error parsing mask %v", v.Name)
		}
		ev.BitMask = true
		in.Masks = append(in.Masks, ev)
	}

	if len(s.Enums) > 0 || len(s.Masks) > 0 {
		in.Imports = append(in.Imports, "fmt")
	}

	if len(s.Masks) > 0 {
		in.Imports = append(in.Imports, "sort", "strings")
	}

	return &in, nil
}

func genCode(s *Specifications) (string, error) {

	buf := bytes.NewBuffer(nil)

	in, err := prepareInput(s)
	if err != nil {
		return "", err
	}

	tmpl := template.New("root")
	tmpl.Funcs(template.FuncMap{
		"ttlvPackage": func() string { return in.TTLVPackage },
	})
	template.Must(tmpl.Parse(global))
	template.Must(tmpl.New("tags").Parse(tags))
	template.Must(tmpl.New("base").Parse(baseTmpl))
	template.Must(tmpl.New("enumeration").Parse(enumerationTmpl))
	template.Must(tmpl.New("mask").Parse(maskTmpl))

	err = tmpl.Execute(buf, in)

	if err != nil {
		return "", merry.Prepend(err, "executing template")
	}

	// format returns the gofmt-ed contents of the Generator's buffer.
	src, err := format.Source(buf.Bytes())
	if err != nil {
		// Should never happen, but can arise when developing this code.
		// The user can compile the output to see the error.
		log.Printf("warning: internal error: invalid Go generated: %s", err)
		log.Printf("warning: compile the package to analyze the error")
		return buf.String(), nil
	}

	return string(src), nil
}

const global = `// Code generated by kmipgen; DO NOT EDIT.
package {{.Package}}

{{with .Imports}}
import (
{{range .}} "{{.}}"
{{end}})
{{end}}

{{with .Tags}}{{template "tags" .}}{{end}}

{{with .Enums}}{{range .}}{{template "enumeration" .}}{{end}}{{end}}

{{with .Masks}}{{range .}}{{template "mask" .}}{{end}}{{end}}
`

const tags = `
const (
{{range .}}	Tag{{.Name}} {{ttlvPackage}}Tag = {{.Value | printf "%#06x"}}
{{end}})

func init() { 
{{range .}} {{ttlvPackage}}RegisterTag({{ttlvPackage}}Tag({{.Value | printf "%#06x"}}), "{{.FullName}}")
{{end}}}
`

const baseTmpl = `
{{ $typeName := .TypeName }}

// {{.Comment}}
type {{.TypeName}} uint32

const ({{range .Vals}}
	{{$typeName}}{{.Name}} {{$typeName}} = {{.Value | printf "%#08x"}}{{end}}
)

var _{{.TypeName}}NameToValueMap = map[string]{{.TypeName}} { {{range .Vals}}
	"{{.Name}}": {{$typeName}}{{.Name}},{{end}}
}

var _{{.TypeName}}ValueToNameMap = map[{{.TypeName}}]string { {{range .Vals}}
	{{$typeName}}{{.Name}}: "{{.Name}}",{{end}}
}

func ({{.Var}} {{.TypeName}}) MarshalText() (text []byte, err error) {
	return []byte({{.Var}}.String()), nil
}

{{ if .Tags }}
{{ $bitMask := .BitMask }}
func init() { {{range .Tags}}
	{{ttlvPackage}}Register{{if $bitMask}}BitMask{{else}}Enum{{end}}({{.}}, {{ttlvPackage}}EnumTypeDef{
		Parse: func(s string) (uint32, bool) {
			v, ok := _{{$typeName}}NameToValueMap[s] 
			return uint32(v), ok
		},
		String: func(v uint32) string {
			return {{$typeName}}(v).String()		
		},
		Typed: func(v uint32) interface{} {
			return {{$typeName}}(v)
		},
	}){{end}}
}{{end}}
`

const enumerationTmpl = `
// {{.Name}} Enumeration
{{template "base" . }}

func ({{.Var}} {{.TypeName}}) MarshalTTLV(enc *{{ttlvPackage}}Encoder, tag {{ttlvPackage}}Tag) error {
	enc.EncodeEnumeration(tag, uint32({{.Var}}))
	return nil
}

func Register{{.TypeName}}({{.Var}} {{.TypeName}}, name string) {
	name = {{ttlvPackage}}NormalizeName(name)
	_{{.TypeName}}NameToValueMap[name] = {{.Var}}
	_{{.TypeName}}ValueToNameMap[{{.Var}}] = name
}

func ({{.Var}} {{.TypeName}}) String() string {
	if s, ok := _{{.TypeName}}ValueToNameMap[{{.Var}}]; ok {
		return s
	}
	return fmt.Sprintf("%#08x", uint32({{.Var}}))
}

`

const maskTmpl = `
// {{.Name}} Bit Mask
{{template "base" . }}

func Register{{.TypeName}}({{.Var}} {{.TypeName}}, name string) {
	name = {{ttlvPackage}}NormalizeName(name)
	_{{.TypeName}}NameToValueMap[name] = {{.Var}}
	_{{.TypeName}}ValueToNameMap[{{.Var}}] = name
	_{{.TypeName}}SortedValues = append(_{{.TypeName}}SortedValues, int({{.Var}}))
	sort.Ints(_{{.TypeName}}SortedValues)
}

var _{{.TypeName}}SortedValues []int

func init() {
	for {{.Var}} := range _{{.TypeName}}ValueToNameMap {
		_{{.TypeName}}SortedValues = append(_{{.TypeName}}SortedValues, int({{.Var}}))
		sort.Ints(_{{.TypeName}}SortedValues)
	}
}

func ({{.Var}} {{.TypeName}}) String() string {
	r := int({{.Var}})

	var sb strings.Builder
	var appending bool
	for _, v := range _{{.TypeName}}SortedValues {
		if v & r == v {
			if name :=_{{.TypeName}}ValueToNameMap[{{.TypeName}}(v)]; name != "" {
				if appending {
					sb.WriteString("|")
				} else {
					appending = true
				}
				sb.WriteString(name)
				r ^= v
			}

		}
		if r == 0 {
			break
		}
	}
	if r != 0 {
		if appending {
			sb.WriteString("|")
		}
		fmt.Fprintf(&sb, "%#08x", uint32(r))
	}
	return sb.String()
}`
