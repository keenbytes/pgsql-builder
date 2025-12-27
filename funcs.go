package pgsqlbuilder

import (
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var intRegex = regexp.MustCompile(`^-?\d+$`)
var floatRegex = regexp.MustCompile(`^-?\d*\.\d+$`)

// IsFieldKindSupported checks if a specific reflect kind of the field is supported by the Builder.
func IsFieldKindSupported(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.String, reflect.Bool:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// IsStructField checks if a field exists in a struct.
func IsStructField(u interface{}, field string) bool {
	v := reflect.ValueOf(u)
	i := reflect.Indirect(v)
	s := i.Type()

	for j := 0; j < s.NumField(); j++ {
		f := s.Field(j)
		k := f.Type.Kind()

		if !IsFieldKindSupported(k) {
			continue
		}

		if f.Name == field {
			return true
		}
	}

	return false
}

// StructFieldValueFromString takes a field value as string and converts it (if possible) to a value type of that field.
func StructFieldValueFromString(obj interface{}, name string, value string) (bool, interface{}) {
	objValue := reflect.ValueOf(obj)
	objIndirect := reflect.Indirect(objValue)
	objType := objIndirect.Type()

	for j := 0; j < objType.NumField(); j++ {
		field := objType.Field(j)
		kind := field.Type.Kind()

		if !IsFieldKindSupported(kind) {
			continue
		}

		if field.Name == name {
			switch kind {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				if intRegex.MatchString(value) {
					v, err := strconv.ParseInt(value, 10, 64)
					if err == nil {
						return true, v
					}
				}
			case reflect.Float32, reflect.Float64:
				if floatRegex.MatchString(value) || intRegex.MatchString(value) {
					v, err := strconv.ParseFloat(value, 64)
					if err == nil {
						return true, v
					}
				}
			case reflect.Bool:
				if strings.ToLower(value) == "true" || strings.ToLower(value) == "false" {
					v, err := strconv.ParseBool(value)
					if err == nil {
						return true, v
					}
				}
			case reflect.String:
				return true, value
			}
			return false, nil
		}
	}

	return false, nil
}

// FieldToColumn converts struct field name to database column name.
func FieldToColumn(s string) string {
	if s == "ID" {
		return "id"
	}

	o := ""

	var prev rune
	for i, ch := range s {
		if i == 0 {
			o += strings.ToLower(string(ch))
			prev = ch
			continue
		}

		if unicode.IsUpper(ch) {
			if prev == 'I' && ch == 'D' {
				o += strings.ToLower(string(ch))
				continue
			}

			o += "_" + strings.ToLower(string(ch))
			prev = ch
			continue
		}

		o += string(ch)
		prev = ch
	}

	return o
}

// ColumnToField converts database column name to struct field name.
func ColumnToField(s string) string {
	parts := strings.Split(s, "_")
	result := ""

	for _, part := range parts {
		if part == "id" {
			result += "ID"
			continue
		}
		result += strings.ToUpper(part[:1]) + part[1:]
	}

	return result
}

// PrettifyCreateTable prettifies SQL query to make it more human-readable.
func PrettifyCreateTable(sql string) string {
	sql = strings.Replace(sql, "(", "(\n  ", 1)
	sql = strings.ReplaceAll(sql, ",", ",\n  ")

	return sql
}

func MapInterface(mapObj map[string]interface{}) []interface{} {
	var interfaces []interface{}

	numObjs := len(mapObj)
	sorted := make([]string, 0, numObjs)
	for field := range mapObj {
		sorted = append(sorted, field)
	}
	sort.Strings(sorted)

	for _, val := range sorted {
		interfaces = append(interfaces, mapObj[val])
	}

	return interfaces
}
