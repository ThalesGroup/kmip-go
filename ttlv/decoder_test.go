package ttlv

import (
	"bytes"
	"errors"
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

			err := Unmarshal(Hex2bytes(sample.exp), v)
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
			in:  CredentialTypeAttestation,
			ptr: new(int64),
			err: ErrUnsupportedTypeError,
		},
		{
			in:       Value{Tag: TagBatchCount, Value: "red"},
			ptr:      new(interface{}),
			expected: "red",
		},
		{
			name: "structtypeerror",
			in: Value{Tag: TagBatchCount, Value: Values{
				Value{Tag: TagComment, Value: "red"},
			}},
			ptr: new(int),
			err: ErrUnsupportedTypeError,
		},
		{
			name: "ttlvStructure",
			in: Value{Tag: TagBatchCount, Value: Values{
				Value{Tag: TagBatchItem, Value: "red"},
				Value{Tag: TagBatchContinueCapability, Value: "blue"},
			}},
			ptr: new(Value),
			expected: Value{Tag: TagBatchCount, Value: Values{
				Value{Tag: TagBatchItem, Value: "red"},
				Value{Tag: TagBatchContinueCapability, Value: "blue"},
			}},
		},
	}

	type A struct {
		Comment string
	}
	tests = append(tests,
		unmarshalTest{
			name: "simplestruct",
			in: Value{Tag: TagBatchCount, Value: Values{
				Value{Tag: TagComment, Value: "red"},
			}},
			ptr:      new(A),
			expected: A{Comment: "red"},
		},
	)

	type B struct {
		S string `ttlv:"Comment"`
	}

	tests = append(tests,
		unmarshalTest{
			name: "fieldtag",
			in: Value{Tag: TagBatchCount, Value: Values{
				Value{Tag: TagComment, Value: "red"},
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
			in: Value{Tag: TagBatchCount, Value: Values{
				Value{Tag: TagComment, Value: "red"},
				Value{Tag: TagBatchCount, Value: "blue"},
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
			in: Value{Tag: TagBatchCount, Value: Values{
				Value{Tag: TagComment, Value: "red"},
				Value{Tag: TagBatchCount, Value: Values{
					Value{Tag: TagComment, Value: "blue"},
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
			in: Value{Tag: TagBatchCount, Value: Values{
				Value{Tag: TagComment, Value: "red"},
				Value{Tag: TagBatchCount, Value: Values{
					Value{Tag: TagComment, Value: "blue"},
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
			in: Value{Tag: TagBatchCount, Value: Values{
				Value{Tag: TagComment, Value: "red"},
				Value{Tag: TagComment, Value: "blue"},
				Value{Tag: TagComment, Value: "green"},
			}},
			ptr:      new(G),
			expected: G{Comment: []string{"red", "blue", "green"}},
		},
	)

	type H struct {
		Comment        string
		Any1           []Value `ttlv:",any"`
		Any2           []Value `ttlv:",any"`
		AttributeValue string
	}

	tests = append(tests,
		unmarshalTest{
			name: "anyflag",
			in: Value{Value: Values{
				Value{Tag: TagComment, Value: "red"},
				Value{Tag: TagNameType, Value: "blue"},
				Value{Tag: TagName, Value: "orange"},
				Value{Tag: TagAttributeValue, Value: "yellow"},
			}},
			ptr: new(H),
			expected: H{Comment: "red", AttributeValue: "yellow", Any1: []Value{
				{Tag: TagNameType, Value: "blue"},
				{Tag: TagName, Value: "orange"},
			}},
		})

	for _, test := range tests {
		if test.name == "" {
			test.name = fmt.Sprintf("%T into %T", test.in, test.ptr)
		}
		t.Run(test.name, func(t *testing.T) {

			b, err := Marshal(Value{Tag: TagBatchCount, Value: test.in})
			require.NoError(t, err)

			t.Log(b.String())

			v := reflect.New(reflect.TypeOf(test.ptr).Elem())
			expected := test.expected
			if expected == nil {
				expected = test.in
			}

			err = Unmarshal(b, v.Interface())

			if test.err != nil {
				require.Error(t, err, "got value instead: %#v", v.Elem().Interface())
				require.True(t, errors.Is(err, test.err), Details(err))
				return
			}

			require.NoError(t, err, Details(err))
			require.Equal(t, expected, v.Elem().Interface())

			// if out type is not a slice, add a test for unmarshaling into
			// a slice of that type, which should work too.  e.g.  you should
			// be able to unmarshal a bool into either bool or []bool

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
				bb, err := Marshal(Value{Tag: TagBatchCount, Value: v.Elem().Interface()})
				require.NoError(t, err)

				if !test.skipExactRoundtripTest {
					assert.Equal(t, b.String(), bb.String())
				}

				t.Log(bb.String())

				vv := reflect.New(reflect.TypeOf(test.ptr).Elem())
				err = Unmarshal(bb, vv.Interface())
				require.NoError(t, err)

				assert.Equal(t, v.Elem().Interface(), vv.Elem().Interface())
			})
		})

	}

}

func TestUnmarshal_tagfield(t *testing.T) {
	// tests unmarshaling into structs which contain
	// a TTLVTag field.  Unmarshal should record the tag of the value
	// in this field

	b, err := Marshal(Value{TagComment, Values{{TagName, "red"}}})
	require.NoError(t, err)

	type M struct {
		TTLVTag Tag
		Name    string
	}

	var m M

	err = Unmarshal(b, &m)
	require.NoError(t, err)

	assert.Equal(t, M{TagComment, "red"}, m)

}

func TestUnmarshal_tagPrecedence(t *testing.T) {
	// tests the order of precedence for matching a field
	// to a ttlv

	b, err := Marshal(Value{TagComment, Values{{TagName, "red"}}})
	require.NoError(t, err)

	// lowest precedence: the name of the field
	type A struct {
		Name string
	}

	var a A

	err = Unmarshal(b, &a)
	require.NoError(t, err)

	assert.EqualValues(t, "red", a.Name)

	// next: the TTLVTag tag of the struct field

	type B struct {
		N struct {
			TTLVTag string `ttlv:"Name"`
			Value
		}
	}

	var bb B

	err = Unmarshal(b, &bb)
	require.NoError(t, err)

	assert.EqualValues(t, "red", bb.N.Value.Value)

	// next: the field's tag

	type C struct {
		N string `ttlv:"Name"`
	}

	var c C

	err = Unmarshal(b, &c)
	require.NoError(t, err)

	assert.EqualValues(t, "red", c.N)

	// conflicts of these result in errors
	cases := []struct {
		name    string
		v       interface{}
		allowed bool
	}{
		{name: "field name and field tag", v: &struct {
			Name string
			N    string `ttlv:"Name"`
		}{}},
		{name: "field name and TTLVTag", v: &struct {
			Name string
			N    struct {
				TTLVTag string `ttlv:"Name"`
				Value
			}
		}{}},
		{name: "field tag and TTLVTag", v: &struct {
			S string `ttlv:"Name"`
			N struct {
				TTLVTag string `ttlv:"Name"`
				Value
			}
		}{}},
		{name: "field tag and TTLVTag", v: &struct {
			N struct {
				TTLVTag string `ttlv:"Name"`
				Value
			} `ttlv:"Comment"`
		}{}},
		{
			name: "field tag and TTLVTag",
			v: &struct {
				N struct {
					TTLVTag string `ttlv:"Name"`
					Value
				} `ttlv:"Name"`
			}{},
			allowed: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Unmarshal(b, tc.v)
			if tc.allowed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.True(t, merry.Is(err, ErrTagConflict), "%+v", err)
			}
		})
	}

	type D struct {
		Name string
		N    string `ttlv:"Name"`
	}

	err = Unmarshal(b, &D{})
	require.Error(t, err)
	assert.True(t, merry.Is(err, ErrTagConflict))
}

func TestDecoder_DisallowUnknownFields(t *testing.T) {
	type A struct {
		Comment    string
		BatchCount int
	}

	tests := []struct {
		name  string
		input Value
	}{
		{
			name: "middle",
			input: Value{
				TagAlternativeName,
				Values{
					{TagComment, "red"},
					{TagArchiveDate, "blue"},
					{TagBatchCount, 5},
				},
			},
		},
		{
			name: "first",
			input: Value{
				TagAlternativeName,
				Values{
					{TagArchiveDate, "blue"},
					{TagComment, "red"},
					{TagBatchCount, 5},
				},
			},
		},
		{
			name: "last",
			input: Value{
				TagAlternativeName,
				Values{
					{TagComment, "red"},
					{TagBatchCount, 5},
					{TagArchiveDate, "blue"},
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
