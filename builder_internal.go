package pgsqlbuilder

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

func (b *Builder) initMaps(numField int) {
	b.fieldColumnName = make(map[string]string, numField)
	b.columnFieldName = make(map[string]string, numField)
	b.fieldFlags = make(map[string]int64, numField)
	b.fieldColumnType = make(map[string]string, numField)
	b.fieldDefault = make(map[string]string, numField)
	b.columnDefinitions = make([]string, 0, numField)
	b.columnNames = make([]string, 0, numField)
}

func (b *Builder) reflect(obj interface{}, tableNamePrefix string) {
	b.reflectFieldTags(obj)
	b.reflectBuildQueries(obj, tableNamePrefix)
}

func (b *Builder) reflectFieldTags(obj interface{}) {
	objValue := reflect.ValueOf(obj)
	objIndirectValue := reflect.Indirect(objValue)
	objType := objIndirectValue.Type()

	b.initMaps(objType.NumField())

	for j := 0; j < objType.NumField(); j++ {
		field := objType.Field(j)
		fieldTypeKind := field.Type.Kind()

		// Only basic golang types are included as columns for the database table.
		// Check the function below for the details.
		if !IsFieldKindSupported(fieldTypeKind) {
			continue
		}

		// Get value of field's 2sql and 2sql_val tags ('2sql' or different when TagName provided in options).
		tagValue := field.Tag.Get(b.tagName)
		valTagValue := field.Tag.Get(b.tagName + "_val")

		// Go through tag values and parse out the ones we're interested in.
		b.setFieldFromTag(tagValue, field.Name)
		if b.reflectError != nil {
			return
		}

		if valTagValue != "" {
			b.fieldDefault[field.Name] = valTagValue
		}
	}
}

func (b *Builder) reflectBuildQueries(obj interface{}, tableNamePrefix string) {
	objValue := reflect.ValueOf(obj)
	objIndirectValue := reflect.Indirect(objValue)
	objType := objIndirectValue.Type()

	if objType.String() == "reflect.Value" {
		objType = reflect.ValueOf(obj.(reflect.Value).Interface()).Type().Elem().Elem()
	}

	objTypeName := objType.Name()
	// if struct is User_Register, then take User as base for table name.
	if strings.Contains(objTypeName, "_") {
		objTypeName = strings.Split(objTypeName, "_")[0]
	}

	b.tableName = fmt.Sprintf(`"%s"`, tableNamePrefix+CamelCaseToSnakeCase(objTypeName))

	var (
		columnNames                string
		columnNamesWithoutId       string
		valuesWithoutId            string
		values                     string
		columnNamesWithValues      string
		columnNamesWithValuesAgain string
		numColumn                  int
		modificationFields         int
	)

	for j := 0; j < objType.NumField(); j++ {
		field := objType.Field(j)
		fieldTypeKind := field.Type.Kind()

		// Only basic golang types are included as columns for the database table.
		// Check the function below for the details.
		if !IsFieldKindSupported(fieldTypeKind) {
			continue
		}

		if fieldTypeKind != reflect.String && b.fieldFlags[field.Name]&FieldFlagNotString == 0 {
			b.fieldFlags[field.Name] += FieldFlagNotString
		}

		columnName := CamelCaseToSnakeCase(field.Name)
		b.fieldColumnName[field.Name] = columnName
		b.columnFieldName[columnName] = field.Name

		unique := false
		if b.fieldFlags[field.Name]&FieldFlagUnique > 0 {
			unique = true
		}

		columnDefinition := b.columnDefinitionFromField(field.Name, field.Type.String(), unique)
		b.columnDefinitions = append(b.columnDefinitions, fmt.Sprintf(`"%s" %s`, columnName, columnDefinition))
		b.columnNames = append(b.columnNames, fmt.Sprintf(`"%s"`, columnName))

		// Assuming that primary field is named ID and that it is always first -> TODO: add check

		if b.isFieldModification(field.Name, fieldTypeKind) {
			modificationFields++
		}
	}

	if modificationFields == 4 && b.flags&FlagHasModificationFields == 0 {
		b.flags += FlagHasModificationFields
	}

	// Assuming that struct has at least 2 fields -> TODO: add check
	numColumn = len(b.columnNames)

	columnNamesWithoutId = strings.Join(b.columnNames[1:], ",")
	columnNames = strings.Join(b.columnNames, ",")
	valuesWithoutId = "?" + strings.Repeat(",?", numColumn-2)

	columnNamesWithValues = strings.Join(b.columnNames[1:], "=?,") + "=?"
	columnNamesWithValuesAgain = columnNamesWithValues
	for i := 1; i <= numColumn*2; i++ {
		columnNamesWithValues = strings.Replace(columnNamesWithValues, "?", fmt.Sprintf("$%d", i), 1)
		valuesWithoutId = strings.Replace(valuesWithoutId, "?", fmt.Sprintf("$%d", i), 1)
		if i > numColumn {
			columnNamesWithValuesAgain = strings.Replace(columnNamesWithValuesAgain, "?", fmt.Sprintf("$%d", i), 1)
		}
	}
	values = valuesWithoutId + fmt.Sprintf(",$%d", numColumn)

	idColumn := `"id"`

	b.queryDropTable = fmt.Sprintf("DROP TABLE IF EXISTS %s", b.tableName)
	b.queryCreateTable = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", b.tableName, strings.Join(b.columnDefinitions, ","))

	b.queryDeleteById = fmt.Sprintf("DELETE FROM %s WHERE %s = $1", b.tableName, idColumn)
	b.queryDeletePrefix = fmt.Sprintf("DELETE FROM %s", b.tableName)

	b.queryUpdateById = fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d", b.tableName, columnNamesWithValues, idColumn, numColumn)
	b.queryUpdatePrefix = fmt.Sprintf("UPDATE %s SET", b.tableName)

	b.queryInsert = fmt.Sprintf("INSERT INTO %s(%s) VALUES (%s) RETURNING %s", b.tableName, columnNamesWithoutId, valuesWithoutId, idColumn)
	b.queryInsertOnConflictUpdate = fmt.Sprintf("INSERT INTO %s(%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s RETURNING %s", b.tableName, columnNames, values, idColumn, columnNamesWithValuesAgain, idColumn)

	b.querySelectById = fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", columnNames, b.tableName, idColumn)
	b.querySelectPrefix = fmt.Sprintf("SELECT %s FROM %s", columnNames, b.tableName)

	b.querySelectCountPrefix = fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", b.tableName)
}

func (b *Builder) setFieldFromTag(tag string, fieldName string) {
	opts := strings.Split(tag, " ")
	for _, opt := range opts {
		b.setFieldFromTagOptWithoutVal(opt, fieldName)
	}
}

func (b *Builder) setFieldFromTagOptWithoutVal(opt string, fieldName string) {
	if opt == "uniq" && b.fieldFlags[fieldName]&FieldFlagUnique == 0 {
		b.fieldFlags[fieldName] += FieldFlagUnique
		return
	}

	if opt == "pass" && b.fieldFlags[fieldName]&FieldFlagPassword == 0 {
		b.fieldFlags[fieldName] += FieldFlagPassword
		return
	}

	if !strings.HasPrefix(opt, "type:") {
		return
	}

	typeArr := strings.Split(opt, ":")
	typeUpperCase := strings.ToUpper(typeArr[1])
	if typeUpperCase == "TEXT" || typeUpperCase == "BPCHAR" {
		b.fieldColumnType[fieldName] = typeUpperCase
		return
	}
	m, _ := regexp.MatchString(`^(VARCHAR|CHARACTER VARYING|BPCHAR|CHAR|CHARACTER)\([0-9]+\)$`, typeUpperCase)
	if m {
		b.fieldColumnType[fieldName] = typeUpperCase
		return
	}
}

// Mapping database column type to struct field type
func (b *Builder) columnDefinitionFromField(fieldName string, fieldType string, isUnique bool) string {
	// 'Id' and 'Flags' are special fields
	if fieldName == "Id" {
		return "SERIAL PRIMARY KEY"
	}

	if fieldName == "Flags" {
		return "BIGINT NOT NULL DEFAULT 0"
	}

	fieldColumnType, ok := b.fieldColumnType[fieldName]
	if ok && fieldColumnType != "" {
		definition := fmt.Sprintf("%s NOT NULL DEFAULT ''", fieldColumnType)

		if isUnique {
			definition += " UNIQUE"
		}

		return definition
	}

	definition := ""
	switch fieldType {
	case "string":
		definition = "VARCHAR(255) NOT NULL DEFAULT ''"
	case "bool":
		definition = "BOOLEAN NOT NULL DEFAULT false"
	case "int64":
		definition = "BIGINT NOT NULL DEFAULT 0"
	case "int32":
		definition = "INTEGER NOT NULL DEFAULT 0"
	case "int16":
		definition = "SMALLINT NOT NULL DEFAULT 0"
	case "int8":
		definition = "SMALLINT NOT NULL DEFAULT 0"
	case "int":
		definition = "BIGINT NOT NULL DEFAULT 0"
	case "uint64":
		definition = "BIGINT NOT NULL DEFAULT 0"
	case "uint32":
		definition = "INTEGER NOT NULL DEFAULT 0"
	case "uint16":
		definition = "SMALLINT NOT NULL DEFAULT 0"
	case "uint8":
		definition = "SMALLINT NOT NULL DEFAULT 0"
	case "uint":
		definition = "BIGINT NOT NULL DEFAULT 0"
	// TODO: Consider something different
	default:
		definition = "VARCHAR(255) NOT NULL DEFAULT ''"
	}

	if isUnique {
		definition += " UNIQUE"
	}

	return definition
}

func (b *Builder) isFieldModification(name string, typeKind reflect.Kind) bool {
	return (name == "CreatedAt" || name == "CreatedBy" || name == "ModifiedAt" || name == "ModifiedBy") && typeKind == reflect.Int64
}

func (b *Builder) queryOrder(order []string) (string, error) {
	if len(order) == 0 {
		return "", nil
	}

	qOrder := ""
	for i := 0; i < len(order); i = i + 2 {
		field := order[i]
		direction := order[i+1]

		fieldColumn, ok := b.fieldColumnName[field]
		if !ok {
			return "", &ErrBuilder{
				Op:  "queryOrder",
				Err: errors.New("invalid order field/column"),
			}
		}

		queryDirection := "ASC"
		if direction == strings.ToLower("desc") {
			queryDirection = "DESC"
		}

		qOrder += fmt.Sprintf(`,"%s" %s`, fieldColumn, queryDirection)
	}

	if qOrder == "" {
		return qOrder, nil
	}

	return qOrder[1:], nil
}

func (b *Builder) queryLimitOffset(limit int, offset int) string {
	if limit == 0 {
		return ""
	}

	if offset > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	}

	return fmt.Sprintf("LIMIT %d", limit)
}

func (b *Builder) querySet(values map[string]interface{}) (string, int, error) {
	if len(values) == 0 {
		return "", 0, nil
	}

	columns := make([]string, 0, len(values))
	for value := range values {
		fieldColumn, ok := b.fieldColumnName[value]
		if !ok {
			return "", 0, &ErrBuilder{
				Op:  "querySet",
				Err: errors.New("invalid field/column to set"),
			}
		}

		columns = append(columns, fieldColumn)
	}

	numColumn := len(columns)
	if numColumn == 0 {
		return "", 0, nil
	}

	sort.Strings(columns)

	querySet := ""
	for i := 1; i <= numColumn; i++ {
		querySet += fmt.Sprintf(`,"%s"=$%d`, columns[i-1], i)
	}

	return querySet[1:], numColumn, nil
}

func (b *Builder) queryFilters(filters *Filters, firstValueNum int) (string, error) {
	if filters == nil || len(*filters) == 0 {
		return "", nil
	}

	sortedNames := make([]string, 0, len(*filters))
	for name := range *filters {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	queryWhere := ""
	valueNum := firstValueNum

	for _, name := range sortedNames {
		// _raw is a special entry that allows almost-raw SQL query
		if name == Raw {
			continue
		}

		fieldColumn, ok := b.fieldColumnName[name]
		if !ok {
			return "", &ErrBuilder{
				Op:  "queryFilters",
				Err: errors.New("invalid field/column to filter"),
			}
		}

		if b.fieldFlags[name]&FieldFlagNotString > 0 && ((*filters)[name].Op == OpLike || (*filters)[name].Op == OpMatch) {
			fieldColumn = fmt.Sprintf(`CAST("%s" AS TEXT)`, fieldColumn)
		} else {
			fieldColumn = fmt.Sprintf(`"%s"`, fieldColumn)
		}

		switch (*filters)[name].Op {
		case OpLike:
			queryWhere += fmt.Sprintf(` AND %s LIKE $%d`, fieldColumn, valueNum)
		case OpMatch:
			queryWhere += fmt.Sprintf(` AND %s ~ $%d`, fieldColumn, valueNum)
		case OpNotEqual:
			queryWhere += fmt.Sprintf(` AND %s!=$%d`, fieldColumn, valueNum)
		case OpGreater:
			queryWhere += fmt.Sprintf(` AND %s>$%d`, fieldColumn, valueNum)
		case OpLower:
			queryWhere += fmt.Sprintf(` AND %s<$%d`, fieldColumn, valueNum)
		case OpGreaterOrEqual:
			queryWhere += fmt.Sprintf(` AND %s>=$%d`, fieldColumn, valueNum)
		case OpLowerOrEqual:
			queryWhere += fmt.Sprintf(` AND %s<=$%d`, fieldColumn, valueNum)
		case OpBit:
			queryWhere += fmt.Sprintf(` AND %s&$%d>0`, fieldColumn, valueNum)
		default:
			queryWhere += fmt.Sprintf(` AND %s=$%d`, fieldColumn, valueNum)
		}

		valueNum++
	}

	if queryWhere != "" {
		queryWhere = queryWhere[5:]
	}

	// _raw
	rawQueryArr, ok := (*filters)[Raw]
	if !ok || len(rawQueryArr.Val.([]interface{})) == 0 {
		return queryWhere, nil
	}

	rawQuery := rawQueryArr.Val.([]interface{})[0].(string)
	if rawQuery == "" {
		return queryWhere, nil
	}

	if queryWhere != "" {
		queryWhere = fmt.Sprintf("(%s)", queryWhere)

		conjunction := rawQueryArr.Op
		if conjunction != OpOR {
			queryWhere += " AND "
		} else {
			queryWhere += " OR "
		}
	}

	queryWhere += "("

	fieldsInRaw := regexpFieldInRaw.FindAllString(rawQuery, -1)
	alreadyReplaced := map[string]bool{}
	for _, fieldInRaw := range fieldsInRaw {
		if alreadyReplaced[fieldInRaw] {
			continue
		}

		fieldName := strings.Replace(fieldInRaw, ".", "", 1)

		fieldColumn, ok := b.fieldColumnName[fieldName]
		if !ok {
			return "", &ErrBuilder{
				Op:  "queryFilters",
				Err: errors.New("invalid field/column to filter in _raw"),
			}
		}

		rawQuery = strings.ReplaceAll(rawQuery, fieldInRaw, fmt.Sprintf(`"%s"`, fieldColumn))
		alreadyReplaced[fieldInRaw] = true
	}

	numRaw := len((*filters)[Raw].Val.([]interface{}))
	for j := 1; j < numRaw; j++ {
		rawType := reflect.TypeOf((*filters)[Raw].Val.([]interface{})[j])
		if rawType.Kind() != reflect.Slice && rawType.Kind() != reflect.Array {
			// Value is a single value so just replace ? with $x, eg $2
			rawQuery = strings.Replace(rawQuery, "?", fmt.Sprintf("$%d", valueNum), 1)
			valueNum++
			continue
		}

		rawNumValue := reflect.ValueOf((*filters)[Raw].Val.([]interface{})[j]).Len()

		queryVal := ""
		for k := 0; k < rawNumValue; k++ {
			if k == 0 {
				queryVal += fmt.Sprintf("$%d", valueNum)
				valueNum++
				continue
			}
			queryVal += fmt.Sprintf(",$%d", valueNum)
			valueNum++
		}
		rawQuery = strings.Replace(rawQuery, "?", queryVal, 1)
	}

	queryWhere += rawQuery + ")"

	return queryWhere, nil
}
