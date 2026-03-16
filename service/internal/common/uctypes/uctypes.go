// Package uctypes contains util types for interfaces
package uctypes

// QueryGetListParams is params for find list
type QueryGetListParams struct {
	WithDeleted         bool
	ForShare            bool
	ForUpdate           bool
	ForUpdateSkipLocked bool
	ForUpdateNoWait     bool
	Limit               uint64
	Offset              uint64
}

// QueryGetOneParams is params for find one
type QueryGetOneParams struct {
	WithDeleted         bool
	ForShare            bool
	ForUpdate           bool
	ForUpdateSkipLocked bool
	ForUpdateNoWait     bool
}

// CompareType is compare type
type CompareType int

const (
	// CompareEqual - equal
	CompareEqual CompareType = iota

	// CompareNotEqual - not equal
	CompareNotEqual

	// CompareLess - less
	CompareLess

	// CompareMore - more
	CompareMore

	// CompareLessOrEqual - less or equal
	CompareLessOrEqual

	// CompareMoreOrEqual - more or equal
	CompareMoreOrEqual
)

type CompareOption[T comparable] struct {
	Value T
	Type  CompareType
}

type SortOption[T comparable] struct {
	Field  T
	IsDesc bool
}
