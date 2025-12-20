package pgsqlbuilder

import (
	"fmt"
	"reflect"
	"sort"
)

type Filters map[string]OpVal

func (f Filters) Add(name string, value OpVal) {
	f[name] = value
}

const (
	Raw = "_"
)

// Interfaces returns list of interfaces from filters map (used in querying)
func Interfaces(filters *Filters) []interface{} {
	var interfaces []interface{}

	if filters == nil || len(*filters) == 0 {
		return interfaces
	}

	numFilters := len(*filters)
	sorted := make([]string, 0, numFilters)
	for k := range *filters {
		if k == Raw {
			continue
		}
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	for _, filter := range sorted {
		interfaces = append(interfaces, (*filters)[filter].Val)
	}

	// Get pointers to values from raw query
	rawFilters, ok := (*filters)[Raw]
	if !ok {
		return interfaces
	}

	rawFilterType := reflect.TypeOf(rawFilters.Val)
	if rawFilterType.Kind() != reflect.Slice && rawFilterType.Kind() != reflect.Array {
		return interfaces
	}
	if reflect.ValueOf(rawFilters.Val).Len() < 2 {
		return interfaces
	}

	// first raw item in the array is a query
	for i := 1; i < reflect.ValueOf(rawFilters.Val).Len(); i++ {
		placeholderValue := reflect.TypeOf(rawFilters.Val.([]interface{})[i])

		if placeholderValue.Kind() != reflect.Slice && placeholderValue.Kind() != reflect.Array {
			interfaces = append(interfaces, rawFilters.Val.([]interface{})[i])

			continue
		}

		valInt8s, ok := rawFilters.Val.([]interface{})[i].([]int8)
		if ok {
			for j := 0; j < len(valInt8s); j++ {
				interfaces = append(interfaces, valInt8s[j])
			}
			continue
		}
		valInt16s, ok := rawFilters.Val.([]interface{})[i].([]int16)
		if ok {
			for j := 0; j < len(valInt16s); j++ {
				interfaces = append(interfaces, valInt16s[j])
			}
			continue
		}
		valInt32s, ok := rawFilters.Val.([]interface{})[i].([]int32)
		if ok {
			for j := 0; j < len(valInt32s); j++ {
				interfaces = append(interfaces, valInt32s[j])
			}
			continue
		}
		valInt64s, ok := rawFilters.Val.([]interface{})[i].([]int64)
		if ok {
			for j := 0; j < len(valInt64s); j++ {
				interfaces = append(interfaces, valInt64s[j])
			}
			continue
		}
		valInts, ok := rawFilters.Val.([]interface{})[i].([]int)
		if ok {
			for j := 0; j < len(valInts); j++ {
				interfaces = append(interfaces, valInts[j])
			}
			continue
		}
		valUint8s, ok := rawFilters.Val.([]interface{})[i].([]uint8)
		if ok {
			for j := 0; j < len(valUint8s); j++ {
				interfaces = append(interfaces, valUint8s[j])
			}
			continue
		}
		valUint16s, ok := rawFilters.Val.([]interface{})[i].([]uint16)
		if ok {
			for j := 0; j < len(valUint16s); j++ {
				interfaces = append(interfaces, valUint16s[j])
			}
			continue
		}
		valUint32s, ok := rawFilters.Val.([]interface{})[i].([]uint32)
		if ok {
			for j := 0; j < len(valUint32s); j++ {
				interfaces = append(interfaces, valUint32s[j])
			}
			continue
		}
		valUint64s, ok := rawFilters.Val.([]interface{})[i].([]uint64)
		if ok {
			for j := 0; j < len(valUint64s); j++ {
				interfaces = append(interfaces, valUint64s[j])
			}
			continue
		}
		valUints, ok := rawFilters.Val.([]interface{})[i].([]uint)
		if ok {
			for j := 0; j < len(valUints); j++ {
				interfaces = append(interfaces, valUints[j])
			}
			continue
		}
		valFloat32s, ok := rawFilters.Val.([]interface{})[i].([]float32)
		if ok {
			for j := 0; j < len(valFloat32s); j++ {
				interfaces = append(interfaces, valFloat32s[j])
			}
			continue
		}
		valFloat64s, ok := rawFilters.Val.([]interface{})[i].([]float64)
		if ok {
			for j := 0; j < len(valFloat64s); j++ {
				interfaces = append(interfaces, valFloat64s[j])
			}
			continue
		}
		valBools, ok := rawFilters.Val.([]interface{})[i].([]bool)
		if ok {
			for j := 0; j < len(valBools); j++ {
				interfaces = append(interfaces, valBools[j])
			}
			continue
		}
		valStrings, ok := rawFilters.Val.([]interface{})[i].([]string)
		if ok {
			for j := 0; j < len(valStrings); j++ {
				interfaces = append(interfaces, valStrings[j])
			}
		}
	}

	return interfaces
}

func SetObjFields(obj interface{}, values *Filters) error {
	if values == nil || len(*values) == 0 {
		return nil
	}

	objValue := reflect.ValueOf(obj)

	if objValue.Kind() != reflect.Ptr || objValue.IsNil() {
		return fmt.Errorf("obj must be a non‑nil pointer to a struct")
	}

	structVal := objValue.Elem()
	if structVal.Kind() != reflect.Struct {
		return fmt.Errorf("obj must point to a struct, got %s", structVal.Kind())
	}

	var firstErr error
	typ := structVal.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		if field.PkgPath != "" { // PkgPath is non‑empty for unexported fields
			continue
		}

		if !IsFieldKindSupported(field.Type.Kind()) {
			continue
		}

		value, ok := (*values)[field.Name]
		if !ok {
			continue
		}

		dest := structVal.Field(i)

		// TODO: We do not support pointers at this stage
		if dest.Kind() == reflect.Ptr {
			continue
		}

		inVal := reflect.ValueOf(value.Val)

		if inVal.Type().AssignableTo(dest.Type()) {
			dest.Set(inVal)
			continue
		}

		if inVal.Type().ConvertibleTo(dest.Type()) {
			dest.Set(inVal.Convert(dest.Type()))
			continue
		}

		err := fmt.Errorf("cannot set field %s (%s) with value of type %T",
			field.Name, dest.Type(), value)
		if firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
