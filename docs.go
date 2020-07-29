// Package kmip is a general purpose KMIP library for implementing KMIP services and clients.
//
// Features
//
// TTLV: This is a low-level parser for the TTLV binary format.  It can parse the binary format, and
// can marshal/unmarshal to/from the XML and JSON KMIP formats.  Note: this library is built around
// the binary TTLV format.  XML and JSON KMIP values need to be converted to TTLV values first.
//
// Encoder/Decoder: These types marshal and unmarshal TTLV to Go structs.  They work much like json.Encoder,
// json.Decoder, xml.Encoder, and xml.Decoder.  kmip.Encoder can also be used to directly encode TTLV values from
// primitive values.
package kmip
