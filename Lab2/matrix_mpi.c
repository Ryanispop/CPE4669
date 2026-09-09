#include <stdio.h>
#include <stdlib.h>
#include <time.h>
#include <mpi.h>
#include <math.h>

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

   float Cseq[n][n];

   // initialize only on process 0
   if (rank == 0) {
      
      // get random matrices A and B
      //srand(time(NULL));
      srand(42);
      for (int i = 0; i < n; i++) {
         for (int j = 0; j < n; j++) {
            A[i][j] = (float)(rand() % 10);
            B[i][j] = (float)(rand() % 10);
         }
      }

      // print matrices A and B for verification
      printf("Matrix A:\n");
      for (int i = 0; i < n; i++) {
         for (int j = 0; j < n; j++) {
            printf("%.0f ", A[i][j]);
         }
         printf("\n");
      }

      printf("Matrix B:\n");
      for (int i = 0; i < n; i++) {
         for (int j = 0; j < n; j++) {
            printf("%.0f ", B[i][j]);
         }
         printf("\n");
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

      // perform sequential matrix multiplication for verification
      for (int i = 0; i < n; i++) {
         for (int j = 0; j < n; j++) {
            Cseq[i][j] = 0;
            for (int k = 0; k < n; k++) {
               Cseq[i][j] += A[i][k] * B[k][j];
            }
         }
      }
      
      // check for correctness
      int correct = 1;
      for (int i = 0; i < n; i++) {
         for (int j = 0; j < n; j++) {
            if (fabs(C[i][j] - Cseq[i][j]) > 1e-6) {
               correct = 0;
               break;
            }
         }
         if (!correct) break;
      }
      if (correct) {
         printf("Matrix multiplication is correct.\n");
      } else {
         printf("Matrix multiplication is incorrect.\n");
      }

      printf("Result matrix C:\n");
      for (int i = 0; i < n; i++) {
         for (int j = 0; j < n; j++) {
            printf("%.0f ", C[i][j]);
         }
         printf("\n");
      }
   }

   MPI_Finalize();

   return 0;

}