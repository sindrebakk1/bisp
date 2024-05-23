package tcp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"reflect"
)

type decoderFunc func(*Decoder, reflect.Value) (interface{}, error)

type Decoder struct {
	buf               *bytes.Buffer
	primitiveDecoders map[reflect.Kind]decoderFunc
	reader            io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		buf: new(bytes.Buffer),
		primitiveDecoders: map[reflect.Kind]decoderFunc{
			reflect.Uint:    func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeUint(v) },
			reflect.Uint8:   func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeUint8(v) },
			reflect.Uint16:  func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeUint16(v) },
			reflect.Uint32:  func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeUint32(v) },
			reflect.Uint64:  func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeUint64(v) },
			reflect.Int8:    func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeInt8(v) },
			reflect.Int:     func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeInt(v) },
			reflect.Int16:   func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeInt16(v) },
			reflect.Int32:   func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeInt32(v) },
			reflect.Int64:   func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeInt64(v) },
			reflect.Float32: func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeFloat32(v) },
			reflect.Float64: func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeFloat64(v) },
			reflect.Bool:    func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeBool(v) },
			reflect.String:  func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeString(v) },
			reflect.Slice:   func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeArrayOrSlice(v) },
			reflect.Array:   func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeArrayOrSlice(v) },
			reflect.Struct:  func(d *Decoder, v reflect.Value) (interface{}, error) { return d.decodeStruct(v) },
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
	body, err = d.DecodeBody(header.Type, uint16(header.Length))
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
		length  uint16
	)
	if version, err = d.decodeUint8(reflect.ValueOf(version)); err != nil {
		return nil, err
	}
	if Version(version) != CurrentVersion {
		return nil, errors.New("unsupported version")
	}
	if flags, err = d.decodeUint8(reflect.ValueOf(flags)); err != nil {
		return nil, err
	}
	if typeID, err = d.decodeUint16(reflect.ValueOf(typeID)); err != nil {
		return nil, err
	}
	if (Flag(flags) & FTransactionID) == FTransactionID {
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
	if length, err = d.decodeUint16(reflect.ValueOf(length)); err != nil {
		return nil, err
	}

	header.Version = Version(version)
	header.Flags = Flag(flags)
	header.Type = TypeID(typeID)
	header.TransactionID = transID
	header.Length = Length(length)

	return &header, nil
}

func (d *Decoder) DecodeBody(typeID TypeID, length uint16) (interface{}, error) {
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
	if err = d.decodeValue(val); err != nil {
		return nil, err
	}
	return val.Interface(), nil
}

func (d *Decoder) decodeValue(v reflect.Value) error {
	if decoder, ok := d.primitiveDecoders[v.Kind()]; ok {
		val, err := decoder(d, v)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(val))
		return nil
	} else {
		return errors.New("unsupported type")
	}
}

func (d *Decoder) decodeUint(_ reflect.Value) (uint, error) {
	var value uint32
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return uint(value), err
}

func (d *Decoder) decodeUint8(_ reflect.Value) (uint8, error) {
	var value uint8
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeUint16(_ reflect.Value) (uint16, error) {
	var value uint16
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeUint32(_ reflect.Value) (uint32, error) {
	var value uint32
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeUint64(_ reflect.Value) (uint64, error) {
	var value uint64
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeInt(_ reflect.Value) (int, error) {
	var value int32
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return int(value), err
}

func (d *Decoder) decodeInt8(_ reflect.Value) (int8, error) {
	var value int8
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeInt16(_ reflect.Value) (int16, error) {
	var value int16
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeInt32(_ reflect.Value) (int32, error) {
	var value int32
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeInt64(_ reflect.Value) (int64, error) {
	var value int64
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeFloat32(_ reflect.Value) (float32, error) {
	var value float32
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeFloat64(_ reflect.Value) (float64, error) {
	var value float64
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err
}

func (d *Decoder) decodeBool(_ reflect.Value) (bool, error) {
	var value bool
	err := binary.Read(d.buf, binary.BigEndian, &value)
	return value, err

}

func (d *Decoder) decodeString(v reflect.Value) (string, error) {
	length, err := d.decodeUint16(v)
	if err != nil {
		return "", err
	}
	buf := make([]byte, length)
	_, err = d.buf.Read(buf)
	return string(buf), err
}

func (d *Decoder) decodeArrayOrSlice(v reflect.Value) (interface{}, error) {
	t := v.Type()
	elType := t.Elem()
	length, err := d.decodeUint32(v)
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
		val, err = decoder(d, reflect.New(elType).Elem())
		if err != nil {
			return nil, err
		}
		slice = reflect.Append(slice, reflect.ValueOf(val))
	}
	return slice.Interface(), nil
}

func (d *Decoder) decodeStruct(v reflect.Value) (interface{}, error) {
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
		newFieldValue, err := d.primitiveDecoders[fieldKind](d, field)
		if err != nil {
			return nil, err
		}
		if field.IsValid() && field.CanSet() {
			field.Set(reflect.ValueOf(newFieldValue))
		}
	}

	return structPtr.Elem().Interface(), nil
}
