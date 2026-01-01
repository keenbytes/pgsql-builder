package pgsqlbuilder

import "regexp"

var (
	regexpFieldInRaw = regexp.MustCompile(`\.[a-zA-Z0-9_]+`)
)
