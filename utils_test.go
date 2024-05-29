package bisp_test

import (
	"bytes"
	"encoding/binary"
	"github.com/sindrebakk1/bisp"
	"reflect"
)

type testCase struct {
	value    any
	name     string
	expected any
}

func encodeTestHeader(header *bisp.Header, l32 bool) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(byte(header.Version))
	buf.WriteByte(byte(header.Flags))
	_ = binary.Write(buf, binary.BigEndian, header.Type)
	_ = binary.Write(buf, binary.BigEndian, header.TransactionID)
	if l32 {
		_ = binary.Write(buf, binary.BigEndian, uint32(header.Length))
		return buf.Bytes()
	}
	_ = binary.Write(buf, binary.BigEndian, uint16(header.Length))
	return buf.Bytes()
}

func encodeTestValue(testValue any, l32 bool) ([]byte, error) {
	buf := new(bytes.Buffer)

	t := reflect.TypeOf(testValue)
	switch t.Kind() {
	case reflect.Int:
		if err := binary.Write(buf, binary.BigEndian, int64(testValue.(int))); err != nil {
			return nil, err
		}
		break
	case reflect.Uint:
		if err := binary.Write(buf, binary.BigEndian, uint64(testValue.(uint))); err != nil {
			return nil, err
		}
		break
	case reflect.String:
		if err := encodeLength(buf, len(testValue.(string)), l32); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, []byte(testValue.(string))); err != nil {
			return nil, err
		}
		break
	case reflect.Slice:
		slice := reflect.ValueOf(testValue)
		if err := encodeLength(buf, slice.Len(), l32); err != nil {
			return nil, err
		}
		for i := 0; i < slice.Len(); i++ {
			v := slice.Index(i).Interface()
			encodedBytes, err := encodeTestValue(v, l32)
			if err != nil {
				return nil, err
			}
			if _, err = buf.Write(encodedBytes); err != nil {
				return nil, err
			}
		}
		break
	case reflect.Array:
		arr := reflect.ValueOf(testValue)
		for i := 0; i < arr.Len(); i++ {
			v := arr.Index(i).Interface()
			encodedBytes, err := encodeTestValue(v, l32)
			if err != nil {
				return nil, err
			}
			if _, err = buf.Write(encodedBytes); err != nil {
				return nil, err
			}
		}
	case reflect.Struct:
		v := reflect.ValueOf(testValue)
		for i := 0; i < v.NumField(); i++ {
			fieldVal := v.Field(i)
			fieldType := v.Type().Field(i)
			var err error
			var encodedBytes []byte
			if fieldType.IsExported() {
				encodedBytes, err = encodeTestValue(fieldVal.Interface(), l32)
				if err != nil {
					return nil, err
				}
				if _, err = buf.Write(encodedBytes); err != nil {
					return nil, err
				}
			}
		}
	case reflect.Map:
		m := reflect.ValueOf(testValue)
		if err := encodeLength(buf, m.Len(), l32); err != nil {
			return nil, err
		}
		for _, key := range m.MapKeys() {
			keyBytes, err := encodeTestValue(key.Interface(), l32)
			if err != nil {
				return nil, err
			}
			if _, err = buf.Write(keyBytes); err != nil {
				return nil, err
			}
			valBytes, err := encodeTestValue(m.MapIndex(key).Interface(), l32)
			if err != nil {
				return nil, err
			}
			if _, err = buf.Write(valBytes); err != nil {
				return nil, err
			}
		}
	default:
		if err := binary.Write(buf, binary.BigEndian, testValue); err != nil {
			return nil, err
		}
		break
	}

	return buf.Bytes(), nil
}

func encodeLength(buf *bytes.Buffer, length int, l32 bool) error {
	if l32 {
		if err := binary.Write(buf, binary.BigEndian, uint32(length)); err != nil {
			return err
		}
	} else {
		if err := binary.Write(buf, binary.BigEndian, uint16(length)); err != nil {
			return err
		}
	}
	return nil
}

type TestEnum uint8

const (
	TestEnum1 TestEnum = iota
	TestEnum2
	TestEnum3
)

func (t TestEnum) String() string {
	return [...]string{"TestEnum1", "TestEnum2", "TestEnum3"}[t]
}

type testStructEnum struct {
	A TestEnum
	B TestEnum
	C []TestEnum
}

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

func init() {
	bisp.RegisterType([3]int{})
	bisp.RegisterType([3]uint{})
	bisp.RegisterType([3]float32{})
	bisp.RegisterType([3]float64{})
	bisp.RegisterType([3]string{})
	bisp.RegisterType([3]bool{})
	bisp.RegisterType([3]testStruct{})
	bisp.RegisterType(TestEnum(0))
	bisp.RegisterType(testStructEnum{})
	bisp.RegisterType(map[int]string{})
	bisp.RegisterType(map[TestEnum]string{})
	bisp.RegisterType(map[string]int{})
	bisp.RegisterType(map[string]TestEnum{})
	bisp.RegisterType(map[string][]int{})
	bisp.RegisterType(testStruct{})
	bisp.RegisterType(testStructSliceField{})
	bisp.RegisterType(testStructStructField{})
	bisp.RegisterType(testStructStructFieldSliceField{})
	bisp.RegisterType(testStructEmbeddedPrivateStruct{})
	bisp.RegisterType(testStructEmbeddedStruct{})
	bisp.RegisterType(testStructPrivateFields{})
}
