package bisp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
)

type Encoder struct {
	buf    *bytes.Buffer
	writer io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		buf:    new(bytes.Buffer),
		writer: w,
	}
}

func (e *Encoder) Encode(m *Message) error {
	e.buf.Reset()
	var (
		err      error
		typeID   TypeID
		length   int
		msgBytes []byte
	)
	typeID, err = GetIDFromType(m.Body)
	if err != nil {
		return err
	}
	l32 := m.Header.HasFlag(F32b)
	err = e.EncodeBody(m.Body, l32)
	if err != nil {
		return err
	}
	length = e.buf.Len()
	if l32 && length > Max32bMessageBodySize {
		return errors.New(fmt.Sprintf("message body too large. length: %d max: %d", length, Max32bMessageBodySize))
	}
	if !l32 && length > MaxTcpMessageBodySize {
		return errors.New(fmt.Sprintf("message body too large. length: %d max: %d", length, MaxTcpMessageBodySize))
	}
	msgBytes, err = e.EncodeHeader(&m.Header, typeID, length)
	if err != nil {
		return err
	}
	msgBytes = append(msgBytes, e.buf.Bytes()...)
	_, err = e.writer.Write(msgBytes)
	return err
}

func (e *Encoder) EncodeProcedure(p struct{}, kind ProcedureKind) error {
	e.buf.Reset()
	var (
		err           error
		procedureID   TypeID
		transactionID TransactionID
		length        int
		msgBytes      []byte
		header        Header
	)
	procedureID, err = GetProcedureID(p)
	if err != nil {
		return err
	}
	header.SetFlag(FProcedure)

	v := reflect.ValueOf(p)
	if transactionID, err = e.encodeProcedure(v, procedureID, kind == Response); err != nil {
		return err
	}
	header.TransactionID = transactionID
	length = e.buf.Len()
	if length > MaxTcpMessageBodySize {
		return errors.New(fmt.Sprintf("message body too large. length: %d max: %d", length, MaxTcpMessageBodySize))
	}
	msgBytes, err = e.EncodeHeader(&header, procedureID, length)
	if err != nil {
		return err
	}
	msgBytes = append(msgBytes, e.buf.Bytes()...)
	_, err = e.writer.Write(msgBytes)
	return err
}

func (e *Encoder) EncodeHeader(h *Header, typeID TypeID, length int) ([]byte, error) {
	h.Version = CurrentVersion
	h.Type = typeID
	h.Length = Length(length)

	buf := new(bytes.Buffer)
	buf.Grow(h.Len() + length)

	buf.WriteByte(byte(h.Version))
	buf.WriteByte(byte(h.Flags))
	if err := binary.Write(buf, binary.BigEndian, uint16(h.Type)); err != nil {
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
		if err := binary.Write(buf, binary.BigEndian, h.TransactionID); err != nil {
			return nil, err
		}
	}
	if h.HasFlag(F32b) {
		if err := binary.Write(buf, binary.BigEndian, h.Length); err != nil {
			return nil, err
		}
	} else {
		if err := binary.Write(buf, binary.BigEndian, uint16(h.Length)); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (e *Encoder) EncodeBody(v any, l32 bool) error {
	val := reflect.ValueOf(v)
	kind := val.Kind()
	if kind == reflect.Invalid {
		return nil
	}
	if kind == reflect.Ptr {
		val = val.Elem()
	}
	underlyingType := getUnderlyingType(val, kind)
	if val.Type() != underlyingType {
		val = val.Convert(underlyingType)
	}
	err := e.encodeValue(val, kind, l32)
	if err != nil {
		return err
	}
	return nil
}

func (e *Encoder) Bytes() []byte {
	return e.buf.Bytes()
}

func (e *Encoder) Reset() {
	e.buf.Reset()
}

func (e *Encoder) encodeProcedure(p reflect.Value, id TypeID, isReturn bool) (TransactionID, error) {
	var transactionID TransactionID
	t, ok := pReverseRegistry[id]
	if !ok {
		return transactionID, errors.New(fmt.Sprintf("procedure %s not registered", p.Type().Name()))
	}

	transactionField := p.FieldByName("TransactionID")
	if transactionField.IsValid() && transactionField.Type() == reflect.TypeOf(transactionID) {
		transactionID = transactionField.Interface().(TransactionID)
	}

	if isReturn {
		field := p.FieldByName("Out")
		if err := e.encodeValue(field, field.Kind(), false); err != nil {
			return transactionID, err
		}
		return transactionID, nil
	}
	for i := range t.NumField() {
		tField := t.Field(i)
		if tField.Name == "Procedure" || tField.Name == "TransactionID" || tField.Name == "Out" {
			continue
		}
		field := p.FieldByName(tField.Name)
		if err := e.encodeValue(field, field.Kind(), false); err != nil {
			return transactionID, err
		}
	}
	return transactionID, nil
}

func (e *Encoder) encodeValue(val reflect.Value, kind reflect.Kind, l32 bool) error {
	var err error
	switch kind {
	case reflect.Uint:
		err = e.encodeUint(val, l32)
	case reflect.Uint8:
		err = e.encodeUint8(val, l32)
	case reflect.Uint16:
		err = e.encodeUint16(val, l32)
	case reflect.Uint32:
		err = e.encodeUint32(val, l32)
	case reflect.Uint64:
		err = e.encodeUint64(val, l32)
	case reflect.Int:
		err = e.encodeInt(val, l32)
	case reflect.Int8:
		err = e.encodeInt8(val, l32)
	case reflect.Int16:
		err = e.encodeInt16(val, l32)
	case reflect.Int32:
		err = e.encodeInt32(val, l32)
	case reflect.Int64:
		err = e.encodeInt64(val, l32)
	case reflect.Float32:
		err = e.encodeFloat32(val, l32)
	case reflect.Float64:
		err = e.encodeFloat64(val, l32)
	case reflect.Bool:
		err = e.encodeBool(val, l32)
	case reflect.String:
		err = e.encodeString(val, l32)
	case reflect.Slice:
		err = e.encodeSlice(val, l32)
	case reflect.Array:
		err = e.encodeArray(val, l32)
	case reflect.Struct:
		err = e.encodeStruct(val, l32)
	case reflect.Map:
		err = e.encodeMap(val, l32)
	default:
		return errors.New("unsupported type")
	}
	if err != nil {
		return err
	}
	return nil
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

func (e *Encoder) encodeUint8(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, uint8(v.Uint()))
}

func (e *Encoder) encodeUint16(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, uint16(v.Uint()))
}

func (e *Encoder) encodeUint32(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, uint32(v.Uint()))
}

func (e *Encoder) encodeUint64(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, v.Uint())
}

func (e *Encoder) encodeUint(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, v.Uint())
}

func (e *Encoder) encodeInt(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, v.Int())
}

func (e *Encoder) encodeInt8(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, int8(v.Int()))
}

func (e *Encoder) encodeInt16(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, int16(v.Int()))
}

func (e *Encoder) encodeInt32(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, int32(v.Int()))
}

func (e *Encoder) encodeInt64(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, v.Int())
}

func (e *Encoder) encodeFloat32(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, float32(v.Float()))
}

func (e *Encoder) encodeFloat64(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, v.Float())
}

func (e *Encoder) encodeBool(v reflect.Value, _ bool) error {
	return binary.Write(e.buf, binary.BigEndian, v.Bool())
}

func (e *Encoder) encodeString(v reflect.Value, l32 bool) error {
	if err := e.encodeLength(v.Len(), l32); err != nil {
		return err
	}
	_, err := e.buf.WriteString(v.String())
	return err
}

func (e *Encoder) encodeSlice(v reflect.Value, l32 bool) error {
	length := v.Len()
	if err := e.encodeLength(length, l32); err != nil {
		return err
	}
	if length == 0 {
		return nil
	}
	elemKind := v.Type().Elem().Kind()
	for i := 0; i < length; i++ {
		v := v.Index(i)
		if err := e.encodeValue(v, elemKind, l32); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeArray(v reflect.Value, l32 bool) error {
	elemKind := v.Type().Elem().Kind()
	for i := 0; i < v.Len(); i++ {
		val := v.Index(i)
		if err := e.encodeValue(val, elemKind, l32); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeStruct(v reflect.Value, l32 bool) error {
	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := v.Type().Field(i)
		if fieldType.IsExported() {
			if err := e.encodeValue(fieldVal, fieldVal.Kind(), l32); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Encoder) encodeMap(v reflect.Value, l32 bool) error {
	t := v.Type()
	valKind := t.Elem().Kind()
	keyKind := t.Key().Kind()
	keys := v.MapKeys()
	if err := e.encodeLength(len(keys), l32); err != nil {
		return err
	}
	for _, key := range keys {
		if err := e.encodeValue(key, keyKind, l32); err != nil {
			return err
		}
		value := v.MapIndex(key)
		if err := e.encodeValue(value, valKind, l32); err != nil {
			return err
		}
	}
	return nil
}
