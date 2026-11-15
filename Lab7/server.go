package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var client = &http.Client{Timeout: 600 * time.Millisecond}

func request(method, address string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, address, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", address, resp.Status)
	}
	if output != nil {
		return json.NewDecoder(resp.Body).Decode(output)
	}
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

func writeRecord(node *Node, key string, record Record) error {
	if node.Address != "" {
		return request("POST", node.Address+"/record?key="+url.QueryEscape(key), record, nil)
	}
	current, exists := node.Data[key]
	if !exists || record.Version > current.Version {
		node.Data[key] = record
	}
	return nil
}

func readAll(node *Node) (map[string]Record, error) {
	if node.Address == "" {
		return node.Data, nil
	}
	var records map[string]Record
	err := request("GET", node.Address+"/records", nil, &records)
	return records, err
}

func readRecord(node *Node, key string) (Record, bool, error) {
	if node.Address == "" {
		r, ok := node.Data[key]
		return r, ok, nil
	}
	var result struct {
		Record Record
		Exists bool
	}
	err := request("GET", node.Address+"/record?key="+url.QueryEscape(key), nil, &result)
	return result.Record, result.Exists, err
}

// Each process has its own map and membership view. The mutex protects HTTP
// handlers from the background gossip loop; it is never held during requests.
type server struct {
	mu      sync.Mutex
	id      string
	data    map[string]Record
	peers   map[string]string
	members map[string]Member
}

func (s *server) snapshot() map[string]Member {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make(map[string]Member)
	for id, member := range s.members {
		result[id] = member
	}
	return result
}

func (s *server) merge(incoming map[string]Member) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, member := range incoming {
		if id == s.id {
			continue
		}
		if _, known := s.peers[id]; !known {
			continue
		}
		old := s.members[id]
		if member.Heartbeat > old.Heartbeat {
			old.ServerID, old.Heartbeat = id, member.Heartbeat
			old.LastHeartbeat, old.Available = time.Now(), true
			s.members[id] = old
		}
	}
}

func (s *server) gossip() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		own := s.members[s.id]
		// Wall-clock nanoseconds also let a restarted process have a larger
		// heartbeat on this one-machine demo (not a solution to clock skew).
		own.Heartbeat = uint64(time.Now().UnixNano())
		own.LastHeartbeat, own.Available = time.Now(), true
		s.members[s.id] = own
		peers := []string{}
		for id, address := range s.peers {
			if id != s.id {
				peers = append(peers, address)
			}
		}
		for id, member := range s.members {
			if id != s.id && member.Available && time.Since(member.LastHeartbeat) > 2*time.Second {
				member.Available = false
				s.members[id] = member
				fmt.Println(s.id, "detected failure of", id)
			}
		}
		s.mu.Unlock()
		if len(peers) == 0 {
			continue
		}
		peer := peers[rand.Intn(len(peers))]
		var response map[string]Member
		if request("POST", peer+"/gossip", s.snapshot(), &response) == nil {
			s.merge(response)
		}
	}
}

func serve(id, address string, peers map[string]string) error {
	s := &server{id: id, data: make(map[string]Record), peers: peers, members: make(map[string]Member)}
	for peer := range peers {
		s.members[peer] = Member{ServerID: peer, LastHeartbeat: time.Now(), Available: true}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/record", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if len(key) > 20 {
			http.Error(w, "key too long", 400)
			return
		}
		for i := 0; i < len(key); i++ {
			if key[i] > 127 {
				http.Error(w, "key must be ASCII", 400)
				return
			}
		}
		if r.Method == "POST" {
			var incoming Record
			if json.NewDecoder(r.Body).Decode(&incoming) != nil {
				http.Error(w, "invalid record", 400)
				return
			}
			s.mu.Lock()
			old, exists := s.data[key]
			if !exists || incoming.Version > old.Version {
				s.data[key] = incoming
			}
			s.mu.Unlock()
			w.WriteHeader(http.StatusOK)
		} else if r.Method == "GET" {
			s.mu.Lock()
			record, exists := s.data[key]
			s.mu.Unlock()
			json.NewEncoder(w).Encode(struct {
				Record Record
				Exists bool
			}{record, exists})
		} else {
			http.Error(w, "method not allowed", 405)
		}
	})
	mux.HandleFunc("/records", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		json.NewEncoder(w).Encode(s.data)
	})
	mux.HandleFunc("/members", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(s.snapshot()) })
	mux.HandleFunc("/gossip", func(w http.ResponseWriter, r *http.Request) {
		var incoming map[string]Member
		if json.NewDecoder(r.Body).Decode(&incoming) != nil {
			http.Error(w, "invalid membership", 400)
			return
		}
		s.merge(incoming)
		json.NewEncoder(w).Encode(s.snapshot())
	})
	go s.gossip()
	fmt.Println(id, "listening on", address)
	return (&http.Server{Addr: address, Handler: mux, ReadHeaderTimeout: 2 * time.Second}).ListenAndServe()
}
