package main

import (
	"Lab5/shared"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/rpc"
)

func main() {
	// create a Membership list
	nodes := shared.NewMembershipStore()
	requests := shared.NewRequests()

	// register nodes with `rpc.DefaultServer`
	rpc.RegisterName("Membership", nodes)
	rpc.Register(requests)

	// register an HTTP handler for RPC communication
	rpc.HandleHTTP()

	// sample test endpoint
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		io.WriteString(res, "RPC SERVER LIVE!")
	})

	// Listen before printing so the message means the server is ready.
	listener, err := net.Listen("tcp", "localhost:9005")
	if err != nil {
		fmt.Println("Server failed to start:", err)
		return
	}
	fmt.Println("Started server")
	if err := http.Serve(listener, nil); err != nil {
		fmt.Println("Server stopped:", err)
	}
}
