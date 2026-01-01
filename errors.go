package pgsqlbuilder

import "errors"

type BuilderError struct {
	Op  string
	Tag string
	Err error
}

func (e *BuilderError) Error() string {
	return e.Op + ": " + e.Err.Error()
}

var fieldNameNotFoundError = errors.New("field name not found")

var getColumnNameBuilderError = func(source string) *BuilderError {
	return &BuilderError{
		Op:  "get column name from " + source + " field",
		Err: fieldNameNotFoundError,
	}
}
var getClauseBuilderError = func(clause, source string, err error) *BuilderError {
	return &BuilderError{
		Op:  "get " + clause + " clause from " + source,
		Err: err,
	}
}
