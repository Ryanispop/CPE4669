package main

import (
	"fmt"
	"hash/fnv"
	"os"
	"sort"
	"time"
)

type Config struct {
	ServerCount       int
	VNodesPerServer   int
	ReplicationFactor int
}

// The coordinator assigns increasing versions to writes.
type Record struct {
	Value   string
	Version uint64
}

type Node struct {
	ID      string
	Address string // Empty for local tests; HTTP address for a server process.
	Data    map[string]Record
}

type VirtualNode struct {
	ID       string
	ServerID string
	Position uint64 // Hash once when creating the virtual node; reuse this value.
}

type Ring struct {
	VNodes []VirtualNode // Keep sorted by Position.
}

// Membership is separate from ring placement: an outage is not a deletion.
type Member struct {
	ServerID      string
	Heartbeat     uint64
	LastHeartbeat time.Time // Local time when heartbeat progress was observed.
	Available     bool
}

// The coordinator owns the ring and serializes client operations.
type Cluster struct {
	Config      Config
	Ring        Ring
	Nodes       map[string]*Node
	Members     map[string]*Member
	NextVersion uint64
	Verbose     bool
}

func NewCluster(config Config) *Cluster {
	return &Cluster{
		Config:  config,
		Nodes:   make(map[string]*Node),
		Members: make(map[string]*Member),
	}
}

func hashString(input string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(input))
	return h.Sum64()
}

func (c *Cluster) AddNode(node *Node) error {
	if c.Config.VNodesPerServer <= 0 || c.Config.ReplicationFactor <= 0 {
		return fmt.Errorf("virtual node and replica counts must be positive")
	}
	if node == nil || node.ID == "" {
		return fmt.Errorf("node cannot be nil")
	}
	if _, exists := c.Nodes[node.ID]; exists {
		return fmt.Errorf("node %s already exists", node.ID)
	}
	// When joining a populated ring, retain existing data for redistribution.
	var records map[string]Record
	if c.NextVersion > 0 {
		var err error
		records, err = c.collectRecords("")
		if err != nil {
			return err
		}
	}
	oldRing := append([]VirtualNode(nil), c.Ring.VNodes...)
	if node.Data == nil {
		node.Data = make(map[string]Record)
	}
	for i := 0; i < c.Config.VNodesPerServer; i++ {
		vnodeID := fmt.Sprintf("%s-vnode%d", node.ID, i)
		vnode := VirtualNode{
			ID:       vnodeID,
			ServerID: node.ID,
			Position: hashString(vnodeID),
		}
		c.Ring.VNodes = append(c.Ring.VNodes, vnode)
	}
	sort.Slice(c.Ring.VNodes, func(i, j int) bool {
		return c.Ring.VNodes[i].Position < c.Ring.VNodes[j].Position
	})

	c.Nodes[node.ID] = node

	c.Members[node.ID] = &Member{
		ServerID:      node.ID,
		Heartbeat:     0,
		LastHeartbeat: time.Now(),
		Available:     true,
	}

	// Copy affected records before callers start using the expanded ring.
	for key, record := range records {
		owners, err := c.Ring.Owners(key, c.Config.ReplicationFactor)
		if err == nil {
			for _, id := range owners {
				if !c.Members[id].Available {
					err = fmt.Errorf("owner %s unavailable", id)
					break
				}
				if err = writeRecord(c.Nodes[id], key, record); err != nil {
					break
				}
			}
		}
		if err != nil {
			c.Ring.VNodes = oldRing
			delete(c.Nodes, node.ID)
			delete(c.Members, node.ID)
			return err
		}
	}
	return nil
}

func (r *Ring) Owners(key string, replicaCount int) ([]string, error) {
	if len(r.VNodes) == 0 {
		return nil, fmt.Errorf("ring is empty")
	}
	if replicaCount <= 0 {
		return nil, fmt.Errorf("replica count must be positive")
	}

	keyPosition := hashString(key)

	start := sort.Search(len(r.VNodes), func(i int) bool {
		return r.VNodes[i].Position >= keyPosition
	})
	if start == len(r.VNodes) {
		start = 0
	}
	owners := []string{}
	seen := make(map[string]bool)

	for offset := 0; offset < len(r.VNodes); offset++ {
		index := (start + offset) % len(r.VNodes)

		serverID := r.VNodes[index].ServerID

		if !seen[serverID] {
			owners = append(owners, serverID)
			seen[serverID] = true
		}
		if len(owners) == replicaCount {
			return owners, nil
		}

	}
	return nil, fmt.Errorf(
		"need %d distinct servers, found %d",
		replicaCount, len(owners),
	)
}

func (c *Cluster) Put(key, value string) error {
	if len(key) > 20 {
		return fmt.Errorf("key exceeds 20 characters")
	}
	for i := 0; i < len(key); i++ {
		if key[i] > 127 {
			return fmt.Errorf("key must contain only ASCII characters")
		}
	}
	owners, err := c.Ring.Owners(key, c.Config.ReplicationFactor)
	if err != nil {
		return err
	}

	c.NextVersion++

	record := Record{
		Value:   value,
		Version: c.NextVersion,
	}

	written := 0
	for _, serverID := range owners {
		member := c.Members[serverID]
		if !member.Available {
			continue
		}
		node := c.Nodes[serverID]
		if err := writeRecord(node, key, record); err != nil {
			continue
		}
		written++
	}
	if written == 0 {
		return fmt.Errorf("no available owners for key %q", key)
	}
	return nil
}

// Read all reachable owners and choose the newest copy. A failed owner is
// different from a reachable server that reports that the key does not exist.
func (c *Cluster) Get(key string) (string, error) {
	owners, err := c.Ring.Owners(key, c.Config.ReplicationFactor)
	if err != nil {
		return "", err
	}
	var newest Record
	found, incomplete := false, false
	answered := ""
	for _, id := range owners {
		if !c.Members[id].Available {
			incomplete = true
			continue
		}
		record, exists, err := readRecord(c.Nodes[id], key)
		if err != nil {
			incomplete = true
			continue
		}
		if exists && (!found || record.Version > newest.Version) {
			newest, found, answered = record, true, id
		}
	}
	if found {
		if c.Verbose {
			fmt.Printf("Read %s answered by: %s\n", key, answered)
		}
		return newest.Value, nil
	}
	if incomplete {
		return "", fmt.Errorf("key %q could not be read: owners unavailable", key)
	}
	return "", fmt.Errorf("key %q not found", key)
}

// Gather the newest records from available servers. Do not read a failed
// server's memory: permanent recovery must work even when that data is gone.
func (c *Cluster) collectRecords(exclude string) (map[string]Record, error) {
	records := make(map[string]Record)
	sources := 0
	for id, node := range c.Nodes {
		if id == exclude || !c.Members[id].Available {
			continue
		}
		data, err := readAll(node)
		if err != nil {
			return nil, err
		}
		sources++
		for key, candidate := range data {
			old, exists := records[key]
			if !exists || candidate.Version > old.Version {
				records[key] = candidate
			}
		}
	}
	if sources == 0 {
		return nil, fmt.Errorf("no available recovery sources")
	}
	return records, nil
}

func (c *Cluster) DeleteNode(serverID string) error {
	if _, exists := c.Nodes[serverID]; !exists {
		return fmt.Errorf("unknown server %q", serverID)
	}
	if len(c.Nodes)-1 < c.Config.ReplicationFactor {
		return fmt.Errorf("removal leaves too few servers")
	}
	records, err := c.collectRecords(serverID)
	if err != nil {
		return err
	}
	remaining := Ring{}
	for _, vnode := range c.Ring.VNodes {
		if vnode.ServerID != serverID {
			remaining.VNodes = append(remaining.VNodes, vnode)
		}
	}
	// Transfer first; commit the new ring only if every required copy succeeds.
	for key, record := range records {
		owners, err := remaining.Owners(key, c.Config.ReplicationFactor)
		if err != nil {
			return err
		}
		oldOwners, _ := c.Ring.Owners(key, c.Config.ReplicationFactor)
		for _, id := range owners {
			if !c.Members[id].Available {
				return fmt.Errorf("replacement %s unavailable", id)
			}
			if err := writeRecord(c.Nodes[id], key, record); err != nil {
				return err
			}
			if c.Verbose && !contains(oldOwners, id) {
				fmt.Printf("Replacement: %s -> %s\n", key, id)
			}
		}
	}
	c.Ring = remaining
	delete(c.Nodes, serverID)
	delete(c.Members, serverID)
	return nil
}

func contains(ids []string, target string) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

// Poll actual gossip tables. Only an advancing heartbeat refreshes the timer.
// A returning server remains unavailable to clients until RecoverNode finishes.
func (c *Cluster) GossipOnce() error {
	reached := 0
	for _, node := range c.Nodes {
		if node.Address == "" {
			continue
		}
		var members map[string]Member
		if err := request("GET", node.Address+"/members", nil, &members); err != nil {
			continue
		}
		reached++
		for id, incoming := range members {
			local := c.Members[id]
			if local != nil && incoming.Heartbeat > local.Heartbeat {
				local.Heartbeat = incoming.Heartbeat
				local.LastHeartbeat = time.Now()
			}
		}
	}
	if reached == 0 {
		return fmt.Errorf("no gossip server reachable")
	}
	return nil
}

func (c *Cluster) DetectFailures(now time.Time, timeout time.Duration) {
	for id, member := range c.Members {
		if member.Available && now.Sub(member.LastHeartbeat) > timeout {
			member.Available = false
			if c.Verbose {
				fmt.Println("Failure detected:", id)
			}
		}
	}
}

func (c *Cluster) RecoverNode(serverID string) error {
	node, exists := c.Nodes[serverID]
	if !exists {
		return fmt.Errorf("unknown server %q", serverID)
	}
	c.Members[serverID].Available = false
	records, err := c.collectRecords(serverID)
	if err != nil {
		return err
	}
	for key, candidate := range records {
		owners, err := c.Ring.Owners(key, c.Config.ReplicationFactor)
		if err != nil {
			return err
		}
		if !contains(owners, serverID) {
			continue
		}
		if err := writeRecord(node, key, candidate); err != nil {
			return err
		}
	}
	c.Members[serverID].LastHeartbeat = time.Now()
	c.Members[serverID].Available = true
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
