package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// The demo starts real processes on localhost. Each process owns its own data.
type processes struct {
	cluster  *Cluster
	children map[string]*exec.Cmd
	peers    map[string]string
	logs     map[string]*os.File
}

func startProcesses(count, vnodes, replicas int, logDir string) (*processes, error) {
	p := &processes{cluster: NewCluster(Config{count, vnodes, replicas}), children: make(map[string]*exec.Cmd), peers: make(map[string]string), logs: make(map[string]*os.File)}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}
	// Reserve distinct ephemeral ports together, then release before startup.
	listeners := []net.Listener{}
	for i := 1; i <= count; i++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			for _, l := range listeners {
				l.Close()
			}
			return nil, err
		}
		listeners = append(listeners, listener)
		p.peers[fmt.Sprintf("Server%d", i)] = "http://" + listener.Addr().String()
	}
	for _, listener := range listeners {
		listener.Close()
	}
	for i := 1; i <= count; i++ {
		id := fmt.Sprintf("Server%d", i)
		log, err := os.Create(filepath.Join(logDir, id+".log"))
		if err != nil {
			p.close()
			return nil, err
		}
		p.logs[id] = log
		if err := p.start(id); err != nil {
			p.close()
			return nil, err
		}
		if err := p.cluster.AddNode(&Node{ID: id, Address: p.peers[id]}); err != nil {
			p.close()
			return nil, err
		}
	}
	return p, nil
}

func (p *processes) start(id string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	peers, _ := json.Marshal(p.peers)
	cmd := exec.Command(executable, "-mode", "server", "-id", id, "-listen", strings.TrimPrefix(p.peers[id], "http://"), "-peers", string(peers))
	cmd.Stdout, cmd.Stderr = p.logs[id], p.logs[id]
	if err := cmd.Start(); err != nil {
		return err
	}
	p.children[id] = cmd
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var members map[string]Member
		if request("GET", p.peers[id]+"/members", nil, &members) == nil {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("%s did not start; check its log", id)
}

func (p *processes) stop(id string) error {
	cmd := p.children[id]
	if cmd == nil {
		return fmt.Errorf("%s not running", id)
	}
	if err := cmd.Process.Kill(); err != nil {
		return err
	}
	cmd.Wait() // A killed process normally returns an exit error.
	delete(p.children, id)
	return nil
}

func (p *processes) close() {
	for id := range p.children {
		p.stop(id)
	}
	for _, log := range p.logs {
		log.Close()
	}
}

func placement(c *Cluster, keys []string) error {
	fmt.Printf("%-10s %-10s %-10s %-10s\n", "Key", "Primary", "Replica 1", "Replica 2")
	for _, key := range keys {
		owners, err := c.Ring.Owners(key, c.Config.ReplicationFactor)
		if err != nil {
			return err
		}
		fmt.Printf("%-10s %s\n", key, strings.Join(owners, "    "))
	}
	return nil
}

func verify(c *Cluster, expected map[string]string) error {
	for key, value := range expected {
		got, err := c.Get(key)
		if err != nil || got != value {
			return fmt.Errorf("%s: got %q, want %q, error %v", key, got, value, err)
		}
		owners, err := c.Ring.Owners(key, c.Config.ReplicationFactor)
		if err != nil {
			return err
		}
		var version uint64
		for _, id := range owners {
			record, exists, err := readRecord(c.Nodes[id], key)
			if err != nil || !exists || record.Value != value {
				return fmt.Errorf("incorrect replica %s/%s", id, key)
			}
			if version != 0 && record.Version != version {
				return fmt.Errorf("version mismatch for %s", key)
			}
			version = record.Version
		}
	}
	return nil
}

func demo(vnodes int, logDir string) error {
	p, err := startProcesses(5, vnodes, 3, logDir)
	if err != nil {
		return err
	}
	defer p.close()
	c := p.cluster
	c.Verbose = true
	fmt.Println("Started 5 server processes, each with", vnodes, "virtual nodes; replication factor 3")
	// Allow at least a few gossip exchanges before printing views.
	time.Sleep(time.Second)
	if err := c.GossipOnce(); err != nil {
		return err
	}
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("Server%d", i)
		var members map[string]Member
		if err := request("GET", p.peers[id]+"/members", nil, &members); err != nil {
			return err
		}
		observed := 0
		for _, member := range members {
			if member.Heartbeat > 0 {
				observed++
			}
		}
		fmt.Printf("%s membership: %d peers with advancing heartbeats\n", id, observed)
	}
	expected := map[string]string{"Maria": "100", "John": "20", "Anna": "40", "Tim": "100", "Alex": "10", "Lauren": "30", "Thomas": "50"}
	keys := []string{"Anna", "Maria", "Lauren", "John", "Thomas", "Tim", "Alex"}
	for _, key := range keys {
		if err := c.Put(key, expected[key]); err != nil {
			return err
		}
	}
	fmt.Println("\nBEFORE FAILURE")
	if err := placement(c, keys); err != nil {
		return err
	}
	if err := verify(c, expected); err != nil {
		return err
	}
	fmt.Println("Normal operation: all values and three replicas verified")
	fmt.Println("\nStopping Server2 (actual process termination)")
	if err := p.stop("Server2"); err != nil {
		return err
	}
	deadline := time.Now().Add(8 * time.Second)
	for c.Members["Server2"].Available && time.Now().Before(deadline) {
		c.GossipOnce()
		c.DetectFailures(time.Now(), 2*time.Second)
		time.Sleep(100 * time.Millisecond)
	}
	if c.Members["Server2"].Available {
		return fmt.Errorf("failure detection timed out")
	}
	fmt.Println("\nDURING OUTAGE: serving replicas (ring ownership is unchanged)")
	fmt.Printf("%-10s %s\n", "Key", "Available owners")
	for _, key := range keys {
		owners, err := c.Ring.Owners(key, c.Config.ReplicationFactor)
		if err != nil {
			return err
		}
		available := []string{}
		for _, id := range owners {
			if c.Members[id].Available {
				available = append(available, id)
			}
		}
		fmt.Printf("%-10s %s\n", key, strings.Join(available, " "))
	}
	for _, key := range keys {
		if _, err := c.Get(key); err != nil {
			return err
		}
	}
	if err := c.Put("Thomas", "75"); err != nil {
		return err
	}
	expected["Thomas"] = "75"
	value, err := c.Get("Thomas")
	if err != nil || value != "75" {
		return fmt.Errorf("outage update failed: %v", err)
	}
	fmt.Println("Thomas updated to 75 while Server2 was down")
	fmt.Println("\nRestarting Server2 with empty memory; unavailable until synchronized")
	if err := p.start("Server2"); err != nil {
		return err
	}
	before, exists, err := readRecord(c.Nodes["Server2"], "Thomas")
	if err != nil {
		return err
	}
	fmt.Printf("Server2 before synchronization: Thomas=%q exists=%t version=%d\n", before.Value, exists, before.Version)
	if err := c.RecoverNode("Server2"); err != nil {
		return err
	}
	if err := verify(c, expected); err != nil {
		return err
	}
	after, exists, err := readRecord(c.Nodes["Server2"], "Thomas")
	if err != nil {
		return err
	}
	fmt.Printf("Server2 after synchronization: Thomas=%q exists=%t version=%d\n", after.Value, exists, after.Version)
	fmt.Println("Temporary recovery: all values and replica versions verified")
	fmt.Println("\nPermanently removing Server2")
	if err := p.stop("Server2"); err != nil {
		return err
	}
	c.Members["Server2"].Available = false
	if err := c.DeleteNode("Server2"); err != nil {
		return err
	}
	if err := placement(c, keys); err != nil {
		return err
	}
	if err := verify(c, expected); err != nil {
		return err
	}
	fmt.Println("Permanent recovery: all seven keys verified on three surviving owners")
	return nil
}

func workload(c *Cluster, count int) error {
	for i := 0; i < count; i++ {
		if err := c.Put(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i)); err != nil {
			return err
		}
	}
	for i := 0; i < count; i++ {
		value, err := c.Get(fmt.Sprintf("key%d", i))
		if err != nil || value != fmt.Sprintf("value%d", i) {
			return fmt.Errorf("incorrect result for key%d: %v", i, err)
		}
	}
	return nil
}

// Use the same sequential workload on 1, 2, and 4 server processes.
func benchmark(count, repeats, vnodes int, logDir string) error {
	if count < 1 || repeats < 1 {
		return fmt.Errorf("workload and repeats must be positive")
	}
	expected := make(map[string]string)
	for i := 0; i < count; i++ {
		expected[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}
	fmt.Printf("Workload: %d PUTs and %d GETs; median of %d runs; startup excluded\n", count, count, repeats)
	fmt.Println("Configuration,Replicas,Seconds,Speedup,LabTimeRatio,Correct")
	baseline := 0.0
	for _, nodes := range []int{1, 2, 4} {
		// Three distinct copies are impossible with fewer than three servers.
		replicas := min(nodes, 3)
		times := []float64{}
		for repeat := 0; repeat < repeats; repeat++ {
			directory := filepath.Join(logDir, fmt.Sprintf("bench-%d-%d", nodes, repeat))
			p, err := startProcesses(nodes, vnodes, replicas, directory)
			if err != nil {
				return err
			}
			start := time.Now()
			err = workload(p.cluster, count)
			elapsed := time.Since(start).Seconds()
			// Check every stored copy outside the timed interval.
			if err == nil {
				err = verify(p.cluster, expected)
			}
			p.close()
			if err != nil {
				return err
			}
			times = append(times, elapsed)
		}
		sort.Float64s(times)
		elapsed := times[len(times)/2]
		if nodes == 1 {
			baseline = elapsed
		}
		fmt.Printf("%d-node,%d,%.6f,%.3f,%.3f,Yes\n", nodes, replicas, elapsed, baseline/elapsed, elapsed/baseline)
	}
	return nil
}

func run() error {
	mode := flag.String("mode", "demo", "demo, server, or bench")
	id := flag.String("id", "Server1", "server ID")
	listen := flag.String("listen", "127.0.0.1:8001", "server listen address")
	peersJSON := flag.String("peers", "{}", "JSON map of server IDs to HTTP addresses")
	vnodes := flag.Int("vnodes", 4, "virtual nodes per physical server")
	count := flag.Int("workload", 1000, "number of keys for benchmark")
	repeats := flag.Int("repeats", 3, "benchmark repetitions")
	logDir := flag.String("logs", "results/logs", "directory for server logs")
	flag.Parse()
	if *vnodes <= 0 {
		return fmt.Errorf("vnodes must be positive")
	}
	switch *mode {
	case "server":
		peers := make(map[string]string)
		if err := json.Unmarshal([]byte(*peersJSON), &peers); err != nil {
			return err
		}
		peers[*id] = "http://" + *listen
		return serve(*id, *listen, peers)
	case "demo":
		return demo(*vnodes, *logDir)
	case "bench":
		return benchmark(*count, *repeats, *vnodes, *logDir)
	default:
		return fmt.Errorf("unknown mode %q", *mode)
	}
}
