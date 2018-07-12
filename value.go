package kmip

import "io"

/*
Structure: struct
Integer: int32
LongInteger: int64
BigInteger: big.Int
Boolean: bool
DateTime: time.Time
ByteString: []byte
TextString: string
Interval: time.Duration
Enumeration: Enumeration (uint32)

struct {
	TagName bool  // infer length and type, tag from name
	Cust int32   `tag:"TagName" type:"ByteString"`
	NoType int // tag from name, then infer length and type from tag info?
}
 */


 func writeInteger(w io.Writer, tag Tag, i int32) {

 }