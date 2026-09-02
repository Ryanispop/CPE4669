package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func makeMatrix(n int) [][]float32 {
	matrix := make([][]float32, n)

	for i := 0; i < n; i++ {
		matrix[i] = make([]float32, n)
	}

	return matrix
}

func multiplySequential(a, b [][]float32) [][]float32 {
	n := len(a)
	c := makeMatrix(n)

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			for k := 0; k < n; k++ {
				c[i][j] += a[i][k] * b[k][j]
			}
		}
	}

	return c
}

func multiplyConcurrent(a, b [][]float32) [][]float32 {
	n := len(a)
	c := makeMatrix(n)

	var wg sync.WaitGroup

	rowsPerGoroutine := 1

	// Start one goroutine for each band of `rowsPerGoroutine` rows in C.
	for start := 0; start < n; start += rowsPerGoroutine {
		end := start + rowsPerGoroutine
		if end > n {
			end = n
		}

		wg.Add(1)

		go func(start, end int) {
			defer wg.Done()

			for row := start; row < end; row++ {
				for j := 0; j < n; j++ {
					for k := 0; k < n; k++ {
						c[row][j] += a[row][k] * b[k][j]
					}
				}
			}
		}(start, end)
	}

	wg.Wait()
	return c
}

func resultsMatch(a, b [][]float32) bool {
	for i := range a {
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}

	return true
}

func main() {
	n := 500
	rng := rand.New(rand.NewSource(42))

	a := makeMatrix(n)
	b := makeMatrix(n)

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			a[i][j] = rng.Float32()
			b[i][j] = rng.Float32()
		}
	}

	// Sequential timing
	start := time.Now()
	seqResult := multiplySequential(a, b)
	seqTime := time.Since(start)

	// Concurrent timing
	start = time.Now()
	conResult := multiplyConcurrent(a, b)
	conTime := time.Since(start)

	fmt.Println("Matrix size:", n, "x", n)
	fmt.Println("Sequential time:", seqTime)
	fmt.Println("Concurrent time:", conTime)
	fmt.Println("Results match:", resultsMatch(seqResult, conResult))
}
