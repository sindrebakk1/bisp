package bisp

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	typeRegistry           = make(map[reflect.Type]TypeID, 32)
	reverseRegistry        = make(map[TypeID]reflect.Type, 32)
	nextID          TypeID = 0
)

func RegisterType(value interface{}) TypeID {
	t := reflect.TypeOf(value)
	if _, ok := typeRegistry[t]; !ok {
		typeRegistry[t] = nextID
		reverseRegistry[nextID] = t

		nextID++
	}
	if t != nil && t.Kind() != reflect.Slice && t.Kind() != reflect.Array && t.Kind() != reflect.Map {
		slice := reflect.New(reflect.SliceOf(t)).Elem().Interface()
		RegisterType(slice)
	}
	return typeRegistry[t]
}

func GetIDFromType(value interface{}) (TypeID, error) {
	ID, exists := typeRegistry[reflect.TypeOf(value)]
	if !exists {
		return 0, errors.New("type not registered")
	}
	return ID, nil
}

func GetTypeFromID(id TypeID) (reflect.Type, error) {
	typ, exists := reverseRegistry[id]
	if !exists {
		return nil, errors.New("type not registered")
	}
	return typ, nil
}

func SyncTypeRegistry(other map[reflect.Type]TypeID) []error {
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

func GetTypeRegistry() map[reflect.Type]TypeID {
	return typeRegistry
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
