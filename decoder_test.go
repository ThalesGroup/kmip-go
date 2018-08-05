package kmip

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"
)

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
	}

	type A struct {
		Comment string
	}
	tests = append(tests,
		unmarshalTest{
			name: "simplestruct",
			in: Structure{Tag: TagBatchCount, Values: []interface{}{
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
			in: Structure{Tag: TagBatchCount, Values: []interface{}{
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
			in: Structure{Tag: TagBatchCount, Values: []interface{}{
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
			in: Structure{Tag: TagBatchCount, Values: []interface{}{
				TaggedValue{Tag: TagComment, Value: "red"},
				Structure{Tag: TagBatchCount, Values: []interface{}{
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
			in: Structure{Tag: TagBatchCount, Values: []interface{}{
				TaggedValue{Tag: TagComment, Value: "red"},
				Structure{Tag: TagBatchCount, Values: []interface{}{
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
			in: Structure{Tag: TagBatchCount, Values: []interface{}{
				TaggedValue{Tag: TagComment, Value: "red"},
				TaggedValue{Tag: TagComment, Value: "blue"},
				TaggedValue{Tag: TagComment, Value: "green"},
			}},
			ptr:      new(G),
			expected: G{Comment: []string{"red", "blue", "green"}},
		},
	)

	for _, test := range tests {
		if test.name == "" {
			test.name = fmt.Sprintf("%T into %T", test.in, test.ptr)
		}
		t.Run(test.name, func(t *testing.T) {

			tv := test.in

			switch ttv := tv.(type) {
			case Marshaler:
			default:
				tv = TaggedValue{Tag: TagBatchCount, Value: ttv}
			}

			b, err := Marshal(tv)
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
