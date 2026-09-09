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
   // check for divisibility of matrix size by number of processes
   if (n % size != 0) {
      if (rank == 0) {
         printf("Matrix size not divisible by number of processes.\n");
      }
      MPI_Finalize();
      return -1;
   }
   int rowsperprocess = n / size;

   // dynamically allocated matrices
   float *A = (float *)malloc(n * n * sizeof(float));
   float *B = (float *)malloc(n * n * sizeof(float));
   float *C = (float *)malloc(n * n * sizeof(float));

   float *Alocal = (float *)malloc(rowsperprocess * n * sizeof(float));
   float *Clocal = (float *)malloc(rowsperprocess * n * sizeof(float));

   float *Cseq = (float *)malloc(n * n * sizeof(float));

   // initialize only on process 0
   if (rank == 0) {
      
      // get random matrices A and B
      //srand(time(NULL));
      srand(42);
      for (int i = 0; i < n; i++) {
         for (int j = 0; j < n; j++) {
            A[i * n + j] = (float)(rand() % 10);
            B[i * n + j] = (float)(rand() % 10);
         }
      }

      // print matrices A and B for verification
      printf("Matrix A:\n");
      for (int i = 0; i < n; i++) {
         for (int j = 0; j < n; j++) {
            printf("%.0f ", A[i * n + j]);
         }
         printf("\n");
      }

      printf("Matrix B:\n");
      for (int i = 0; i < n; i++) {
         for (int j = 0; j < n; j++) {
            printf("%.0f ", B[i * n + j]);
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
         Clocal[i * n + j] = 0;
         for (int k = 0; k < n; k++) {
            Clocal[i * n + j] += Alocal[i * n + k] * B[k * n + j];
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
            Cseq[i * n + j] = 0;
            for (int k = 0; k < n; k++) {
               Cseq[i * n + j] += A[i * n + k] * B[k * n + j];
            }
         }
      }
      
      // check for correctness
      int correct = 1;
      for (int i = 0; i < n; i++) {
         for (int j = 0; j < n; j++) {
            if (fabs(C[i * n + j] - Cseq[i * n + j]) > 1e-6) {
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
            printf("%.0f ", C[i * n + j]);
         }
         printf("\n");
      }
   }

   // free allocated memory
   free(Alocal);
   free(Clocal);
   if (rank == 0) {
      free(A);
      free(B);
      free(C);
      free(Cseq);
   }

   MPI_Finalize();

   return 0;

}