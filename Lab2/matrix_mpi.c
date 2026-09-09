#include <stdio.h>
#include <stdlib.h>
#include <mpi.h>

#define n 4

int main(int argc, char *argv[]) {

   // 1. matrix multiplication using MPI
   int rank, size;

   // start MPI, get process ID and number of processes
   MPI_Init(&argc, &argv);
   MPI_Comm_rank(MPI_COMM_WORLD, &rank);
   MPI_Comm_size(MPI_COMM_WORLD, &size);

   // calculate number of rows to be processed by each MPI process
   int rowsperprocess = n / size;

   float A[n][n];
   float B[n][n];
   float C[n][n];

   float Alocal[rowsperprocess][n];
   float Clocal[rowsperprocess][n];

   // initialize only on process 0
   if (rank == 0) {
      float tempA[n][n] = {
         {1, 2, 3, 4},
         {5, 6, 7, 8},
         {9, 10, 11, 12},
         {13, 14, 15, 16}
      };

      float tempB[n][n] = {
         {1, 0, 0, 0},
         {0, 1, 0, 0},
         {0, 0, 1, 0},
         {0, 0, 0, 1}
      };

      // copy tempA and tempB to A and B
      for (int i = 0; i < n; i++) {
         for (int j = 0; j < n; j++) {
            A[i][j] = tempA[i][j];
            B[i][j] = tempB[i][j];
         }
      }
   }

   // broadcast B to all processes
   MPI_Bcast(B, n*n, MPI_FLOAT, 0, MPI_COMM_WORLD);

   // scatter rows of A to all processes
   MPI_Scatter(A, rowsperprocess*n, MPI_FLOAT, Alocal, rowsperprocess*n, MPI_FLOAT, 0, MPI_COMM_WORLD);

   // perform local matrix multiplication
   for (int i = 0; i < rowsperprocess; i++) {
      for (int j = 0; j < n; j++) {
         Clocal[i][j] = 0;
         for (int k = 0; k < n; k++) {
            Clocal[i][j] += Alocal[i][k] * B[k][j];
         }
      }
   }

   // gather the results to process 0
   MPI_Gather(Clocal, rowsperprocess*n, MPI_FLOAT, C, rowsperprocess*n, MPI_FLOAT, 0, MPI_COMM_WORLD);

   // print the result matrix C on process 0
   if (rank == 0) {
      printf("Result matrix C:\n");
      for (int i = 0; i < n; i++) {
         for (int j = 0; j < n; j++) {
            printf("%.2f ", C[i][j]);
         }
         printf("\n");
      }
   }

   MPI_Finalize();

   return 0;

}