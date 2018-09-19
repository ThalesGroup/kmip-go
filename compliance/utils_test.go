package compliance

import (
	"encoding/xml"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTTLV_MarshalXML(t *testing.T) {
	ttlv := TTLV{
		Tag:   "blue",
		Type:  "white",
		Value: "green",
		Children: []*TTLV{
			{
				XMLName: xml.Name{Local: "brown"},
				Tag:     "orange",
				Type:    "black",
				Value:   "white",
			},
		},
	}
	b, err := xml.Marshal(ttlv)
	require.NoError(t, err)
	require.Equal(t, string(b), `<TTLV tag="blue" type="white" value="green"><brown tag="orange" type="black" value="white"></brown></TTLV>`)
}

//func TestTTLV_Cmp(t *testing.T) {
//	tests := []struct{
//		name string
//		isEq bool
//		v1 *TTLV
//		v2 *TTLV
//	}{
//
//	}
//}
