# Lab 5: Gossip Membership

This program runs eight nodes that exchange membership tables through one RPC server. Each node increases its heartbeat every second, sends its table to two randomly chosen neighbors every two seconds, and marks another node dead when its last heartbeat timestamp is more than six seconds old. Each node simulates a crash after a random 50–99 seconds.

## Run

Install Go 1.25 or newer. Open a terminal in the `Lab5` directory and start the server:

```bash
go run server.go
```

Leave it running. Open eight more terminals in the same directory, and run one client per terminal, using a different ID from 1 through 8:

```bash
go run client.go 1
```

Replace `1` with `2`, `3`, and so on in the other terminals. The clients connect to the server at `localhost:9005`, so run them on the same computer as the server.

## What to watch

- The server prints `Started server` and `Added node N` as clients register.
- Each client prints its two neighbors and a membership table every two seconds. Heartbeat counters should increase, and nodes learned through gossip should appear in the table.
- After a client's printed crash time, it stops updating and sending. Other clients that know about it should mark it dead after the six-second timeout.

Press Ctrl+C in each terminal to stop the processes. A simulated crash stops the client's activity, but its process stays open until you stop it.
