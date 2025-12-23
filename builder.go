package pgsqlbuilder

import (
	"fmt"
	"regexp"
)

// Builder reflects the object to generate and cache PostgreSQL queries (CREATE TABLE, INSERT, UPDATE etc.).
// Database table and column names are lowercase with underscore and they are generated from field names.
type Builder struct {
	tagName string
	flags   int64

	queryCreateTable            string
	queryDropTable              string
	queryInsert                 string
	queryUpdateById             string
	queryInsertOnConflictUpdate string
	querySelectById             string
	queryDeleteById             string
	querySelectPrefix           string
	querySelectCountPrefix      string
	queryDeletePrefix           string
	queryUpdatePrefix           string

	tableName string

	fieldColumnName   map[string]string
	columnFieldName   map[string]string
	fieldFlags        map[string]int64
	fieldColumnType   map[string]string
	fieldDefault      map[string]string
	columnDefinitions []string
	columnNames       []string

	reflectError error
}

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

var (
	regexpFieldInRaw = regexp.MustCompile(`\.[a-zA-Z0-9_]+`)
)

// New takes a struct and returns a Builder instance.
func New(obj interface{}, options Options) *Builder {
	builder := &Builder{}

	builder.tagName = "sql"
	if options.TagName != "" {
		builder.tagName = options.TagName
	}

	builder.reflect(obj, options.TableNamePrefix)
	return builder
}

// Err returns an error that appeared during reflecting the struct.
func (b *Builder) Err() error {
	return b.reflectError
}

// Flags returns flags.
func (b *Builder) Flags() int64 {
	return b.flags
}

// DropTable returns an SQL query for dropping the table.
func (b *Builder) DropTable() string {
	return b.queryDropTable + ";"
}

// CreateTable returns an SQL query for creating the table.
func (b *Builder) CreateTable() string {
	return b.queryCreateTable + ";"
}

// Insert returns an SQL query for inserting a new object to the table.
func (b *Builder) Insert() string {
	return b.queryInsert + ";"
}

// UpdateById returns an SQL query for updating an object by their ID.
func (b *Builder) UpdateById() string {
	return b.queryUpdateById + ";"
}

// InsertOnConflictUpdate returns an SQL query for inserting when conflict is detected.
func (b *Builder) InsertOnConflictUpdate() string {
	return b.queryInsertOnConflictUpdate + ";"
}

// SelectById returns an SQL query for selecting object by its ID.
func (b *Builder) SelectById() string {
	return b.querySelectById + ";"
}

// DeleteById returns an SQL query for deleting object by its ID.
func (b *Builder) DeleteById() string {
	return b.queryDeleteById + ";"
}

// Select returns a SELECT query with WHERE condition built from 'filters' (field-value pairs).
// Struct fields in 'filters' argument are sorted alphabetically. Hence, when used with database connection, their values (or pointers to it) must be sorted as well.
// Columns in the SELECT query are ordered the same way as they are defined in the struct, eg. SELECT field1_column, field2_column, ... etc.
func (b *Builder) Select(order []string, limit int, offset int, filters *Filters) (string, error) {
	query := b.querySelectPrefix

	qOrder, err := b.queryOrder(order)
	if err != nil {
		return "", &ErrBuilder{
			Op:  "Select",
			Err: err,
		}
	}

	qLimitOffset := b.queryLimitOffset(limit, offset)
	qWhere, err := b.queryFilters(filters, 1)
	if err != nil {
		return "", &ErrBuilder{
			Op:  "Select",
			Err: err,
		}
	}

	if qWhere != "" {
		query += " WHERE " + qWhere
	}
	if qOrder != "" {
		query += " ORDER BY " + qOrder
	}
	if qLimitOffset != "" {
		query += " " + qLimitOffset
	}

	return query + ";", nil
}

// SelectCount returns a SELECT COUNT(*) query to count rows with WHERE condition built from 'filters' (field-value pairs).
// Struct fields in 'filters' argument are sorted alphabetically. Hence, when used with database connection, their values (or pointers to it) must be sorted as well.
func (b *Builder) SelectCount(filters *Filters) (string, error) {
	query := b.querySelectCountPrefix

	qWhere, err := b.queryFilters(filters, 1)
	if err != nil {
		return "", &ErrBuilder{
			Op:  "SelectCount",
			Err: err,
		}
	}

	if qWhere != "" {
		query += " WHERE " + qWhere
	}

	return query + ";", nil
}

// Delete returns a DELETE query with WHERE condition built from 'filters' (field-value pairs).
// Struct fields in 'filters' argument are sorted alphabetically. Hence, when used with database connection, their values (or pointers to it) must be sorted as well.
func (b *Builder) Delete(filters *Filters) (string, error) {
	query := b.queryDeletePrefix

	qWhere, err := b.queryFilters(filters, 1)
	if err != nil {
		return "", &ErrBuilder{
			Op:  "Delete",
			Err: err,
		}
	}

	if qWhere != "" {
		query += " WHERE " + qWhere
	}

	return query + ";", nil
}

// DeleteReturningId returns a DELETE query with WHERE condition built from 'filters' (field-value pairs) with RETURNING id.
// Struct fields in 'filters' argument are sorted alphabetically. Hence, when used with database connection, their values (or pointers to it) must be sorted as well.
func (b *Builder) DeleteReturningId(filters *Filters) (string, error) {
	query := b.queryDeletePrefix

	qWhere, err := b.queryFilters(filters, 1)
	if err != nil {
		return "", &ErrBuilder{
			Op:  "Delete",
			Err: err,
		}
	}

	if qWhere != "" {
		query += " WHERE " + qWhere
	}
	query += fmt.Sprintf(` RETURNING "%s";`, b.fieldColumnName["Id"])

	return query, nil
}

// Update returns an UPDATE query where specified struct fields (columns) are updated and rows match specific WHERE condition built from 'filters' (field-value pairs).
// Struct fields in 'values' and 'filters' arguments, are sorted alphabetically. Hence, when used with database connection, their values (or pointers to it) must be sorted as well.
func (b *Builder) Update(values map[string]interface{}, filters *Filters) (string, error) {
	query := b.queryUpdatePrefix

	qSet, lastVarNumber, err := b.querySet(values)
	if err != nil {
		return "", &ErrBuilder{
			Op:  "Update",
			Err: err,
		}
	}

	query += " " + qSet

	qWhere, err := b.queryFilters(filters, lastVarNumber+1)
	if err != nil {
		return "", &ErrBuilder{
			Op:  "Update",
			Err: err,
		}
	}

	if qWhere != "" {
		query += " WHERE " + qWhere
	}

	return query + ";", nil
}

// DatabaseColumnToFieldName takes a database column and converts it to a struct field name.
func (b *Builder) DatabaseColumnToFieldName(n string) string {
	return b.columnFieldName[n]
}

// HasModificationFields returns true if all the following int64 fields are present: CreatedAt, CreatedBy, ModifiedAt, ModifiedBy.
func (b *Builder) HasModificationFields() bool {
	return b.flags&FlagHasModificationFields > 0
}

// UniqueFields returns a list with field names that are unique.
func (b *Builder) UniqueFields() []string {
	uniqFields := make([]string, 0, len(b.fieldColumnName))
	for field := range b.fieldColumnName {
		if b.fieldFlags[field]&FieldFlagUnique > 0 {
			uniqFields = append(uniqFields, field)
		}
	}

	return uniqFields
}

// PasswordFields returns a list with field names that are passwords.
func (b *Builder) PasswordFields() []string {
	passFields := make([]string, 0, len(b.fieldColumnName))
	for field := range b.fieldColumnName {
		if b.fieldFlags[field]&FieldFlagPassword > 0 {
			passFields = append(passFields, field)
		}
	}

	return passFields
}
