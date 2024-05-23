package bisp_test

import (
	"github.com/sindrebakk1/bisp"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestEncodeDecodeMessage_String(t *testing.T) {
	tcs := []testCase{
		{
			value: bisp.Message{
				Body: "Hello",
			},
			name: "Hello",
		},
		{
			value: bisp.Message{
				Body: "World",
			},
			name: "World",
		},
		{
			value: bisp.Message{
				Body: "GoLang!%&",
			},
			name: "GoLang",
		},
		{
			value: bisp.Message{
				Body: "12345",
			},
			name: "number string",
		},
		{
			value: bisp.Message{
				Body: "",
			},
			name: "empty string",
		},
	}
	testEncodeDecodeMessages(t, tcs)
}

func TestEncodeDecodeMessage_Numbers(t *testing.T) {
	tcs := []testCase{
		{
			value: bisp.Message{
				Body: 1234,
			},
			name: "int",
		},
		{
			value: bisp.Message{
				Body: uint8(123),
			},
			name: "uint8",
		},
		{
			value: bisp.Message{
				Body: uint16(1234),
			},
			name: "uint16",
		},
		{
			value: bisp.Message{
				Body: uint32(1234),
			},
			name: "uint32",
		},
		{
			value: bisp.Message{
				Body: uint64(1234),
			},
			name: "uint64",
		},
		{
			value: bisp.Message{
				Body: int8(123),
			},
			name: "int8",
		},
		{
			value: bisp.Message{
				Body: int16(1234),
			},
			name: "int16",
		},
		{
			value: bisp.Message{
				Body: int32(12345),
			},
			name: "int32",
		},
		{
			value: bisp.Message{
				Body: int64(123456),
			},
			name: "int64",
		},
		{
			value: bisp.Message{
				Body: float32(1234.567),
			},
			name: "float32",
		},
		{
			value: bisp.Message{
				Body: uint16(1234),
			},
			name: "uint16",
		},
		{
			value: bisp.Message{
				Body: float64(1234.567),
			},
			name: "float64",
		},
	}
	testEncodeDecodeMessages(t, tcs)
}

func TestEncodeDecodeMessage_Boolean(t *testing.T) {
	tcs := []testCase{
		{
			value: bisp.Message{
				Body: false,
			},
			name: "false",
		},
		{
			value: bisp.Message{
				Body: true,
			},
			name: "true",
		},
	}
	testEncodeDecodeMessages(t, tcs)
}

func TestEncodeDecodeMessage_Slice(t *testing.T) {
	type testStruct3 struct {
		A int
		B string
		C bool
	}
	bisp.RegisterType(testStruct3{})
	tcs := []testCase{
		{
			value: bisp.Message{
				Body: []int{1, 2, 3},
			},
			name: "int slice",
		},
		{
			value: bisp.Message{
				Body: []uint{4, 5, 6},
			},
			name: "uint slice",
		},
		{
			value: bisp.Message{
				Body: []float32{1.1, 2.2, 3.3},
			},
			name: "float32 slice",
		},
		{
			value: bisp.Message{
				Body: []float64{4.4, 5.5, 6.6},
			},
			name: "float64 slice",
		},
		{
			value: bisp.Message{
				Body: []string{"a", "b", "c"},
			},
			name: "string slice",
		},
		{
			value: bisp.Message{
				Body: []bool{true, false, true},
			},
			name: "bool slice",
		},
		{
			value: bisp.Message{
				Body: []testStruct3{{1, "a", true}, {2, "B", false}},
			},
			name: "struct slice",
		},
	}
	testEncodeDecodeMessages(t, tcs)
}

func TestEncodeDecodeMessage_Struct(t *testing.T) {
	type testStruct3 struct {
		A int
		B string
		C bool
	}
	type TestStruct3 struct {
		A int
		B string
		C bool
	}
	type testStructSliceField3 struct {
		Slice []int
	}
	type testStructStructField3 struct {
		Struct testStruct3
		B      string
	}
	type testStructStructFieldSliceField3 struct {
		Structs []testStruct3
	}
	type testStructEmbeddedPrivateStruct3 struct {
		testStruct3
		B string
	}
	type testStructEmbeddedStruct3 struct {
		TestStruct3
		B string
	}
	type testStructPrivateFields3 struct {
		a int
		b string
		c bool
	}
	bisp.RegisterType(testStruct3{})
	bisp.RegisterType(testStructSliceField3{})
	bisp.RegisterType(testStructStructField3{})
	bisp.RegisterType(testStructStructFieldSliceField3{})
	bisp.RegisterType(testStructEmbeddedPrivateStruct3{})
	bisp.RegisterType(testStructEmbeddedStruct3{})
	bisp.RegisterType(testStructPrivateFields3{})

	var (
		testStructEmbeddedPrivateStruct3ID bisp.TypeID
		testStructPrivateFields3ID         bisp.TypeID
		err                                error
	)
	testStructEmbeddedPrivateStruct3ID, err = bisp.GetIDFromType(testStructEmbeddedPrivateStruct3{})
	assert.NoError(t, err)
	testStructPrivateFields3ID, err = bisp.GetIDFromType(testStructPrivateFields3{})
	assert.NoError(t, err)

	tcs := []testCase{
		{
			value: bisp.Message{
				Body: testStruct3{1, "a", true},
			},
			name: "struct",
		},
		{
			value: bisp.Message{
				Body: testStructSliceField3{[]int{1, 2, 3}},
			},
			name: "struct with slice",
		},
		{
			value: bisp.Message{
				Body: testStructStructField3{testStruct3{1, "a", true}, "b"},
			},
			name: "struct with struct",
		},
		{
			value: bisp.Message{
				Body: testStructStructFieldSliceField3{[]testStruct3{{1, "a", true}, {1, "a", true}}},
			},
			name: "struct with struct slice",
		},
		{
			value: bisp.Message{
				Body: testStructEmbeddedPrivateStruct3{testStruct3{1, "a", true}, "b"},
			},
			expected: bisp.Message{
				Header: bisp.Header{
					Version: bisp.V1,
					Type:    testStructEmbeddedPrivateStruct3ID,
					Length:  0x3,
				},
				Body: testStructEmbeddedPrivateStruct3{testStruct3{}, "b"},
			},
			name: "struct with embedded private struct",
		},
		{
			value: bisp.Message{
				Body: testStructEmbeddedStruct3{TestStruct3{1, "a", true}, "b"},
			},
			name: "struct with embedded struct",
		},
		{
			value: bisp.Message{
				Body: testStructPrivateFields3{1, "a", true},
			},
			expected: bisp.Message{
				Header: bisp.Header{
					Version: bisp.V1,
					Type:    testStructPrivateFields3ID,
					Length:  0,
				},
				Body: testStructPrivateFields3{},
			},
			name: "struct with private fields",
		},
	}
	testEncodeDecodeMessages(t, tcs)
}

func TestEncodeDecodeMessage_Header(t *testing.T) {
	tcs := []testCase{
		{
			value: bisp.Message{
				Header: bisp.Header{
					Version:       bisp.V1,
					Flags:         bisp.FTransactionID | bisp.FError,
					TransactionID: bisp.TransactionID(make([]byte, bisp.TransactionIDSize)),
					Type:          0,
					Length:        0,
				},
				Body: "Hello",
			},
			name: "all fields",
		},
		{
			value: bisp.Message{
				Header: bisp.Header{
					TransactionID: bisp.TransactionID(make([]byte, bisp.TransactionIDSize)),
				},
				Body: "Hello",
			},
			name: "transaction id",
		},
		{
			value: bisp.Message{
				Header: bisp.Header{},
				Body:   "Hello",
			},
			name: "empty",
		},
	}
	testEncodeDecodeMessages(t, tcs)
}

func TestEncodeDecodeMessage_Nil(t *testing.T) {
	tcs := []testCase{
		{
			value: bisp.Message{
				Body: nil,
			},
			name: "nil",
		},
		{
			value: bisp.Message{},
			name:  "empty initializer",
		},
	}
	testEncodeDecodeMessages(t, tcs)
}

func testEncodeDecodeMessages(t *testing.T, tcs []testCase) {
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			testEncodeMessage(t, tc)
		})
	}
}

func testEncodeMessage(t *testing.T, tc testCase) {
	msg := tc.value.(bisp.Message)
	client, server := net.Pipe()
	// Write to server
	go func() {
		encoder := bisp.NewEncoder(server)
		err := encoder.Encode(&msg)
		assert.NoError(t, err)
		server.Close()
	}()

	// Read from client
	decoder := bisp.NewDecoder(client)
	var res bisp.Message
	err := decoder.Decode(&res)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	expected := msg
	if tc.expected != nil {
		expected = tc.expected.(bisp.Message)
	}
	assert.Equal(t, expected, res)
}
