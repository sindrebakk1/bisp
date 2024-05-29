package bisp

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	nameRegistry       = make(map[string]ID, 32)
	typeRegistry       = make(map[reflect.Type]ID, 32)
	reverseRegistry    = make(map[ID]reflect.Type, 32)
	nextID          ID = 0
)

var (
	tUint    = reflect.TypeOf(uint(0))
	tUint8   = reflect.TypeOf(uint8(0))
	tUint16  = reflect.TypeOf(uint16(0))
	tUint32  = reflect.TypeOf(uint32(0))
	tUint64  = reflect.TypeOf(uint64(0))
	tInt     = reflect.TypeOf(int(0))
	tInt8    = reflect.TypeOf(int8(0))
	tInt16   = reflect.TypeOf(int16(0))
	tInt32   = reflect.TypeOf(int32(0))
	tInt64   = reflect.TypeOf(int64(0))
	tFloat32 = reflect.TypeOf(float32(0))
	tFloat64 = reflect.TypeOf(float64(0))
	tBool    = reflect.TypeOf(false)
	tString  = reflect.TypeOf("")
)

func RegisterType(value interface{}) ID {
	t := reflect.TypeOf(value)
	var name string
	if t == nil {
		name = "nil"
	} else {
		name = t.String()
	}
	if _, ok := nameRegistry[name]; !ok {
		nameRegistry[name] = nextID
		typeRegistry[t] = nextID
		reverseRegistry[nextID] = t

		nextID++
		if t != nil && t.Kind() != reflect.Slice && t.Kind() != reflect.Array && t.Kind() != reflect.Map {
			slice := reflect.New(reflect.SliceOf(t)).Elem().Interface()
			RegisterType(slice)
		}
	}
	return nameRegistry[name]
}

func GetIDFromType(value interface{}) (ID, error) {
	t := reflect.TypeOf(value)
	var name string
	if t == nil {
		name = "nil"
	} else {
		name = t.String()
	}
	ID, exists := nameRegistry[name]
	if !exists {
		return 0, errors.New("type not registered")
	}
	return ID, nil
}

func GetTypeFromID(id ID) (reflect.Type, error) {
	typ, exists := reverseRegistry[id]
	if !exists {
		return nil, errors.New("type not registered")
	}
	return typ, nil
}

func SyncTypeRegistry(other map[reflect.Type]ID) []error {
	errs := make([]error, 0)
	for typ, id := range other {
		if _, ok := typeRegistry[typ]; !ok {
			errs = append(errs, errors.New(fmt.Sprintf("type %s not registered", typ)))
		} else {
			typeRegistry[typ] = id
			reverseRegistry[id] = typ
		}
	}
	for typ, _ := range typeRegistry {
		if _, ok := other[typ]; !ok {
			errs = append(errs, errors.New(fmt.Sprintf("type %s registered locally, but is not supported by server", typ)))
		}
	}
	return errs
}

func GetTypeRegistry() map[reflect.Type]ID {
	return typeRegistry
}

func getUnderlyingType(v reflect.Value, k reflect.Kind) reflect.Type {
	switch k {
	case reflect.Uint:
		return tUint
	case reflect.Uint8:
		return tUint8
	case reflect.Uint16:
		return tUint16
	case reflect.Uint32:
		return tUint32
	case reflect.Uint64:
		return tUint64
	case reflect.Int:
		return tInt
	case reflect.Int8:
		return tInt8
	case reflect.Int16:
		return tInt16
	case reflect.Int32:
		return tInt32
	case reflect.Int64:
		return tInt64
	case reflect.Float32:
		return tFloat32
	case reflect.Float64:
		return tFloat64
	case reflect.Bool:
		return tBool
	case reflect.String:
		return tString
	default:
		return v.Type()
	}
}

func init() {
	RegisterType(nil)
	RegisterType(byte(0))
	RegisterType(0)
	RegisterType(int8(0))
	RegisterType(int16(0))
	RegisterType(int32(0))
	RegisterType(int64(0))
	RegisterType(uint(0))
	RegisterType(uint8(0))
	RegisterType(uint16(0))
	RegisterType(uint32(0))
	RegisterType(uint64(0))
	RegisterType(float32(0))
	RegisterType(float64(0))
	RegisterType(false)
	RegisterType(rune(0))
	RegisterType("")
}
