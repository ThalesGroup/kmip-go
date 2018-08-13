package kmip

// 4.26

type DiscoverVersionsRequestPayload struct {
	ProtocolVersion []ProtocolVersion
}

//func (d *DiscoverVersionsRequestPayload) UnmarshalTTLV(ttlv TTLV) error {
//	var pv []ProtocolVersion
//	err := Unmarshal(ttlv, &pv)
//	if err != nil {
//		return err
//	}
//	if len(pv) > 0 {
//		if d == nil {
//			*d = DiscoverVersionsRequestPayload{}
//		}
//		d.ProtocolVersion = pv
//	}
//	return nil
//}
//
//func (d *DiscoverVersionsRequestPayload) MarshalTTLV(e *Encoder, tag Tag) error {
//	if d == nil {
//		return nil
//	}
//
//	return e.EncodeValue(tag, d.ProtocolVersion)
//}

type DiscoverVersionsResponsePayload struct {
	ProtocolVersion []ProtocolVersion
}

//func (d *DiscoverVersionsResponsePayload) UnmarshalTTLV(ttlv TTLV) error {
//	var pv []ProtocolVersion
//	err := Unmarshal(ttlv, &pv)
//	if err != nil {
//		return err
//	}
//	if len(pv) > 0 {
//		if d == nil {
//			*d = DiscoverVersionsResponsePayload{}
//		}
//		d.ProtocolVersion = pv
//	}
//	return nil
//}
//
//func (d *DiscoverVersionsResponsePayload) MarshalTTLV(e *Encoder, tag Tag) error {
//	if d == nil {
//		return nil
//	}
//
//	return e.EncodeValue(tag, d.ProtocolVersion)
//}
