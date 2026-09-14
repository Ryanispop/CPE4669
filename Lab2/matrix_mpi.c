#include <math.h>
#include <mpi.h>
#include <stdio.h>
#include <stdlib.h>

#ifndef N
#define N 1000
#endif

static void multiply_rows(const float *a, const float *b, float *c, int rows) {
    for (int i = 0; i < rows; ++i)
        for (int j = 0; j < N; ++j) {
            float sum = 0.0f;
            for (int k = 0; k < N; ++k) sum += a[i * N + k] * b[k * N + j];
            c[i * N + j] = sum;
        }
}

int main(int argc, char **argv) {
    int rank, processes;
    MPI_Init(&argc, &argv);
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &processes);
    if (processes > N) {
        if (rank == 0) fprintf(stderr, "Number of processes cannot exceed N (%d).\n", N);
        MPI_Finalize(); return 1;
    }

    int base_rows = N / processes, remainder = N % processes;
    int local_rows = base_rows + (rank < remainder ? 1 : 0);
    int *counts = malloc((size_t)processes * sizeof(*counts));
    int *displacements = malloc((size_t)processes * sizeof(*displacements));
    for (int p = 0, offset = 0; p < processes; ++p) {
        int rows = base_rows + (p < remainder ? 1 : 0);
        counts[p] = rows * N; displacements[p] = offset; offset += counts[p];
    }

    float *b = malloc((size_t)N * N * sizeof(*b));
    float *local_a = malloc((size_t)local_rows * N * sizeof(*local_a));
    float *local_c = malloc((size_t)local_rows * N * sizeof(*local_c));
    float *a = NULL, *c = NULL, *sequential_c = NULL;
    if (rank == 0) {
        a = malloc((size_t)N * N * sizeof(*a));
        c = malloc((size_t)N * N * sizeof(*c));
        sequential_c = malloc((size_t)N * N * sizeof(*sequential_c));
        if (!a || !b || !c || !sequential_c) MPI_Abort(MPI_COMM_WORLD, 2);
        srand(42); /* deterministic input makes verification reproducible */
        for (int i = 0; i < N * N; ++i) { a[i] = rand() % 10; b[i] = rand() % 10; }
    }
    if (!b || !local_a || !local_c) MPI_Abort(MPI_COMM_WORLD, 2);

    MPI_Barrier(MPI_COMM_WORLD);
    double start = MPI_Wtime();
    MPI_Bcast(b, N * N, MPI_FLOAT, 0, MPI_COMM_WORLD);
    MPI_Scatterv(a, counts, displacements, MPI_FLOAT, local_a, local_rows * N,
                 MPI_FLOAT, 0, MPI_COMM_WORLD);
    multiply_rows(local_a, b, local_c, local_rows);
    MPI_Gatherv(local_c, local_rows * N, MPI_FLOAT, c, counts, displacements,
                MPI_FLOAT, 0, MPI_COMM_WORLD);
    double elapsed = MPI_Wtime() - start;
    double max_elapsed = 0.0;
    /* The job completes when its slowest rank completes. */
    MPI_Reduce(&elapsed, &max_elapsed, 1, MPI_DOUBLE, MPI_MAX, 0, MPI_COMM_WORLD);

    if (rank == 0) {
        /* Reference calculation is outside the measured distributed interval. */
        multiply_rows(a, b, sequential_c, N);
        int correct = 1;
        for (int i = 0; i < N * N; ++i)
            if (fabsf(c[i] - sequential_c[i]) > 1e-4f) { correct = 0; break; }
        printf("N=%d, processes=%d\n", N, processes);
        printf("Distributed matrix multiplication time (slowest rank): %.6f s\n", max_elapsed);
        printf("Verification: %s\n", correct ? "PASS" : "FAIL");
    }

    free(counts); free(displacements); free(b); free(local_a); free(local_c);
    if (rank == 0) { free(a); free(c); free(sequential_c); }
    MPI_Finalize(); return 0;
}
