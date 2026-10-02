package shared

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	MAX_NODES = 8
)

// Node struct represents a computing node.
type Node struct {
	ID        int
	Hbcounter int
	Time      float64
	Alive     bool
}

// Generate random crash time from 10-60 seconds
func (n Node) CrashTime() int {
	rand.Seed(time.Now().UnixNano())
	max := 60
	min := 10
	return rand.Intn(max-min) + min
}

func (n Node) InitializeNeighbors(id int) [2]int {
	neighbor1 := RandInt()
	for neighbor1 == id {
		neighbor1 = RandInt()
	}
	neighbor2 := RandInt()
	for neighbor1 == neighbor2 || neighbor2 == id {
		neighbor2 = RandInt()
	}
	return [2]int{neighbor1, neighbor2}
}

func RandInt() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(MAX_NODES-1+1) + 1
}

/*---------------*/
//Server Side stored membership
type MembershipStore struct {
	mu      sync.Mutex
	Members map[int]Node
}

func NewMembershipStore() *MembershipStore {
	return &MembershipStore{
		Members: make(map[int]Node),
	}
}

// Membership struct represents participanting nodes
type Membership struct {
	Members map[int]Node
}

// Returns a new instance of a Membership (pointer).
func NewMembership() *Membership {
	return &Membership{
		Members: make(map[int]Node),
	}
}

// Adds a node to the membership list.
func (m *MembershipStore) Add(payload Node, reply *Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Members[payload.ID] = payload
	*reply = payload
	fmt.Printf("Added node %d\n", payload.ID)
	return nil
}

// Updates a node in the membership list.
func (m *MembershipStore) Update(payload Node, reply *Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.Members[payload.ID]; !exists {
		return fmt.Errorf("node %d not found", payload.ID)
	}
	m.Members[payload.ID] = payload
	*reply = payload
	return nil
}

// Returns a node with specific ID.
func (m *MembershipStore) Get(payload int, reply *Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	node, exists := m.Members[payload]
	if !exists {
		return fmt.Errorf("node %d not found", payload)
	}
	*reply = node
	return nil
}

/*---------------*/

// Request struct represents a new message request to a client
type Request struct {
	ID    int
	Table Membership
}

// Requests struct represents pending message requests
type Requests struct {
	mu      sync.Mutex
	Pending map[int]Membership
}

// Returns a new instance of Requests.
func NewRequests() *Requests {
	return &Requests{
		Pending: make(map[int]Membership),
	}
}

// Adds a new message request to the pending list
func (req *Requests) Add(payload Request, reply *bool) error {
	req.mu.Lock()
	defer req.mu.Unlock()

	if pending, exists := req.Pending[payload.ID]; exists {
		req.Pending[payload.ID] = *CombineTables(&pending, &payload.Table)
	} else {
		req.Pending[payload.ID] = payload.Table
	}
	*reply = true
	return nil
}

// Listens to communication from neighboring nodes.
func (req *Requests) Listen(ID int, reply *Membership) error {
	req.mu.Lock()
	defer req.mu.Unlock()

	table, exists := req.Pending[ID]
	if !exists {
		return fmt.Errorf("no pending messages for node %d", ID)
	}
	*reply = table
	delete(req.Pending, ID)
	return nil
}

func CombineTables(table1 *Membership, table2 *Membership) *Membership {
	combined := NewMembership()
	for id, node := range table1.Members {
		combined.Members[id] = node
	}
	for id, node := range table2.Members {
		current, exists := combined.Members[id]
		if !exists || node.Hbcounter > current.Hbcounter ||
			(node.Hbcounter == current.Hbcounter && node.Time > current.Time) {
			combined.Members[id] = node
		}
	}
	return combined
}
