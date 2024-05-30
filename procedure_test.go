package bisp_test

import (
	"bytes"
	"github.com/sindrebakk1/bisp"
	"github.com/stretchr/testify/assert"
	"net"
	"reflect"
	"testing"
)

type PStruct struct {
	A int
	B string
	C bool
}
type TestProcedureString struct {
	bisp.Procedure[string]
	String string
}
type TestProcedureInt struct {
	bisp.Procedure[int]
	Int int
}
type TestProcedureSlice struct {
	bisp.Procedure[[]int]
	Slice []int
}
type TestProcedureArray struct {
	bisp.Procedure[[3]int]
	Slice [3]int
}
type TestProcedureStruct struct {
	bisp.Procedure[PStruct]
	Struct PStruct
}
type TestProcedureMap struct {
	bisp.Procedure[map[string]int]
	Map map[string]int
}
type TestProcedureEnum struct {
	bisp.Procedure[TestEnum]
	Enum TestEnum
}
type TestProcedureMultipleParams struct {
	bisp.Procedure[string]
	Int    int
	String string
	Bool   bool
	Slice  []int
	Array  [3]int
	Struct PStruct
	Map    map[string]int
	Enum   TestEnum
}

var (
	pString = TestProcedureString{
		Procedure: bisp.Procedure[string]{
			Out: "World",
		},
		String: "Hello",
	}
	pInt = TestProcedureInt{
		Procedure: bisp.Procedure[int]{
			Out: 42,
		},
		Int: 42,
	}
	pSlice = TestProcedureSlice{
		Procedure: bisp.Procedure[[]int]{
			Out: []int{4, 5, 6},
		},
		Slice: []int{1, 2, 3},
	}
	pArray = TestProcedureArray{
		Procedure: bisp.Procedure[[3]int]{
			Out: [3]int{4, 5, 6},
		},
		Slice: [3]int{1, 2, 3},
	}
	pStruct = TestProcedureStruct{
		Procedure: bisp.Procedure[PStruct]{
			Out: PStruct{A: 1, B: "a", C: true},
		},
		Struct: PStruct{A: 1, B: "a", C: true},
	}
	pMap = TestProcedureMap{
		Procedure: bisp.Procedure[map[string]int]{
			Out: map[string]int{"a": 1},
		},
		Map: map[string]int{"a": 1},
	}
	pEnum = TestProcedureEnum{
		Procedure: bisp.Procedure[TestEnum]{
			Out: TestEnum2,
		},
		Enum: TestEnum3,
	}
	pMultipleParams = TestProcedureMultipleParams{
		Procedure: bisp.Procedure[string]{
			Out: "World",
		},
		Int:    42,
		String: "Hello",
		Bool:   true,
		Slice:  []int{1, 2, 3},
		Array:  [3]int{4, 5, 6},
		Struct: PStruct{A: 1, B: "a", C: true},
		Map:    map[string]int{"a": 1},
		Enum:   TestEnum2,
	}
	testTransactionID = bisp.TransactionID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
)

func TestEncodeProcedure_Call(t *testing.T) {
	tcs := []testCase{
		{name: "string", value: pString},
		{name: "int", value: pInt},
		{name: "slice", value: pSlice},
		{name: "array", value: pArray},
		{name: "struct", value: pStruct},
		{name: "map", value: pMap},
		{name: "enum", value: pEnum},
		{name: "multiple", value: pMultipleParams},
	}

	testEncodeProcedures(t, tcs, bisp.Call)
}

func TestEncodeProcedure_Response(t *testing.T) {
	tcs := []testCase{
		{name: "string", value: pString},
		{name: "int", value: pInt},
		{name: "slice", value: pSlice},
		{name: "array", value: pArray},
		{name: "struct", value: pStruct},
		{name: "map", value: pMap},
		{name: "enum", value: pEnum},
		{name: "multiple", value: pMultipleParams},
	}

	testEncodeProcedures(t, tcs, bisp.Response)
}

func TestEncodeProcedure_TransactionID(t *testing.T) {
	pWithTransactionID := TestProcedureString{
		bisp.Procedure[string]{
			Out: "World",
		},
		"Hello",
	}
	opts := &bisp.EncodeProcedureOpts{
		TransactionID: testTransactionID,
	}
	t.Run("call", func(t *testing.T) {
		buf := new(bytes.Buffer)
		enc := bisp.NewEncoder(buf)
		err := enc.EncodeProcedure(pWithTransactionID, bisp.Call, opts)
		if err != nil {
			t.Fatal(err)
		}
		res := buf.Bytes()
		expected := encodeTestProcedure(t, pWithTransactionID, bisp.Call, true)
		assert.Equal(t, expected, res)
	})
	t.Run("response", func(t *testing.T) {
		buf := new(bytes.Buffer)
		enc := bisp.NewEncoder(buf)
		err := enc.EncodeProcedure(pWithTransactionID, bisp.Response, opts)
		if err != nil {
			t.Fatal(err)
		}
		res := buf.Bytes()
		expected := encodeTestProcedure(t, pWithTransactionID, bisp.Response, true)
		assert.Equal(t, expected, res)
	})
}

func TestDecodeProcedure_Call(t *testing.T) {
	tcs := []testCase{
		{name: "string", value: pString, expected: TestProcedureString{Procedure: bisp.Procedure[string]{Kind: bisp.Call}, String: "Hello"}},
		{name: "int", value: pInt, expected: TestProcedureInt{Procedure: bisp.Procedure[int]{Kind: bisp.Call}, Int: 42}},
		{name: "slice", value: pSlice, expected: TestProcedureSlice{Procedure: bisp.Procedure[[]int]{Kind: bisp.Call}, Slice: []int{1, 2, 3}}},
		{name: "array", value: pArray, expected: TestProcedureArray{Procedure: bisp.Procedure[[3]int]{Kind: bisp.Call}, Slice: [3]int{1, 2, 3}}},
		{name: "struct", value: pStruct, expected: TestProcedureStruct{Procedure: bisp.Procedure[PStruct]{Kind: bisp.Call}, Struct: PStruct{A: 1, B: "a", C: true}}},
		{name: "map", value: pMap, expected: TestProcedureMap{Procedure: bisp.Procedure[map[string]int]{Kind: bisp.Call}, Map: map[string]int{"a": 1}}},
		{name: "enum", value: pEnum, expected: TestProcedureEnum{Procedure: bisp.Procedure[TestEnum]{Kind: bisp.Call}, Enum: 0x2}},
		{name: "multiple", value: pMultipleParams, expected: TestProcedureMultipleParams{Procedure: bisp.Procedure[string]{Kind: bisp.Call}, Int: 42, String: "Hello", Bool: true, Slice: []int{1, 2, 3}, Array: [3]int{4, 5, 6}, Struct: PStruct{A: 1, B: "a", C: true}, Map: map[string]int{"a": 1}, Enum: TestEnum2}},
	}

	testDecodeProcedures(t, tcs, bisp.Call)
}

func TestDecodeProcedure_Response(t *testing.T) {
	tcs := []testCase{
		{name: "string", value: pString, expected: TestProcedureString{Procedure: bisp.Procedure[string]{Kind: bisp.Response, Out: "World"}}},
		{name: "int", value: pInt, expected: TestProcedureInt{Procedure: bisp.Procedure[int]{Kind: bisp.Response, Out: 42}}},
		{name: "slice", value: pSlice, expected: TestProcedureSlice{Procedure: bisp.Procedure[[]int]{Kind: bisp.Response, Out: []int{4, 5, 6}}}},
		{name: "array", value: pArray, expected: TestProcedureArray{Procedure: bisp.Procedure[[3]int]{Kind: bisp.Response, Out: [3]int{4, 5, 6}}}},
		{name: "struct", value: pStruct, expected: TestProcedureStruct{Procedure: bisp.Procedure[PStruct]{Kind: bisp.Response, Out: PStruct{A: 1, B: "a", C: true}}}},
		{name: "map", value: pMap, expected: TestProcedureMap{Procedure: bisp.Procedure[map[string]int]{Kind: bisp.Response, Out: map[string]int{"a": 1}}}},
		{name: "enum", value: pEnum, expected: TestProcedureEnum{Procedure: bisp.Procedure[TestEnum]{Kind: bisp.Response, Out: 0x1}, Enum: 0x0}},
		{name: "multiple", value: pMultipleParams, expected: TestProcedureMultipleParams{Procedure: bisp.Procedure[string]{Kind: bisp.Response, Out: "World"}}},
	}

	testDecodeProcedures(t, tcs, bisp.Response)
}

func TestTDecodeProcedure_Call(t *testing.T) {
	p := TestProcedureMultipleParams{
		Procedure: bisp.Procedure[string]{
			Out: "World",
		},
		Int:    42,
		String: "Hello",
		Bool:   true,
		Slice:  []int{1, 2, 3},
		Array:  [3]int{4, 5, 6},
		Struct: PStruct{A: 1, B: "a", C: true},
		Map:    map[string]int{"a": 1},
		Enum:   TestEnum2,
	}
	expected := TestProcedureMultipleParams{
		Procedure: bisp.Procedure[string]{
			Kind: bisp.Call,
		},
		Int:    42,
		String: "Hello",
		Bool:   true,
		Slice:  []int{1, 2, 3},
		Array:  [3]int{4, 5, 6},
		Struct: PStruct{A: 1, B: "a", C: true},
		Map:    map[string]int{"a": 1},
		Enum:   TestEnum2,
	}
	encoded := encodeTestProcedure(t, p, bisp.Call, false)

	client, server := net.Pipe()
	go func() {
		_, err := server.Write(encoded)
		assert.Nil(t, err)
		server.Close()
	}()

	decoder := bisp.NewDecoder(client)
	msg, err := bisp.TDecodeProcedure[TestProcedureMultipleParams](decoder)
	assert.Nil(t, err)
	assert.Equal(t, expected.Procedure.Kind, msg.Body.Kind)
	assert.Equal(t, expected, msg.Body)
}

func TestTDecodeProcedure_Receive(t *testing.T) {
	p := TestProcedureMultipleParams{
		Procedure: bisp.Procedure[string]{
			Out: "World",
		},
		Int:    42,
		String: "Hello",
		Bool:   true,
		Slice:  []int{1, 2, 3},
		Array:  [3]int{4, 5, 6},
		Struct: PStruct{A: 1, B: "a", C: true},
		Map:    map[string]int{"a": 1},
		Enum:   TestEnum2,
	}
	expected := TestProcedureMultipleParams{
		Procedure: bisp.Procedure[string]{
			Kind: bisp.Response,
			Out:  "World",
		},
	}
	encoded := encodeTestProcedure(t, p, bisp.Response, false)

	client, server := net.Pipe()
	go func() {
		_, err := server.Write(encoded)
		assert.Nil(t, err)
		server.Close()
	}()

	decoder := bisp.NewDecoder(client)
	msg, err := bisp.TDecodeProcedure[TestProcedureMultipleParams](decoder)
	assert.Nil(t, err)
	assert.Equal(t, expected.Procedure.Kind, msg.Body.Kind)
	assert.Equal(t, expected, msg.Body)
}

func encodeProcedure(t *testing.T, p any, kind bisp.PKind) []byte {
	buf := new(bytes.Buffer)
	enc := bisp.NewEncoder(buf)
	err := enc.EncodeProcedure(p, kind, nil)
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func testEncodeProcedures(t *testing.T, tcs []testCase, kind bisp.PKind) {
	if kind == bisp.Unknown {
		t.Fatal("Unknown procedure kind")
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			expected := encodeTestProcedure(t, tc.value, kind, false)
			res := encodeProcedure(t, tc.value, kind)
			assert.Equal(t, expected, res)
		})
	}
}

func testDecodeProcedures(t *testing.T, tcs []testCase, kind bisp.PKind) {
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			encoded := encodeTestProcedure(t, tc.value, kind, false)

			client, server := net.Pipe()
			go func() {
				_, err := server.Write(encoded)
				assert.Nil(t, err)
				server.Close()
			}()

			decoder := bisp.NewDecoder(client)
			var msg bisp.Message
			err := decoder.Decode(&msg)
			assert.Nil(t, err)
			expected := tc.value
			if tc.expected != nil {
				expected = tc.expected
			}
			assert.Equal(t, expected, msg.Body)
		})
	}
}

func encodeTestProcedure(t *testing.T, p any, kind bisp.PKind, transaction bool) []byte {
	pID, err := bisp.GetProcedureID(p)
	if err != nil {
		t.Fatal(err)
	}
	pVal := reflect.ValueOf(p)
	pType := pVal.Type()
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(kind))
	var expectedBytes []byte
	if kind == bisp.Response {
		out := pVal.FieldByName("Out")
		expectedBytes, err = encodeTestValue(out.Interface(), false)
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(expectedBytes)
	} else if kind == bisp.Call {
		for i := range pType.NumField() {
			fType := pType.Field(i)
			if fType.Name == "Procedure" {
				continue
			}
			field := pVal.FieldByName(fType.Name)
			expectedBytes, err = encodeTestValue(field.Interface(), false)
			if err != nil {
				t.Fatal(err)
			}
			buf.Write(expectedBytes)
		}
	}
	header := bisp.Header{
		Version:       bisp.CurrentVersion,
		Flags:         bisp.FProcedure,
		TransactionID: testTransactionID,
		Type:          pID,
		Length:        bisp.Length(buf.Len()),
	}
	headerBytes := encodeTestHeader(&header, false, transaction)
	headerBytes = append(headerBytes, buf.Bytes()...)
	return headerBytes
}

func init() {
	bisp.RegisterProcedure[TestProcedureString]()
	bisp.RegisterProcedure[TestProcedureInt]()
	bisp.RegisterProcedure[TestProcedureSlice]()
	bisp.RegisterProcedure[TestProcedureStruct]()
	bisp.RegisterProcedure[TestProcedureArray]()
	bisp.RegisterProcedure[TestProcedureMap]()
	bisp.RegisterProcedure[TestProcedureEnum]()
	bisp.RegisterProcedure[TestProcedureMultipleParams]()
}
