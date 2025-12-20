package pgsqlbuilder

type OpVal struct {
	Op  int
	Val interface{}
}

const (
	OpOR = iota * 1
	OpAND
)

const (
	OpEqual = iota * 1
	OpNotEqual
	OpLike
	OpMatch
	OpGreater
	OpLower
	OpGreaterOrEqual
	OpLowerOrEqual
	OpBit
)
