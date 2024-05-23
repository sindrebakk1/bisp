package tcp_test

import (
	"github.com/stretchr/testify/assert"
	"net"
	"tcp"
	"testing"
)

func TestNewDecoder(t *testing.T) {
	client, _ := net.Pipe()
	decoder := tcp.NewDecoder(client)
	assert.NotNil(t, decoder)
	client.Close()
}

func TestDecodeHeader(t *testing.T) {
	client, server := net.Pipe()

	header := &tcp.Header{
		Version:       tcp.V1,
		Flags:         tcp.FError | tcp.FHuff | tcp.FTransactionID,
		Type:          1,
		TransactionID: tcp.TransactionID(make([]byte, 32)),
		Length:        5,
	}

	go func() {
		encodedHeader := encodeTestHeader(header)
		_, err := server.Write(encodedHeader)
		assert.Nil(t, err)
		server.Close()
	}()

	decoder := tcp.NewDecoder(client)

	res, err := decoder.DecodeHeader()
	assert.Nil(t, err)
	assert.Equal(t, header, res)
	client.Close()
}

func TestDecodeBody_String(t *testing.T) {
	testCases := []testCase{
		{value: "Hello", name: "Hello"},
		{value: "World", name: "World"},
		{value: "GoLang!%&", name: "GoLang"},
		{value: "12345", name: "number string"},
		{value: "", name: "empty string"},
	}
	testDecodeBody(t, testCases)
}

func TestDecodeBody_Numbers(t *testing.T) {
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
	testDecodeBody(t, testCases)
}

func TestDecodeBody_Boolean(t *testing.T) {
	testCases := []testCase{
		{value: true, name: "true"},
		{value: false, name: "false"},
	}
	testDecodeBody(t, testCases)
}

func TestDecodeBody_Slice(t *testing.T) {
	type testStruct struct {
		A int
		B string
		C bool
	}
	tcp.RegisterType(testStruct{})
	testCases := []testCase{
		{value: []int{1, 2, 3}, name: "int slice"},
		{value: []uint{4, 5, 6}, name: "uint slice"},
		{value: []float32{1.1, 2.2, 3.3}, name: "float32 slice"},
		{value: []float64{4.4, 5.5, 6.6}, name: "float64 slice"},
		{value: []string{"a", "b", "c"}, name: "string slice"},
		{value: []bool{true, false, true}, name: "bool slice"},
		{value: []testStruct{{1, "a", true}, {2, "b", false}}, name: "struct slice"},
	}
	testDecodeBody(t, testCases)
}

func TestDecodeBody_Struct(t *testing.T) {
	type testStruct struct {
		A int
		B string
		C bool
	}
	type TestStruct struct {
		A int
		B string
		C bool
	}
	type testStructSliceField struct {
		Slice []int
	}
	type testStructStructField struct {
		Struct testStruct
		B      string
	}
	type testStructStructFieldSliceField struct {
		Structs []testStruct
	}
	type testStructEmbeddedPrivateStruct struct {
		testStruct
		B string
	}
	type testStructEmbeddedStruct struct {
		TestStruct
		B string
	}
	type testStructPrivateFields struct {
		a int
		b string
		c bool
	}
	tcp.RegisterType(testStruct{})
	tcp.RegisterType(testStructSliceField{})
	tcp.RegisterType(testStructStructField{})
	tcp.RegisterType(testStructStructFieldSliceField{})
	tcp.RegisterType(testStructEmbeddedPrivateStruct{})
	tcp.RegisterType(testStructEmbeddedStruct{})
	tcp.RegisterType(testStructPrivateFields{})
	testCases := []testCase{
		{value: testStruct{1, "a", true}, name: "struct"},
		{value: testStructSliceField{[]int{1, 2, 3}}, name: "struct with slice"},
		{value: testStructStructField{testStruct{1, "a", true}, "b"}, name: "struct with struct"},
		{value: testStructStructFieldSliceField{[]testStruct{{1, "a", true}, {2, "b", false}}}, name: "struct with struct slice"},
		{value: testStructEmbeddedPrivateStruct{testStruct{1, "a", true}, "b"}, expected: testStructEmbeddedPrivateStruct{testStruct{}, "b"}, name: "struct with embedded private struct"},
		{value: testStructEmbeddedStruct{TestStruct{1, "a", true}, "b"}, name: "struct with embedded struct"},
		{value: testStructPrivateFields{1, "a", true}, expected: testStructPrivateFields{}, name: "struct with private fields"},
	}
	testDecodeBody(t, testCases)
}

func TestDecodeMessage_String(t *testing.T) {
	testMsg := tcp.Message{
		Header: tcp.Header{
			Version:       tcp.V1,
			Flags:         tcp.FTransactionID,
			TransactionID: tcp.TransactionID(make([]byte, tcp.TransactionIDSize)),
		},
		Body: "Hello",
	}
	typeID, err := tcp.GetIDFromType(testMsg.Body)
	assert.Nil(t, err)
	bodyBytes, err := encodeTestValue("Hello")
	assert.Nil(t, err)
	testMsg.Header.Type = typeID
	testMsg.Header.Length = tcp.Length(len(bodyBytes))
	headerBytes := encodeTestHeader(&testMsg.Header)
	msgBytes := append(headerBytes, bodyBytes...)

	client, server := net.Pipe()

	go func() {
		_, err = server.Write(msgBytes)
		assert.Nil(t, err)
		server.Close()
	}()

	decoder := tcp.NewDecoder(client)
	var msg tcp.Message
	err = decoder.Decode(&msg)
	assert.Nil(t, err)
	assert.Equal(t, msg, testMsg)
	client.Close()
}

func testDecodeBody(t *testing.T, testCases []testCase) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := encodeTestValue(tc.value)
			assert.Nil(t, err)

			client, server := net.Pipe()
			go func() {
				_, err = server.Write(encoded)
				assert.Nil(t, err)
				server.Close()
			}()

			decoder := tcp.NewDecoder(client)

			var typeID tcp.TypeID
			typeID, err = tcp.GetIDFromType(tc.value)

			var res interface{}
			res, err = decoder.DecodeBody(typeID, uint16(len(encoded)))
			assert.Nil(t, err)
			expected := tc.value
			if tc.expected != nil {
				expected = tc.expected
			}
			assert.Equal(t, expected, res)
		})
	}
}
