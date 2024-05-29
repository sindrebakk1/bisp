package bisp

import (
	"errors"
	"fmt"
	"reflect"
)

type PKind uint8

const (
	Call PKind = iota
	Response
)

func (p PKind) String() string {
	if p == Call {
		return "Call"
	}
	return "Response"
}

var (
	pNameRegistry       = make(map[string]ID, 16)
	pTypeRegistry       = make(map[reflect.Type]ID, 16)
	pReverseRegistry    = make(map[ID]reflect.Type, 16)
	nextPID          ID = 1
)

type Procedure[T any] struct {
	Out           T
	TransactionID TransactionID
}

func RegisterProcedure[P any]() ID {
	var p P
	t := reflect.TypeOf(p)
	if t.Kind() != reflect.Struct {
		panic("procedure must be a struct")
	}
	name := t.Name()
	if _, ok := pNameRegistry[name]; ok {
		return pNameRegistry[name]
	}
	if _, ok := t.FieldByName("Procedure"); !ok {
		panic("procedure must have an embedded Procedure")
	}
	outField, ok := t.FieldByName("Out")
	if !ok {
		panic("procedure must have an Out field")
	}
	kind := outField.Type.Kind()
	if kind == reflect.Interface || kind == reflect.Ptr || kind == reflect.Invalid {
		panic("procedure Out field must be a concrete type and not invalid")
	}
	if _, ok = pNameRegistry[name]; !ok {
		registerParamTypes(t)

		pNameRegistry[name] = nextPID
		pTypeRegistry[t] = nextPID
		pReverseRegistry[nextPID] = t

		nextPID++
	}

	return pNameRegistry[name]
}

func registerParamTypes(t reflect.Type) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Name == "Procedure" || field.Name == "TransactionID" {
			continue
		}
		RegisterType(reflect.New(field.Type).Elem().Interface())
	}
}

func GetProcedureID(p any) (ID, error) {
	ID, ok := pTypeRegistry[reflect.TypeOf(p)]
	if !ok {
		return 0, errors.New(fmt.Sprintf("procedure not registered: %s", reflect.TypeOf(p)))
	}
	return ID, nil
}

func GetProcedureFromID(id ID) (reflect.Type, error) {
	typ, exists := pReverseRegistry[id]
	if !exists {
		return nil, errors.New(fmt.Sprintf("procedure with id %d not registered", id))
	}
	return typ, nil
}
