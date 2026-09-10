package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"
)

func countOccurencesRange(text []byte, search string, startIndex int, endIndex int) int {
	if len(search) == 0 {
		return 0
	}

	count := 0
	searchBytes := []byte(search)

	for i := startIndex; i < endIndex && i+len(searchBytes) <= len(text); i++ {
		match := true

		for j := 0; j < len(searchBytes); j++ {
			if text[i+j] != searchBytes[j] {
				match = false
				break
			}
		}

		if match {
			count++
		}
	}

	return count

}

func countOccurencesParallel(text []byte, search string, workers int) int {
	if workers < 1 {
		workers = 1
	}

	var wg sync.WaitGroup
	results := make([]int, workers)

	for worker := 0; worker < workers; worker++ {
		startIndex := worker * len(text) / workers
		endIndex := (worker + 1) * len(text) / workers

		wg.Add(1)

		go func(workerID, start, end int) {
			defer wg.Done()

			results[workerID] = countOccurencesRange(
				text,
				search,
				start,
				end,
			)
		}(worker, startIndex, endIndex)
	}
	wg.Wait()

	total := 0
	for _, result := range results {
		total += result
	}

	return total
}

func main() {
	fileName := flag.String("file", "data.txt", "text file to search")
	searchString := flag.String("search", "needle", "string to search for")
	runs := flag.Int("runs", 10, "number of timing runs")
	workers := flag.Int("workers", 4, "number of goroutines")

	flag.Parse()

	text, err := os.ReadFile(*fileName)

	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	if *searchString == "" {
		fmt.Println("Search string can't be empty")
		return
	}

	var totalTime time.Duration
	var result int

	for i := 0; i < *runs; i++ {
		start := time.Now()

		result = countOccurencesParallel(text, *searchString, *workers)

		totalTime += time.Since(start)
	}

	averageTime := totalTime.Seconds() * 1000 / float64(*runs)

	fmt.Println("File:", *fileName)
	fmt.Println("Search string:", *searchString)
	fmt.Println("Occurrences:", result)
	fmt.Printf("Average search time over %d runs: %.3f ms\n", *runs, averageTime)
}
