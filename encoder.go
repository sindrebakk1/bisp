package bisp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
)

type encoderFunc func(*Encoder, interface{}) error

type Encoder struct {
	buf               *bytes.Buffer
	primitiveEncoders map[reflect.Kind]encoderFunc
	writer            io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		buf: new(bytes.Buffer),
		primitiveEncoders: map[reflect.Kind]encoderFunc{
			reflect.Uint:    func(e *Encoder, value interface{}) error { return e.encodeUint(value.(uint)) },
			reflect.Uint8:   func(e *Encoder, value interface{}) error { return e.encodeUint8(value.(uint8)) },
			reflect.Uint16:  func(e *Encoder, value interface{}) error { return e.encodeUint16(value.(uint16)) },
			reflect.Uint32:  func(e *Encoder, value interface{}) error { return e.encodeUint32(value.(uint32)) },
			reflect.Uint64:  func(e *Encoder, value interface{}) error { return e.encodeUint64(value.(uint64)) },
			reflect.Int:     func(e *Encoder, value interface{}) error { return e.encodeInt(value.(int)) },
			reflect.Int8:    func(e *Encoder, value interface{}) error { return e.encodeInt8(value.(int8)) },
			reflect.Int16:   func(e *Encoder, value interface{}) error { return e.encodeInt16(value.(int16)) },
			reflect.Int32:   func(e *Encoder, value interface{}) error { return e.encodeInt32(value.(int32)) },
			reflect.Int64:   func(e *Encoder, value interface{}) error { return e.encodeInt64(value.(int64)) },
			reflect.Float32: func(e *Encoder, value interface{}) error { return e.encodeFloat32(value.(float32)) },
			reflect.Float64: func(e *Encoder, value interface{}) error { return e.encodeFloat64(value.(float64)) },
			reflect.Bool:    func(e *Encoder, value interface{}) error { return e.encodeBool(value.(bool)) },
			reflect.String:  func(e *Encoder, value interface{}) error { return e.encodeString(value.(string)) },
			reflect.Slice:   func(e *Encoder, value interface{}) error { return e.encodeArrayOrSlice(value) },
			reflect.Array:   func(e *Encoder, value interface{}) error { return e.encodeArrayOrSlice(value) },
			reflect.Struct:  func(e *Encoder, value interface{}) error { return e.encodeStruct(value) },
		},
		writer: w,
	}
}

func (e *Encoder) Encode(m *Message) error {
	var (
		err        error
		typeID     TypeID
		bodyBuf    []byte
		msgBytes   []byte
		bodyLength int
	)
	bodyLength, bodyBuf, err = e.EncodeBody(m.Body)
	if err != nil {
		return err
	}
	e.buf.Reset()
	if bodyLength > MaxMessageBodySize {
		return errors.New(fmt.Sprintf("message body too large. length: %d max: %d", bodyLength, MaxMessageBodySize))
	}
	typeID, err = GetIDFromType(m.Body)
	if err != nil {
		return err
	}
	m.Header.Version = CurrentVersion
	m.Header.Type = typeID
	m.Header.Length = Length(bodyLength)
	msgBytes, err = e.EncodeHeader(&m.Header)
	if err != nil {
		return err
	}
	msgBytes = append(msgBytes, bodyBuf...)
	_, err = e.writer.Write(msgBytes)
	return err
}

func (e *Encoder) EncodeHeader(h *Header) ([]byte, error) {
	var err error
	buf := new(bytes.Buffer)
	if err = binary.Write(buf, binary.BigEndian, h.Version); err != nil {
		return nil, err
	}
	if err = binary.Write(buf, binary.BigEndian, h.Flags); err != nil {
		return nil, err
	}
	if err = binary.Write(buf, binary.BigEndian, h.Type); err != nil {
		return nil, err
	}
	hasTransactionID := false
	for _, b := range h.TransactionID {
		if b != 0 {
			hasTransactionID = true
			break
		}
	}
	if hasTransactionID {
		h.Flags |= FTransactionID
	}
	if (h.Flags & FTransactionID) != 0 {
		if err = binary.Write(buf, binary.BigEndian, h.TransactionID); err != nil {
			return nil, err
		}
	}
	if err = binary.Write(buf, binary.BigEndian, h.Length); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (e *Encoder) EncodeBody(v interface{}) (int, []byte, error) {
	val := reflect.ValueOf(v)
	res := new(bytes.Buffer)
	err := e.encodeValue(val)
	if err != nil {
		return 0, nil, err
	}
	var n int64
	n, err = io.Copy(res, e.buf)
	return int(n), res.Bytes(), err
}

func (e *Encoder) encodeValue(value reflect.Value) error {
	kind := value.Kind()
	if kind == reflect.Invalid {
		return nil
	}
	value = castValueToUnderlying(value)
	if encoder, ok := e.primitiveEncoders[kind]; ok {
		return encoder(e, value.Interface())
	}
	return errors.New("unsupported type")
}

func castValueToUnderlying(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	underlyingType := getUnderlyingType(v.Type())
	return v.Convert(underlyingType)
}

func getUnderlyingType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Uint:
		return reflect.TypeOf(uint(0))
	case reflect.Uint8:
		return reflect.TypeOf(uint8(0))
	case reflect.Uint16:
		return reflect.TypeOf(uint16(0))
	case reflect.Uint32:
		return reflect.TypeOf(uint32(0))
	case reflect.Uint64:
		return reflect.TypeOf(uint64(0))
	case reflect.Int:
		return reflect.TypeOf(int(0))
	case reflect.Int8:
		return reflect.TypeOf(int8(0))
	case reflect.Int16:
		return reflect.TypeOf(int16(0))
	case reflect.Int32:
		return reflect.TypeOf(int32(0))
	case reflect.Int64:
		return reflect.TypeOf(int64(0))
	case reflect.Float32:
		return reflect.TypeOf(float32(0))
	case reflect.Float64:
		return reflect.TypeOf(float64(0))
	case reflect.Bool:
		return reflect.TypeOf(bool(false))
	case reflect.String:
		return reflect.TypeOf("")
	default:
		return t
	}
}

func (e *Encoder) encodeUint(value uint) error {
	return binary.Write(e.buf, binary.BigEndian, uint64(value))
}

func (e *Encoder) encodeUint8(value uint8) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeUint16(value uint16) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeUint32(value uint32) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeUint64(value uint64) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeInt(value int) error {
	return binary.Write(e.buf, binary.BigEndian, int64(value))
}

func (e *Encoder) encodeInt8(value int8) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeInt16(value int16) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeInt32(value int32) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeInt64(value int64) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeFloat32(value float32) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeFloat64(value float64) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeBool(value bool) error {
	var v uint8
	if value {
		v = 1
	}
	return binary.Write(e.buf, binary.BigEndian, v)
}

func (e *Encoder) encodeString(value string) error {
	if err := e.encodeUint16(uint16(len(value))); err != nil {
		return err
	}
	return binary.Write(e.buf, binary.BigEndian, []byte(value))
}

func (e *Encoder) encodeArrayOrSlice(value interface{}) error {
	sliceValue := reflect.ValueOf(value)
	if err := e.encodeUint32(uint32(sliceValue.Len())); err != nil {
		return err
	}
	if sliceValue.Len() == 0 {
		return nil
	}
	for i := 0; i < sliceValue.Len(); i++ {
		v := sliceValue.Index(i)
		if err := e.encodeValue(v); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeStruct(val interface{}) error {
	v := reflect.ValueOf(val)
	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := v.Type().Field(i)
		if fieldType.IsExported() {
			if err := e.encodeValue(fieldVal); err != nil {
				return err
			}
		}
	}
	return nil
}
