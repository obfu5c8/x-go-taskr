package util

func ApplyOpts[T any](opts []func(T) T, obj T) T {
	for _, opt := range opts {
		obj = opt(obj)
	}
	return obj
}
