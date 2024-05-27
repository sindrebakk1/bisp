package bisp_test

import (
	"bytes"
	"fmt"
	"github.com/sindrebakk1/bisp"
	"strings"
	"testing"
)

type benchCase struct {
	name string
	msg  *bisp.Message
}

type testBenchmarkStruct struct {
	A int
	B string
	C bool
}

func BenchmarkEncodeDecode(b *testing.B) {
	bcs := []benchCase{
		{
			name: "empty message",
			msg:  &bisp.Message{},
		},
		{
			name: "string message",
			msg:  &bisp.Message{Body: "Hello"},
		},
		{
			name: "int message",
			msg:  &bisp.Message{Body: 42},
		},
		{
			name: "bool message",
			msg:  &bisp.Message{Body: true},
		},
		{
			name: "struct message",
			msg:  &bisp.Message{Body: testBenchmarkStruct{A: 1, B: "a", C: true}},
		},
		{
			name: "slice message",
			msg:  &bisp.Message{Body: []int{1, 2, 3}},
		},
		{
			name: "array message",
			msg:  &bisp.Message{Body: [3]int{1, 2, 3}},
		},
		{
			name: "map message",
			msg:  &bisp.Message{Body: map[string]int{"a": 1, "b": 2, "c": 3}},
		},
		{
			name: "enum message",
			msg:  &bisp.Message{Body: TestEnum1},
		},
	}
	b.Run("encode", func(b *testing.B) {
		bench(b, bcs, benchEncodeMsg)
	})
	b.Run("decode", func(b *testing.B) {
		bench(b, bcs, benchDecodeMsg)
	})
}

func BenchmarkReceiveAndRespond(b *testing.B) {
	bcs := []benchCase{
		{
			name: "empty message",
			msg:  &bisp.Message{},
		},
		{
			name: "string message",
			msg:  &bisp.Message{Body: "Hello"},
		},
		{
			name: "int message",
			msg:  &bisp.Message{Body: 42},
		},
		{
			name: "bool message",
			msg:  &bisp.Message{Body: true},
		},
		{
			name: "struct message",
			msg:  &bisp.Message{Body: testBenchmarkStruct{A: 1, B: "a", C: true}},
		},
		{
			name: "slice message",
			msg:  &bisp.Message{Body: []int{1, 2, 3}},
		},
		{
			name: "array message",
			msg:  &bisp.Message{Body: [3]int{1, 2, 3}},
		},
		{
			name: "map message",
			msg:  &bisp.Message{Body: map[string]int{"a": 1, "b": 2, "c": 3}},
		},
		{
			name: "enum message",
			msg:  &bisp.Message{Body: TestEnum1},
		},
	}

	bench(b, bcs, benchReceiveMsgAndSendRes)
}

func BenchmarkBigString(b *testing.B) {
	stringSize := bisp.MaxTcpMessageBodySize - 2
	stringSize32 := (bisp.Max32bMessageBodySize - 4) / 8
	var bigString strings.Builder
	bigString.Grow(stringSize)
	for i := 0; i < stringSize; i++ {
		bigString.WriteByte('a')
	}
	var bigString32 strings.Builder
	bigString32.Grow(stringSize32)
	for i := 0; i < stringSize32; i++ {
		bigString32.WriteByte('a')
	}
	bcs := []benchCase{
		{
			name: fmt.Sprintf("big string %d", bigString.Len()),
			msg: &bisp.Message{
				Body: bigString.String(),
			},
		},
		{
			name: fmt.Sprintf("big string 32b %d", bigString32.Len()),
			msg: &bisp.Message{
				Header: bisp.Header{
					Flags: bisp.F32b,
				},
				Body: bigString32.String(),
			},
		},
	}
	b.Run("encode", func(b *testing.B) {
		bench(b, bcs, benchEncodeMsg)
	})
	b.Run("decode", func(b *testing.B) {
		bench(b, bcs, benchDecodeMsg)
	})
}

func BenchmarkBigSlice(b *testing.B) {
	sliceSize := bisp.MaxTcpMessageBodySize - 2
	sliceSize32 := bisp.Max32bMessageBodySize / 1024
	bigSlice := make([]uint8, sliceSize)
	for i := 0; i < sliceSize; i++ {
		bigSlice[i] = uint8(i)
	}
	bigSlice32 := make([]uint32, sliceSize32)
	for i := 0; i < sliceSize32; i++ {
		bigSlice32[i] = uint32(i)
	}
	bcs := []benchCase{
		{
			name: fmt.Sprintf("slice %d", len(bigSlice)),
			msg: &bisp.Message{
				Body: bigSlice,
			},
		},
		{
			name: fmt.Sprintf("slice 32b %d", len(bigSlice32)),
			msg: &bisp.Message{
				Header: bisp.Header{
					Flags: bisp.F32b,
				},
				Body: bigSlice32,
			},
		},
	}
	b.Run("encode", func(b *testing.B) {
		bench(b, bcs, benchEncodeMsg)
	})
	b.Run("decode", func(b *testing.B) {
		bench(b, bcs, benchDecodeMsg)
	})
}

func BenchmarkBigArray(b *testing.B) {
	arraySize := bisp.MaxTcpMessageBodySize
	arraySize32 := bisp.Max32bMessageBodySize / 1024
	bigArray := [bisp.MaxTcpMessageBodySize]uint8{}
	for i := 0; i < arraySize; i++ {
		bigArray[i] = uint8(i)
	}
	bigArray32 := [bisp.Max32bMessageBodySize / 1024]uint8{}
	for i := 0; i < arraySize32; i++ {
		bigArray32[i] = uint8(i)
	}
	bcs := []benchCase{
		{
			name: fmt.Sprintf("array %d", len(bigArray)),
			msg: &bisp.Message{
				Body: bigArray,
			},
		},
		{
			name: fmt.Sprintf("array 32b %d", len(bigArray32)),
			msg: &bisp.Message{
				Header: bisp.Header{
					Flags: bisp.F32b,
				},
				Body: bigArray32,
			},
		},
	}
	b.Run("encode", func(b *testing.B) {
		bench(b, bcs, benchEncodeMsg)
	})
	b.Run("decode", func(b *testing.B) {
		bench(b, bcs, benchDecodeMsg)
	})
}

func BenchmarkBigMap(b *testing.B) {
	mapSize := bisp.MaxTcpMessageBodySize / 4
	mapSize32 := bisp.Max32bMessageBodySize / 4096
	bigMap := make(map[uint16]uint16, mapSize)
	for i := 0; i < mapSize; i++ {
		bigMap[uint16(i)] = uint16(i)
	}
	bigMap32 := make(map[uint32]uint32, mapSize32)
	for i := 0; i < mapSize32; i++ {
		bigMap32[uint32(i)] = uint32(i)
	}
	bcs := []benchCase{
		{
			name: fmt.Sprintf("map %d", len(bigMap)),
			msg: &bisp.Message{
				Body: bigMap,
			},
		},
		{
			name: fmt.Sprintf("map 32b %d", len(bigMap32)),
			msg: &bisp.Message{
				Header: bisp.Header{
					Flags: bisp.F32b,
				},
				Body: bigMap32,
			},
		},
	}
	b.Run("encode", func(b *testing.B) {
		bench(b, bcs, benchEncodeMsg)
	})
	b.Run("decode", func(b *testing.B) {
		bench(b, bcs, benchDecodeMsg)
	})
}

func init() {
	bisp.RegisterType(TestEnum(0))
	bisp.RegisterType(map[string]int{})
	bisp.RegisterType(testBenchmarkStruct{})
	bisp.RegisterType([3]int{})
	bisp.RegisterType([bisp.MaxTcpMessageBodySize]uint8{})
	bisp.RegisterType([bisp.Max32bMessageBodySize / 1024]uint8{})
	bisp.RegisterType(map[uint16]uint16{})
	bisp.RegisterType(map[uint32]uint32{})
}

func bench(b *testing.B, bcs []benchCase, f func(b *testing.B, msg *bisp.Message)) {
	for _, bc := range bcs {
		b.Run(bc.name, func(b *testing.B) {
			f(b, bc.msg)
		})
	}
}

func benchEncodeMsg(b *testing.B, msg *bisp.Message) {
	buf := new(bytes.Buffer)
	encoder := bisp.NewEncoder(buf)

	for i := 0; i < b.N; i++ {
		err := encoder.Encode(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchDecodeMsg(b *testing.B, msg *bisp.Message) {
	buf := new(bytes.Buffer)
	encoder := bisp.NewEncoder(buf)
	decoder := bisp.NewDecoder(buf)

	for i := 0; i < b.N; i++ {
		err := encoder.Encode(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res bisp.Message
		err := decoder.Decode(&res)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchReceiveMsgAndSendRes(b *testing.B, msg *bisp.Message) {
	in := new(bytes.Buffer)
	out := new(bytes.Buffer)
	e1 := bisp.NewEncoder(in)
	d1 := bisp.NewDecoder(in)
	e2 := bisp.NewEncoder(out)

	for i := 0; i < b.N; i++ {
		err := e1.Encode(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var res bisp.Message
		err := d1.Decode(&res)
		if err != nil {
			b.Fatal(err)
		}
		err = e2.Encode(&res)
		if err != nil {
			b.Fatal(err)
		}
	}
}
