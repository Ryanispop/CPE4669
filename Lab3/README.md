# Lab 3: Distributed Sorting with MPI





## Build

From the `Lab3` directory, run:

```bash
make
```

This builds the `mpi_sort` executable with optimization and compiler warnings
enabled. Remove the executable with:

```bash
make clean
```

## Usage

Run the program with the number of MPI ranks and the total number of integers:

```bash
mpirun -np <ranks> ./mpi_sort <N>
```

For example:

```bash
mpirun -np 5 ./mpi_sort 100003
```

Rank 0 prints one machine-readable result line:

```text
SUMMARY N=100003 ranks=5 seconds=0.012345678 status=PASS
```

## Correctness tests

Run all required correctness cases with:

```bash
make test
```

This executes:

```bash
mpirun -np 1 ./mpi_sort 0
mpirun -np 3 ./mpi_sort 17
mpirun -np 5 ./mpi_sort 100003
```

Each case should report `status=PASS`.

## Algorithm

1. Rank 0 generates `N` random integers.
2. Chunk sizes differ by at most one, including when `N` is not divisible by
   the number of ranks.
3. `MPI_Scatterv` distributes the chunks.
4. Every rank sorts its local chunk with `qsort`.
5. A binary merge tree combines the sorted chunks. At each round, selected
   sender ranks transmit their count and sorted data to receiver ranks. The
   receivers perform a linear merge. The partner distance doubles each round
   until rank 0 owns all `N` values.
6. Rank 0 checks the final count, nondecreasing order, and an additive checksum.


