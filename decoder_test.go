package kmip

import (
	"testing"
	"github.com/stretchr/testify/require"
	"reflect"
	"time"
	"fmt"
	"math"
	"math/big"
)

func TestUnmarshal(t *testing.T) {

	tests := []struct {
		name              string
		in, ptr, expected interface{}
		err               error
		skipSliceOfTest bool
	}{
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
			in:  parseTime("Friday, March 14, 2008, 11:56:40 UTC"),
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
			in:   math.MaxInt8+1,
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
			in:       5,
			ptr:      new(uint8),
			expected: uint8(5),
			skipSliceOfTest:true, // []uint8 is an alias for []byte, which is handled differently
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
			in:       5,
			ptr:      new(uint64),
			err: ErrUnsupportedTypeError,
		},
		{
			name:"uintoverflow",
			in:  math.MaxUint8 + 1,
			ptr: new(uint8),
			err: ErrIntOverflow,
		},
		{
			in: int64(5),
			ptr:new(int64),
		},
		{
			in:  int64(5),
			ptr: new(uint64),
			expected:uint64(5),
		},
		{
			in:       []byte{0x01, 0x02, 0x03},
			ptr:      new([]byte),
		},
		{
			in: big.NewInt(5),
			ptr: new(big.Int),
			expected: *(big.NewInt(5)),
		},
		{
			in: CredentialTypeAttestation,
			ptr: new(CredentialType),
		},
		{
			in:  CredentialTypeAttestation,
			ptr: new(uint),
			expected:uint(CredentialTypeAttestation),
		},
		{
			in:       CredentialTypeAttestation,
			ptr:      new(int64),
			expected: int64(CredentialTypeAttestation),
		},
	}

	for _, test := range tests {
		if test.name == "" {
			test.name = fmt.Sprintf("%T into %T", test.in, test.ptr)
		}
		t.Run(test.name, func(t *testing.T) {

			tv := test.in

			switch ttv := tv.(type) {
			case TaggedValue, Structure, Marshaler:
			default:
				tv = TaggedValue{Value: ttv}
			}

			b, err := MarshalTTLV(tv)
			require.NoError(t, err)

			v := reflect.New(reflect.TypeOf(test.ptr).Elem())
			expected := test.expected
			if expected == nil {
				expected = test.in
			}

			t.Log(TTLV(b).String())

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
				bb, err := MarshalTTLV(TaggedValue{Value: v.Elem().Interface()})
				require.NoError(t, err)

				// TODO: need to understand what the policy is on roundtrips
				// you can unmarshal into something that would re-marshal
				// into a different TTLV.  JSON seems to allow for this,
				// with the "golden" flag on test cases.
				require.Equal(t, TTLV(b).String(), TTLV(bb).String())

				t.Log(TTLV(bb).String())

				vv := reflect.New(reflect.TypeOf(test.ptr).Elem())
				err = Unmarshal(bb, vv.Interface())
				require.NoError(t, err)

				require.Equal(t, v.Elem().Interface(), vv.Elem().Interface())
			})
		})

	}

	var b bool

	ttlv, err := MarshalTTLV(TaggedValue{Value: true})
	require.NoError(t, err)

	err = Unmarshal(ttlv, &b)
	require.NoError(t, err)

	require.True(t, b)

}
