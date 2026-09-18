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
            global_data[i] = rand() % 1000;
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

    MPI_Gatherv(
        local_data,
        local_count,
        MPI_INT,
        global_data,
        counts,
        displacements,
        MPI_INT,
        0,
        MPI_COMM_WORLD
    );

    int *sortedData = NULL;
    int mergeCount = 0;

    if (rank == 0) {
        int capacity = n > 0 ? n : 1;
        sortedData = malloc((size_t)capacity * sizeof(int));

        for (int p = 0; p < processes; p++) {

            int chunkCount = counts[p];
            if (chunkCount == 0) continue;

            if (mergeCount == 0) {
                memcpy(
                    sortedData,
                    global_data + displacements[p],
                    (size_t)chunkCount * sizeof(int)
                );
                mergeCount = chunkCount;
            } else {
                int newCount = mergeCount + chunkCount;
                int *temp = malloc((size_t)newCount * sizeof(int));
                mergeArrays(
                    sortedData,
                    mergeCount,
                    global_data + displacements[p],
                    chunkCount,
                    temp
                );

                memcpy(
                    sortedData,
                    temp,
                    (size_t)newCount * sizeof(int)
                );

                free(temp);
                mergeCount = newCount;
            }
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
        int sortedCorrect = 1;

        for (int i = 1; i < n; i++) {
            if (sortedData[i - 1] > sortedData[i]) {
                sortedCorrect = 0;
                break;
            }
        }

        long long outputChecksum = 0;

        for (int i = 0; i < n; i++) {
            outputChecksum += sortedData[i];
        }

        int countCorrect = (mergeCount == n);
        int checksumCorrect = (inputChecksum == outputChecksum);
        int passed = sortedCorrect && countCorrect && checksumCorrect;
        printf("N=%d ranks=%d time=%f status=%s\n", n, processes, maxElapsed, passed ? "Pass!" : "Fail!");
    }

    printf("Rank %d sorted chunk:", rank);

    for (int i = 0; i < local_count; i++) {
        printf(" %d", local_data[i]);
    }

    printf("\n");

    free(global_data);
    free(local_data);
    free(counts);
    free(displacements);
    free(sortedData);

    MPI_Finalize(); 
    return 0;
}