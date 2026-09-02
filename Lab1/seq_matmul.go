package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	n := 1000 // Change this later, 100 or 1000

	// Create A, B, and result C.
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

	// C = A × B
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			for k := 0; k < n; k++ {
				c[i][j] += a[i][k] * b[k][j]
			}
		}
	}

	fmt.Println("Matrix multiplication finished.")
	fmt.Println("Execution time:", time.Since(start))
	fmt.Println("First value in result:", c[0][0])
}
