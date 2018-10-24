package compliance

import (
	"encoding/xml"
	"github.com/pmezard/go-difflib/difflib"
	"gitlab.protectv.local/regan/kmip.git"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type TTLV struct {
	XMLName       xml.Name
	Tag           string  `xml:"tag,omitempty,attr"`
	Type          string  `xml:"type,attr,omitempty"`
	Value         string  `xml:"value,attr,omitempty"`
	VariableValue string  `xml:"-"`
	Ignored       bool    `xml:"-"`
	Children      []*TTLV `xml:",any"`
}

func (v *TTLV) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if v.Ignored {
		return nil
	}
	v2 := *v
	if v.VariableValue != "" {
		v2.Value = v.VariableValue
	}

	start.Name = v.XMLName
	type ttlv TTLV
	return e.EncodeElement((*ttlv)(v), start)
}

func (v *TTLV) GetTag() string {
	if v.XMLName.Local != "" {
		return v.XMLName.Local
	}
	return v.Tag
}

func Compare(v1, v2 *TTLV) (eq bool, vars map[string]string, diff string) {
	vars = map[string]string{}
	var c1, c2 *TTLV
	eq, c1, c2 = compare(v1, v2, vars)
	if !eq {
		diff = diffS(c1, c2, vars)
	}
	return
}

func compare(v1, v2 *TTLV, vars map[string]string) (eq bool, v1out *TTLV, v2out *TTLV) {
	switch {
	case v1 == nil && v2 == nil:
		return true, v1, v2
	case v1 == nil && v2 != nil:
		return false, v1, v2
	case v1 != nil && v2 == nil:
		return false, v1, v2
	case v1.GetTag() != v2.GetTag():
		return false, v1, v2
	case v1.Type != v2.Type:
		return false, v1, v2
	case v1.Value != v2.Value:
		if !isVariable(v1.Value) {
			return false, v1, v2
		}
		if varV, ok := vars[v1.Value]; ok {
			v1.VariableValue = varV
			if varV != v2.Value {
				return false, v1, v2
			}
		} else {
			v2.VariableValue = v2.Value
			vars[v1.Value] = v2.Value
		}
	}

	// compare non-attribute children.  must be in same order
	var attrs, attrs2 []*TTLV
	var i2 = -1
	for _, child := range v1.Children {
		var child2 *TTLV
	Search:
		for {
			i2++
			if i2 >= len(v2.Children) {
				// we've run out of children on the target, and having
				// found the match yet
				return false, v1, v2
			}
			child2 = v2.Children[i2]
			if child2.Tag == child.Tag {
				// found the match
				break Search
			}
			if isIgnoredTag(child2.Tag) {
				// these tags are ignored, keep searching
				child2.Ignored = true
			} else {
				return false, child, child2
			}
		}
		if child.GetTag() == "Attribute" {
			if child2.GetTag() != "Attribute" {
				return false, child, child2
			}
			attrs = append(attrs, child)
			attrs2 = append(attrs2, child2)
		} else {
			if eq, c1, c2 := compare(child, child2, vars); !eq {
				return false, c1, c2
			}
		}
	}

	// scrub through the remaining children of v2, skipping any ignored tags
	// if there are any unmatched, not-ignored values, it's not a match
	for {
		i2++
		if i2 >= len(v2.Children) {
			// we've run out of children on the target, and having
			// found the match yet
			break
		}
		if !isIgnoredTag(v2.Children[i2].Tag) {
			return false, v1, v2
		} else {
			v2.Children[i2].Ignored = true
		}
	}

	type parsedAttribute struct {
		name  string
		value string
		index string
		orig  *TTLV
	}

	parse := func(t *TTLV) parsedAttribute {
		var parsed parsedAttribute
		for _, attrC := range t.Children {
			switch attrC.GetTag() {
			case "AttributeName":
				parsed.name = attrC.Value
			case "AttributeValue":
				parsed.value = attrC.Value
			case "AttributeIndex":
				parsed.index = attrC.Value
			}
		}
		return parsed
	}

	var parsed1, parsed2 []parsedAttribute

	for _, attr := range attrs {
		parsed1 = append(parsed1, parse(attr))
	}

	for _, attr := range attrs2 {
		parsed2 = append(parsed2, parse(attr))
	}

	sort.Slice(parsed1, func(i, j int) bool {
		if parsed1[i].name < parsed1[j].name {
			return true
		}
		if parsed1[i].name > parsed1[j].name {
			return false
		}
		var idx1, idx2 int
		if parsed1[i].index == "" {
			idx1 = 0
		} else {
			var err error
			idx1, err = strconv.Atoi(parsed1[i].index)
			if err != nil {
				panic(err)
			}
		}
		if parsed1[j].index == "" {
			idx2 = 0
		} else {
			var err error
			idx2, err = strconv.Atoi(parsed1[j].index)
			if err != nil {
				panic(err)
			}
		}
		if idx1 < idx2 {
			return true
		}
		return false
	})

	sort.Slice(parsed2, func(i, j int) bool {
		if parsed2[i].name < parsed2[j].name {
			return true
		}
		if parsed2[i].name > parsed2[j].name {
			return false
		}
		var idx1, idx2 int
		if parsed2[i].index == "" {
			idx1 = 0
		} else {
			var err error
			idx1, err = strconv.Atoi(parsed2[i].index)
			if err != nil {
				panic(err)
			}
		}
		if parsed2[j].index == "" {
			idx2 = 0
		} else {
			var err error
			idx2, err = strconv.Atoi(parsed2[j].index)
			if err != nil {
				panic(err)
			}
		}
		if idx1 < idx2 {
			return true
		}
		return false
	})

	i2 = -1
	for _, attr := range parsed1 {
		var attr2 parsedAttribute
	SearchAttr:
		for {
			i2++
			if i2 >= len(parsed2) {
				// we've run out of children on the target, and having
				// found the match yet
				return false, v1, v2
			}
			attr2 = parsed2[i2]
			if attr2.name == attr.name {
				// found the match
				break SearchAttr
			}
			//if isIgnoredTag(attr2.Tag) {
			//	these tags are ignored, keep searching
			//attr2.Ignored = true
			//} else {
			//	return false, child, attr2
			//}
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
		mv1 := attrMap[key]
		mv2 := attrMap2[key]
		if mv1 != mv2 {
			if !isVariable(mv1) {
				return false, v1, v2
			}
			if varV, ok := vars[mv1]; ok {
				if varV != v2.Value {
					return false, v1, v2
				}
			} else {
				vars[v1.Value] = v2.Value
			}

		}
	}

	return true, v1, v2
}

func isIgnoredTag(tag string) bool {
	switch tag {
	case kmip.TagServerCorrelationValue.String(), kmip.TagClientCorrelationValue.String():
		return true
	}
	return false
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
