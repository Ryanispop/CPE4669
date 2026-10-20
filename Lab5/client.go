package main

import (
	"Lab5/shared"
	"fmt"
	"math/rand"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	MAX_NODES       = 8
	X_TIME          = 1
	Y_TIME          = 2
	Z_TIME_MAX      = 100
	Z_TIME_MIN      = 50
	FAILURE_TIMEOUT = 6
)

var self_node shared.Node
var clientMu sync.Mutex

// Send the current membership table to a neighboring node with the provided ID
func sendMessage(server rpc.Client, id int, membership shared.Membership) {
	request := shared.Request{ID: id, Table: membership}
	var accepted bool
	if err := server.Call("Requests.Add", request, &accepted); err != nil {
		fmt.Println("send failed:", err)
	}
}

// Read incoming messages from other nodes
func readMessages(server rpc.Client, id int, membership shared.Membership) *shared.Membership {
	var incoming shared.Membership
	if err := server.Call("Requests.Listen", id, &incoming); err != nil {
		// Listen currently returns an error when the queue is empty.
		return &membership
	}

	// Merge incoming.Members into membership.Members here.
	return shared.CombineTables(&membership, &incoming)
}

func calcTime() float64 {
	return float64(time.Now().UnixNano()) / float64(time.Second)
}

var wg = &sync.WaitGroup{}

func main() {
	rand.Seed(time.Now().UnixNano())
	Z_TIME := rand.Intn(Z_TIME_MAX-Z_TIME_MIN) + Z_TIME_MIN

	// Connect to RPC server
	server, err := rpc.DialHTTP("tcp", "localhost:9005")
	if err != nil {
		fmt.Println("Could not connect to server:", err)
		return
	}
	defer server.Close()

	args := os.Args[1:]

	// Get ID from command line argument
	if len(args) == 0 {
		fmt.Println("No args given")
		return
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("Found Error", err)
	}

	fmt.Println("Node", id, "will fail after", Z_TIME, "seconds")

	currTime := calcTime()
	// Construct self
	self_node = shared.Node{ID: id, Hbcounter: 0, Time: currTime, Alive: true}
	var self_node_response shared.Node // Allocate space for a response to overwrite this

	// Add node with input ID
	if err := server.Call("Membership.Add", self_node, &self_node_response); err != nil {
		fmt.Println("Error:2 Membership.Add()", err)
	} else {
		fmt.Printf("Success: Node created with id= %d\n", id)
	}

	neighbors := self_node.InitializeNeighbors(id)
	fmt.Println("Neighbors:", neighbors)

	membership := shared.NewMembership()
	membership.Members[id] = self_node

	sendMessage(*server, neighbors[0], *membership)

	// crashTime := self_node.CrashTime()

	time.AfterFunc(time.Second*X_TIME, func() { runAfterX(server, &self_node, &membership, id) })
	time.AfterFunc(time.Second*Y_TIME, func() { runAfterY(server, neighbors, &membership, id) })
	time.AfterFunc(time.Second*time.Duration(Z_TIME), func() { runAfterZ(server, id) })

	wg.Add(1)
	wg.Wait()
}

func runAfterX(server *rpc.Client, node *shared.Node, membership **shared.Membership, id int) {
	clientMu.Lock()
	if !node.Alive {
		clientMu.Unlock()
		return
	}

	node.Hbcounter++
	node.Time = calcTime()
	(*membership).Members[id] = *node
	clientMu.Unlock()

	time.AfterFunc(time.Second*X_TIME, func() {
		runAfterX(server, node, membership, id)
	})
}

func runAfterY(server *rpc.Client, neighbors [2]int, membership **shared.Membership, id int) {
	clientMu.Lock()
	if !self_node.Alive {
		clientMu.Unlock()
		return
	}

	snapshot := shared.CombineTables(*membership, shared.NewMembership())
	clientMu.Unlock()

	for _, neighbor := range neighbors {
		sendMessage(*server, neighbor, *snapshot)
	}

	received := readMessages(*server, id, *shared.NewMembership())
	clientMu.Lock()
	*membership = shared.CombineTables(*membership, received)

	now := calcTime()
	for nodeID, node := range (*membership).Members {
		if nodeID != id && now-node.Time > FAILURE_TIMEOUT {
			node.Alive = false
			(*membership).Members[nodeID] = node
		}
	}

	printMembership(**membership)

	clientMu.Unlock()
	time.AfterFunc(time.Second*Y_TIME, func() {
		runAfterY(server, neighbors, membership, id)
	})

}

func runAfterZ(server *rpc.Client, id int) {
	clientMu.Lock()
	self_node.Alive = false
	clientMu.Unlock()

	fmt.Println("Node", id, "crashed")
}

func printMembership(m shared.Membership) {
	fmt.Println("<<<<<Membership Table>>>>>")
	for _, val := range m.Members {
		status := "is Alive"
		if !val.Alive {
			status = "is Dead"
		}
		fmt.Printf("Node %d has hb %d, time %.1f and %s\n", val.ID, val.Hbcounter, val.Time, status)
	}
	fmt.Println("")
}
