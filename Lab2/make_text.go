package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	const targetSize = 1024 * 1024 // 1 MiB = 1,048,576 bytes

	outputFile, err := os.Create("data.txt")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	line := "Parallel programming is useful for large problems. " +
		"This text contains the word needle for string-search testing. " +
		"Each worker can search an independent chunk of the text. " +
		"needle appears repeatedly so the program has matches to count.\n"

	written := 0

	for written < targetSize {
		remaining := targetSize - written
		textToWrite := line

		// Do not exceed exactly 1 MiB.
		if len(textToWrite) > remaining {
			textToWrite = textToWrite[:remaining]
		}

		n, err := writer.WriteString(textToWrite)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}

		written += n
	}

	fmt.Printf("Created data.txt with %d bytes\n", written)
}
