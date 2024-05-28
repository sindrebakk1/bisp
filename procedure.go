package bisp

import (
	"fmt"
	"reflect"
)

type ProcedureFunc func(args []any) any

var (
	pRegistry               = make(map[reflect.Type]TypeID, 32)
	reversePRegistry        = make(map[TypeID]reflect.Type, 32)
	nextProcedureID  TypeID = 1
)

func GetProcedureIDFromType(t reflect.Type) (TypeID, error) {
	typeID, exists := pRegistry[t]
	if !exists {
		return 0, fmt.Errorf("type not registered")
	}
	return typeID, nil
}

func RegisterProcedure(fn ProcedureFunc) TypeID {
	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func || t.NumOut() == 0 {
		return 0
	}
	if _, ok := pRegistry[t]; !ok {
		pRegistry[t] = nextProcedureID
		reversePRegistry[nextProcedureID] = t
		nextProcedureID++
	}
	return pRegistry[t]
}
