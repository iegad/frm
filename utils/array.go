package utils

func IsValueInArray[T comparable](v *T, arr []T) bool {
	for _, item := range arr {
		if item == *v {
			return true
		}
	}

	return false
}

func CloneSlice[T any](src []T) []T {
	dst := make([]T, len(src))
	copy(dst, src)
	return dst
}

func CompareSlice(a, b []byte) bool {
	n := len(a)
	if n != len(b) {
		return false
	}

	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
