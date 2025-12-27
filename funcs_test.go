package pgsqlbuilder

import "testing"

type structFieldValueFromString struct {
	Int64Field int64
}

func TestStructFieldValueFromString(t *testing.T) {
	testObj := &structFieldValueFromString{}

	ok, value := StructFieldValueFromString(testObj, "Int64Field", "0")
	if !ok {
		t.Fatal("Failed to parse string value")
	}

	if value.(int64) != 0 {
		t.Fatal("Parsed value is invalid")
	}
}
