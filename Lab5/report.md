# Lab 5 Report: Gossip Membership and Heartbeat Protocol

## Timing Choices and Failure Detection

- **Heartbeat interval (X): 1 second.** Each active node increments its heartbeat counter once per second and refreshes its last-update timestamp. This gives the nodes frequent liveness updates that are easy to see in the output.
- **Gossip interval (Y): 2 seconds.** Every two seconds, each active node sends its current membership table to its two randomly selected neighbors and checks for a table addressed to itself. This lets heartbeat and membership information spread while keeping the output readable.
- **Failure simulation (Z): 50–99 seconds per node.** Each client independently chooses a random integer delay in this range after it starts. When its delay expires, it stops updating its heartbeat and sending or receiving gossip. Because each client chooses its own delay, failures do not occur on one shared, fixed Z-second schedule.
- **Failure-detection timeout: more than 6 seconds.** During each gossip cycle, a node marks another node as dead in its local table when the recorded heartbeat timestamp is more than six seconds old. The timeout is longer than the one-second heartbeat interval, allowing for timing variation; detection is checked about every two seconds and depends on the node having learned that member's table entry.

## How to Run and Observe the Protocol

The server and all eight clients use `localhost:9005`, so run them on the same computer. Use Go 1.25 or newer.

1. Open a terminal in the `Lab5` directory and start the RPC server:

   ```bash
   go run server.go
   ```

2. Leave the server running. Open eight additional terminals in the same directory and start one client in each, using a unique ID from 1 through 8:

   ```bash
   go run client.go 1
   ```

   Repeat with `2`, `3`, `4`, `5`, `6`, `7`, and `8` in the other terminals.

3. In the server terminal, observe `Started server` and an `Added node N` message as each client registers.

4. In each client terminal, observe the selected neighbor IDs and the client's chosen failure delay. About every two seconds, the client prints its membership table. Check that the local heartbeat counter increases, and watch for entries learned through gossip to appear in the table.

5. After a client's displayed delay, that client prints `Node N crashed` and stops its heartbeat and gossip activity. In the other client terminals, watch for that node's last-update timestamp to become more than six seconds old and for the node to be marked dead in each local membership table that contains it. Detection is evaluated during the gossip cycle, so it may occur after the timeout boundary.

6. Press Ctrl+C in each terminal to stop the server and clients. A simulated crash stops a client's protocol activity but leaves its process open until it is stopped manually.

Neighbor selection is random and independent for each node, so the topology and the speed at which membership information spreads can vary between runs.
