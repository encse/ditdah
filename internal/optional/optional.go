// Package optional provides value-based optional values without exposing
// pointers or reserving a value of T as a sentinel.
package optional

// Value either contains a T or represents an absent value. Its zero value is
// absent.
type Value[T any] struct {
	value   T
	present bool
}

// Some returns an optional containing value.
func Some[T any](value T) Value[T] {
	return Value[T]{value: value, present: true}
}

// None returns an absent optional.
func None[T any]() Value[T] {
	return Value[T]{}
}

// Get returns the contained value and whether it is present.
func (o Value[T]) Get() (T, bool) {
	return o.value, o.present
}

// IsSome reports whether the optional contains a value.
func (o Value[T]) IsSome() bool {
	return o.present
}

// IsNone reports whether the optional is absent.
func (o Value[T]) IsNone() bool {
	return !o.present
}
