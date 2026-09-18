#include <mpi.h>
#include <stdio.h>
#include <stdlib.h>
#include <limits.h>

static int compare_ints(const void *a, const void *b)
{
    int left = *(const int *)a;
    int right = *(const int *)b;

    return (left > right) - (left < right);
}

int main(int argc, char **argv) {
    int rank, processes;
    MPI_Init(&argc, &argv);
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &processes);

    if (argc != 2) {
        if (rank == 0) {
            fprintf(stderr, "Usage: %s N\n", argv[0]);
        }
        MPI_Finalize();
        return EXIT_FAILURE;
    }

    char *end;
    long input_n = strtol(argv[1], &end, 10);

    if (argv[1][0] == '\0' || *end != '\0' ||
        input_n < 0 || input_n > INT_MAX) {
        if (rank == 0) {
            fprintf(stderr, "N must be a nonnegative integer\n");
        }
        MPI_Finalize();
        return EXIT_FAILURE;
    }

    int n = (int)input_n;
    int base = n / processes;
    int remainder = n % processes;
    int local_count = base + (rank < remainder ? 1 : 0);
    int *counts = NULL;
    int *displacements = NULL;
    int *global_data = NULL;
    int *local_data = NULL;
    int local_capacity = local_count > 0 ? local_count : 1;
    local_data = malloc((size_t)local_capacity * sizeof(int));

    if (rank == 0) {
        counts = malloc(processes * sizeof(int));
        displacements = malloc(processes * sizeof(int));

        int offset = 0;
        int global_capacity = n > 0 ? n : 1;
        global_data = malloc((size_t)global_capacity * sizeof(int));

        srand(4669);  // Fixed seed

        for (int i = 0; i < n; i++) {
            global_data[i] = rand() % 1000;
        }

        for (int i = 0; i < processes; i++) {
            counts[i] = base + (i < remainder ? 1 : 0);
            displacements[i] = offset;
            offset += counts[i];
        }

    }

    MPI_Barrier(MPI_COMM_WORLD);
        double start = MPI_Wtime();

    MPI_Scatterv(
        global_data,
        counts,
        displacements,
        MPI_INT,
        local_data,
        local_count,
        MPI_INT,
        0,
        MPI_COMM_WORLD
    );

    qsort(
        local_data,
        (size_t)local_count,
        sizeof(int),
        compare_ints
    );

    printf("Rank %d sorted chunk:", rank);

    for (int i = 0; i < local_count; i++) {
        printf(" %d", local_data[i]);
    }

    printf("\n");

    free(global_data);
    free(local_data);
    free(counts);
    free(displacements);

    MPI_Finalize(); 
    return 0;
}