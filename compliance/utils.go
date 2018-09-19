package compliance

import (
	"encoding/xml"
	"regexp"
)

type TTLV struct {
	XMLName   xml.Name
	Tag       string  `xml:"tag,omitempty,attr"`
	Type      string  `xml:"type,attr,omitempty"`
	Value     string  `xml:"value,attr,omitempty"`
	Children  []*TTLV `xml:",any"`
	variables map[string]string
}

func (v *TTLV) GetTag() string {
	if v.XMLName.Local != "" {
		return v.XMLName.Local
	}
	return v.Tag
}

func (v *TTLV) Cmp(v2 *TTLV) bool {
	switch {
	case v == nil && v2 == nil:
		return true
	case v == nil && v2 != nil:
		return false
	case v != nil && v2 == nil:
		return false
	case v.Tag != v2.Tag:
		return false
	case v.Type != v2.Type:
		return false
	case v.Value != v2.Value:
		if !isVariable(v.Value) {
			return false
		}
		if v.variables == nil {
			v.variables = map[string]string{}
		}
		if varV, ok := v.variables[v.Value]; ok {
			if varV != v2.Value {
				return false
			}
		} else {
			v.variables[v.Value] = v2.Value
		}
	}
	if len(v.Children) != len(v2.Children) {
		// TODO: this probably isn't quite right, since some attributes
		// are optional.  Not quite sure how to handle attributes yet
		return false
	}

	// compare non-attribute children.  must be in same order
	var attrs, attrs2 []*TTLV
	for i, child := range v.Children {
		child2 := v2.Children[i]
		if child.GetTag() == "Attribute" {
			if child2.GetTag() != "Attribute" {
				return false
			}
			attrs = append(attrs, child)
			attrs2 = append(attrs2, child2)
		} else {
			if !child.Cmp(child2) {
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
			if !isVariable(v.Value) {
				return false
			}
			if v.variables == nil {
				v.variables = map[string]string{}
			}
			if varV, ok := v.variables[v.Value]; ok {
				if varV != v2 {
					return false
				}
			} else {
				v.variables[v.Value] = v2
			}

		}
	}

	return true
}

func isVariable(s string) bool {
	return regexp.MustCompile(`\$[A-Z0-9_]+`).MatchString(s)
}
