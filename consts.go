package pgsqlbuilder

const (
	_ = 1 << iota
	FlagHasModificationFields
)

const (
	_ = 1 << iota
	FieldFlagUnique
	FieldFlagNotString
	FieldFlagPassword
)

const (
	DefaultTagName = "sql"
)
