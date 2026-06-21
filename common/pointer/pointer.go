package pointer

import "reflect"

func Val[T any](val T) *T {
	return &val
}

func Extract[T any](val *T) T {
	var result T
	if val == nil {
		return result
	}
	return *val
}

func EmptyNil[T any](val T) *T {
	if reflect.ValueOf(&val).Elem().IsZero() {
		return nil
	}
	return &val
}

// SliceVal returns a pointer to the slice, or nil if the slice itself is nil.
// Use this when nil and empty slice carry distinct meaning (e.g. "not provided" vs "provided but empty").
func SliceVal[T any](val []T) *[]T {
	if val == nil {
		return nil
	}
	return &val
}
