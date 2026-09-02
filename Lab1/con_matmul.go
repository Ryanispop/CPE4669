package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func main() {
	n := 10000

	// Create matrices A, B, and result matrix C.
	a := make([][]float64, n)
	b := make([][]float64, n)
	c := make([][]float64, n)

	for i := 0; i < n; i++ {
		a[i] = make([]float64, n)
		b[i] = make([]float64, n)
		c[i] = make([]float64, n)

		for j := 0; j < n; j++ {
			a[i][j] = rand.Float64()
			b[i][j] = rand.Float64()
		}
	}

	start := time.Now()

	var wg sync.WaitGroup

	// One goroutine calculates one complete row of C.
	for i := 0; i < n; i++ {
		wg.Add(1)

		go func(row int) {
			defer wg.Done()

			for j := 0; j < n; j++ {
				for k := 0; k < n; k++ {
					c[row][j] += a[row][k] * b[k][j]
				}
			}
		}(i) // Pass i as row so each goroutine gets the correct row number.
	}

	// Wait until every row has finished.
	wg.Wait()

	fmt.Println("Concurrent matrix multiplication finished.")
	fmt.Println("Execution time:", time.Since(start))
	fmt.Println("First value in result:", c[0][0])
}
