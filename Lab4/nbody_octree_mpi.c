#define main nbody_single_process_main
#include "nbody_octree.c"
#undef main

#include <mpi.h>
#include <string.h>

static void make_partition(int n, int processes, int *counts, int *displacements)
{
   int base = n / processes;
   int remainder = n % processes;
   int offset = 0;

   for (int rank = 0; rank < processes; rank++) {
      counts[rank] = base + (rank < remainder ? 1 : 0);
      displacements[rank] = offset;
      offset += counts[rank];
   }
}

static void distributed_step(
   Body *bodies,
   int n,
   int start,
   int local_count,
   const int *acceleration_counts,
   const int *acceleration_displacements,
   double *local_accelerations,
   double *all_accelerations,
   MPI_Comm communicator)
{
   OctreeNode *root = buildOctree(bodies, n);
   computeMassDistribution(root, bodies);

   for (int i = 0; i < local_count; i++) {
      int body_index = start + i;
      bodies[body_index].ax = 0.0;
      bodies[body_index].ay = 0.0;
      bodies[body_index].az = 0.0;
      computeForceFromTree(bodies, body_index, root);

      local_accelerations[3 * i] = bodies[body_index].ax;
      local_accelerations[3 * i + 1] = bodies[body_index].ay;
      local_accelerations[3 * i + 2] = bodies[body_index].az;
   }

   MPI_Allgatherv(
      local_accelerations,
      3 * local_count,
      MPI_DOUBLE,
      all_accelerations,
      acceleration_counts,
      acceleration_displacements,
      MPI_DOUBLE,
      communicator
   );

   for (int i = 0; i < n; i++) {
      bodies[i].ax = all_accelerations[3 * i];
      bodies[i].ay = all_accelerations[3 * i + 1];
      bodies[i].az = all_accelerations[3 * i + 2];
   }

   update_bodies(bodies, n, DT);
   freeOctree(root);
}

static int verify_distributed_correctness(int rank, int processes, MPI_Comm communicator)
{
   const int test_n = 4;
   Body initial[test_n];
   Body distributed[test_n];
   Body direct[test_n];
   int *counts = malloc((size_t)processes * sizeof(int));
   int *displacements = malloc((size_t)processes * sizeof(int));
   int *acceleration_counts = malloc((size_t)processes * sizeof(int));
   int *acceleration_displacements = malloc((size_t)processes * sizeof(int));

   make_partition(test_n, processes, counts, displacements);
   for (int i = 0; i < processes; i++) {
      acceleration_counts[i] = 3 * counts[i];
      acceleration_displacements[i] = 3 * displacements[i];
   }

   if (rank == 0) {
      srand(0);
      initialize_bodies(initial, test_n);
   }
   MPI_Bcast(initial, (int)sizeof(initial), MPI_BYTE, 0, communicator);
   memcpy(distributed, initial, sizeof(initial));

   int local_count = counts[rank];
   double *local_accelerations = calloc(
      (size_t)(3 * (local_count > 0 ? local_count : 1)), sizeof(double));
   double all_accelerations[3 * test_n];

   distributed_step(
      distributed,
      test_n,
      displacements[rank],
      local_count,
      acceleration_counts,
      acceleration_displacements,
      local_accelerations,
      all_accelerations,
      communicator
   );

   int passed = 1;
   if (rank == 0) {
      memcpy(direct, initial, sizeof(initial));
      compute_forces(direct, test_n);
      update_bodies(direct, test_n, DT);

      double max_difference = 0.0;
      for (int i = 0; i < test_n; i++) {
         max_difference = fmax(max_difference, fabs(direct[i].x - distributed[i].x));
         max_difference = fmax(max_difference, fabs(direct[i].y - distributed[i].y));
         max_difference = fmax(max_difference, fabs(direct[i].z - distributed[i].z));
         max_difference = fmax(max_difference, fabs(direct[i].vx - distributed[i].vx));
         max_difference = fmax(max_difference, fabs(direct[i].vy - distributed[i].vy));
         max_difference = fmax(max_difference, fabs(direct[i].vz - distributed[i].vz));
      }

      passed = max_difference <= 1e-6;
      printf(
         "MPI correctness check: %s (max difference %.6e)\n",
         passed ? "PASS" : "FAIL",
         max_difference
      );
   }

   MPI_Bcast(&passed, 1, MPI_INT, 0, communicator);
   free(local_accelerations);
   free(counts);
   free(displacements);
   free(acceleration_counts);
   free(acceleration_displacements);
   return passed;
}

int main(int argc, char **argv)
{
   int rank;
   int processes;
   int num_bodies = 1000;
   int num_steps = 100;

   MPI_Init(&argc, &argv);
   MPI_Comm_rank(MPI_COMM_WORLD, &rank);
   MPI_Comm_size(MPI_COMM_WORLD, &processes);

   if (argc >= 2)
      num_bodies = atoi(argv[1]);
   if (argc >= 3)
      num_steps = atoi(argv[2]);

   if (num_bodies < 4 || num_steps < 0) {
      if (rank == 0)
         fprintf(stderr, "Usage: %s <bodies>=4 <steps>=0\n", argv[0]);
      MPI_Finalize();
      return EXIT_FAILURE;
   }

   if (!verify_distributed_correctness(rank, processes, MPI_COMM_WORLD)) {
      MPI_Finalize();
      return EXIT_FAILURE;
   }

   Body *bodies = malloc((size_t)num_bodies * sizeof(Body));
   int *counts = malloc((size_t)processes * sizeof(int));
   int *displacements = malloc((size_t)processes * sizeof(int));
   int *acceleration_counts = malloc((size_t)processes * sizeof(int));
   int *acceleration_displacements = malloc((size_t)processes * sizeof(int));

   if (rank == 0) {
      srand(0);
      initialize_bodies(bodies, num_bodies);
   }
   MPI_Bcast(
      bodies,
      (int)((size_t)num_bodies * sizeof(Body)),
      MPI_BYTE,
      0,
      MPI_COMM_WORLD
   );

   make_partition(num_bodies, processes, counts, displacements);
   for (int i = 0; i < processes; i++) {
      acceleration_counts[i] = 3 * counts[i];
      acceleration_displacements[i] = 3 * displacements[i];
   }

   int local_count = counts[rank];
   double *local_accelerations = calloc(
      (size_t)(3 * (local_count > 0 ? local_count : 1)), sizeof(double));
   double *all_accelerations = calloc(
      (size_t)(3 * num_bodies), sizeof(double));

   MPI_Barrier(MPI_COMM_WORLD);
   double start = MPI_Wtime();

   for (int step = 0; step < num_steps; step++) {
      distributed_step(
         bodies,
         num_bodies,
         displacements[rank],
         local_count,
         acceleration_counts,
         acceleration_displacements,
         local_accelerations,
         all_accelerations,
         MPI_COMM_WORLD
      );
   }

   double local_elapsed = MPI_Wtime() - start;
   double elapsed = 0.0;
   MPI_Reduce(
      &local_elapsed,
      &elapsed,
      1,
      MPI_DOUBLE,
      MPI_MAX,
      0,
      MPI_COMM_WORLD
   );

   if (rank == 0) {
      printf(
         "SUMMARY bodies=%d steps=%d ranks=%d seconds=%.9f status=PASS\n",
         num_bodies,
         num_steps,
         processes,
         elapsed
      );
   }

   free(all_accelerations);
   free(local_accelerations);
   free(acceleration_displacements);
   free(acceleration_counts);
   free(displacements);
   free(counts);
   free(bodies);
   MPI_Finalize();
   return EXIT_SUCCESS;
}
