package dsa

type Less[T any] func(T, T) bool

func zero[T any]() T {
	var z T
	return z
}
