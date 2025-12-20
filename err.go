package pgsqlbuilder

// ErrBuilder is error that is returned by the builder.
type ErrBuilder struct {
	Op  string
	Tag string
	Err error
}

func (e ErrBuilder) Error() string {
	return e.Err.Error()
}

func (e ErrBuilder) Unwrap() error {
	return e.Err
}
