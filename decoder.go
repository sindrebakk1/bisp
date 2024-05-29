package bisp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"unsafe"
)

type Decoder struct {
	buf    *bytes.Buffer
	reader io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		buf:    new(bytes.Buffer),
		reader: r,
	}
}

func (d *Decoder) Decode(msg *Message) error {
	d.buf.Reset()
	header, err := d.DecodeHeader()
	if err != nil {
		return err
	}
	var body interface{}
	if header.HasFlag(FProcedure) {
		body, _, err = d.DecodeProcedure(header.Type, uint32(header.Length))
		if err != nil {
			return err
		}
	} else {
		body, err = d.DecodeBody(header.Type, uint32(header.Length), header.HasFlag(F32b))
		if err != nil {
			return err
		}
	}
	msg.Header = *header
	msg.Body = body
	return nil
}

func TDecode[T any](d *Decoder) (*TMessage[T], error) {
	var (
		msg  Message
		body T
		ok   bool
	)
	err := d.Decode(&msg)
	if err != nil {
		return nil, err
	}
	if body, ok = msg.Body.(T); !ok {
		return nil, errors.New(fmt.Sprintf("expected body to be of type %s, got %s", reflect.TypeOf(body), reflect.TypeOf(msg.Body)))
	}
	return &TMessage[T]{
		Header: msg.Header,
		Body:   body,
	}, nil
}

func TDecodeProcedure[P any](d *Decoder) (*TMessage[P], PKind, error) {
	var (
		procedureID ID
	)
	header, err := d.DecodeHeader()
	if err != nil {
		return nil, 0, err
	}
	if !header.HasFlag(FProcedure) {
		return nil, 0, errors.New("expected procedure message")
	}
	var p P
	procedureID, err = GetProcedureID(p)
	if err != nil {
		return nil, 0, err
	}
	if header.Type != procedureID {
		return nil, 0, errors.New(fmt.Sprintf("expected procedure type %d, got %d", procedureID, header.Type))
	}
	var pBody any
	var pKind PKind
	pBody, pKind, err = d.DecodeProcedure(procedureID, uint32(header.Length))
	if err != nil {
		return nil, 0, err
	}
	p = pBody.(P)
	return &TMessage[P]{
		Header: *header,
		Body:   p,
	}, pKind, nil
}

func (d *Decoder) DecodeHeader() (*Header, error) {
	var header Header
	n, err := io.CopyN(d.buf, d.reader, HeaderSize)
	if err != nil {
		return nil, err
	}
	if n != HeaderSize {
		return nil, errors.New("unexpected end of message")
	}
	var (
		version uint8
		flags   uint8
		typeID  uint16
		transID [TransactionIDSize]byte
		length  uint32
	)
	if version, err = d.decodeUint8(reflect.ValueOf(version), false); err != nil {
		return nil, err
	}
	if Version(version) != CurrentVersion {
		return nil, errors.New("unsupported version")
	}
	if flags, err = d.decodeUint8(reflect.ValueOf(flags), false); err != nil {
		return nil, err
	}
	if typeID, err = d.decodeUint16(reflect.ValueOf(typeID), false); err != nil {
		return nil, err
	}
	if (Flag(flags) & FTransaction) == FTransaction {
		n, err = io.CopyN(d.buf, d.reader, TransactionIDSize)
		var tn int
		if tn, err = d.buf.Read(transID[:]); err != nil {
			return nil, err
		}
		if tn != TransactionIDSize {
			return nil, errors.New("unexpected end of transaction ID")
		}
	}
	if (Flag(flags) & F32b) == F32b {
		n, err = io.CopyN(d.buf, d.reader, LengthSize)
		if err != nil {
			return nil, err
		}
		if n != LengthSize {
			return nil, errors.New("unexpected end of length")
		}
		if length, err = d.decodeUint32(reflect.ValueOf(length), false); err != nil {
			return nil, err
		}
	} else {
		var length16 uint16
		if length16, err = d.decodeUint16(reflect.ValueOf(length), false); err != nil {
			return nil, err
		}
		length = uint32(length16)
	}

	header.Version = Version(version)
	header.Flags = Flag(flags)
	header.Type = ID(typeID)
	header.TransactionID = transID
	header.Length = Length(length)

	return &header, nil
}

func (d *Decoder) DecodeBody(typeID ID, l uint32, l32 bool) (any, error) {
	d.buf.Reset()
	d.buf.Grow(int(l))
	n, err := io.CopyN(d.buf, d.reader, int64(l))
	if err != nil {
		return nil, err
	}
	if n != int64(l) {
		return nil, errors.New("unexpected end of body")
	}
	var typ reflect.Type
	typ, err = GetTypeFromID(typeID)
	if err != nil {
		return nil, err
	}
	if typ == nil {
		return nil, nil
	}
	val := reflect.New(typ).Elem()
	kind := val.Kind()
	if kind == reflect.Ptr {
		val = val.Elem()
	}
	if err = d.decodeValue(val, typ, kind, l32); err != nil {
		return nil, err
	}
	return val.Interface(), nil
}

func (d *Decoder) DecodeProcedure(procedureID ID, l uint32) (any, PKind, error) {
	d.buf.Reset()
	d.buf.Grow(int(l))
	n, err := io.CopyN(d.buf, d.reader, int64(l))
	if err != nil {
		return nil, 0, err
	}
	if n != int64(l) {
		return nil, 0, errors.New("unexpected end of procedure")
	}
	var typ reflect.Type
	typ, err = GetProcedureFromID(procedureID)
	if err != nil {
		return nil, 0, err
	}
	if typ == nil {
		return nil, 0, nil
	}
	val := reflect.New(typ).Elem()
	kind := val.Kind()
	if kind == reflect.Ptr {
		val = val.Elem()
	}
	var pKind PKind
	if pKind, err = d.decodeProcedure(val, procedureID); err != nil {
		return nil, 0, err
	}
	return val.Interface(), pKind, nil
}

func (d *Decoder) decodeProcedure(p reflect.Value, procedureID ID) (PKind, error) {
	typ, ok := pReverseRegistry[procedureID]
	if !ok {
		return 0, errors.New(fmt.Sprintf("procedure %s not registered", p.Type().Name()))
	}
	t := p.Type()
	if typ != t {
		return 0, errors.New(fmt.Sprintf("procedure type mismatch: %s != %s", typ, t))
	}
	k, err := d.decodeUint8(p, false)
	if err != nil {
		return 0, err
	}
	kind := PKind(k)
	if kind == Response {
		field := p.FieldByName("Out")
		if !field.IsValid() {
			return 0, errors.New("procedure must have a valid Out field")
		}
		err = d.decodeValue(field, field.Type(), field.Kind(), false)
		if err != nil {
			return 0, err
		}
	}
	for i := range t.NumField() {
		tField := t.Field(i)
		if tField.Name == "Procedure" || tField.Name == "TransactionID" || tField.Name == "Out" {
			continue
		}
		field := reflect.New(tField.Type).Elem()
		err = d.decodeValue(field, field.Type(), field.Kind(), false)
		if err != nil {
			return 0, err
		}
	}
	return kind, nil
}

func (d *Decoder) decodeValue(v reflect.Value, t reflect.Type, k reflect.Kind, l32 bool) error {
	var val interface{}
	var err error
	switch k {
	case reflect.Uint:
		val, err = d.decodeUint(v, l32)
		if err != nil {
			return err
		}
		if t !=
			tUint {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Uint8:
		val, err = d.decodeUint8(v, l32)
		if err != nil {
			return err
		}
		if t != tUint8 {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Uint16:
		val, err = d.decodeUint16(v, l32)
		if err != nil {
			return err
		}
		if t != tUint16 {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Uint32:
		val, err = d.decodeUint32(v, l32)
		if err != nil {
			return err
		}
		if t != tUint32 {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Uint64:
		val, err = d.decodeUint64(v, l32)
		if err != nil {
			return err
		}
		if t != tUint64 {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Int8:
		val, err = d.decodeInt8(v, l32)
		if err != nil {
			return err
		}
		if t != tInt8 {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Int:
		val, err = d.decodeInt(v, l32)
		if err != nil {
			return err
		}
		if t != tInt {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Int16:
		val, err = d.decodeInt16(v, l32)
		if err != nil {
			return err
		}
		if t != tInt16 {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Int32:
		val, err = d.decodeInt32(v, l32)
		if err != nil {
			return err
		}
		if t != tInt32 {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Int64:
		val, err = d.decodeInt64(v, l32)
		if err != nil {
			return err
		}
		if t != tInt64 {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Float32:
		val, err = d.decodeFloat32(v, l32)
		if err != nil {
			return err
		}
		if t != tFloat32 {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Float64:
		val, err = d.decodeFloat64(v, l32)
		if err != nil {
			return err
		}
		if t != tFloat64 {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Bool:
		val, err = d.decodeBool(v, l32)
		if err != nil {
			return err
		}
		if t != tBool {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.String:
		val, err = d.decodeString(v, l32)
		if err != nil {
			return err
		}
		if t != tString {
			v.Set(reflect.ValueOf(val).Convert(t))
			return nil
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Slice:
		val, err = d.decodeSlice(v, l32)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Array:
		val, err = d.decodeArray(v, l32)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Struct:
		val, err = d.decodeStruct(v, l32)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(val))
		return nil
	case reflect.Map:
		val, err = d.decodeMap(v, l32)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(val))
		return nil
	default:
		return errors.New("unsupported type")
	}
}

func (d *Decoder) decodeLength(v reflect.Value, l32 bool) (int, error) {
	if l32 {
		len32, err := d.decodeUint32(v, l32)
		return int(len32), err
	}
	len16, err := d.decodeUint16(v, l32)
	return int(len16), err
}

func (d *Decoder) decodeUint(_ reflect.Value, _ bool) (uint, error) {
	var value uint64
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return uint(value), err
}

func (d *Decoder) decodeUint8(_ reflect.Value, _ bool) (uint8, error) {
	var value uint8
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeUint16(_ reflect.Value, _ bool) (uint16, error) {
	var value uint16
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeUint32(_ reflect.Value, _ bool) (uint32, error) {
	var value uint32
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeUint64(_ reflect.Value, _ bool) (uint64, error) {
	var value uint64
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeInt(_ reflect.Value, _ bool) (int, error) {
	var value int64
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return int(value), err
}

func (d *Decoder) decodeInt8(_ reflect.Value, _ bool) (int8, error) {
	var value int8
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeInt16(_ reflect.Value, _ bool) (int16, error) {
	var value int16
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeInt32(_ reflect.Value, _ bool) (int32, error) {
	var value int32
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeInt64(_ reflect.Value, _ bool) (int64, error) {
	var value int64
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeFloat32(_ reflect.Value, _ bool) (float32, error) {
	var value float32
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeFloat64(_ reflect.Value, _ bool) (float64, error) {
	var value float64
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeBool(_ reflect.Value, _ bool) (bool, error) {
	var value bool
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeString(v reflect.Value, l32 bool) (string, error) {
	length, err := d.decodeLength(v, l32)
	if err != nil {
		return "", err
	}
	var n int
	buf := make([]byte, length)
	n, err = io.ReadFull(d.buf, buf)
	if err != nil {
		return "", err
	}
	if n != length {
		return "", io.ErrUnexpectedEOF
	}
	return *(*string)(unsafe.Pointer(&buf)), nil
}

func (d *Decoder) decodeSlice(v reflect.Value, l32 bool) (interface{}, error) {
	elType := v.Type().Elem()
	kind := elType.Kind()
	length, err := d.decodeLength(v, l32)
	if err != nil {
		return nil, err
	}
	slice := reflect.MakeSlice(reflect.SliceOf(elType), length, length)
	if length == 0 {
		return slice.Interface(), nil
	}
	for i := 0; i < length; i++ {
		val := reflect.New(elType).Elem()
		err = d.decodeValue(val, elType, kind, l32)
		if err != nil {
			return nil, err
		}
		slice.Index(i).Set(val)
	}
	return slice.Interface(), nil
}

func (d *Decoder) decodeArray(v reflect.Value, l32 bool) (interface{}, error) {
	t := v.Type().Elem()
	kind := t.Kind()
	array := reflect.New(reflect.ArrayOf(v.Len(), t)).Elem()
	if v.Len() == 0 {
		return array.Interface(), nil
	}
	for i := 0; i < v.Len(); i++ {
		val := v.Index(i)
		err := d.decodeValue(val, t, kind, l32)
		if err != nil {
			return nil, err
		}
		array.Index(i).Set(val)
	}
	return array.Interface(), nil
}

func (d *Decoder) decodeStruct(v reflect.Value, l32 bool) (interface{}, error) {
	t := v.Type()
	n := t.NumField()
	if n == 0 {
		return v.Interface(), nil
	}
	for i := 0; i < n; i++ {
		field := v.Field(i)
		if field.IsValid() && field.CanSet() {
			err := d.decodeValue(field, field.Type(), field.Kind(), l32)
			if err != nil {
				return nil, err
			}
		}
	}

	return v.Interface(), nil
}

func (d *Decoder) decodeMap(v reflect.Value, l32 bool) (interface{}, error) {
	t := v.Type()
	v = reflect.MakeMap(t)
	keyType := t.Key()
	keyKind := keyType.Kind()
	valueType := t.Elem()
	valueKind := valueType.Kind()
	length, err := d.decodeLength(v, l32)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return v.Interface(), nil
	}
	for i := 0; i < length; i++ {
		key := reflect.New(keyType).Elem()
		value := reflect.New(valueType).Elem()
		err = d.decodeValue(key, keyType, keyKind, l32)
		if err != nil {
			return nil, err
		}
		err = d.decodeValue(value, valueType, valueKind, l32)
		if err != nil {
			return nil, err
		}
		v.SetMapIndex(key, value)
	}
	return v.Interface(), nil
}
