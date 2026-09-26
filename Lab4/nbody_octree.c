#include <stdio.h>
#include <stdlib.h>
#include <math.h>
#include <time.h>

#define G 6.67430e-11
#define SOFTENING 1e-9
#define DT 0.01
#define THETA 0.5

typedef struct {
    double x, y, z;      // Position
    double vx, vy, vz;   // Velocity
    double ax, ay, az;   // Acceleration
    double mass;
} Body;

// define the Octree structure and related functions
typedef struct OctreeNode {
   double x, y, z;      // center of cube
   double halfSize;     // half of cube's width
   double mass;
   double comX, comY, comZ;  // center of mass
   int isLeaf;
   int bodyIndex;  // index of the body contained in this leaf node, if any

   struct OctreeNode *children[8];
} OctreeNode;

OctreeNode *createOctreeNode(double x, double y, double z, double halfSize) {
   OctreeNode *node = (OctreeNode *)malloc(sizeof(OctreeNode));
   
   if (node == NULL) {
      fprintf(stderr, "Failed to allocate memory for OctreeNode\n");
      exit(EXIT_FAILURE);
   }

   node->x = x;
   node->y = y;
   node->z = z;
   node->halfSize = halfSize;
   node->mass = 0.0;
   node->comX = 0.0;
   node->comY = 0.0;
   node->comZ = 0.0;
   node->isLeaf = 1;
   node->bodyIndex = -1;
   for (int i = 0; i < 8; i++)
      node->children[i] = NULL;

   return node;
}

int getOctant(OctreeNode *node, Body *body) {
   int oct = 0;
   if (body->x >= node->x) oct |= 1;
   if (body->y >= node->y) oct |= 2;
   if (body->z >= node->z) oct |= 4;
   return oct;
}

OctreeNode *createChild(OctreeNode *parent, int octant) {
   double offset = parent->halfSize / 2.0;
   double x = parent->x + ((octant & 1) ? offset : -offset);
   double y = parent->y + ((octant & 2) ? offset : -offset);
   double z = parent->z + ((octant & 4) ? offset : -offset);
   return createOctreeNode(x, y, z, offset);
}

void insertBody(OctreeNode *node, Body *bodies, int bodyIndex) {
   Body *body = &bodies[bodyIndex];

   // empty leaf, so store the body here
   if (node->isLeaf && node->bodyIndex == -1) {
      node->bodyIndex = bodyIndex;
      return;
   }

   // leaf already contains a body
   if (node->isLeaf) {
      int oldBodyIndex = node->bodyIndex;
      Body *oldBody = &bodies[oldBodyIndex];

      // node is no longer a leaf
      node->isLeaf = 0;
      node->bodyIndex = -1;

      // move the old body into the correct child
      int oldOctant = getOctant(node, oldBody);

      if (node->children[oldOctant] == NULL) {
         node->children[oldOctant] = createChild(node, oldOctant);
      }

      insertBody(node->children[oldOctant], bodies, oldBodyIndex);
   }

   // insert the new body into the correct child
   int newOctant = getOctant(node, body);

   if (node->children[newOctant] == NULL) {
      node->children[newOctant] = createChild(node, newOctant);
   }

   insertBody(node->children[newOctant], bodies, bodyIndex);
}

OctreeNode *buildOctree(Body *bodies, int n) {
   double minX = bodies[0].x;
   double maxX = bodies[0].x;
   double minY = bodies[0].y;
   double maxY = bodies[0].y;
   double minZ = bodies[0].z;
   double maxZ = bodies[0].z;

   // find the bounds of all bodies
   for (int i = 1; i < n; i++) {
      if (bodies[i].x < minX) minX = bodies[i].x;
      if (bodies[i].x > maxX) maxX = bodies[i].x;
      if (bodies[i].y < minY) minY = bodies[i].y;
      if (bodies[i].y > maxY) maxY = bodies[i].y;
      if (bodies[i].z < minZ) minZ = bodies[i].z;
      if (bodies[i].z > maxZ) maxZ = bodies[i].z;
   }

   // bounding cube centers
   double centerX = (minX + maxX) / 2.0;
   double centerY = (minY + maxY) / 2.0;
   double centerZ = (minZ + maxZ) / 2.0;

   // find the largest dimension
   double sizeX = maxX - minX;
   double sizeY = maxY - minY;
   double sizeZ = maxZ - minZ;

   double maxSize = sizeX;

   if (sizeY > maxSize) maxSize = sizeY;
   if (sizeZ > maxSize) maxSize = sizeZ;

   double halfSize = maxSize / 2.0 + 1.0;

   OctreeNode *root = createOctreeNode(centerX, centerY, centerZ, halfSize);

   for (int i = 0; i < n; i++) {
      insertBody(root, bodies, i);
   }

   return root;
}

void computeMassDistribution(OctreeNode *node, Body *bodies) {
   if (node == NULL)
      return;

   // leaf containing a body
   if (node->isLeaf) {
      if (node->bodyIndex != -1) {
         Body *body = &bodies[node->bodyIndex];

         node->mass = body->mass;
         node->comX = body->x;
         node->comY = body->y;
         node->comZ = body->z;
      }

      return;
   }

   // internal node
   node->mass = 0.0;
   node->comX = 0.0;
   node->comY = 0.0;
   node->comZ = 0.0;

   for (int i = 0; i < 8; i++) {

      OctreeNode *child = node->children[i];

      if (child == NULL)
         continue;

      // calculate the child's mass
      computeMassDistribution(child, bodies);

      node->mass += child->mass;
      node->comX += child->mass * child->comX;
      node->comY += child->mass * child->comY;
      node->comZ += child->mass * child->comZ;
   }

   // convert weighted sums into center of mass
   if (node->mass > 0.0) {
      node->comX /= node->mass;
      node->comY /= node->mass;
      node->comZ /= node->mass;
   }
}

void applyForce(Body *body, double mass, double x, double y, double z) {
   double dx = x - body->x;
   double dy = y - body->y;
   double dz = z - body->z;

   double distSquared = dx * dx + dy * dy + dz * dz + SOFTENING;
   double dist = sqrt(distSquared);
   double acc = G * mass / distSquared;

   body->ax += acc * dx / dist;
   body->ay += acc * dy / dist;
   body->az += acc * dz / dist;
}

void computeForceFromTree(Body *bodies, int bodyIndex, OctreeNode *node) {
   if (node == NULL || node->mass == 0.0)
      return;
   
   Body *body = &bodies[bodyIndex];

   if (node->isLeaf) {

      // don't calculate a body's force on itself
      if (node->bodyIndex == -1 || node->bodyIndex == bodyIndex)
         return;

      Body *other = &bodies[node->bodyIndex];

      applyForce(body, other->mass, other->x, other->y, other->z);

      return;
   }

   // distance from body to this node's center of mass
   double dx = node->comX - body->x;
   double dy = node->comY - body->y;
   double dz = node->comZ - body->z;

   double distance = sqrt(dx * dx + dy * dy + dz * dz + SOFTENING);
   double size = node->halfSize * 2.0;

   // Barnes-Hut approximation condition
   if ((size / distance) < THETA) {

      applyForce(body, node->mass, node->comX, node->comY, node->comZ);
      return;
   }

   // node is too close, so examine their children
   for (int i = 0; i < 8; i++) {
      computeForceFromTree(bodies, bodyIndex, node->children[i]);
   }
}

void computeForcesOctree(Body *bodies, int n, OctreeNode *root) {
   for (int i = 0; i < n; i++) {

      bodies[i].ax = 0.0;
      bodies[i].ay = 0.0;
      bodies[i].az = 0.0;

      // calculate force using Octree
      computeForceFromTree(bodies, i, root);
   }
}

void freeOctree(OctreeNode *node) {
   if (node == NULL)
      return;

   for (int i = 0; i < 8; i++) {
      freeOctree(node->children[i]);
   }

   free(node);
}

/* Function prototypes */
void initialize_bodies(Body *bodies, int n);
void compute_forces(Body *bodies, int n);
void update_bodies(Body *bodies, int n, double dt);
void print_bodies(Body *bodies, int n);


/*
 * Initialize bodies with random positions, velocities, and masses.
 */
void initialize_bodies(Body *bodies, int n)
{
    for (int i = 0; i < n; i++) {

        bodies[i].x = ((double)rand() / RAND_MAX) * 100.0;
        bodies[i].y = ((double)rand() / RAND_MAX) * 100.0;
        bodies[i].z = ((double)rand() / RAND_MAX) * 100.0;

        bodies[i].vx = 0.0;
        bodies[i].vy = 0.0;
        bodies[i].vz = 0.0;

        bodies[i].ax = 0.0;
        bodies[i].ay = 0.0;
        bodies[i].az = 0.0;

        bodies[i].mass =
            1.0e20 + ((double)rand() / RAND_MAX) * 1.0e20;
    }
}


/*
 * Sequential O(N^2) force calculation.
 *
 * This is the baseline implementation.
 * Later, this function can be replaced with an Octree /
 * Barnes-Hut version.
 */
void compute_forces(Body *bodies, int n)
{
    /* Reset acceleration */
    for (int i = 0; i < n; i++) {
        bodies[i].ax = 0.0;
        bodies[i].ay = 0.0;
        bodies[i].az = 0.0;
    }

    /* Compute gravitational forces */
    for (int i = 0; i < n; i++) {

        for (int j = 0; j < n; j++) {

            if (i == j)
                continue;

            double dx = bodies[j].x - bodies[i].x;
            double dy = bodies[j].y - bodies[i].y;
            double dz = bodies[j].z - bodies[i].z;

            double distance_squared =
                dx * dx +
                dy * dy +
                dz * dz +
                SOFTENING;

            double distance = sqrt(distance_squared);

            double acceleration =
                G * bodies[j].mass /
                distance_squared;

            bodies[i].ax += acceleration * dx / distance;
            bodies[i].ay += acceleration * dy / distance;
            bodies[i].az += acceleration * dz / distance;
        }
    }
}


/*
 * Update velocity and position using a simple Euler integration.
 */
void update_bodies(Body *bodies, int n, double dt)
{
   for (int i = 0; i < n; i++) {

      /* Update velocity */
      bodies[i].vx += bodies[i].ax * dt;
      bodies[i].vy += bodies[i].ay * dt;
      bodies[i].vz += bodies[i].az * dt;

      /* Update position */
      bodies[i].x += bodies[i].vx * dt;
      bodies[i].y += bodies[i].vy * dt;
      bodies[i].z += bodies[i].vz * dt;
   }
}


/*
* Print body information.
*
* Useful for debugging with a small number of bodies.
*/
void print_bodies(Body *bodies, int n)
{
   for (int i = 0; i < n; i++) {

      printf(
         "Body %d: "
         "pos=(%.4f, %.4f, %.4f) "
         "vel=(%.4f, %.4f, %.4f)\n",
         i,
         bodies[i].x,
         bodies[i].y,
         bodies[i].z,
         bodies[i].vx,
         bodies[i].vy,
         bodies[i].vz
      );
   }
}


int main(int argc, char *argv[])
{
   int num_bodies = 1000;
   int num_steps = 100;

   /*
   * Allow command-line arguments:
   *
   * ./nbody 10000 100
   *
   * argv[1] = number of bodies
   * argv[2] = number of simulation steps
   */
   if (argc >= 2)
      num_bodies = atoi(argv[1]);

   if (argc >= 3)
      num_steps = atoi(argv[2]);

   printf("N-Body Simulation\n");
   printf("-----------------\n");
   printf("Bodies: %d\n", num_bodies);
   printf("Steps : %d\n\n", num_steps);

   Body *bodies =
      (Body *)malloc(num_bodies * sizeof(Body));

   if (bodies == NULL) {
      fprintf(stderr, "Error allocating memory.\n");
      return EXIT_FAILURE;
   }

   srand(0);

   initialize_bodies(bodies, num_bodies);
   clock_t start = clock();

   for (int step = 0; step < num_steps; step++) {

      // build Octree using current body positions
      OctreeNode *root = buildOctree(bodies, num_bodies);

      // calculate mass and center of mass
      computeMassDistribution(root, bodies);

      // calculate forces using Barnes-Hut
      computeForcesOctree(bodies, num_bodies, root);

      update_bodies(bodies, num_bodies, DT);
      freeOctree(root);
   }

   clock_t end = clock();

   double elapsed =
      (double)(end - start) / CLOCKS_PER_SEC;

   printf("Simulation completed.\n");
   printf("Execution time: %.6f seconds\n", elapsed);

   free(bodies);
   return EXIT_SUCCESS;
}