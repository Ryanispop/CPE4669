package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func countOccurences(text []byte, search string) int {
	if len(search) == 0 {
		return 0
	}

	count := 0
	searchBytes := []byte(search)

	for i := 0; i+len(searchBytes) <= len(text); i++ {
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

func main() {
	fileName := flag.String("file", "data.txt", "text file to search")
	searchString := flag.String("search", "needle", "string to search for")
	runs := flag.Int("runs", 10, "number of timing runs")

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

		result = countOccurences(text, *searchString)

		totalTime += time.Since(start)
	}

	averageTime := totalTime.Seconds() * 1000 / float64(*runs)

	fmt.Println("File:", *fileName)
	fmt.Println("Search string:", *searchString)
	fmt.Println("Occurrences:", result)
	fmt.Printf("Average search time over %d runs: %.3f ms\n", *runs, averageTime)
}
