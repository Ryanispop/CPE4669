# Simplified Raft Leader Election — 8 Nodes

This program implements the leader-election portion of Raft. It starts exactly
eight nodes as Followers, elects a Leader with at least five votes, sends
heartbeats, retries unsuccessful elections, and automatically elects a new
Leader after the current one crashes. It is not Paxos or a complete Raft log
replication system.

## Quick start

You need **Python 3.10 or newer**. Only the Python standard library is used;
there are no packages to install. Extract the ZIP, open a terminal in the
`raft_leader_election` folder, and run:

```sh
python main.py
```

This starts **all 8 nodes in one process**. You do not need eight terminals.
The default demonstration runs for about three seconds:

1. All eight nodes start as Followers in term 0.
2. A randomized timeout triggers an election.
3. A candidate receives five votes and becomes Leader.
4. The Leader sends heartbeats every 50 ms.
5. One simulated second after the first election, the harness crashes the Leader.
6. The remaining nodes time out, hold another election, and elect a replacement.
7. A `PASS` line confirms the demonstration completed.

On macOS/Linux, use `python3` if `python` is unavailable. On Windows, you can use
`py -3` instead of `python`. In VS Code, open the extracted folder, choose
**Terminal > New Terminal**, and run the same command. WSL is not required.
Check your version with `python --version`; installation instructions are at
[python.org](https://www.python.org/downloads/).

## Compile and run

Python runs source files directly, so a separate compilation step is not needed.
For an optional syntax/bytecode compilation check:

```sh
python -m py_compile main.py raft.py test_raft.py
python main.py
```

Keep `main.py` and `raft.py` in the same directory. Run commands from that
directory. Press Ctrl+C to stop a running demonstration early.

## Demonstrations

| What to observe | Command |
| --- | --- |
| Initial election, automatic Leader crash, and replacement | `python main.py` |
| Repeatable 4–4 split vote, retry, and successful election | `python main.py --scenario split` |
| Healthy cluster with periodic heartbeats and no injected crash | `python main.py --scenario election` |
| Every received heartbeat and timer reset | `python main.py --verbose-heartbeats` |
| Run without real-time waiting | `python main.py --fast` |
| Try a different random schedule | `python main.py --seed 19` |
| Run longer and crash two seconds after the first election | `python main.py --duration 6 --fail-after 2` |
| Show all options | `python main.py --help` |

The default seed is **7**. Using the same seed reproduces the same simulated
message/timer schedule. `--duration` is the total simulated run time;
`--fail-after` is measured from the **first successful election**, not program
startup, and applies only to `failure` mode. Choose a duration long enough to
include the crash and recovery. An unfinished demonstration prints
`INCOMPLETE` and exits with status 1; a completed demonstration exits with 0.

### Split votes and re-elections

Run:

```sh
python main.py --scenario split
```

The harness initially blocks messages between nodes 1–4 and nodes 5–8. All
election timeouts still come from the normal randomized 150–300 ms range.
Each four-node group can give a candidate four votes, but **neither group has
the required five votes**. Candidates send requests to all seven peers;
cross-group requests are logged as `DROP` during the temporary partition.

When two candidates have four votes each in the same term, the program prints
`SPLIT` and reconnects the groups. Healing the connection does not assign a
Leader or change anyone's votes. The nodes continue waiting for their existing
randomized deadlines. With the default seed, a candidate times out, prints
`RETRY`, increments its term, requests votes again, and wins a majority.

This controlled network fault makes a split vote easy to demonstrate. The
ordinary election and failure modes start fully connected. Other seeds can
produce additional competing candidates or let a Follower start the next
election before either original candidate. Use seed 7 for the supplied trace.

## Reading the output

Each line begins with elapsed **simulated milliseconds**, such as `[00178ms]`.
The program normally paces simulated time at approximately real time.
`--fast` removes the waits while preserving the same event timestamps and rules.

| Log label | Meaning |
| --- | --- |
| `INIT` | Node starts as a Follower; initial random timeout is shown. |
| `TIMEOUT` | A Follower's election deadline expires. |
| `STATE` | Follower, Candidate, or Leader transition. |
| `TERM` | A received message causes a node to adopt a higher term. |
| `ELECTION` | New election term, self-vote, and next random timeout. |
| `REQUEST` | Vote request sent to an individual peer. |
| `VOTE` | Request received, grant/denial reason, and response sent. |
| `RESPONSE` | Candidate receives a vote response and updates its count. |
| `LEADER` | At least five distinct votes have elected a Leader. |
| `HEARTBEAT` | Leader broadcast or a Follower's timer reset. |
| `SPLIT` | Two candidates have four votes each; no majority. |
| `RETRY` | Candidate's election timed out; another term is beginning. |
| `NETWORK` / `DROP` | Temporary partition, blocked messages, or recovery. |
| `CRASH` | Current Leader stops processing events and sending messages. |
| `STALE` | An old-term heartbeat is rejected. |
| `STATUS` | Table of all eight nodes, normally every 500 ms. |
| `SUMMARY` / `PASS` | Election history and scenario outcome. |

Every node stores its ID, term, state, `voted_for`, distinct received votes,
last heartbeat time, known leader, and election deadline. A node's term is local:
nodes learn about newer terms through messages. The `ELECTION` line also shows
term increments for candidates that remain Candidates across retries.

The status table retains a crashed node's last role and shows `alive=no`.
That row is **not a functioning second Leader**. `alive` is separate from the
three Raft roles. `voted_for` may differ from the eventual Leader because a node
can vote for a losing candidate and then accept the winning Leader's heartbeat.
`votes` is the number of votes this node collected for its own candidacy, not the
number it cast. A Leader's last received heartbeat can be `never` or an older
timestamp because it does not send heartbeats to itself.

The default output shows every heartbeat broadcast and each node's first
heartbeat from a newly recognized Leader. Use `--verbose-heartbeats` to see
every received heartbeat and its new timeout.

## Implementation and design decisions

- **Single-process event simulation:** each node has independent protocol state.
  A message queue and one event loop model asynchronous delivery, avoiding
  dependence on operating-system thread scheduling. These are eight simulated
  nodes, not eight network services or worker threads. A node makes decisions
  only from its own state, timeouts, and received messages.
- **Timing:** each timer reset independently draws an integer from 150 through
  300 ms, inclusive. Candidates use the same range to wait for election
  responses. Heartbeats are sent immediately after election and every 50 ms
  afterward. Messages normally take 5 simulated ms one way. If a message and
  timeout share a timestamp, the message is processed first.
- **Election rules:** a candidate increments its term, votes for itself, and
  requests every other node's vote. A set of voter IDs prevents duplicate
  responses from inflating its count. It becomes Leader immediately upon
  receiving its fifth distinct vote; it need not wait for all responses.
- **Terms and votes:** a higher term in any request or response causes a node
  to become a Follower and clear its previous-term vote. A node grants at most
  one candidate its vote in each term. Repeated requests from that same
  candidate receive an idempotent reply. Granted votes reset the election
  timer; denied and stale requests do not.
- **Heartbeats:** valid current/new-term heartbeats reset the timer and make a
  Candidate return to Follower. A same-term transition preserves `voted_for`.
  Older heartbeats are rejected without resetting the timer. Heartbeat replies
  let an outdated Leader discover a higher term and step down.
- **Failure model:** a crash stops the whole simulated node, including its
  heartbeats and voting. It does not restart. Messages already sent before the
  crash may finish their 5 ms trip. Messages arriving at a crashed node are
  ignored. Cluster membership remains eight, so a majority always means five;
  it never shrinks to four just because one node failed.
- **Scope:** this is leader election only. There are no replicated logs, client
  commands, persistence, restart recovery, membership changes, or log freshness
  comparisons. The transient partition is a demonstration mechanism. Recovery
  assumes at least five live nodes can communicate; four cannot elect a Leader.

The demo harness can crash a node, partition/reconnect the network, and observe
results. It does not elect a Leader centrally or supply fabricated votes.

## Files and checks

| File | Purpose |
| --- | --- |
| `main.py` | Command-line options, scenario harness, pacing, and summaries. |
| `raft.py` | Node state machine, messages, timers, and simulated transport. |
| `test_raft.py` | Automated behavioral checks for safety and recovery. |
| `examples/leader_failure.txt` | Actual captured output from the default scenario. |
| `examples/split_vote.txt` | Actual captured output from the split scenario. |
| `README.md` | Setup, execution, explanations, and observation instructions. |

Run all checks:

```sh
python -m unittest -v
```

Tests cover one vote per term, duplicate/stale messages, higher-term step-down,
the five-vote threshold, heartbeat timer resets, candidate retries, the 4–4
partition, failure recovery, and behavior with only four or five survivors.
The tests also check both recovery scenarios across 25 different seeds and
verify that no two different Leaders were elected in the same term.

To capture your own demonstration output:

```sh
python main.py --fast > leader_failure.txt
python main.py --scenario split --fast > split_vote.txt
```

The supplied examples were generated with seed 7. They are recorded program
output, not hard-coded output used by the simulation.

## Reference

Diego Ongaro and John Ousterhout, *In Search of an Understandable Consensus
Algorithm (Extended Version)*, especially Figure 2 and Section 5.2:
[Raft paper](https://raft.github.io/raft.pdf).

The project follows the assignment's simplified election scope. Full Raft adds
log replication, log-dependent voting restrictions, and durable state.
