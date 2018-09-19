package compliance

import (
	"encoding/xml"
	"github.com/pmezard/go-difflib/difflib"
	"regexp"
	"strings"
)

type TTLV struct {
	XMLName  xml.Name
	Tag      string  `xml:"tag,omitempty,attr"`
	Type     string  `xml:"type,attr,omitempty"`
	Value    string  `xml:"value,attr,omitempty"`
	Children []*TTLV `xml:",any"`
}

func (v *TTLV) GetTag() string {
	if v.XMLName.Local != "" {
		return v.XMLName.Local
	}
	return v.Tag
}

func Compare(v1, v2 *TTLV) (eq bool, vars map[string]string, diff string) {
	vars = map[string]string{}
	eq = compare(v1, v2, vars)
	if !eq {
		diff = diffS(v1, v2, vars)
	}
	return
}

func compare(v1, v2 *TTLV, vars map[string]string) bool {
	switch {
	case v1 == nil && v2 == nil:
		return true
	case v1 == nil && v2 != nil:
		return false
	case v1 != nil && v2 == nil:
		return false
	case v1.GetTag() != v2.GetTag():
		return false
	case v1.Type != v2.Type:
		return false
	case v1.Value != v2.Value:
		if !isVariable(v1.Value) {
			return false
		}
		if varV, ok := vars[v1.Value]; ok {
			if varV != v2.Value {
				return false
			}
		} else {
			vars[v1.Value] = v2.Value
		}
	}
	if len(v1.Children) != len(v2.Children) {
		// TODO: this probably isn't quite right, since some attributes
		// are optional.  Not quite sure how to handle attributes yet
		return false
	}

	if len(v1.Children) == 0 {
		return true
	}

	// compare non-attribute children.  must be in same order
	var attrs, attrs2 []*TTLV
	for i, child := range v1.Children {
		child2 := v2.Children[i]
		if child.GetTag() == "Attribute" {
			if child2.GetTag() != "Attribute" {
				return false
			}
			attrs = append(attrs, child)
			attrs2 = append(attrs2, child2)
		} else {
			if !compare(child, child2, vars) {
				return false
			}
		}
	}

	// compare attrs, order independent
	attrMap := map[string]string{}
	attrMap2 := map[string]string{}

	mapize := func(m map[string]string, attrs []*TTLV) {
		for _, attr := range attrs {
			var name, value string
			idx := "0"
			for _, attrC := range attr.Children {
				switch attrC.GetTag() {
				case "AttributeName":
					name = attrC.Value
				case "AttributeValue":
					value = attrC.Value
				case "AttributeIndex":
					idx = attrC.Value
				}
			}
			m[name+idx] = value
		}
	}
	mapize(attrMap, attrs)
	mapize(attrMap2, attrs2)

	var keys []string
	for key := range attrMap {
		keys = append(keys, key)
	}

	for _, key := range keys {
		v1 := attrMap[key]
		v2 := attrMap2[key]
		if v1 != v2 {
			if !isVariable(v1) {
				return false
			}
			if varV, ok := vars[v1]; ok {
				if varV != v2 {
					return false
				}
			} else {
				vars[v1] = v2
			}

		}
	}

	return true
}

func diffS(v1, v2 *TTLV, vars map[string]string) string {
	xml1, err := xml.MarshalIndent(v1, "", "  ")
	if err != nil {
		panic(err)
	}
	s1 := string(xml1)
	xml2, err := xml.MarshalIndent(v2, "", "  ")
	if err != nil {
		panic(err)
	}
	s2 := string(xml2)
	for k, v := range vars {
		s1 = strings.Replace(s1, k, v, -1)
	}
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(s1),
		B:        difflib.SplitLines(s2),
		FromFile: "v1",
		FromDate: "",
		ToFile:   "v2",
		ToDate:   "",
		Context:  1,
	})
	return diff

}

func isVariable(s string) bool {
	return regexp.MustCompile(`\$[A-Z0-9_]+`).MatchString(s)
}
