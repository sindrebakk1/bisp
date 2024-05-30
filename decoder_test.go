package bisp_test

import (
	"github.com/sindrebakk1/bisp"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestNewDecoder(t *testing.T) {
	client, _ := net.Pipe()
	decoder := bisp.NewDecoder(client)
	assert.NotNil(t, decoder)
	client.Close()
}

func TestDecodeHeader(t *testing.T) {
	client, server := net.Pipe()

	header := &bisp.Header{
		Version:       bisp.V1,
		Flags:         bisp.FError | bisp.FHuff | bisp.FTransaction,
		Type:          1,
		TransactionID: bisp.TransactionID(make([]byte, 32)),
		Length:        5,
	}

	go func() {
		encodedHeader := encodeTestHeader(header, false, true)
		_, err := server.Write(encodedHeader)
		assert.Nil(t, err)
		server.Close()
	}()

	decoder := bisp.NewDecoder(client)

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

func TestDecodeBody_Array(t *testing.T) {
	testCases := []testCase{
		{value: [3]int{1, 2, 3}, name: "int array"},
		{value: [3]uint{4, 5, 6}, name: "uint array"},
		{value: [3]float32{1.1, 2.2, 3.3}, name: "float32 array"},
		{value: [3]float64{4.4, 5.5, 6.6}, name: "float64 array"},
		{value: [3]string{"a", "b", "c"}, name: "string array"},
		{value: [3]bool{true, false, true}, name: "bool array"},
		{value: [3]testStruct{{1, "a", true}, {2, "b", false}, {3, "c", true}}, name: "struct array"},
	}
	testDecodeBody(t, testCases)
}

func TestDecodeBody_Struct(t *testing.T) {
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

func TestDecodeBody_Map(t *testing.T) {
	testCases := []testCase{
		{value: map[int]string{1: "a", 2: "b", 3: "c"}, name: "int > string map"},
		{value: map[TestEnum]string{TestEnum1: "a", TestEnum2: "b", TestEnum3: "c"}, name: "enum > string map"},
		{value: map[string]int{"a": 1, "b": 2, "c": 3}, name: "string > int map"},
	}
	testDecodeBody(t, testCases)
}

func TestDecodeMessage_String(t *testing.T) {
	testMsg := bisp.Message{
		Header: bisp.Header{
			Version:       bisp.V1,
			Flags:         bisp.FTransaction,
			TransactionID: bisp.TransactionID(make([]byte, bisp.TransactionIDSize)),
		},
		Body: "Hello",
	}
	typeID, err := bisp.GetIDFromType(testMsg.Body)
	assert.Nil(t, err)
	bodyBytes, err := encodeTestValue("Hello", false)
	assert.Nil(t, err)
	testMsg.Header.Type = typeID
	testMsg.Header.Length = bisp.Length(len(bodyBytes))
	headerBytes := encodeTestHeader(&testMsg.Header, false, true)
	msgBytes := append(headerBytes, bodyBytes...)

	client, server := net.Pipe()

	go func() {
		_, err = server.Write(msgBytes)
		assert.Nil(t, err)
		server.Close()
	}()

	decoder := bisp.NewDecoder(client)
	var msg bisp.Message
	err = decoder.Decode(&msg)
	assert.Nil(t, err)
	assert.Equal(t, msg, testMsg)
	client.Close()
}

func TestDecodeMessage_32bLengths(t *testing.T) {
	var body string
	l := bisp.MaxTcpMessageBodySize * 2
	for i := 0; i < l; i++ {
		body += "a"
	}
	typeID, err := bisp.GetIDFromType(body)
	assert.Nil(t, err)
	testMsg := bisp.Message{
		Header: bisp.Header{
			Version:       bisp.V1,
			Flags:         bisp.FTransaction | bisp.F32b,
			TransactionID: bisp.TransactionID(make([]byte, bisp.TransactionIDSize)),
			Type:          typeID,
			Length:        bisp.Length(len(body) + 4),
		},
		Body: body,
	}
	bodyBytes, err := encodeTestValue(body, true)
	assert.Nil(t, err)
	headerBytes := encodeTestHeader(&testMsg.Header, true, true)
	msgBytes := append(headerBytes, bodyBytes...)

	client, server := net.Pipe()

	go func() {
		_, err = server.Write(msgBytes)
		assert.Nil(t, err)
		server.Close()
	}()

	decoder := bisp.NewDecoder(client)
	var msg bisp.Message
	err = decoder.Decode(&msg)
	assert.Nil(t, err)
	assert.Equal(t, msg, testMsg)
	client.Close()
}

func testDecodeBody(t *testing.T, testCases []testCase) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := encodeTestValue(tc.value, false)
			assert.Nil(t, err)

			client, server := net.Pipe()
			go func() {
				_, err = server.Write(encoded)
				assert.Nil(t, err)
				server.Close()
			}()

			decoder := bisp.NewDecoder(client)

			var typeID bisp.ID
			typeID, err = bisp.GetIDFromType(tc.value)
			assert.Nil(t, err)

			var res interface{}
			res, err = decoder.DecodeBody(typeID, uint32(len(encoded)), false)
			assert.Nil(t, err)
			expected := tc.value
			if tc.expected != nil {
				expected = tc.expected
			}
			assert.Equal(t, expected, res)
		})
	}
}
