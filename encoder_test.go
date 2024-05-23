package tcp_test

import (
	"github.com/stretchr/testify/assert"
	"net"
	"tcp"
	"testing"
)

func TestNewEncoder(t *testing.T) {
	_, server := net.Pipe()
	encoder := tcp.NewEncoder(server)
	assert.NotNil(t, encoder)
	server.Close()
}

func TestEncodeHeader(t *testing.T) {
	_, server := net.Pipe()

	header := tcp.Header{
		Version:       tcp.V1,
		Flags:         tcp.FError | tcp.FHuff | tcp.FTransactionID,
		Type:          1,
		TransactionID: tcp.TransactionID(make([]byte, tcp.TransactionIDSize)),
		Length:        5,
	}

	encoder := tcp.NewEncoder(server)
	bytes, err := encoder.EncodeHeader(&header)
	assert.NoError(t, err)

	expected := encodeTestHeader(&header)
	assert.Equal(t, expected, bytes)
	server.Close()
}

func TestEncodeBody_String(t *testing.T) {
	testCases := []testCase{
		{value: "Hello", name: "Hello"},
		{value: "World", name: "World"},
		{value: "GoLang!%&", name: "GoLang"},
		{value: "12345", name: "number string"},
		{value: "", name: "empty string"},
	}
	testEncodeBody(t, testCases)
}

func TestEncodeBody_Numbers(t *testing.T) {
	testCases := []testCase{
		{value: uint8(1), name: "uint8"},
		{value: uint16(2), name: "uint16"},
		{value: uint32(3), name: "uint32"},
		{value: uint64(4), name: "uint64"},
		{value: int8(5), name: "int8"},
		{value: int16(6), name: "int16"},
		{value: int32(7), name: "int32"},
		{value: int64(8), name: "int64"},
		{value: 9, name: "int"},
		{value: uint(10), name: "uint"},
		{value: float32(11.2), name: "float32"},
		{value: float64(12.3), name: "float64"},
	}
	testEncodeBody(t, testCases)
}

func TestEncodeBody_Boolean(t *testing.T) {
	testCases := []testCase{
		{value: true, name: "true"},
		{value: false, name: "false"},
	}
	testEncodeBody(t, testCases)
}

func TestEncodeBody_Slice(t *testing.T) {
	type testStruct2 struct {
		A int
		B string
		C bool
	}
	tcp.RegisterType(testStruct2{})
	testCases := []testCase{
		{value: []int{1, 2, 3}, name: "int slice"},
		{value: []uint{4, 5, 6}, name: "uint slice"},
		{value: []float32{1.1, 2.2, 3.3}, name: "float32 slice"},
		{value: []float64{4.4, 5.5, 6.6}, name: "float64 slice"},
		{value: []string{"a", "b", "c"}, name: "string slice"},
		{value: []bool{true, false, true}, name: "bool slice"},
		{value: []testStruct2{{1, "a", true}, {2, "b", false}}, name: "struct slice"},
	}
	testEncodeBody(t, testCases)
}

func TestEncodeBody_Struct(t *testing.T) {
	type testStruct2 struct {
		A int
		B string
		C bool
	}
	type TestStruct2 struct {
		A int
		B string
		C bool
	}
	type testStructSliceField2 struct {
		Slice []int
	}
	type testStructStructField2 struct {
		Struct testStruct2
		B      string
	}
	type testStructStructFieldSliceField2 struct {
		Structs []testStruct2
	}
	type testStructEmbeddedPrivateStruct2 struct {
		testStruct2
		B string
	}
	type testStructEmbeddedStruct2 struct {
		TestStruct2
		B string
	}
	type testStructPrivateFields2 struct {
		a int
		b string
		c bool
	}
	tcp.RegisterType(testStruct2{})
	tcp.RegisterType(testStructSliceField2{})
	tcp.RegisterType(testStructStructField2{})
	tcp.RegisterType(testStructStructFieldSliceField2{})
	tcp.RegisterType(testStructEmbeddedPrivateStruct2{})
	tcp.RegisterType(testStructEmbeddedStruct2{})
	tcp.RegisterType(testStructPrivateFields2{})
	testCases := []testCase{
		{value: testStruct2{1, "a", true}, name: "struct"},
		{value: testStructSliceField2{[]int{1, 2, 3}}, name: "struct with slice"},
		{value: testStructStructField2{testStruct2{1, "a", true}, "b"}, name: "struct with struct"},
		{value: testStructStructFieldSliceField2{[]testStruct2{{1, "a", true}, {2, "b", false}}}, name: "struct with struct slice"},
		{value: testStructEmbeddedPrivateStruct2{testStruct2{1, "a", true}, "b"}, expected: testStructEmbeddedPrivateStruct2{testStruct2{}, "b"}, name: "struct with embedded private struct"},
		{value: testStructEmbeddedStruct2{TestStruct2{1, "a", true}, "b"}, name: "struct with embedded struct"},
		{value: testStructPrivateFields2{1, "a", true}, expected: testStructPrivateFields2{}, name: "struct with private fields"},
	}
	testEncodeBody(t, testCases)
}

func TestEncodeMessage_String(t *testing.T) {
	client, server := net.Pipe()
	message := &tcp.Message{
		Header: tcp.Header{
			Flags:         tcp.FTransactionID,
			TransactionID: tcp.TransactionID(make([]byte, tcp.TransactionIDSize)),
		},
		Body: "Hello",
	}
	go func() {
		encoder := tcp.NewEncoder(server)
		err := encoder.Encode(message)
		assert.NoError(t, err)
		server.Close()
	}()
	bytes := make([]byte, 29)
	_, err := client.Read(bytes)
	assert.NoError(t, err)
	expected := encodeTestHeader(&message.Header)
	b, err := encodeTestValue("Hello")
	if err != nil {
		t.Fatal(err)
	}
	expected = append(expected, b...)
	assert.Equal(t, expected, bytes)
}

func testEncodeBody(t *testing.T, testCases []testCase) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, server := net.Pipe()

			encoder := tcp.NewEncoder(server)

			_, bytes, err := encoder.EncodeBody(tc.value)
			assert.NoError(t, err)

			expected := tc.value
			if tc.expected != nil {
				expected = tc.expected
			}
			var expectedBytes []byte
			expectedBytes, err = encodeTestValue(expected)
			assert.Equal(t, expectedBytes, bytes)

			server.Close()
		})
	}
}
