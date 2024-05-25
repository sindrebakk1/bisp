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

type TestEnum uint8

const (
	TestEnum1 TestEnum = iota
	TestEnum2
	TestEnum3
)

func (t TestEnum) String() string {
	return [...]string{"TestEnum1", "TestEnum2", "TestEnum3"}[t]
}

func encodeTestHeader(header *bisp.Header) []byte {
	headerBytes := make([]byte, bisp.HeaderSizeWithTransactionID)
	headerBytes[0] = byte(header.Version)
	headerBytes[bisp.VersionSize] = byte(header.Flags)
	binary.BigEndian.PutUint16(headerBytes[bisp.VersionSize+bisp.FlagsSize:], uint16(header.Type))
	copy(headerBytes[bisp.VersionSize+bisp.FlagsSize+bisp.TypeIDSize:], header.TransactionID[:])
	binary.BigEndian.PutUint16(headerBytes[bisp.VersionSize+bisp.FlagsSize+bisp.TypeIDSize+bisp.TransactionIDSize:], uint16(header.Length))

	return headerBytes
}

func encodeTestValue(testValue any, bigLengths bool) ([]byte, error) {
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
		if err := encodeLength(buf, len(testValue.(string)), bigLengths); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, []byte(testValue.(string))); err != nil {
			return nil, err
		}
		break
	case reflect.Slice:
		slice := reflect.ValueOf(testValue)
		if err := encodeLength(buf, slice.Len(), bigLengths); err != nil {
			return nil, err
		}
		for i := 0; i < slice.Len(); i++ {
			v := slice.Index(i).Interface()
			encodedBytes, err := encodeTestValue(v, bigLengths)
			if err != nil {
				return nil, err
			}
			if _, err = buf.Write(encodedBytes); err != nil {
				return nil, err
			}
		}
		break
	case reflect.Struct:
		v := reflect.ValueOf(testValue)
		for i := 0; i < v.NumField(); i++ {
			fieldVal := v.Field(i)
			fieldType := v.Type().Field(i)
			var err error
			var encodedBytes []byte
			if fieldType.IsExported() {
				encodedBytes, err = encodeTestValue(fieldVal.Interface(), bigLengths)
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
		if err := encodeLength(buf, m.Len(), bigLengths); err != nil {
			return nil, err
		}
		for _, key := range m.MapKeys() {
			keyBytes, err := encodeTestValue(key.Interface(), bigLengths)
			if err != nil {
				return nil, err
			}
			if _, err = buf.Write(keyBytes); err != nil {
				return nil, err
			}
			valBytes, err := encodeTestValue(m.MapIndex(key).Interface(), bigLengths)
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

func encodeLength(buf *bytes.Buffer, length int, bigLengths bool) error {
	if bigLengths {
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

func init() {
	bisp.RegisterType(TestEnum(0))
}
