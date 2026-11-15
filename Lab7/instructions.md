Dynamo / DynamoDB-Style Distributed Key-Value Store
Implement a simplified Dynamo-style distributed key-value store in Go.

Your implementation should demonstrate:

Consistent hashing.
Virtual nodes.
Gossip membership.
Replication.
Failure detection.
Temporary failure recovery.
Permanent failure recovery.
Basic PUT and GET operations.
Required Functions
At minimum, implement functions equivalent to:

AddNode(node)
DeleteNode(node)
Put(key, value)
Get(key)
Get(key) should identify the server responsible for the key and retrieve its value.

Put(key, value) should place the key-value pair on the correct server and its replicas.

You may use Go's hashing libraries, such as MD5 or SHA-based hashing, to place nodes and keys on the hash ring.

Your implementation should avoid unnecessary recomputation. For example, do not repeatedly hash values that can be stored and reused.

Initial Configuration
For testing, assume:

5 physical servers.
Servers are distributed around the consistent-hashing ring.
Each physical server is represented by 4 virtual nodes to improve load distribution.
Each key is replicated on at least 3 physical servers.
Keys are ASCII strings no longer than 20 characters.
Your program should be written so that the number of servers and virtual nodes can be changed easily.

Example Key-Value Inserts
Test your program using at least:

Maria -> 100
John -> 20
Anna -> 40
Tim -> 100
Alex -> 10
You should also test key placement using:

Anna
Maria
Lauren
John
Thomas
Required Failure Experiment
After inserting the data, simulate the failure of the second physical server in your server list.

Show:

Where each key was located before the failure.
The primary server and replicas for each key.
Which replica answers a request while Server 2 is unavailable.
What happens if Server 2 returns after writes occurred during its outage.
How Server 2 receives updated information after recovery.
What happens if Server 2 is permanently removed.
Which new server receives the data necessary to restore the replication factor.
Your report should contain tables similar to:

Key	Primary	Replica 1	Replica 2
Anna	Server ?	Server ?	Server ?
Maria	Server ?	Server ?	Server ?
Lauren	Server ?	Server ?	Server ?
And after Server 2 fails:

Key	Available Primary/Replica	Replacement Replica
Anna	Server ?	Server ?
Maria	Server ?	Server ?
Your actual answer depends on your hash function and placement of the virtual nodes.

Performance Evaluation
Every group must include a performance study.

At minimum, compare:

Sequential / Single Node
2 Nodes
4 Nodes
Use additional nodes where appropriate.

Include tables such as:

Configuration	Execution Time	Speedup	Correct Result?
Sequential		1.00	Yes
2 Nodes			Yes
4 Nodes			Yes
Use:

Speedup=Tdistributed​/Tsequential​​

Your analysis should discuss why performance does or does not improve when additional nodes are added. Consider:

Communication overhead.
Synchronization.
Serialization.
Network latency.
Load balancing.
Failure detection.
Replication overhead.
Size of the workload.
Demonstration Requirement
You will demonstrate your implementation during lab.

Your demonstration should clearly show:

The distributed system starting.
Nodes discovering or communicating with each other.
Normal operation.
Data or work distributed among nodes.
Replication, if applicable.
A node failure.
Failure detection.
Continued system operation after the failure.
Recovery of a temporarily failed node or replacement of a permanently failed node.
Verification that the final result is correct.
Submission
Submit one ZIP file containing:

All source code.
A README with instructions for compiling and running the project.
Any scripts or configuration files needed to start the nodes.
A PDF report.
Screenshots showing the implementation running.
The PDF report must include:

Group members and individual contributions.
Description of the selected distributed system.
Membership protocol.
Consensus/coordination mechanism.
Replication mechanism.
Failure detection and recovery mechanism.
Any additional algorithms specific to the selected architecture.
Design decisions and modifications from the original paper/system.
Explanation of how to compile and execute the program.
Testing methodology.
Correctness results.
Failure and recovery experiments.
Timing tables.
Speedup/scalability analysis.
Screenshots of the system running.
You may also use the provided KeyValue.go starter code as a starting point for implementing the Put and Get operations.

A small shell of a keyValue put and get functions : KeyValue.go.pdfDownload KeyValue.go.pdf


Conclusions.
Dynamo References
For groups implementing the Dynamo-style project, the following references may be useful:

[1] David Karger, Eric Lehman, Tom Leighton, Matthew Levine, Daniel Lewin, and Rina Panigrahy, Consistent Hashing and Random Trees: Distributed Caching Protocols for Relieving Hot Spots on the World Wide Web, Proceedings of the 29th ACM Symposium on Theory of Computing (STOC), 1997.

[2] John Lamping and Eric Veach, A Fast, Minimal Memory, Consistent Hash Algorithm.