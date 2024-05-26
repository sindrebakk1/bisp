package bisp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"reflect"
)

type decoderFunc func(*Decoder, reflect.Value, bool) (interface{}, error)

type Decoder struct {
	buf               *bytes.Buffer
	primitiveDecoders map[reflect.Kind]decoderFunc
	reader            io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		buf: new(bytes.Buffer),
		primitiveDecoders: map[reflect.Kind]decoderFunc{
			reflect.Uint:    func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeUint(v, b) },
			reflect.Uint8:   func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeUint8(v, b) },
			reflect.Uint16:  func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeUint16(v, b) },
			reflect.Uint32:  func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeUint32(v, b) },
			reflect.Uint64:  func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeUint64(v, b) },
			reflect.Int8:    func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeInt8(v, b) },
			reflect.Int:     func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeInt(v, b) },
			reflect.Int16:   func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeInt16(v, b) },
			reflect.Int32:   func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeInt32(v, b) },
			reflect.Int64:   func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeInt64(v, b) },
			reflect.Float32: func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeFloat32(v, b) },
			reflect.Float64: func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeFloat64(v, b) },
			reflect.Bool:    func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeBool(v, b) },
			reflect.String:  func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeString(v, b) },
			reflect.Slice:   func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeSlice(v, b) },
			reflect.Array:   func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeArray(v, b) },
			reflect.Struct:  func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeStruct(v, b) },
			reflect.Map:     func(d *Decoder, v reflect.Value, b bool) (interface{}, error) { return d.decodeMap(v, b) },
		},
		reader: r,
	}
}

func (d *Decoder) Decode(msg *Message) error {
	header, err := d.DecodeHeader()
	if err != nil {
		return err
	}
	var body interface{}
	body, err = d.DecodeBody(header.Type, uint32(header.Length), header.HasFlag(F32b))
	if err != nil {
		return err
	}
	d.buf.Reset()
	msg.Header = *header
	msg.Body = body
	return nil
}

func (d *Decoder) DecodeHeader() (*Header, error) {
	var header Header
	limitReader := io.LimitReader(d.reader, HeaderSize)
	n, err := io.Copy(d.buf, limitReader)
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
		tIDReader := io.LimitReader(d.reader, TransactionIDSize)
		n, err = io.Copy(d.buf, tIDReader)
		var tn int
		if tn, err = d.buf.Read(transID[:]); err != nil {
			return nil, err
		}
		if tn != TransactionIDSize {
			return nil, errors.New("unexpected end of transaction ID")
		}
	}
	if (Flag(flags) & F32b) == F32b {
		tIDReader := io.LimitReader(d.reader, LengthSize)
		n, err = io.Copy(d.buf, tIDReader)
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
	header.Type = TypeID(typeID)
	header.TransactionID = transID
	header.Length = Length(length)

	return &header, nil
}

func (d *Decoder) DecodeBody(typeID TypeID, length uint32, l32 bool) (interface{}, error) {
	limitReader := io.LimitReader(d.reader, int64(length))
	n, err := io.Copy(d.buf, limitReader)
	if err != nil {
		return nil, err

	}
	if n != int64(length) {
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
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if err = d.decodeValue(val, l32); err != nil {
		return nil, err
	}
	return val.Interface(), nil
}

func (d *Decoder) decodeValue(v reflect.Value, l32 bool) error {
	if decoder, ok := d.primitiveDecoders[v.Kind()]; ok {
		val, err := decoder(d, v, l32)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(val).Convert(v.Type()))
		return nil
	} else {
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
	buf := make([]byte, length)
	_, err = d.buf.Read(buf)
	return string(buf), err
}

func (d *Decoder) decodeSlice(v reflect.Value, l32 bool) (interface{}, error) {
	t := v.Type()
	elType := t.Elem()
	length, err := d.decodeLength(v, l32)
	if err != nil {
		return nil, err
	}
	slice := reflect.MakeSlice(reflect.SliceOf(elType), 0, int(length))
	if length == 0 {
		return slice.Interface(), nil
	}
	decoder, ok := d.primitiveDecoders[elType.Kind()]
	if !ok {
		return nil, errors.New("unsupported type")
	}
	for i := 0; i < int(length); i++ {
		var val interface{}
		val, err = decoder(d, reflect.New(elType).Elem(), l32)
		if err != nil {
			return nil, err
		}
		slice = reflect.Append(slice, reflect.ValueOf(val).Convert(elType))
	}
	return slice.Interface(), nil
}

func (d *Decoder) decodeArray(v reflect.Value, l32 bool) (interface{}, error) {
	t := v.Type()
	elType := t.Elem()
	array := reflect.New(reflect.ArrayOf(t.Len(), elType)).Elem()
	if t.Len() == 0 {
		return array.Interface(), nil
	}
	decoder, ok := d.primitiveDecoders[elType.Kind()]
	if !ok {
		return nil, errors.New("unsupported type")
	}
	for i := 0; i < t.Len(); i++ {
		val, err := decoder(d, reflect.New(elType).Elem(), l32)
		if err != nil {
			return nil, err
		}
		array.Index(i).Set(reflect.ValueOf(val).Convert(elType))
	}
	return array.Interface(), nil
}

func (d *Decoder) decodeStruct(v reflect.Value, l32 bool) (interface{}, error) {
	msgType := v.Type()
	structPtr := reflect.New(msgType)
	structValue := structPtr.Elem()
	for i := 0; i < msgType.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := msgType.Field(i)
		if !fieldType.IsExported() {
			continue
		}
		fieldKind := fieldVal.Kind()
		field := structValue.Field(i)
		newFieldValue, err := d.primitiveDecoders[fieldKind](d, field, l32)
		if err != nil {
			return nil, err
		}
		if field.IsValid() && field.CanSet() {
			field.Set(reflect.ValueOf(newFieldValue).Convert(field.Type()))
		}
	}

	return structPtr.Elem().Interface(), nil
}

func (d *Decoder) decodeMap(v reflect.Value, l32 bool) (interface{}, error) {
	mapType := v.Type()
	mapVal := reflect.MakeMap(mapType)
	keyType := mapType.Key()
	valueType := mapType.Elem()
	length, err := d.decodeLength(v, l32)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return mapVal.Interface(), nil
	}
	var (
		keyDecoder   decoderFunc
		valueDecoder decoderFunc
		ok           bool
	)
	keyDecoder, ok = d.primitiveDecoders[keyType.Kind()]
	if !ok {
		return nil, errors.New("unsupported key type")
	}
	valueDecoder, ok = d.primitiveDecoders[valueType.Kind()]
	if !ok {
		return nil, errors.New("unsupported value type")
	}
	for i := 0; i < length; i++ {
		key := reflect.New(keyType).Elem()
		value := reflect.New(valueType).Elem()
		var (
			newKey   interface{}
			newValue interface{}
		)
		newKey, err = keyDecoder(d, key, l32)
		if err != nil {
			return nil, err
		}
		newValue, err = valueDecoder(d, value, l32)
		if err != nil {
			return nil, err
		}
		mapVal.SetMapIndex(reflect.ValueOf(newKey).Convert(keyType), reflect.ValueOf(newValue).Convert(valueType))
	}
	return mapVal.Interface(), nil
}
