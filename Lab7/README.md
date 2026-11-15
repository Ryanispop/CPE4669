# Lab 7: Dynamo-style key-value store

Requires Go 1.22 or newer and Bash. Uses only Go's standard library.
Run from this directory:

```bash
./run.sh demo
./run.sh bench 1000 3
./run.sh test
```

Or build with `go build -o lab7 .` and run `./lab7 -mode demo`.
Use `go run .` instead of `go run main.go` because there are several Go files.
The demo starts five separate server processes on localhost and cleans them up
when it finishes. Server logs are in `results/logs`. Change virtual nodes with
`-vnodes`; the default is four per server.

## Files

- `main.go`: hash ring, AddNode, DeleteNode, Put, Get, failure detection, recovery.
- `server.go`: HTTP storage handlers and gossip between servers.
- `experiments.go`: starting processes, the failure demo, and timing.
- `main_test.go`: correctness tests.

## How it works

FNV-1a hashes keys and virtual-node names. The sorted ring selects the first
clockwise position and then three distinct physical owners. Virtual-node
positions are stored instead of hashed repeatedly. One coordinator assigns
increasing write versions. Put writes to available owners; Get checks the
owners and returns the newest available record.

Every 200 ms each server exchanges heartbeat tables with a random configured
peer. Only heartbeat progress refreshes the timer. After two seconds without
progress, a server is considered unavailable. The coordinator reads these
membership tables to detect failures. A failed server stays on the ring.

The demo kills Server2, updates Thomas while it is down, restarts Server2 with
empty memory, and copies current records to it before making it available.
It then permanently removes Server2, copies data to replacement owners, and
checks that every key has three correct copies. Replacement owners are printed.
Adding a server to a populated ring also copies data to its new owners.

## Timing and limitations

The benchmark uses the same 1,000 PUTs and 1,000 GETs on 1, 2, and 4 servers.
It reports the median of three runs. Startup and final replica verification
are excluded. Replication is min(3, server count), since one or two servers
cannot hold three distinct copies. The single-server HTTP run is the baseline.
`Speedup` is T1/Tn; `LabTimeRatio` is Tn/T1 as written in the assignment.
Client operations are sequential, so adding servers does not add parallel work.

This is an in-memory system with one coordinator, not a full Dynamo deployment.
There is no consensus, durable storage, or concurrent client coordination.
Put succeeds with at least one copy; losing all current copies loses the data.
Recovery scans records with client writes paused. Gossip uses a static peer list
and wall-clock heartbeats on one host. Removed servers stay failed in worker
membership tables but are removed from the coordinator's ring.

The report and measured results are in `report.pdf` and `results/`. Screenshots
show browser views of actual captured output. Fill in individual contributions
in `report.html`, then open it in a browser and Print to PDF before submitting.
