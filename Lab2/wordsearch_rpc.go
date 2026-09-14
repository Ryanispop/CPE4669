package main

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strings"
	"sync"
	"time"
)

type SearchRequest struct {
	Text        []byte
	Search      string
	ValidStarts int
}
type SearchReply struct{ Occurrences int }

func searchRPC(text []byte, needle string, clients []*rpc.Client) (int, error) {
	if len(needle) == 0 {
		return 0, fmt.Errorf("search string cannot be empty")
	}
	results, errs := make([]SearchReply, len(clients)), make([]error, len(clients))
	var wg sync.WaitGroup
	for w := range clients {
		start, end := w*len(text)/len(clients), (w+1)*len(text)/len(clients)
		chunkEnd := end + len(needle) - 1
		if chunkEnd > len(text) {
			chunkEnd = len(text)
		}
		request := SearchRequest{Text: text[start:chunkEnd], Search: needle, ValidStarts: end - start}
		wg.Add(1)
		go func(i int, req SearchRequest) {
			defer wg.Done()
			errs[i] = clients[i].Call("SearchWorker.Count", req, &results[i])
		}(w, request)
	}
	wg.Wait()
	total := 0
	for i := range results {
		if errs[i] != nil {
			return 0, errs[i]
		}
		total += results[i].Occurrences
	}
	return total, nil
}

func main() {
	fileName := flag.String("file", "data.txt", "text file to search")
	search := flag.String("search", "needle", "string to search for")
	addresses := flag.String("addrs", "localhost:9001", "comma-separated RPC worker addresses")
	runs := flag.Int("runs", 10, "number of timed RPC searches")
	flag.Parse()
	text, err := os.ReadFile(*fileName)
	if err != nil {
		log.Fatal(err)
	}
	if *runs < 1 {
		log.Fatal("runs must be at least 1")
	}
	parts := strings.Split(*addresses, ",")
	clients := make([]*rpc.Client, len(parts))
	for i, address := range parts {
		clients[i], err = rpc.Dial("tcp", strings.TrimSpace(address))
		if err != nil {
			log.Fatalf("dial %s: %v", address, err)
		}
		defer clients[i].Close()
	}
	var total time.Duration
	var occurrences int
	for i := 0; i < *runs; i++ {
		start := time.Now()
		occurrences, err = searchRPC(text, *search, clients)
		total += time.Since(start)
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("File:", *fileName)
	fmt.Println("Search string:", *search)
	fmt.Println("RPC workers:", len(clients))
	fmt.Println("Occurrences:", occurrences)
	fmt.Printf("Average RPC search time over %d runs: %.3f ms\n", *runs, total.Seconds()*1000/float64(*runs))
}
