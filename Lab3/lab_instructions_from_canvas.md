> [!NOTE]
> This file is a converted markdown copy of the Canvas assignment instructions for Lab 3 generated from the Canvas page print PDF (`lab 3.pdf`). Please verify against the original Canvas assignment page or PDF if any discrepancies arise.

# Lab 3: Distributed Sorting with MPI

- **Due:** Friday by 11:59pm
- **Points:** 100
- **Submission Type:** Text entry box (containing submission archive / details as required)

---

## Objective

Design, implement, and evaluate a distributed-memory sorting algorithm using MPI on an SDSC supercomputing system. Your program must sort a large array of integers across multiple processes, verify the result, and demonstrate how its performance changes as the number of MPI ranks increases.

---

## Learning Outcomes

By completing this assignment, you will be able to:
- Distribute data and work among MPI processes;
- Use collective and point-to-point MPI communication appropriately;
- Design a distributed sorting strategy that produces one globally sorted result;
- Measure parallel run time correctly; and
- Evaluate strong-scaling speedup and efficiency.

---

## Requirements

Write an MPI program named `mpi_sort` that accepts one command-line argument:

```bash
./mpi_sort N
```

where `N` is the total number of integers to sort.

Your program must:
1. **Input Generation:** Have rank 0 create an input array of `N` integers. Use a fixed seed so that runs are reproducible.
2. **Even Work Distribution:** Divide the input among all ranks as evenly as possible. Your design must work when `N` is not divisible by the number of ranks.
3. **Local Sort:** Sort each rank's local portion.
4. **Global Redistribution & Collection:** Use MPI communication to combine, exchange, or redistribute local results until rank 0 owns one globally sorted array.
5. **Correctness Verification (Rank 0):**
   - All adjacent values are in nondecreasing order; and
   - The output has the same number of values and a checksum (or equivalent invariant) matching the input.
6. **Summary Output:** Print a single machine-readable summary line from rank 0 containing at least `N`, rank count, elapsed seconds, and pass/fail status.
7. **Robustness:** Work for one rank, a non-power-of-two rank count, and values of `N` that do not divide evenly among ranks.

> [!IMPORTANT]
> Do not use a library routine that sorts the entire global input for you. A local serial sort is allowed; the distributed partitioning and global ordering strategy must be your own MPI design.

You may choose an appropriate distributed sorting design, such as:
- Sample sort
- Parallel merge sort
- Odd-even transposition sort
- Another approach approved by the instructor

Describe and justify your algorithm, communication pattern, and data distribution in the report.

---

## Timing Rules

- Time **only** the parallel sorting operation: data distribution, local sorting, and the communication needed to establish global order.
- **Exclude** input setup, correctness checking, and printing.
- Synchronize before timing, then report the **maximum elapsed time over all ranks**, not rank 0's time alone.

---

## SDSC Execution

Run the program through Slurm on your assigned SDSC system. Your submission must include a Slurm script that specifies the resources you request. Set the account, partition, time limit, node count, and MPI tasks per node according to the documentation and allocation for the SDSC system you use.

At minimum, complete these correctness tests before your performance study:

```bash
srun -n 1 ./mpi_sort 0
srun -n 3 ./mpi_sort 17
srun -n 5 ./mpi_sort 100003
```

Each run must report success.

---

## Scalability Experiment: Strong Scaling

Use a fixed, sufficiently large value of `N` for all performance runs. Choose rank counts appropriate for your allocation—for example: $1, 2, 4, 8, 16, 32, 64,$ and $128$. Run each configuration at least three times and use the median execution time $T_p$.

### Formulas to Report

| Metric | Formula |
| :--- | :--- |
| **Speedup** | $S(p) = \frac{T_1}{T_p}$ |
| **Parallel Efficiency** | $E(p) = \frac{S(p)}{p}$ |

### Results Table & Plot

Include a table and plot with one row per rank count:

| Ranks $p$ | Median time $T_p$ (s) | Speedup $S(p)$ | Efficiency $E(p)$ | Correct? |
| :--- | :--- | :--- | :--- | :--- |
| 1 | | 1.00 | 1.00 | |
| 2 | | | | |
| 4 | | | | |
| 8 | | | | |
| ... | | | | |

### Analysis

Discuss where performance stops improving and why. Consider:
- Communication cost
- Increasingly small local sort sizes
- Memory bandwidth
- Process placement
- Load imbalance
- Algorithm-specific global exchange or merge costs

Base conclusions on your measurements rather than ideal speedup alone.

---

## Submission

Submit a single archive containing:
- Source files and build instructions (`Makefile` or equivalent);
- Your Slurm job script;
- A short report (PDF preferred) with:
  - Algorithm description
  - Experimental environment
  - Correctness evidence
  - Results table
  - Plot
  - Scalability discussion; and
- Raw timing output or CSV data.

In the report, identify the:
- SDSC system/partition
- Node configuration
- MPI implementation and version
- Compiler and flags
- Input size
- Number of trials

### Deliverables Summary
- Project report PDF
- Code in ZIP
- Demo/explain during the lab session

---

## Academic Integrity

You may discuss MPI concepts and use course-approved references, but the design and implementation you submit must be your own. Cite any external code, algorithms, or sources as required by course policy.
