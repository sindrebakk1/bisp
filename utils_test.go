package tcp_test

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"tcp"
)

type testCase struct {
	value    any
	name     string
	expected any
}

func encodeTestHeader(header *tcp.Header) []byte {
	headerBytes := make([]byte, tcp.HeaderSizeWithTransactionID)
	headerBytes[0] = byte(header.Version)
	headerBytes[tcp.VersionSize] = byte(header.Flags)
	binary.BigEndian.PutUint16(headerBytes[tcp.VersionSize+tcp.FlagsSize:], uint16(header.Type))
	copy(headerBytes[tcp.VersionSize+tcp.FlagsSize+tcp.TypeIDSize:], header.TransactionID[:])
	binary.BigEndian.PutUint16(headerBytes[tcp.VersionSize+tcp.FlagsSize+tcp.TypeIDSize+tcp.TransactionIDSize:], uint16(header.Length))

	return headerBytes
}

func encodeTestValue(testValue any) ([]byte, error) {
	buf := new(bytes.Buffer)

	t := reflect.TypeOf(testValue)
	switch t.Kind() {
	case reflect.Int:
		if err := binary.Write(buf, binary.BigEndian, int32(testValue.(int))); err != nil {
			return nil, err
		}
		break
	case reflect.Uint:
		if err := binary.Write(buf, binary.BigEndian, uint32(testValue.(uint))); err != nil {
			return nil, err
		}
		break
	case reflect.String:
		if err := binary.Write(buf, binary.BigEndian, uint16(len(testValue.(string)))); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, []byte(testValue.(string))); err != nil {
			return nil, err
		}
		break
	case reflect.Slice:
		slice := reflect.ValueOf(testValue)
		if err := binary.Write(buf, binary.BigEndian, uint32(slice.Len())); err != nil {
			return nil, err
		}
		for i := 0; i < slice.Len(); i++ {
			v := slice.Index(i).Interface()
			var encodedBytes []byte
			encodedBytes, err := encodeTestValue(v)
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
				encodedBytes, err = encodeTestValue(fieldVal.Interface())
				if err != nil {
					return nil, err
				}
				if _, err = buf.Write(encodedBytes); err != nil {
					return nil, err
				}
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
