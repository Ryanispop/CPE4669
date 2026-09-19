CC = mpicc
CFLAGS = -O3 -std=c11 -Wall -Wextra -Wpedantic
TARGET = mpi_sort

.PHONY: all test clean

all: $(TARGET)

$(TARGET): mpi_sort.c
	$(CC) $(CFLAGS) mpi_sort.c -o $(TARGET)

test: $(TARGET)
	mpirun -np 1 ./$(TARGET) 0
	mpirun -np 3 ./$(TARGET) 17
	mpirun -np 5 ./$(TARGET) 100003

clean:
	rm -f $(TARGET)