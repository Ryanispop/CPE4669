#include <mpi.h>
#include <stdio.h>
#include <stdlib.h>
#include <limits.h>
#include <string.h>

static int compare_ints(const void *a, const void *b)
{
    int left = *(const int *)a;
    int right = *(const int *)b;

    return (left > right) - (left < right);
}

static void mergeArrays(const int *a, int aCount, const int *b, int bCount, int *result) {
    int i = 0;
    int j = 0;
    int k = 0;

    while (i < aCount && j < bCount) {
        if (a[i] <= b[j]) {
            result[k++] = a[i++];
        } else {
            result[k++] = b[j++];
        }
    }

    while (i < aCount) {
        result[k++] = a[i++];
    }

    while (j < bCount) {
        result[k++] = b[j++];
    }
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

    long long inputChecksum = 0;

    if (rank == 0) {
        counts = malloc(processes * sizeof(int));
        displacements = malloc(processes * sizeof(int));

        int offset = 0;
        int global_capacity = n > 0 ? n : 1;
        global_data = malloc((size_t)global_capacity * sizeof(int));

        srand(4669);  // Fixed seed

        for (int i = 0; i < n; i++) {
            global_data[i] = rand();
            inputChecksum += global_data[i];
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

    for (int step = 1; step < processes; step *= 2) {
        int group_size = 2 * step;

        if (rank % group_size == 0) {
            /* This rank is a receiver. */
            int partner = rank + step;

            if (partner < processes) {
                int incoming_count;

                MPI_Recv(
                    &incoming_count,
                    1,
                    MPI_INT,
                    partner,
                    0,
                    MPI_COMM_WORLD,
                    MPI_STATUS_IGNORE
                );

                int incoming_capacity = incoming_count > 0 ? incoming_count : 1;
                int *incoming_data = malloc((size_t)incoming_capacity * sizeof(int));

                int merged_count = local_count + incoming_count;
                int merged_capacity = merged_count > 0 ? merged_count : 1;
                int *merged_data = malloc((size_t)merged_capacity * sizeof(int));

                MPI_Recv(
                    incoming_data,
                    incoming_count,
                    MPI_INT,
                    partner,
                    1,
                    MPI_COMM_WORLD,
                    MPI_STATUS_IGNORE
                );

                mergeArrays(
                    local_data,
                    local_count,
                    incoming_data,
                    incoming_count,
                    merged_data
                );

                free(local_data);
                free(incoming_data);

                local_data = merged_data;
                local_count = merged_count;
            }
        } else if (rank % group_size == step) {
            /* This rank is a sender. */
            int partner = rank - step;

            MPI_Send(
                &local_count,
                1,
                MPI_INT,
                partner,
                0,
                MPI_COMM_WORLD
            );

            MPI_Send(
                local_data,
                local_count,
                MPI_INT,
                partner,
                1,
                MPI_COMM_WORLD
            );

            break;
        }
    }


    double endTime = MPI_Wtime();
    double localElapsed = endTime - start;
    double maxElapsed = 0.0;

    MPI_Reduce(
        &localElapsed,
        &maxElapsed,
        1,
        MPI_DOUBLE,
        MPI_MAX,
        0,
        MPI_COMM_WORLD
    );

    if (rank == 0) {
        int countCorrect = (local_count == n);
        int sortedCorrect = 1;

        for (int i = 1; i < n; i++) {
            if (local_data[i - 1] > local_data[i]) {
                sortedCorrect = 0;
                break;
            }
        }

        long long outputChecksum = 0;

        for (int i = 0; i < n; i++) {
            outputChecksum += local_data[i];
        }

        int checksumCorrect = (inputChecksum == outputChecksum);
        int passed = sortedCorrect && countCorrect && checksumCorrect;
        printf(
            "SUMMARY N=%d ranks=%d seconds=%.9f status=%s\n",
            n,
            processes,
            maxElapsed,
            passed ? "PASS" : "FAIL"
        );
    }


    free(global_data);
    free(local_data);
    free(counts);
    free(displacements);

    MPI_Finalize(); 
    return 0;
}