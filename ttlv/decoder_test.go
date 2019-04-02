package ttlv

import (
	"bytes"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"
)

func TestUnmarshal_known(t *testing.T) {
	for _, sample := range knownGoodSamples {
		tname := sample.name
		if tname == "" {
			tname = fmt.Sprintf("%T", sample.v)
		}
		t.Run(tname, func(t *testing.T) {
			typ := reflect.ValueOf(sample.v).Type()
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			v := reflect.New(typ).Interface()

			err := Unmarshal(hex2bytes(sample.exp), v)
			require.NoError(t, err)
			switch tv := sample.v.(type) {
			case *big.Int:
				require.Zero(t, tv.Cmp(v.(*big.Int)))
			case big.Int:
				require.Zero(t, tv.Cmp(v.(*big.Int)))
			default:
				require.Equal(t, sample.v, reflect.ValueOf(v).Elem().Interface())
			}

		})

	}
}

func TestUnmarshal(t *testing.T) {

	type unmarshalTest struct {
		name                   string
		in, ptr, expected      interface{}
		err                    error
		skipSliceOfTest        bool
		skipExactRoundtripTest bool
	}

	tests := []unmarshalTest{
		{
			in:  true,
			ptr: new(bool),
		},
		{
			in:  "red",
			ptr: new(string),
		},
		{
			in:  time.Second * 10,
			ptr: new(time.Duration),
		},
		{
			in:  parseTime("2008-03-14T11:56:40Z"),
			ptr: new(time.Time),
		},
		{
			in:  5,
			ptr: new(int),
		},
		{
			in:       5,
			ptr:      new(int8),
			expected: int8(5),
		},
		{
			name: "intoverflow",
			in:   math.MaxInt8 + 1,
			ptr:  new(int8),
			err:  ErrIntOverflow,
		},
		{
			in:       5,
			ptr:      new(int16),
			expected: int16(5),
		},
		{
			in:       5,
			ptr:      new(int32),
			expected: int32(5),
		},
		{
			in:  5,
			ptr: new(int64),
			err: ErrUnsupportedTypeError,
		},
		{
			in:       5,
			ptr:      new(uint),
			expected: uint(5),
		},
		{
			in:              5,
			ptr:             new(uint8),
			expected:        uint8(5),
			skipSliceOfTest: true, // []uint8 is an alias for []byte, which is handled differently
		},
		{
			in:       5,
			ptr:      new(uint16),
			expected: uint16(5),
		},
		{
			in:       5,
			ptr:      new(uint32),
			expected: uint32(5),
		},
		{
			in:  5,
			ptr: new(uint64),
			err: ErrUnsupportedTypeError,
		},
		{
			name: "uintoverflow",
			in:   math.MaxUint8 + 1,
			ptr:  new(uint8),
			err:  ErrIntOverflow,
		},
		{
			in:  int64(5),
			ptr: new(int64),
		},
		{
			in:       int64(5),
			ptr:      new(uint64),
			expected: uint64(5),
		},
		{
			in:  []byte{0x01, 0x02, 0x03},
			ptr: new([]byte),
		},
		{
			in:       big.NewInt(5),
			ptr:      new(big.Int),
			expected: *(big.NewInt(5)),
		},
		{
			in:  CredentialTypeAttestation,
			ptr: new(CredentialType),
		},
		{
			in:                     CredentialTypeAttestation,
			ptr:                    new(uint),
			expected:               uint(CredentialTypeAttestation),
			skipExactRoundtripTest: true,
		},
		{
			in:                     CredentialTypeAttestation,
			ptr:                    new(int64),
			expected:               int64(CredentialTypeAttestation),
			skipExactRoundtripTest: true,
		},
		{
			in:  TaggedValue{Tag: TagBatchCount, Value: "red"},
			ptr: new(interface{}),
			expected: func() interface{} {
				b, err := Marshal(TaggedValue{Tag: TagBatchCount, Value: "red"})
				require.NoError(t, err)
				return TTLV(b)
			}(),
		},
		{
			name: "structtypeerror",
			in: Structure{TTLVTag: TagBatchCount, Values: []interface{}{
				TaggedValue{Tag: TagComment, Value: "red"},
			}},
			ptr: new(int),
			err: ErrUnsupportedTypeError,
		},
		{
			name: "ttlvStructure",
			in: Structure{TTLVTag: TagBatchCount, Values: []interface{}{
				TaggedValue{Tag: TagBatchItem, Value: "red"},
				TaggedValue{Tag: TagBatchContinueCapability, Value: "blue"},
			}},
			ptr: new(Structure),
			expected: func() interface{} {
				s := Structure{TTLVTag: TagBatchCount}
				for _, v := range []TaggedValue{
					{Tag: TagBatchItem, Value: "red"},
					{Tag: TagBatchContinueCapability, Value: "blue"},
				} {
					b, err := Marshal(v)
					require.NoError(t, err)
					s.Values = append(s.Values, TTLV(b))
				}
				return s
			}(),
		},
	}

	type A struct {
		Comment string
	}
	tests = append(tests,
		unmarshalTest{
			name: "simplestruct",
			in: Structure{TTLVTag: TagBatchCount, Values: []interface{}{
				TaggedValue{Tag: TagComment, Value: "red"},
			}},
			ptr:      new(A),
			expected: A{Comment: "red"},
		},
	)

	type B struct {
		S string `kmip:"Comment"`
	}

	tests = append(tests,
		unmarshalTest{
			name: "fieldtag",
			in: Structure{TTLVTag: TagBatchCount, Values: []interface{}{
				TaggedValue{Tag: TagComment, Value: "red"},
			}},
			ptr:      new(B),
			expected: B{S: "red"},
		},
	)

	type D struct {
		Comment    string
		BatchCount string
	}

	tests = append(tests,
		unmarshalTest{
			name: "multifields",
			in: Structure{TTLVTag: TagBatchCount, Values: []interface{}{
				TaggedValue{Tag: TagComment, Value: "red"},
				TaggedValue{Tag: TagBatchCount, Value: "blue"},
			}},
			ptr:      new(D),
			expected: D{Comment: "red", BatchCount: "blue"},
		},
	)

	type E struct {
		Comment    string
		BatchCount A
	}

	tests = append(tests,
		unmarshalTest{
			name: "nested",
			in: Structure{TTLVTag: TagBatchCount, Values: []interface{}{
				TaggedValue{Tag: TagComment, Value: "red"},
				Structure{TTLVTag: TagBatchCount, Values: []interface{}{
					TaggedValue{Tag: TagComment, Value: "blue"},
				}},
			}},
			ptr:      new(E),
			expected: E{Comment: "red", BatchCount: A{Comment: "blue"}},
		},
	)

	type F struct {
		Comment    string
		BatchCount *A
	}

	tests = append(tests,
		unmarshalTest{
			name: "nestedptr",
			in: Structure{TTLVTag: TagBatchCount, Values: []interface{}{
				TaggedValue{Tag: TagComment, Value: "red"},
				Structure{TTLVTag: TagBatchCount, Values: []interface{}{
					TaggedValue{Tag: TagComment, Value: "blue"},
				}},
			}},
			ptr:      new(F),
			expected: F{Comment: "red", BatchCount: &A{Comment: "blue"}},
		},
	)

	type G struct {
		Comment []string
	}

	tests = append(tests,
		unmarshalTest{
			name: "slice",
			in: Structure{TTLVTag: TagBatchCount, Values: []interface{}{
				TaggedValue{Tag: TagComment, Value: "red"},
				TaggedValue{Tag: TagComment, Value: "blue"},
				TaggedValue{Tag: TagComment, Value: "green"},
			}},
			ptr:      new(G),
			expected: G{Comment: []string{"red", "blue", "green"}},
		},
	)

	type H struct {
		Comment        string
		Any1           []TaggedValue `kmip:",any"`
		Any2           []TaggedValue `kmip:",any"`
		AttributeValue string
	}

	tests = append(tests,
		unmarshalTest{
			name: "anyflag",
			in: Structure{Values: []interface{}{
				TaggedValue{Tag: TagComment, Value: "red"},
				TaggedValue{Tag: TagNameType, Value: "blue"},
				TaggedValue{Tag: TagName, Value: "orange"},
				TaggedValue{Tag: TagAttributeValue, Value: "yellow"},
			}},
			ptr: new(H),
			expected: H{Comment: "red", AttributeValue: "yellow", Any1: []TaggedValue{
				{Tag: TagNameType, Value: "blue"},
				{Tag: TagName, Value: "orange"},
			}},
		})

	for _, test := range tests {
		if test.name == "" {
			test.name = fmt.Sprintf("%T into %T", test.in, test.ptr)
		}
		t.Run(test.name, func(t *testing.T) {

			b, err := Marshal(TaggedValue{Tag: TagBatchCount, Value: test.in})
			require.NoError(t, err)

			t.Log(TTLV(b).String())

			v := reflect.New(reflect.TypeOf(test.ptr).Elem())
			expected := test.expected
			if expected == nil {
				expected = test.in
			}

			err = Unmarshal(b, v.Interface())

			if test.err != nil {
				require.Error(t, err, "got value instead: %#v", v.Elem().Interface())
				require.True(t, Is(err, test.err), Details(err))
				return
			}

			require.NoError(t, err, Details(err))
			require.Equal(t, expected, v.Elem().Interface())

			// if out type is not a slice, add a test for unmarshaling into
			// a slice of that type, which should work to.  e.g.  you should
			// be able to unmarshal a bool into either a bool or a slice of bools

			if !test.skipSliceOfTest {
				t.Run("sliceof", func(t *testing.T) {
					sltype := reflect.SliceOf(reflect.TypeOf(test.ptr).Elem())
					v := reflect.New(sltype)
					err = Unmarshal(b, v.Interface())
					require.NoError(t, err, Details(err))
					expv := reflect.Zero(sltype)
					expv = reflect.Append(expv, reflect.ValueOf(expected))
					require.Equal(t, expv.Interface(), v.Elem().Interface())
				})
			}

			t.Run("roundtrip", func(t *testing.T) {
				bb, err := Marshal(TaggedValue{Tag: TagBatchCount, Value: v.Elem().Interface()})
				require.NoError(t, err)

				if !test.skipExactRoundtripTest {
					assert.Equal(t, TTLV(b).String(), TTLV(bb).String())
				}

				t.Log(TTLV(bb).String())

				vv := reflect.New(reflect.TypeOf(test.ptr).Elem())
				err = Unmarshal(bb, vv.Interface())
				require.NoError(t, err)

				assert.Equal(t, v.Elem().Interface(), vv.Elem().Interface())
			})
		})

	}

}

func TestDecoder_DisallowUnknownFields(t *testing.T) {
	type A struct {
		Comment    string
		BatchCount int
	}

	tests := []struct {
		name  string
		input Structure
	}{
		{
			name: "middle",
			input: Structure{
				TagAlternativeName,
				[]interface{}{
					TaggedValue{TagComment, "red"},
					TaggedValue{TagArchiveDate, "blue"},
					TaggedValue{TagBatchCount, 5},
				},
			},
		},
		{
			name: "first",
			input: Structure{
				TagAlternativeName,
				[]interface{}{
					TaggedValue{TagArchiveDate, "blue"},
					TaggedValue{TagComment, "red"},
					TaggedValue{TagBatchCount, 5},
				},
			},
		},
		{
			name: "last",
			input: Structure{
				TagAlternativeName,
				[]interface{}{
					TaggedValue{TagComment, "red"},
					TaggedValue{TagBatchCount, 5},
					TaggedValue{TagArchiveDate, "blue"},
				},
			},
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			b, err := Marshal(testcase.input)
			require.NoError(t, err)

			// verify that it works find if flag is off
			dec := NewDecoder(bytes.NewReader(b))
			a := A{}
			err = dec.Decode(&a)
			require.NoError(t, err)

			require.Equal(t, A{"red", 5}, a)

			// verify that it bombs is flag is on
			dec = NewDecoder(bytes.NewReader(b))
			dec.DisallowExtraValues = true
			err = dec.Decode(&a)
			require.Error(t, err)
			require.True(t, merry.Is(err, ErrUnexpectedValue))

		})
	}
}
