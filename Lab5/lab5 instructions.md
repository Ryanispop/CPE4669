Group Project — Gossip Membership & Heartbeat Protocol
This project may be completed in a group using the same group membership as the previous project.

Objective
Implement a Gossip-Based Membership and Heartbeat Protocol that allows a set of distributed computing nodes to exchange membership information and detect node failures.

Requirements
Your system should consist of 8 computing nodes. Each node communicates with two neighboring nodes.

The two neighbors may be:

Selected randomly, or
Assigned using a fixed topology.
Each node must maintain a heartbeat table containing information such as:

Node ID	Heartbeat Counter	Last Update Time
Node 1	15	timestamp
Node 2	12	timestamp
...	...	...
Node Behavior
Each node must perform the following operations:

Maintain a Heartbeat Counter
Each node has its own heartbeat counter (HBcounter).
Increment the counter every X seconds, where your group chooses an appropriate value for X.
Exchange Heartbeat Tables
Every Y seconds, each node sends its current heartbeat/membership table to its two neighbors.
Your group may choose an appropriate value for Y.
When a node receives a table, it should update its local information based on the received heartbeat values and timestamps.
Detect Node Failures
A node should be considered failed if its heartbeat has not been updated within a specified timeout.
Clearly define and document the failure-detection rule used by your implementation.
Simulate Failures
Simulate one node failing every Z seconds, where your group chooses an appropriate value for Z.
Demonstrate how information about the failed node propagates through the system.
Show how the heartbeat/membership tables at the remaining nodes change after a failure occurs.
Demonstrate Gossip Propagation
Your output should make it possible to observe:
Heartbeat counters increasing.
Nodes exchanging heartbeat tables.
Membership information propagating between nodes.
Failed nodes being detected by other nodes.
Implementation
You may use Go's RPC (net/rpc) library for communication between nodes.

Some starter code is provided in netRPCStart.zip. You may modify or replace as much of the starter code as necessary. Its primary purpose is to provide an example of setting up communication using the RPC library.

Submission Requirements
Upload one ZIP file containing:

All source code.
A README with clear instructions explaining how to compile and run the system.
Any configuration files or scripts required to start the 8 nodes.
A brief description of your choices for X, Y, and Z and your failure-detection timeout.
Instructions explaining how to observe the gossip protocol, heartbeat updates, and simulated node failures.
Make sure your program can demonstrate all 8 nodes running, exchanging membership information, and detecting simulated failures.