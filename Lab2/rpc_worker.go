package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
)

type SearchRequest struct {
	Text        []byte
	Search      string
	ValidStarts int
}
type SearchReply struct{ Occurrences int }
type SearchWorker struct{}

func (SearchWorker) Count(request SearchRequest, reply *SearchReply) error {
	needle := []byte(request.Search)
	if len(needle) == 0 || request.ValidStarts <= 0 {
		return nil
	}
	limit := request.ValidStarts
	if limit > len(request.Text) {
		limit = len(request.Text)
	}
	for i := 0; i < limit && i+len(needle) <= len(request.Text); i++ {
		matched := true
		for j := range needle {
			if request.Text[i+j] != needle[j] {
				matched = false
				break
			}
		}
		if matched {
			reply.Occurrences++
		}
	}
	return nil
}

func main() {
	listen := flag.String("listen", ":9001", "TCP address to listen on")
	flag.Parse()
	if err := rpc.RegisterName("SearchWorker", new(SearchWorker)); err != nil {
		log.Fatal(err)
	}
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("RPC search worker listening on", listener.Addr())
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go rpc.ServeConn(connection)
	}
}
