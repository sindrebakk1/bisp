package bisp_test

import (
	"github.com/sindrebakk1/bisp"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestNewEncoder(t *testing.T) {
	_, server := net.Pipe()
	encoder := bisp.NewEncoder(server)
	assert.NotNil(t, encoder)
	server.Close()
}

func TestEncodeHeader(t *testing.T) {
	_, server := net.Pipe()

	header := bisp.Header{
		Version:       bisp.CurrentVersion,
		Flags:         bisp.FError | bisp.FHuff | bisp.FTransaction,
		Type:          1,
		TransactionID: bisp.TransactionID(make([]byte, bisp.TransactionIDSize)),
		Length:        0,
	}

	encoder := bisp.NewEncoder(server)
	bytes, err := encoder.EncodeHeader(&header, 1, 0)
	assert.NoError(t, err)

	expected := encodeTestHeader(&header, false, true)
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
		{value: float32(11.1), name: "float32"},
		{value: float64(12.2), name: "float64"},
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
	testCases := []testCase{
		{value: []int{1, 2, 3}, name: "int slice"},
		{value: []uint{4, 5, 6}, name: "uint slice"},
		{value: []float32{1.1, 2.2, 3.3}, name: "float32 slice"},
		{value: []float64{4.4, 5.5, 6.6}, name: "float64 slice"},
		{value: []string{"a", "b", "c"}, name: "string slice"},
		{value: []bool{true, false, true}, name: "bool slice"},
		{value: []testStruct{{1, "a", true}, {2, "b", false}}, name: "struct slice"},
	}
	testEncodeBody(t, testCases)
}

func TestEncodeBody_Array(t *testing.T) {
	testCases := []testCase{
		{value: [3]int{1, 2, 3}, name: "int array"},
		{value: [3]uint{4, 5, 6}, name: "uint array"},
		{value: [3]float32{1.1, 2.2, 3.3}, name: "float32 array"},
		{value: [3]float64{4.4, 5.5, 6.6}, name: "float64 array"},
		{value: [3]string{"a", "b", "c"}, name: "string array"},
		{value: [3]bool{true, false, true}, name: "bool array"},
		{value: [3]testStruct{{1, "a", true}, {2, "b", false}}, name: "struct array"},
	}
	testEncodeBody(t, testCases)
}

func TestEncodeBody_Struct(t *testing.T) {
	testCases := []testCase{
		{value: testStruct{1, "a", true}, name: "struct"},
		{value: testStructSliceField{[]int{1, 2, 3}}, name: "struct with slice"},
		{value: testStructStructField{testStruct{1, "a", true}, "b"}, name: "struct with struct"},
		{value: testStructStructFieldSliceField{[]testStruct{{1, "a", true}, {2, "b", false}}}, name: "struct with struct slice"},
		{value: testStructEmbeddedPrivateStruct{testStruct{1, "a", true}, "b"}, expected: testStructEmbeddedPrivateStruct{testStruct{}, "b"}, name: "struct with embedded private struct"},
		{value: testStructEmbeddedStruct{TestStruct{1, "a", true}, "b"}, name: "struct with embedded struct"},
		{value: testStructPrivateFields{1, "a", true}, expected: testStructPrivateFields{}, name: "struct with private fields"},
	}
	testEncodeBody(t, testCases)
}

func TestEncodeBody_Enum(t *testing.T) {
	testCases := []testCase{
		{value: TestEnum1, name: "TestEnum21"},
		{value: TestEnum2, name: "TestEnum22"},
		{value: TestEnum3, name: "TestEnum23"},
	}
	testEncodeBody(t, testCases)
}

func TestEncodeBody_Map(t *testing.T) {
	testCases := []testCase{
		{value: map[int]string{1: "a"}, name: "int > string map"},
		{value: map[TestEnum]string{TestEnum1: "a"}, name: "enum > string map"},
		{value: map[string]int{"a": 1}, name: "string > int map"},
	}
	testEncodeBody(t, testCases)
}

func TestEncodeMessage_String(t *testing.T) {
	client, server := net.Pipe()
	body := "Hello"
	typeID, err := bisp.GetIDFromType(body)
	assert.NoError(t, err)
	message := &bisp.Message{
		Header: bisp.Header{
			Version:       bisp.CurrentVersion,
			Flags:         bisp.FTransaction,
			Type:          typeID,
			TransactionID: bisp.TransactionID(make([]byte, bisp.TransactionIDSize)),
			Length:        bisp.Length(2 + len(body)),
		},
		Body: body,
	}
	go func() {
		encoder := bisp.NewEncoder(server)
		err = encoder.Encode(message)
		assert.NoError(t, err)
		server.Close()
	}()
	expected := encodeTestHeader(&message.Header, false, true)
	var b []byte
	b, err = encodeTestValue("Hello", false)
	if err != nil {
		t.Fatal(err)
	}
	expected = append(expected, b...)
	bytes := make([]byte, len(expected))
	_, err = client.Read(bytes)
	assert.NoError(t, err)
	assert.Equal(t, expected, bytes)
}

func TestEncodeMessage_32bLengths(t *testing.T) {
	client, server := net.Pipe()
	var body string
	l := bisp.MaxTcpMessageBodySize * 2
	for i := 0; i < l; i++ {
		body += "a"
	}
	typeID, err := bisp.GetIDFromType(body)
	assert.NoError(t, err)
	msg := bisp.Message{
		Header: bisp.Header{
			Version: bisp.CurrentVersion,
			Flags:   bisp.F32b | bisp.FTransaction,
			Type:    typeID,
			Length:  bisp.Length(len(body) + 4),
		},
		Body: body,
	}
	go func() {
		encoder := bisp.NewEncoder(server)
		err = encoder.Encode(&msg)
		assert.NoError(t, err)
		server.Close()
	}()
	expected := encodeTestHeader(&msg.Header, true, true)
	bytes := make([]byte, len(body)+4+len(expected))
	_, err = client.Read(bytes)
	assert.NoError(t, err)
	var b []byte
	b, err = encodeTestValue(body, true)
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

			encoder := bisp.NewEncoder(server)

			err := encoder.EncodeBody(tc.value, false)
			assert.NoError(t, err)

			expected := tc.value
			if tc.expected != nil {
				expected = tc.expected
			}
			var expectedBytes []byte
			expectedBytes, err = encodeTestValue(expected, false)

			bytes := encoder.Bytes()
			assert.Equal(t, expectedBytes, bytes)

			server.Close()
		})
	}
}
