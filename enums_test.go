package kmip

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBitMaskString(t *testing.T) {
	tests := []struct {
		in  CryptographicUsageMask
		out string
	}{
		{
			in:  CryptographicUsageMaskSign,
			out: "Sign",
		},
		{
			in:  CryptographicUsageMask(0x00100000),
			out: "0x00100000",
		},
		{
			in:  CryptographicUsageMaskSign | CryptographicUsageMaskExport,
			out: "Sign|Export",
		},
		{
			in:  CryptographicUsageMaskSign | CryptographicUsageMaskExport | CryptographicUsageMask(0x00100000),
			out: "Sign|Export|0x00100000",
		},
		{
			in:  CryptographicUsageMaskSign | CryptographicUsageMaskExport | CryptographicUsageMask(0x00100000) | CryptographicUsageMask(0x00200000),
			out: "Sign|Export|0x00300000",
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.out, func(t *testing.T) {
			assert.Equal(t, testcase.out, testcase.in.String())
		})
	}
}

func TestParseInteger(t *testing.T) {
	tests := []struct {
		out CryptographicUsageMask
		in  string
	}{
		{
			out: CryptographicUsageMaskSign,
			in:  "Sign",
		},
		{
			out: CryptographicUsageMaskDecrypt,
			in:  "0x00000008",
		},
		{
			out: CryptographicUsageMaskDecrypt,
			in:  "8",
		},
		{
			out: CryptographicUsageMask(0x00100000),
			in:  "0x00100000",
		},
		{
			out: CryptographicUsageMask(0x00100000),
			in:  "1048576",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskExport,
			in:  "Sign|Export",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskExport,
			in:  "Sign Export",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskExport,
			in:  "0x00000001 0x00000040",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskExport,
			in:  "0x00000001|0x00000040",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskVerify,
			in:  "1|2",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskVerify | CryptographicUsageMaskEncrypt,
			in:  "1 2 4",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskVerify | CryptographicUsageMaskEncrypt,
			in:  " 1   2      4  ",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskVerify | CryptographicUsageMaskEncrypt,
			in:  " 1 || 2|||   4  ",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskExport | CryptographicUsageMask(0x00100000),
			in:  "Sign|Export|0x00100000",
		},
		{
			out: CryptographicUsageMaskSign | CryptographicUsageMaskExport | CryptographicUsageMask(0x00100000) | CryptographicUsageMask(0x00200000),
			in:  "Sign|Export|0x00300000",
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.in, func(t *testing.T) {
			mask, e := ParseInteger(TagCryptographicUsageMask, testcase.in)
			require.NoError(t, e)
			assert.Equal(t, int32(testcase.out), mask)
		})
	}
}
