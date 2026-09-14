# Lab 2 commands

Run commands from `Lab2`. `data.txt` is exactly 1 MiB; regenerate it with `go run make_text.go` if needed.

## Sequential and goroutine search

```bash
go run wordsearch_seq.go -file data.txt -search needle -runs 20
go run wordsearch_par.go -file data.txt -search needle -workers 1 -runs 20
go run wordsearch_par.go -file data.txt -search needle -workers 2 -runs 20
go run wordsearch_par.go -file data.txt -search needle -workers 4 -runs 20
```


## RPC search

Start one worker per terminal:

```bash
go run rpc_worker.go -listen :9001
go run rpc_worker.go -listen :9002
go run rpc_worker.go -listen :9003
go run rpc_worker.go -listen :9004
```

Run the 1-, 2-, and 4-worker tests respectively:

```bash
go run wordsearch_rpc.go -file data.txt -search needle -addrs localhost:9001 -runs 20
go run wordsearch_rpc.go -file data.txt -search needle -addrs localhost:9001,localhost:9002 -runs 20
go run wordsearch_rpc.go -file data.txt -search needle -addrs localhost:9001,localhost:9002,localhost:9003,localhost:9004 -runs 20
```

## MPI matrix multiplication

```bash
mpicc -O3 -Wall -Wextra matrix_mpi.c -o matrix_mpi
mpirun -np 1 ./matrix_mpi
mpirun -np 2 ./matrix_mpi
mpirun -np 4 ./matrix_mpi
```
