package bisp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
)

type encoderFunc func(*Encoder, interface{}, bool) error

type Encoder struct {
	buf               *bytes.Buffer
	primitiveEncoders map[reflect.Kind]encoderFunc
	writer            io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		buf: new(bytes.Buffer),
		primitiveEncoders: map[reflect.Kind]encoderFunc{
			reflect.Uint:    func(e *Encoder, value interface{}, b bool) error { return e.encodeUint(value.(uint), b) },
			reflect.Uint8:   func(e *Encoder, value interface{}, b bool) error { return e.encodeUint8(value.(uint8), b) },
			reflect.Uint16:  func(e *Encoder, value interface{}, b bool) error { return e.encodeUint16(value.(uint16), b) },
			reflect.Uint32:  func(e *Encoder, value interface{}, b bool) error { return e.encodeUint32(value.(uint32), b) },
			reflect.Uint64:  func(e *Encoder, value interface{}, b bool) error { return e.encodeUint64(value.(uint64), b) },
			reflect.Int:     func(e *Encoder, value interface{}, b bool) error { return e.encodeInt(value.(int), b) },
			reflect.Int8:    func(e *Encoder, value interface{}, b bool) error { return e.encodeInt8(value.(int8), b) },
			reflect.Int16:   func(e *Encoder, value interface{}, b bool) error { return e.encodeInt16(value.(int16), b) },
			reflect.Int32:   func(e *Encoder, value interface{}, b bool) error { return e.encodeInt32(value.(int32), b) },
			reflect.Int64:   func(e *Encoder, value interface{}, b bool) error { return e.encodeInt64(value.(int64), b) },
			reflect.Float32: func(e *Encoder, value interface{}, b bool) error { return e.encodeFloat32(value.(float32), b) },
			reflect.Float64: func(e *Encoder, value interface{}, b bool) error { return e.encodeFloat64(value.(float64), b) },
			reflect.Bool:    func(e *Encoder, value interface{}, b bool) error { return e.encodeBool(value.(bool), b) },
			reflect.String:  func(e *Encoder, value interface{}, b bool) error { return e.encodeString(value.(string), b) },
			reflect.Slice:   func(e *Encoder, value interface{}, b bool) error { return e.encodeSlice(value, b) },
			reflect.Array:   func(e *Encoder, value interface{}, b bool) error { return e.encodeArray(value, b) },
			reflect.Struct:  func(e *Encoder, value interface{}, b bool) error { return e.encodeStruct(value, b) },
			reflect.Map:     func(e *Encoder, value interface{}, b bool) error { return e.encodeMap(value, b) },
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
	bodyLength, bodyBuf, err = e.EncodeBody(m.Body, m.Header.HasFlag(F32b))
	if err != nil {
		return err
	}
	e.buf.Reset()
	if m.Header.HasFlag(F32b) && bodyLength > Max32bMessageBodySize {
		return errors.New(fmt.Sprintf("message body too large. length: %d max: %d", bodyLength, Max32bMessageBodySize))
	}
	if bodyLength > MaxTcpMessageBodySize && !m.Header.HasFlag(F32b) {
		return errors.New(fmt.Sprintf("message body too large. length: %d max: %d", bodyLength, MaxTcpMessageBodySize))
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
		h.SetFlag(FTransaction)
	}
	if h.HasFlag(FTransaction) {
		if err = binary.Write(buf, binary.BigEndian, h.TransactionID); err != nil {
			return nil, err
		}
	}
	if h.HasFlag(F32b) {
		if err = binary.Write(buf, binary.BigEndian, h.Length); err != nil {
			return nil, err
		}
	} else {
		if err = binary.Write(buf, binary.BigEndian, uint16(h.Length)); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (e *Encoder) EncodeBody(v interface{}, l32 bool) (int, []byte, error) {
	val := reflect.ValueOf(v)
	res := new(bytes.Buffer)
	err := e.encodeValue(val, l32)
	if err != nil {
		return 0, nil, err
	}
	var n int64
	n, err = io.Copy(res, e.buf)
	return int(n), res.Bytes(), err
}

func (e *Encoder) encodeValue(value reflect.Value, l32 bool) error {
	kind := value.Kind()
	if kind == reflect.Invalid {
		return nil
	}
	value = castValueToUnderlying(value)
	if encoder, ok := e.primitiveEncoders[kind]; ok {
		return encoder(e, value.Interface(), l32)
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

func (e *Encoder) encodeLength(value int, l32 bool) error {
	if l32 && value > 1<<32 {
		return fmt.Errorf("length %d is too large, max: %d", value, 1<<32)
	}
	if !l32 && value > 1<<16 {
		return fmt.Errorf("length %d is too large, max: %d. consider setting the F32b flag", value, 1<<16)
	}
	if l32 {
		return binary.Write(e.buf, binary.BigEndian, uint32(value))
	}
	return binary.Write(e.buf, binary.BigEndian, uint16(value))
}

func (e *Encoder) encodeUint(value uint, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, uint64(value))
}

func (e *Encoder) encodeUint8(value uint8, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeUint16(value uint16, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeUint32(value uint32, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeUint64(value uint64, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeInt(value int, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, int64(value))
}

func (e *Encoder) encodeInt8(value int8, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeInt16(value int16, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeInt32(value int32, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeInt64(value int64, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeFloat32(value float32, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeFloat64(value float64, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, value)
}

func (e *Encoder) encodeBool(value bool, _ bool) error {
	var v uint8
	if value {
		v = 1
	}
	return binary.Write(e.buf, binary.BigEndian, v)
}

func (e *Encoder) encodeString(value string, l32 bool) error {
	if err := e.encodeLength(len(value), l32); err != nil {
		return err
	}
	return binary.Write(e.buf, binary.BigEndian, []byte(value))
}

func (e *Encoder) encodeSlice(value interface{}, l32 bool) error {
	sliceValue := reflect.ValueOf(value)
	if err := e.encodeLength(sliceValue.Len(), l32); err != nil {
		return err
	}
	if sliceValue.Len() == 0 {
		return nil
	}
	for i := 0; i < sliceValue.Len(); i++ {
		v := sliceValue.Index(i)
		if err := e.encodeValue(v, l32); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeArray(value interface{}, l32 bool) error {
	arr := reflect.ValueOf(value)
	for i := 0; i < arr.Len(); i++ {
		v := arr.Index(i)
		if err := e.encodeValue(v, l32); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeStruct(val interface{}, l32 bool) error {
	v := reflect.ValueOf(val)
	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := v.Type().Field(i)
		if fieldType.IsExported() {
			if err := e.encodeValue(fieldVal, l32); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Encoder) encodeMap(val interface{}, l32 bool) error {
	v := reflect.ValueOf(val)
	keys := v.MapKeys()
	if err := e.encodeLength(len(keys), l32); err != nil {
		return err
	}
	for _, key := range keys {
		if err := e.encodeValue(key, l32); err != nil {
			return err
		}
		if err := e.encodeValue(v.MapIndex(key), l32); err != nil {
			return err
		}
	}
	return nil
}
