package utils

func Conditional[T any](expr bool, a, b T) T {
	if expr {
		return a
	}

	return b
}
