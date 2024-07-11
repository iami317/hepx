package util

func EmptyOr[T comparable](v T, fallback T) T {
	var zero T
	if zero == v {
		return fallback
	}
	return v
}
