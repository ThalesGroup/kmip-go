// Package ttlv encodes and decodes the 3 wire formats defined in the KMIP specification:
//
// 1. TTLV (the default, binary wire format)
// 2. JSON
// 3. XML
//
// The core representation of KMIP values is the ttlv.TTLV type, which is
// a []byte encoded in the TTLV binary format.  The ttlv.TTLV type knows how to marshal/
// unmarshal to and from the JSON and XML encoding formats.
//
// This package also knows how to marshal and unmarshal ttlv.TTLV values to golang structs,
// in a way similar to the json or xml packages.
//
// When decoding, you'll first unmarshal JSON or XML to TTLV (if necessary), then unmarshal TTLV into
// go structs.  When encoding, you'll first marshal go structs to TTLV, then marshal that
// into XML or JSON (if necessary).
//
// KMIP TTLV value types are mapped to idiomatic golang types:
//
// | KMIP type | Golang type |
// | --------- | ----------- |
// | Interval  | time.Duration |
// | DateTime  | time.Time |
// | DateTimeExtended | ttlv.DateTimeExtended |
// | ByteString | []byte |
// | TextString | string |
// | Boolean | bool |
// | Enumeration | uint32 |
// | BigInteger | *bit.Int |
// | LongInteger | int64 |
// | Integer | int |
// | Structure | ttlv.TTLV |
//
// The ttlv.TTLV methods convert the value to these types.
//
// Marshaling/Unmarshaling
//
// When mapping golang values to and from KMIP tagged values, two things need to
// be inferred:
//
// 1. The transcoding of golang types and values to KMIP types and values
// 2. The mapping of KMIP tags to golang types and struct fields
//
// Transcoding Golang Values
//
// - nil marshals to nothing (no bytes are encoded)
// - time.Time, time.Duration, string, bool, []byte, big.Int, and ttlv.DateTimeExtended are encoded
//   and decoded according to the table above.
// - marshaling numeric types: the encoded type depends on the tag being encoded.  If
//   the tag is registered as an enum, the value will be encoded as an Enumeration, otherwise
//   it will be encoded as an Integer or LongInteger, depending on the size of the golang
//   type.  int, int8, int16, int32, uint, uint8, uint16, uint32 will encode as Integer.
//   int64 and uint64 will encode as LongInteger.  Some golang types can hold a higher value
//   than the corresponding KMIP type (e.g. uint64).  If the value overflows the KMIP
//   type, ErrIntOverflow or ErrLongIntOverflow is returned
// - unmarshaling to numeric types: Integer can unmarshal into int, int8, int16,
//   int32, uint, uint8, uint16, or uint32.  LongInteger can unmarshal into int64 or uint64.
//   Enumeration can unmarshal into an numeric type.  If the KMIP value overflows the
//   golang type, ErrIntOverflow is returned
// - slices (other than []byte) are encoded as a ttlv.TTLV instance, containing
//   a sequence of concatenated TTLV values, by encoding each of the slice members.  Use
//   TTLV.Next() to iterate through the sequence.
// - structs map to KMIP Structures.  See notes below.
// - when unmarshaling into a slice (other than []byte), the TTLV value will be unmarshaled
//   into an instance of the slice's type, and appended to the end of the slice
// - chan, map, func, uintptr, float32/64, complex64/128, interface do not
//   map to KMIP types.  The encoder/decoder will return ErrUnsupportedTypeError
// - when marshaling interface values or pointers, the marshaler will follow one level
//   of indirection.  If that value is another pointer, ErrUnsupportedTypeError is returned
// - when unmarshaling into interface{}, the value will be set to the raw ttlv.TTLV
// - any type can assume control over the marshaling and unmarshaling process by implementing
//   Marshaler and Unmarshaler
//
// Tags
//
// When marshaling a golang value into a KMIP value, the KMIP tag must be determined.  The tag
// can explicitly specified, using the Encoder.EncodeValue() method.  If no tag is explicitly
// specified, the tag will be inferred from the value, using the following rules:
//
// 1. If the value is a struct, and the struct contains a field named "TTLVTag", and the field
//    has a "ttlv" struct tag, the value of the struct tag will be parsed using ParseTag().  If
//    parsing fails, an error is returned.  The type and value of the field is ignored.
//    In this example example, Foo will marshal to the DeactivationDate tag:
//
//         type Foo struct {
//             TTLVTag struct{} `ttlv:"DeactivationDate"`
//         }
//         ttlv.Marshal(&Foo{})        // marshals to
//
//    If a types with an explicit mapping is used in a field of another struct, the parent
//    struct cannot use it's own struct field tag to map to a different KMIP tag.  For example:
//
//         type Bar struct {
//             foo Foo `ttlv:"DerivationData"`
//         }
//
//    Using a ttlv struct tag on the field "foo" that conflicts with Foo's explicit mapping is an error, and using
//    Bar with marshal/unmarshal/encode/decode will result in a ErrTagConflict.
// 2. If the value is a struct, and the struct contains a field named "TTLVTag", and that field
//    is of type ttlv.Tag and is not empty, the value of the field will be KMIP tag.  For example:
//
//         type Foo struct {
//             TTLVTag ttlv.Tag
//         }
//         f := Foo{TTLVTag: ttlv.TagState}
//
//    This allows you to dynamically set the KMIP tag that a value will marshal to.
// 3. If the value is field in a struct, and the struct field has a "ttlv" tag, the value
//    of that will be parsed with ParseTag(), e.g.:
//
//         type Bar struct {
//             foo Foo   `ttlv:"DerivationData"`
//         }
//
//    The value "foo" will be marshaled to the KMIP DerivationData tag.
// 4. If the value is field in a struct, the field's name will be passed to ParseTag().
//    Parse errors are ignored, but if the parse is successfully, the value will be encoded
//    with that KMIP tag, e.g.:
//
//         type Bar struct {
//             DerivationData int
//         }
//
// 5. Finally, the name of the value's type will be parsed with ParseTag().  Errors are ignored,
//    but if the parse is successful, the value will be encoded with that KMIP tag, e.g.:
//
//         type DerivationData int
//
//         type Bar struct {
//             dd DerivationData
//         }
//
// Marshaling/Unmarshaling Structs
//
// Structs marshal to and from the KMIP Structure type.  The fields of the struct map to the values
// in the structure.
//
// The "ttlv" struct tag can be used to tune the process.
//
//     type Foo struct {
//         i int               `ttlv:"DerivationData"`        // map field to KMIP tag DerivationData
//         j int               `ttlv:"0x420034"`              // tag can also be expressed as hex
//         k int               `ttlv:"-"`                     // skip this field: do not marshal it or unmarshal into it
//         l int               `ttlv:"DestroyDate,omitempty"` // do not marshal if value is zero; no effect on unmarshal
//         StartDate time.Time `ttlv:",dateTimeExtended"`     // marshal as DateTimeExtended; no effect on unmarshal
//         AttributeValue int  `ttlv:",enum"`                 // marshal as an Enumeration; usually unnecessary
//                                                            // since if the KMIP tag for this field is registered
//                                                            // as an enum, it would be encoded as an Enumeration
//                                                            // anyway
//         State string        `ttlv:",enum"`                 // enum flag can also be used on string fields.  The
//                                                            // encoder will try to parse the value using ParseEnum()

// - enum
// - any
// - dateTimeExtended
// - omitempty
//
// - Multiple fields in the same struct can't map to the same tag.  returns ErrTagConflict
// - KMIP Structures are an array of TTLV values, not a map, which means the same tag can appear
//   multiple times.  When unmarshaling, each occurrence of that tag will map to the same golang struct
//   field.  If that field is a scalar value, it will be overwritten each time, so the last occurrence
//   of the tag in the KMIP Structure will win.  If the golang field is a slice, each occurrence of the
//   tag in the KMIP Structure will be appended to the slice.
//
// | Golang type | KMIP type |
// | --------- | ----------- |
// | chan, map, func, uintptr, float32/64, complex64/128, interface | no mapping, encoding error |
// | []byte | ByteString |
// | other slice | sequence of TTLV values |
// |
// | Interval  | time.Duration |
// | DateTime  | time.Time |
// | DateTimeExtended | ttlv.DateTimeExtended |
// | ByteString | []byte |
// | TextString | string |
// | Boolean | bool |
// | Enumeration | uint32 |
// | BigInteger | *bit.Int |
// | LongInteger | int64 |
// | Integer | int |
// | Structure | ttlv.TTLV |
package ttlv
