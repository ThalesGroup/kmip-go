package kmip

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/ansel1/merry"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"
)

const lenTag = 3
const lenLen = 4
const lenInt = 4
const lenDateTime = 8
const lenInterval = 4
const lenEnumeration = 4
const lenLongInt = 8
const lenBool = 8
const lenHeader = lenTag + 1 + lenLen // tag + type + len

type TTLV []byte

func (t TTLV) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(t) == 0 {
		return nil
	}

	out := struct {
		XMLName  xml.Name
		Tag      string `xml:"tag,omitempty,attr"`
		Type     string `xml:"type,attr,omitempty"`
		Value    string `xml:"value,attr,omitempty"`
		Children []TTLV
		Inner    []byte `xml:",innerxml"`
	}{}

	tagS := t.Tag().String()
	if strings.HasPrefix(tagS, "0x") {
		out.XMLName.Local = "TTLV"
		out.Tag = tagS
	} else {
		out.XMLName.Local = tagS
	}

	if t.Type() != TypeStructure {
		out.Type = t.Type().String()
	}

	switch t.Type() {
	case TypeStructure:
		// TODO: handle translation of Attribute structures

		se := xml.StartElement{Name: out.XMLName}
		if out.Type != "" {
			se.Attr = append(se.Attr, xml.Attr{Name: xml.Name{Local: "type"}, Value: out.Type})
		}
		err := e.EncodeToken(se)
		if err != nil {
			return err
		}

		n := t.ValueStructure()
		var attrTag Tag
		for len(n) > 0 {
			// if the struct contains an attribute name, followed by an
			// attribute value, use the name to try and map enumeration values
			// to their string variants
			if n.Tag() == TagAttributeName {
				// try to map the attribute name to a tag
				attrTag, _ = ParseTag(NormalizeName(n.ValueTextString()))
			}
			if n.Tag() == TagAttributeValue && n.Type() == TypeEnumeration {
				e.EncodeToken(xml.StartElement{
					Name: xml.Name{Local: TagAttributeValue.String()},
					Attr: []xml.Attr{
						{
							Name:  xml.Name{Local: "type"},
							Value: TypeEnumeration.String(),
						},
						{
							Name:  xml.Name{Local: "value"},
							Value: EnumToString(attrTag, n.ValueEnumeration()),
						},
					},
				})
				e.EncodeToken(xml.EndElement{Name: xml.Name{Local: "AttributeValue"}})
			} else {
				err := e.Encode(n)
				if err != nil {
					return err
				}
			}
			n = n.Next()
		}
		return e.EncodeToken(xml.EndElement{Name: out.XMLName})

	case TypeInteger:
		if IsBitMask(t.Tag()) {
			out.Value = strings.Replace(EnumToString(t.Tag(), t.ValueEnumeration()), "|", " ", -1)
		} else {
			out.Value = strconv.Itoa(t.ValueInteger())
		}
	case TypeBoolean:
		out.Value = strconv.FormatBool(t.ValueBoolean())
	case TypeLongInteger:
		out.Value = strconv.FormatInt(t.ValueLongInteger(), 10)
	case TypeBigInteger:
		out.Value = hex.EncodeToString(t.ValueRaw())
	case TypeEnumeration:
		out.Value = EnumToString(t.Tag(), t.ValueEnumeration())
	case TypeTextString:
		out.Value = t.ValueTextString()
	case TypeByteString:
		out.Value = hex.EncodeToString(t.ValueByteString())
	case TypeDateTime:
		out.Value = t.ValueDateTime().Format(time.RFC3339Nano)
	case TypeInterval:
		out.Value = strconv.FormatUint(uint64(t.ValueInterval()/time.Second), 10)
	}

	e.Encode(&out)

	return nil
}

type xmltval struct {
	XMLName  xml.Name
	Tag      string     `xml:"tag,omitempty,attr"`
	Type     string     `xml:"type,attr,omitempty"`
	Value    string     `xml:"value,attr,omitempty"`
	Children []*xmltval `xml:",any"`
}

func unmarshalXMLTval(buf *encBuf, tval *xmltval, attrTag Tag) error {
	if tval.Tag == "" {
		tval.Tag = tval.XMLName.Local
	}

	tag, err := ParseTag(tval.Tag)
	if err != nil {
		return err
	}

	var tp Type
	if tval.Type == "" {
		tp = TypeStructure
	} else {
		tp, err = ParseType(tval.Type)
		if err != nil {
			return err
		}
	}

	switch tp {
	case TypeBoolean:
		b, err := strconv.ParseBool(tval.Value)
		if err != nil {
			return err
		}
		buf.encodeBool(tag, b)
	case TypeTextString:
		buf.encodeTextString(tag, tval.Value)
	case TypeByteString:
		b, err := hex.DecodeString(tval.Value)
		if err != nil {
			return err
		}
		buf.encodeByteString(tag, b)
	case TypeInterval:
		u, err := strconv.ParseUint(tval.Value, 10, 64)
		if err != nil {
			return err
		}
		buf.encodeInterval(tag, time.Duration(u)*time.Second)
	case TypeDateTime:
		d, err := time.Parse(time.RFC3339Nano, tval.Value)
		if err != nil {
			return err
		}
		buf.encodeDateTime(tag, d)
	case TypeInteger:
		i, err := ParseInteger(tag, strings.Replace(tval.Value, " ", "|", -1))
		if err != nil {
			return err
		}
		buf.encodeInt(tag, int32(i))
	case TypeLongInteger:
		i, err := strconv.ParseInt(tval.Value, 10, 64)
		if err != nil {
			return err
		}
		buf.encodeLongInt(tag, i)
	case TypeBigInteger:
		b, err := hex.DecodeString(tval.Value)
		if err != nil {
			return err
		}
		n := &big.Int{}
		unmarshalBigInt(n, b)
		buf.encodeBigInt(tag, n)
	case TypeEnumeration:
		enumTag := tag
		if tag == TagAttributeValue && attrTag != TagNone {
			enumTag = attrTag
		}
		e, err := ParseEnum(enumTag, tval.Value)
		if err != nil {
			return err
		}
		buf.encodeEnum(tag, e)
	case TypeStructure:
		i := buf.startStruct(tag)
		var attrTag Tag
		for _, c := range tval.Children {
			offset := buf.Len()
			err := unmarshalXMLTval(buf, c, attrTag)
			if err != nil {
				return err
			}
			// check whether the TTLV we just unmarshaled is an AttributeName
			ttlv := TTLV(buf.Bytes()[offset:])
			if ttlv.Tag() == TagAttributeName {
				// try to parse the value as a tag name, which may be used later
				// when unmarshaling the AttributeValue
				attrTag, _ = ParseTag(NormalizeName(ttlv.ValueTextString()))
			}
		}
		buf.endStruct(i)
	}
	return nil
}

func (t *TTLV) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {

	var out xmltval
	err := d.DecodeElement(&out, &start)
	if err != nil {
		return err
	}

	var buf encBuf
	err = unmarshalXMLTval(&buf, &out, TagNone)
	if err != nil {
		return err
	}
	*t = buf.Bytes()
	return nil
}

var maxJSONInt = int64(1) << 52
var maxJSONBigInt = big.NewInt(maxJSONInt)

func (t *TTLV) UnmarshalJSON(b []byte) error {
	return t.unmarshalJSON(b, TagNone)
}

func (t *TTLV) unmarshalJSON(b []byte, attrTag Tag) error {
	if len(b) == 0 {
		return nil
	}

	type tval struct {
		Tag   string          `json:"tag"`
		Type  string          `json:"type,omitempty"`
		Value json.RawMessage `json:"value"`
	}
	var ttl tval
	err := json.Unmarshal(b, &ttl)
	if err != nil {
		return err
	}

	tag, err := ParseTag(ttl.Tag)
	if err != nil {
		return err
	}

	var tp Type
	var v interface{}
	if ttl.Type == "" {
		tp = TypeStructure
	} else {
		tp, err = ParseType(ttl.Type)
		if err != nil {
			return err
		}
		// for all types besides Structure, unmarshal
		// value into interface{}
		err = json.Unmarshal(ttl.Value, &v)
		if err != nil {
			return err
		}
	}

	// performance note: for some types, like int, long int, and interval,
	// we are essentially decoding from binary into a go native type
	// then re-encoding to binary.  I benchmarked skipping this step
	// and transferring the bytes directly from the decoded hex strings
	// to the binary TTLV.  It turned out not be faster, and added an
	// additional set of paths to test.  Wasn't worth it.

	enc := encBuf{}
	switch tp {
	case TypeBoolean:
		switch tv := v.(type) {
		default:
			return merry.Errorf("%s: invalid Boolean value: must be boolean or hex string", tag.String())
		case bool:
			enc.encodeBool(tag, tv)
		case string:
			switch tv {
			default:
				return merry.Errorf("%s: invalid Boolean value: hex string for Boolean value must be either 0x0000000000000001 (true) or 0x0000000000000000 (false)", tag.String())
			case "0x0000000000000001":
				enc.encodeBool(tag, true)
			case "0x0000000000000000":
				enc.encodeBool(tag, false)
			}
		}
	case TypeTextString:
		switch tv := v.(type) {
		default:
			return merry.Errorf("%s: invalid TextString value: must be string", tag.String())
		case string:
			enc.encodeTextString(tag, tv)
		}
	case TypeByteString:
		switch tv := v.(type) {
		default:
			return merry.Errorf("%s: invalid ByteString value: must be hex string", tag.String())
		case string:
			if len(tv) >= 2 && tv[:2] == "0x" {
				return merry.Errorf("%s: invalid ByteString value: should not have 0x prefix", tag.String())
			}
			b, err := hex.DecodeString(tv)
			if err != nil {
				return merry.Prependf(err, "%s: invalid ByteString value", tag.String())
			}
			enc.encodeByteString(tag, b)
		}
	case TypeInterval:
		switch tv := v.(type) {
		default:
			return merry.Errorf("%s: invalid Interval value: must be number or hex string", tag.String())
		case string:
			if len(tv) >= 2 && tv[:2] != "0x" {
				return merry.Errorf("%s: invalid Interval value: hex value must start with 0x", tag.String())
			}
			b, err := hex.DecodeString(tv[2:])
			if err != nil {
				return merry.Prependf(err, "%s: invalid Interval value", tag.String())
			}
			if len(b) != 4 {
				return merry.Errorf("%s: invalid Interval value: must be 4 bytes (8 hex characters)", tag.String())
			}
			v := binary.BigEndian.Uint32(b)
			enc.encodeInterval(tag, time.Duration(v)*time.Second)
		case float64:
			enc.encodeInterval(tag, time.Duration(tv)*time.Second)
		}
	case TypeDateTime:
		switch tv := v.(type) {
		default:
			return merry.Errorf("%s: invalid DateTime value: must be string", tag.String())
		case string:
			if tv[:2] == "0x" {
				b, err := hex.DecodeString(tv[2:])
				if err != nil {
					return merry.Prependf(err, "%s: invalid DateTime value", tag.String())
				}
				if len(b) != 8 {
					return merry.Errorf("%s: invalid DateTime value: must be 8 bytes (16 hex characters)", tag.String())
				}

				u := binary.BigEndian.Uint64(b)
				tm := time.Unix(int64(u), 0)
				enc.encodeDateTime(tag, tm)
			} else {
				tm, err := time.Parse(time.RFC3339Nano, tv)
				if err != nil {
					return merry.Prependf(err, "%s: invalid DateTime value: must be ISO8601 format, parsing error", tag.String())
				}
				enc.encodeDateTime(tag, tm)
			}
		}
	case TypeInteger:
		switch tv := v.(type) {
		default:
			return merry.Errorf("%s: invalid Integer value: must be number, hex string, or mask value name", tag.String())
		case string:
			i, err := ParseInteger(tag, tv)
			if err != nil {
				return merry.Prependf(err, "%s: invalid Integer value", tag.String())
			}
			enc.encodeInt(tag, int32(i))
		case float64:
			enc.encodeInt(tag, int32(tv))
		}
	case TypeLongInteger:
		switch tv := v.(type) {
		default:
			return merry.Errorf("%s: invalid LongInteger value: must be number or hex string", tag.String())
		case string:
			if len(tv) >= 2 && tv[:2] != "0x" {
				return merry.Errorf("%s: invalid LongInteger value: hex value must start with 0x", tag.String())
			}
			b, err := hex.DecodeString(tv[2:])
			if err != nil {
				return merry.Prependf(err, "%s: invalid LongInteger value", tag.String())
			}
			if len(b) != 8 {
				return merry.Errorf("%s: invalid LongInteger value: must be 8 bytes (16 hex characters)", tag.String())
			}
			v := binary.BigEndian.Uint64(b)
			enc.encodeLongInt(tag, int64(v))
		case float64:
			enc.encodeLongInt(tag, int64(tv))
		}
	case TypeBigInteger:
		switch tv := v.(type) {
		default:
			return merry.Errorf("%s: invalid BigInteger value: must be number or hex string", tag.String())
		case string:
			if len(tv) >= 2 && tv[:2] != "0x" {
				return merry.Errorf("%s: invalid BigInteger value: hex value must start with 0x", tag.String())
			}
			b, err := hex.DecodeString(tv[2:])
			if err != nil {
				return merry.Prependf(err, "%s: invalid BigInteger value", tag.String())
			}
			if len(b)%8 != 0 {
				return merry.Errorf("%s: invalid BigInteger value: must be multiple of 8 bytes (16 hex characters)", tag.String())
			}
			i := &big.Int{}
			unmarshalBigInt(i, unpadBigInt(b))
			enc.encodeBigInt(tag, i)
		case float64:
			enc.encodeBigInt(tag, big.NewInt(int64(tv)))
		}
	case TypeEnumeration:
		switch tv := v.(type) {
		default:
			return merry.Errorf("%s: invalid Enumeration value: must be number or string", tag.String())
		case string:
			enumTag := tag
			if tag == TagAttributeValue && attrTag != TagNone {
				enumTag = attrTag
			}
			u, err := ParseEnum(enumTag, tv)
			if err != nil {
				return merry.Prependf(err, "%s: invalid Enumeration value", tag.String())
			}
			enc.encodeEnum(tag, u)
		case float64:
			enc.encodeEnum(tag, uint32(tv))
		}
	case TypeStructure:
		// unmarshal each sub value
		var children []json.RawMessage
		err := json.Unmarshal(ttl.Value, &children)
		if err != nil {
			return err
		}
		var scratch TTLV
		s := enc.startStruct(tag)
		var attrTag Tag
		for _, c := range children {
			err := (*TTLV)(&scratch).unmarshalJSON(c, attrTag)
			if err != nil {
				return err
			}
			if TagAttributeName == scratch.Tag() {
				attrTag, _ = ParseTag(NormalizeName(scratch.ValueTextString()))
			}
			enc.Write(scratch)
		}
		enc.endStruct(s)
	}
	*t = TTLV(enc.Bytes())
	return nil
}

func (t TTLV) MarshalJSON() ([]byte, error) {
	if len(t) == 0 {
		return []byte("null"), nil
	}
	if err := t.Valid(); err != nil {
		return nil, err
	}

	var sb strings.Builder

	sb.WriteString(`{"tag":"`)
	sb.WriteString(t.Tag().String())
	if t.Type() != TypeStructure {
		sb.WriteString(`","type":"`)
		sb.WriteString(t.Type().String())
	}
	sb.WriteString(`","value":`)

	switch t.Type() {
	case TypeBoolean:
		if t.ValueBoolean() {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	case TypeEnumeration:
		sb.WriteString(`"`)
		sb.WriteString(EnumToString(t.Tag(), t.ValueEnumeration()))
		sb.WriteString(`"`)
	case TypeInteger:
		if IsBitMask(t.Tag()) {
			sb.WriteString(`"`)
			sb.WriteString(EnumToString(t.Tag(), t.ValueEnumeration()))
			sb.WriteString(`"`)
		} else {
			sb.WriteString(strconv.Itoa(t.ValueInteger()))
		}
	case TypeLongInteger:
		v := t.ValueLongInteger()
		if v <= -maxJSONInt || v >= maxJSONInt {
			sb.WriteString(`"0x`)
			sb.WriteString(hex.EncodeToString(t.ValueRaw()))
			sb.WriteString(`"`)
		} else {
			sb.WriteString(strconv.FormatInt(v, 10))
		}
	case TypeBigInteger:
		v := t.ValueBigInteger()
		if v.IsInt64() && v.CmpAbs(maxJSONBigInt) < 0 {
			val, err := v.MarshalJSON()
			if err != nil {
				return nil, err
			}
			sb.Write(val)
		} else {
			sb.WriteString(`"0x`)
			sb.WriteString(hex.EncodeToString(t.ValueRaw()))
			sb.WriteString(`"`)
		}
	case TypeTextString:
		val, err := json.Marshal(t.ValueTextString())
		if err != nil {
			return nil, err
		}
		sb.Write(val)
	case TypeByteString:
		sb.WriteString(`"`)
		sb.WriteString(hex.EncodeToString(t.ValueRaw()))
		sb.WriteString(`"`)
	case TypeStructure:
		sb.WriteString("[")
		c := t.ValueStructure()
		var attrTag Tag
		for len(c) > 0 {
			// if the struct contains an attribute name, followed by an
			// attribute value, use the name to try and map enumeration values
			// to their string variants
			if c.Tag() == TagAttributeName {
				// try to map the attribute name to a tag
				attrTag, _ = ParseTag(NormalizeName(c.ValueTextString()))
			}
			if c.Tag() == TagAttributeValue && c.Type() == TypeEnumeration {
				sb.WriteString(`{"tag":"AttributeValue","type":"Enumeration","value":"`)
				sb.WriteString(EnumToString(attrTag, c.ValueEnumeration()))
				sb.WriteString(`"}`)
			} else {
				v, err := c.MarshalJSON()
				if err != nil {
					return nil, err
				}
				sb.Write(v)
			}
			c = c.Next()
			if len(c) > 0 {
				sb.WriteString(",")
			}
		}
		sb.WriteString("]")
	case TypeDateTime:
		val, err := t.ValueDateTime().MarshalJSON()
		if err != nil {
			return nil, err
		}
		sb.Write(val)
	case TypeInterval:
		sb.WriteString(strconv.FormatUint(uint64(binary.BigEndian.Uint32(t.ValueRaw())), 10))
	}

	sb.WriteString(`}`)
	return []byte(sb.String()), nil
	//return json.Marshal(&tval{Tag: t.Tag().String(), Type: t.Type().String(), Value: val})
}

func (t *TTLV) UnmarshalTTLV(ttlv TTLV, disallowUnknownFields bool) error {
	if ttlv == nil {
		*t = nil
		return nil
	}

	l := len(ttlv)
	if len(*t) < l {
		*t = make([]byte, l)
	} else {
		*t = (*t)[:l]
	}

	copy(*t, ttlv)
	return nil
}

func (t TTLV) Tag() Tag {
	// don't panic if header is truncated
	if len(t) < 3 {
		return Tag(0)
	}
	return Tag(uint32(t[2]) | uint32(t[1])<<8 | uint32(t[0])<<16)
}

func (t TTLV) Type() Type {
	// don't panic if header is truncated
	if len(t) < 4 {
		return Type(0)
	}
	return Type(t[3])
}

func (t TTLV) Len() int {
	// don't panic if header is truncated
	if len(t) < lenHeader {
		return 0
	}
	return int(binary.BigEndian.Uint32(t[4:8]))
}

func (t TTLV) FullLen() int {
	switch t.Type() {
	case TypeInterval, TypeDateTime, TypeBoolean, TypeEnumeration, TypeLongInteger, TypeInteger:
		return lenHeader + 8
	case TypeByteString, TypeTextString:
		l := t.Len() + lenHeader
		if m := l % 8; m > 0 {
			return l + (8 - m)
		}
		return l
	case TypeBigInteger, TypeStructure:
		return t.Len() + lenHeader
	}
	panic(fmt.Sprintf("invalid type: %x", byte(t.Type())))
}

func (t TTLV) ValueRaw() []byte {
	// don't panic if the value is truncated
	l := t.Len()
	if l == 0 {
		return nil
	}
	if len(t) < lenHeader+l {
		return t[lenHeader:]
	}
	return t[lenHeader : lenHeader+l]
}

func (t TTLV) Value() interface{} {
	switch t.Type() {
	case TypeInterval:
		return t.ValueInterval()
	case TypeDateTime:
		return t.ValueDateTime()
	case TypeByteString:
		return t.ValueByteString()
	case TypeTextString:
		return t.ValueTextString()
	case TypeBoolean:
		return t.ValueBoolean()
	case TypeEnumeration:
		return t.ValueEnumeration()
	case TypeBigInteger:
		return t.ValueBigInteger()
	case TypeLongInteger:
		return t.ValueLongInteger()
	case TypeInteger:
		return t.ValueInteger()
	case TypeStructure:
		return t.ValueStructure()
	}
	panic(fmt.Sprintf("invalid type: %x", byte(t.Type())))
}

func (t TTLV) ValueInteger() int {
	return int(binary.BigEndian.Uint32(t.ValueRaw()))
}

func (t TTLV) ValueLongInteger() int64 {
	return int64(binary.BigEndian.Uint64(t.ValueRaw()))
}

func (t TTLV) ValueBigInteger() *big.Int {
	i := new(big.Int)
	unmarshalBigInt(i, unpadBigInt(t.ValueRaw()))
	return i
}

func (t TTLV) ValueEnumeration() uint32 {
	return binary.BigEndian.Uint32(t.ValueRaw())
}

func (t TTLV) ValueBoolean() bool {
	return t.ValueRaw()[7] != 0
}

func (t TTLV) ValueTextString() string {
	// conveniently, KMIP strings are UTF-8 encoded, as are
	// golang strings
	return string(t.ValueRaw())
}

func (t TTLV) ValueByteString() []byte {
	return t.ValueRaw()
}

func (t TTLV) ValueDateTime() time.Time {
	i := t.ValueLongInteger()
	return time.Unix(i, 0).UTC()
}

func (t TTLV) ValueInterval() time.Duration {
	return time.Duration(binary.BigEndian.Uint32(t.ValueRaw())) * time.Second
}

func (t TTLV) ValueStructure() TTLV {
	return t.ValueRaw()
}

func (t TTLV) Valid() error {
	if err := t.ValidHeader(); err != nil {
		return err
	}

	if len(t) < t.FullLen() {
		return ErrValueTruncated
	}

	if t.Type() == TypeStructure {
		inner := t.ValueStructure()
		for {
			if len(inner) <= 0 {
				break
			}
			if err := inner.Valid(); err != nil {
				return merry.Prepend(err, t.Tag().String())
			}
			inner = inner.Next()
		}
	}

	return nil
}

func (t TTLV) validTag() bool {
	switch t[0] {
	case 0x42, 0x54: // valid
		return true
	}
	return false
}

func (t TTLV) ValidHeader() error {
	if l := len(t); l < lenHeader {
		return ErrHeaderTruncated
	}

	switch t.Type() {
	case TypeStructure, TypeTextString, TypeByteString:
		// any length is valid
	case TypeInteger, TypeEnumeration, TypeInterval:
		if t.Len() != lenInt {
			return ErrInvalidLen
		}
	case TypeLongInteger, TypeBoolean, TypeDateTime:
		if t.Len() != lenLongInt {
			return ErrInvalidLen
		}
	case TypeBigInteger:
		if (t.Len() % 8) != 0 {
			return ErrInvalidLen
		}
	default:
		return ErrInvalidType
	}
	if !t.validTag() {
		return ErrInvalidTag
	}
	return nil

}

func (t TTLV) Next() TTLV {
	if t.Valid() != nil {
		return nil
	}
	n := t[t.FullLen():]
	if len(n) == 0 {
		return nil
	}
	return n
}

func (t TTLV) String() string {
	buf := bytes.NewBuffer(nil)
	Print(buf, "", t)
	return buf.String()
}

func Print(w io.Writer, indent string, t TTLV) (err error) {

	tag := t.Tag()
	typ := t.Type()
	l := t.Len()

	fmt.Fprintf(w, "%s%v (%s/%d):", indent, tag, typ.String(), l)

	if err = t.Valid(); err != nil {
		fmt.Fprintf(w, " (%s)", err.Error())
		switch err {
		case ErrHeaderTruncated:
			// print the err, and as much of the truncated header as we have
			fmt.Fprintf(w, " %#x", []byte(t))
			return
		case ErrInvalidLen, ErrValueTruncated:
			// Something is wrong with the value.  Print the error, and the value
			fmt.Fprintf(w, " %#x", t.ValueRaw())
			return
		}
	}

	switch typ {
	case TypeByteString:
		fmt.Fprintf(w, " %#x", t.ValueByteString())
	case TypeStructure:
		indent += "  "
		s := t.ValueStructure()
		for s != nil {
			fmt.Fprint(w, "\n")
			if err = Print(w, indent, s); err != nil {
				// an error means we've hit invalid bytes in the stream
				// there are no markers to pick back up again, so we have to give up
				return
			}
			s = s.Next()
		}
	case TypeEnumeration:
		fmt.Fprint(w, " ", EnumToString(tag, t.ValueEnumeration()))
	case TypeInteger:
		if IsBitMask(tag) {
			fmt.Fprint(w, " ", EnumToString(tag, t.ValueEnumeration()))
		} else {
			fmt.Fprintf(w, " %v", t.Value())
		}
	default:
		fmt.Fprintf(w, " %v", t.Value())
	}
	return
}

var one = big.NewInt(1)

func unpadBigInt(data []byte) []byte {
	if len(data) < 2 {
		return data
	}

	i := 0
	for ; (i + 1) < len(data); i++ {
		switch {
		// first two cases keep looping, skipping pad bytes
		// pad bytes are all the same bit as
		// the first bit of the next byte
		case data[i] == 0xFF && data[i+1]&0x80 > 1:
		case data[i] == 0x00 && data[i+1]&0x80 == 0:
		default:
			// we've hit a byte that doesn't match the pad pattern
			return data[i:]
		}
	}
	// we've reached the last byte
	return data[i:]
}

// unmarshalBigInt sets the value of n to the big-endian two's complement
// value stored in the given data. If data[0]&80 != 0, the number
// is negative. If data is empty, the result will be 0.
func unmarshalBigInt(n *big.Int, data []byte) {
	n.SetBytes(data)
	if len(data) > 0 && data[0]&0x80 > 0 {
		// first byte is 1, so number is negative.
		// left shifting 1 by the length in bits of the data
		// then subtracting the value from that gives us the
		// twos complement.
		// e.g. if the value is 111111111, then 1 << 8 gives us
		//                     1000000000
		// and 100000000 - 11111111 = 00000001
		n.Sub(n, new(big.Int).Lsh(one, uint(len(data))*8))
	}
}
