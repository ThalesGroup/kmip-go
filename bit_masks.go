package kmip

//import (
//	"encoding/binary"
//	"encoding/hex"
//	"fmt"
//	"sort"
//	"strings"
//)
//
//var _CryptographicUsageMaskSortedValues []int
//
//func init() {
//	for v := range _CryptographicUsageMaskValueToNameMap {
//		_CryptographicUsageMaskSortedValues = append(_CryptographicUsageMaskSortedValues, int(v))
//		sort.Ints(_CryptographicUsageMaskSortedValues)
//	}
//}
//
//func (m CryptographicUsageMask) String2() string {
//
//	r := int(m)
//
//	var sb strings.Builder
//	var appending bool
//	for _, v := range _CryptographicUsageMaskSortedValues {
//		if v & r == v {
//			if name :=_CryptographicUsageMaskValueToNameMap[CryptographicUsageMask(v)]; name != "" {
//				if appending {
//					sb.WriteString("|")
//				} else {
//					appending = true
//				}
//				sb.WriteString(name)
//				r ^= v
//			}
//
//		}
//		if r == 0 {
//			break
//		}
//	}
//	if r != 0 {
//		if appending {
//			sb.WriteString("|")
//		}
//		fmt.Fprintf(&sb, "%#08x", uint32(r))
//	}
//	return sb.String()
//}
//
//func parseSingleCryptographicUsageMask(s string) (CryptographicUsageMask, error) {
//	if strings.HasPrefix(s, "0x") && len(s) == 10 {
//		b, err := hex.DecodeString(s[2:])
//		if err != nil {
//			return 0, err
//		}
//		return CryptographicUsageMask(binary.BigEndian.Uint32(b)), nil
//	}
//	if v, ok := _CryptographicUsageMaskNameToValueMap[s]; ok {
//		return v, nil
//	} else {
//		var v CryptographicUsageMask
//		return v, fmt.Errorf("%s is not a valid CryptographicUsageMask", s)
//	}
//}
//
//func ParseCryptoMask(s string) (CryptographicUsageMask, error) {
//
//	if !strings.Contains(s, "|") {
//		return parseSingleCryptographicUsageMask(s)
//	}
//	var v CryptographicUsageMask
//	parts := strings.Split(s, "|")
//	for _, part := range parts {
//		m, err := parseSingleCryptographicUsageMask(part)
//		if err != nil {
//			return 0, err
//		}
//		v |= m
//	}
//	return v, nil
//}
