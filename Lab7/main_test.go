package main

import (
	"fmt"
	"testing"
	"time"
)

func testCluster(t *testing.T) *Cluster {
	t.Helper()
	c := NewCluster(Config{5, 4, 3})
	for i := 1; i <= 5; i++ {
		if err := c.AddNode(&Node{ID: fmt.Sprintf("Server%d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

func TestPlacementAndValidation(t *testing.T) {
	c := testCluster(t)
	if len(c.Ring.VNodes) != 20 {
		t.Fatal("expected 20 positions")
	}
	if c.AddNode(&Node{ID: "Server2"}) == nil || len(c.Ring.VNodes) != 20 {
		t.Fatal("duplicate changed ring")
	}
	for _, key := range []string{"Anna", "Maria", "Lauren", "John", "Thomas"} {
		owners, err := c.Ring.Owners(key, 3)
		if err != nil || len(owners) != 3 {
			t.Fatalf("owners: %v %v", owners, err)
		}
		if owners[0] == owners[1] || owners[0] == owners[2] || owners[1] == owners[2] {
			t.Fatal("duplicate physical server")
		}
	}
	// Key hashes beyond the final position must wrap to the first position.
	r := Ring{VNodes: []VirtualNode{{ServerID: "A", Position: 0}, {ServerID: "B", Position: 1}}}
	owners, err := r.Owners("wrap", 1)
	if err != nil || owners[0] != "A" {
		t.Fatal("wraparound failed")
	}
	if c.Put("thiskeyiswaytoolongforthislab", "x") == nil || c.Put("caf\u00e9", "x") == nil {
		t.Fatal("invalid key accepted")
	}
	if _, err := c.Ring.Owners("x", 6); err == nil {
		t.Fatal("impossible replication accepted")
	}
}

func TestOutageRecoveryAndRemoval(t *testing.T) {
	c := testCluster(t)
	c.Put("Thomas", "50")
	c.Put("Maria", "100")
	c.Members["Server2"].Available = false
	if err := c.Put("Thomas", "75"); err != nil {
		t.Fatal(err)
	}
	if c.Nodes["Server2"].Data["Thomas"].Value != "50" {
		t.Fatal("wrote to failed server")
	}
	if value, err := c.Get("Thomas"); err != nil || value != "75" {
		t.Fatal("replica read failed")
	}
	// Restart with empty memory, as in the process demo.
	c.Nodes["Server2"].Data = make(map[string]Record)
	if err := c.RecoverNode("Server2"); err != nil {
		t.Fatal(err)
	}
	if err := verify(c, map[string]string{"Thomas": "75", "Maria": "100"}); err != nil {
		t.Fatal(err)
	}
	c.Members["Server2"].Available = false
	if err := c.DeleteNode("Server2"); err != nil {
		t.Fatal(err)
	}
	if err := verify(c, map[string]string{"Thomas": "75", "Maria": "100"}); err != nil {
		t.Fatal(err)
	}
	for _, v := range c.Ring.VNodes {
		if v.ServerID == "Server2" {
			t.Fatal("removed vnode remains")
		}
	}
	if err := c.AddNode(&Node{ID: "Server6"}); err != nil {
		t.Fatal(err)
	}
	if err := verify(c, map[string]string{"Thomas": "75", "Maria": "100"}); err != nil {
		t.Fatal(err)
	}
}

func TestUnavailableAndStaleRecords(t *testing.T) {
	c := testCluster(t)
	c.Put("Thomas", "50")
	c.Put("Thomas", "75")
	c.Nodes["Server2"].Data["Thomas"] = Record{Value: "50", Version: 1}
	if value, err := c.Get("Thomas"); err != nil || value != "75" {
		t.Fatal("returned stale primary")
	}
	if _, err := c.Get("missing"); err == nil {
		t.Fatal("missing key succeeded")
	}
	for _, m := range c.Members {
		m.Available = false
	}
	if c.Put("Thomas", "100") == nil {
		t.Fatal("all-down write succeeded")
	}
	if _, err := c.Get("Thomas"); err == nil {
		t.Fatal("all-down read succeeded")
	}
	if c.RecoverNode("Server2") == nil || c.Members["Server2"].Available {
		t.Fatal("recovered without sources")
	}
	if c.RecoverNode("unknown") == nil || c.DeleteNode("unknown") == nil {
		t.Fatal("unknown node accepted")
	}
}

func TestHeartbeatProgress(t *testing.T) {
	now := time.Now()
	s := &server{id: "Server1", peers: map[string]string{"Server2": "unused"}, members: map[string]Member{"Server2": {ServerID: "Server2", Heartbeat: 10, LastHeartbeat: now, Available: true}}}
	s.merge(map[string]Member{"Server2": {Heartbeat: 10}})
	if !s.members["Server2"].LastHeartbeat.Equal(now) {
		t.Fatal("duplicate heartbeat refreshed timer")
	}
	s.merge(map[string]Member{"Server2": {Heartbeat: 11}})
	if s.members["Server2"].Heartbeat != 11 {
		t.Fatal("new heartbeat not merged")
	}
	c := testCluster(t)
	for _, m := range c.Members {
		m.LastHeartbeat = now.Add(-3 * time.Second)
	}
	c.DetectFailures(now, 2*time.Second)
	for _, m := range c.Members {
		if m.Available {
			t.Fatal("timeout not detected")
		}
	}
}
