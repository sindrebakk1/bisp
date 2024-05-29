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
	String string
	bisp.Procedure[string]
}
type TestProcedureInt struct {
	Int int
	bisp.Procedure[int]
}
type TestProcedureSlice struct {
	Slice []int
	bisp.Procedure[[]int]
}
type TestProcedureArray struct {
	Slice [3]int
	bisp.Procedure[[3]int]
}
type TestProcedureStruct struct {
	Struct PStruct
	bisp.Procedure[PStruct]
}
type TestProcedureMap struct {
	Map map[string]int
	bisp.Procedure[map[string]int]
}
type TestProcedureEnum struct {
	Enum TestEnum
	bisp.Procedure[TestEnum]
}
type TestProcedureMultipleParams struct {
	Int    int
	String string
	Bool   bool
	Slice  []int
	Array  [3]int
	Struct PStruct
	Map    map[string]int
	Enum   TestEnum
	bisp.Procedure[string]
}

var (
	pString = TestProcedureString{
		String: "Hello",
		Procedure: bisp.Procedure[string]{
			Out: "World",
		},
	}
	pInt = TestProcedureInt{
		Procedure: bisp.Procedure[int]{
			Out: 42,
		},
	}
	pSlice = TestProcedureSlice{
		Slice: []int{1, 2, 3},
		Procedure: bisp.Procedure[[]int]{
			Out: []int{4, 5, 6},
		},
	}
	pArray = TestProcedureArray{
		Slice: [3]int{1, 2, 3},
		Procedure: bisp.Procedure[[3]int]{
			Out: [3]int{4, 5, 6},
		},
	}
	pStruct = TestProcedureStruct{
		Struct: PStruct{A: 1, B: "a", C: true},
		Procedure: bisp.Procedure[PStruct]{
			Out: PStruct{A: 1, B: "a", C: true},
		},
	}
	pMap = TestProcedureMap{
		Map: map[string]int{"a": 1},
		Procedure: bisp.Procedure[map[string]int]{
			Out: map[string]int{"a": 1},
		},
	}
	pEnum = TestProcedureEnum{
		Enum: TestEnum1,
		Procedure: bisp.Procedure[TestEnum]{
			Out: TestEnum2,
		},
	}
	pMultipleParams = TestProcedureMultipleParams{
		Int:    42,
		String: "Hello",
		Bool:   true,
		Slice:  []int{1, 2, 3},
		Array:  [3]int{4, 5, 6},
		Struct: PStruct{A: 1, B: "a", C: true},
		Map:    map[string]int{"a": 1},
		Enum:   TestEnum1,
		Procedure: bisp.Procedure[string]{
			Out: "World",
		},
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
		String: "Hello",
		Procedure: bisp.Procedure[string]{
			TransactionID: testTransactionID,
			Out:           "World",
		},
	}
	t.Run("call", func(t *testing.T) {
		buf := new(bytes.Buffer)
		enc := bisp.NewEncoder(buf)
		err := enc.EncodeProcedure(pWithTransactionID, bisp.Call)
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
		err := enc.EncodeProcedure(pWithTransactionID, bisp.Response)
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
		{name: "string", value: pString},
		{name: "int", value: pInt},
		{name: "slice", value: pSlice},
		{name: "array", value: pArray},
		{name: "struct", value: pStruct},
		{name: "map", value: pMap},
		{name: "enum", value: pEnum},
		{name: "multiple", value: pMultipleParams},
	}

	testDecodeProcedures(t, tcs, bisp.Call)
}

func encodeProcedure(t *testing.T, p any, kind bisp.PKind) []byte {
	buf := new(bytes.Buffer)
	enc := bisp.NewEncoder(buf)
	err := enc.EncodeProcedure(p, kind)
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func testEncodeProcedures(t *testing.T, tcs []testCase, kind bisp.PKind) {
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
			assert.Equal(t, tc.value, msg.Body)
		})
	}
}

func encodeTestProcedure(t *testing.T, p any, kind bisp.PKind, tID bool) []byte {
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
	} else {
		for i := range pType.NumField() {
			fType := pType.Field(i)
			if fType.Name == "Procedure" || fType.Name == "TransactionID" || fType.Name == "Out" {
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
		Version:       bisp.V1,
		Flags:         bisp.FProcedure,
		TransactionID: testTransactionID,
		Type:          pID,
		Length:        bisp.Length(buf.Len()),
	}
	headerBytes := encodeTestHeader(&header, false, tID)
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
