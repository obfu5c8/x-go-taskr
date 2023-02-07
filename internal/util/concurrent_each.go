package util

import "sync"

func ConcurrentForEach[T any](items []T, handler func(T)) {
	var wg sync.WaitGroup
	wg.Add(len(items))
	for _, item := range items {
		go func(item T) {
			handler(item)
			wg.Done()
		}(item)
	}
	wg.Wait()

}
