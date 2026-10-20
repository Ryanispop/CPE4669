"""Eight-node, message-based simulation of Raft leader election only.

Time is measured in integer simulated milliseconds. A single event loop orders
messages and timers, so a seed reproduces an execution without thread races.
Nodes never inspect other nodes' election state to decide how to vote.
"""

from __future__ import annotations

import heapq
import random
from dataclasses import dataclass
from enum import Enum
from typing import TextIO

NODE_COUNT = 8
MAJORITY = NODE_COUNT // 2 + 1  # Always 5, including after a node crashes.
MIN_TIMEOUT_MS = 150
MAX_TIMEOUT_MS = 300
HEARTBEAT_INTERVAL_MS = 50
MESSAGE_DELAY_MS = 5


class State(str, Enum):
    FOLLOWER = "Follower"
    CANDIDATE = "Candidate"
    LEADER = "Leader"


class Kind(str, Enum):
    REQUEST_VOTE = "RequestVote"
    VOTE_RESPONSE = "VoteResponse"
    HEARTBEAT = "Heartbeat"
    HEARTBEAT_RESPONSE = "HeartbeatResponse"


@dataclass(frozen=True)
class Message:
    kind: Kind
    term: int
    source: int
    destination: int
    granted: bool = False


@dataclass(frozen=True)
class Election:
    time_ms: int
    node_id: int
    term: int
    voters: tuple[int, ...]


class Node:
    def __init__(self, node_id: int, cluster: Cluster):
        self.node_id = node_id
        self.cluster = cluster
        self.current_term = 0
        self.state = State.FOLLOWER
        self.voted_for: int | None = None
        self.votes: set[int] = set()
        self.last_heartbeat: int | None = None
        self.leader_id: int | None = None
        self.alive = True
        self.election_timeout = 0
        self.election_deadline = 0
        self.next_heartbeat = 0
        self.reset_election_timer()
        self.log("INIT", f"Follower; timeout={self.election_timeout}ms")

    @property
    def vote_count(self) -> int:
        return len(self.votes)

    def log(self, tag: str, message: str) -> None:
        self.cluster.log(tag, f"node={self.node_id} term={self.current_term} {message}")

    def reset_election_timer(self) -> None:
        self.election_timeout = self.cluster.rng.randint(MIN_TIMEOUT_MS, MAX_TIMEOUT_MS)
        self.election_deadline = self.cluster.now + self.election_timeout

    def change_state(self, state: State, reason: str) -> None:
        if self.state != state:
            old = self.state
            self.state = state
            self.log("STATE", f"{old.value} -> {state.value}; {reason}")

    def adopt_higher_term(self, term: int, reason: str) -> None:
        """A higher term invalidates the old vote, regardless of message kind."""
        if term <= self.current_term:
            return
        was_leader = self.state == State.LEADER
        old_term = self.current_term
        self.current_term = term
        self.voted_for = None
        self.votes.clear()
        self.leader_id = None
        self.log("TERM", f"{old_term} -> {term}; {reason}")
        self.change_state(State.FOLLOWER, "observed a higher term")
        if was_leader:
            # Leaders do not use election deadlines. Arm one on stepping down.
            self.reset_election_timer()

    def tick(self) -> None:
        if not self.alive:
            return
        if self.state == State.LEADER:
            if self.cluster.now >= self.next_heartbeat:
                self.send_heartbeats()
        elif self.cluster.now >= self.election_deadline:
            self.start_election()

    def start_election(self) -> None:
        if not self.alive:
            return
        if self.state == State.CANDIDATE:
            self.log("RETRY", f"no majority: {self.vote_count}/8 votes; starting a new term")
        else:
            self.log("TIMEOUT", "election timer expired without a valid leader heartbeat")
        self.current_term += 1
        self.change_state(State.CANDIDATE, "starting election")
        self.leader_id = None
        self.voted_for = self.node_id
        self.votes = {self.node_id}
        self.cluster.vote_history.append((self.current_term, self.node_id, self.node_id))
        self.reset_election_timer()
        self.log(
            "ELECTION",
            f"self-vote; votes=1/8; need={MAJORITY}; retry timeout={self.election_timeout}ms",
        )
        for peer in range(1, NODE_COUNT + 1):
            if peer != self.node_id:
                self.log("REQUEST", f"send RequestVote -> node={peer}")
                self.cluster.send(Message(Kind.REQUEST_VOTE, self.current_term, self.node_id, peer))

    def receive(self, message: Message) -> None:
        if not self.alive:
            return
        if message.destination != self.node_id or not 1 <= message.source <= NODE_COUNT:
            return
        if message.source == self.node_id:
            return
        self.adopt_higher_term(message.term, f"{message.kind.value} from node={message.source}")
        if message.kind == Kind.REQUEST_VOTE:
            self.receive_vote_request(message)
        elif message.kind == Kind.VOTE_RESPONSE:
            self.receive_vote_response(message)
        elif message.kind == Kind.HEARTBEAT:
            self.receive_heartbeat(message)
        # Heartbeat responses require only the higher-term check above. Even a
        # rejection can tell an outdated leader to become a follower.

    def receive_vote_request(self, message: Message) -> None:
        granted = False
        if message.term < self.current_term:
            reason = "stale term"
        elif self.voted_for not in (None, message.source):
            reason = f"already voted for node={self.voted_for} in this term"
        else:
            granted = True
            reason = "vote granted"
            if self.voted_for is None:
                self.cluster.vote_history.append((self.current_term, self.node_id, message.source))
            self.voted_for = message.source
            # A duplicate request from the SAME candidate is an idempotent yes,
            # not a second vote. Granting a vote also resets the election timer.
            self.reset_election_timer()
        self.log(
            "VOTE",
            f"RequestVote from node={message.source} request_term={message.term}; "
            f"{'GRANT' if granted else 'DENY'}; {reason}; "
            f"send VoteResponse -> node={message.source}",
        )
        self.cluster.send(
            Message(Kind.VOTE_RESPONSE, self.current_term, self.node_id, message.source, granted)
        )

    def receive_vote_response(self, message: Message) -> None:
        if self.state != State.CANDIDATE or message.term != self.current_term:
            self.log("RESPONSE", f"ignore vote from node={message.source}; not in that election")
            return
        if not message.granted:
            self.log("RESPONSE", f"DENY from node={message.source}; votes={self.vote_count}/8")
            return
        if message.source in self.votes:
            self.log("RESPONSE", f"duplicate GRANT from node={message.source}; ignored")
            return
        self.votes.add(message.source)
        self.log("RESPONSE", f"GRANT from node={message.source}; votes={self.vote_count}/8")
        if self.vote_count >= MAJORITY:
            self.change_state(State.LEADER, f"received {self.vote_count}/8 votes")
            self.leader_id = self.node_id
            voters = tuple(sorted(self.votes))
            self.cluster.elections.append(Election(self.cluster.now, self.node_id, self.current_term, voters))
            self.log("LEADER", f"elected with voters={list(voters)}; majority={MAJORITY}")
            self.send_heartbeats()  # Establish authority immediately.

    def send_heartbeats(self) -> None:
        self.log("HEARTBEAT", "broadcast -> all 7 peers")
        for peer in range(1, NODE_COUNT + 1):
            if peer != self.node_id:
                self.cluster.send(Message(Kind.HEARTBEAT, self.current_term, self.node_id, peer))
        self.next_heartbeat = self.cluster.now + HEARTBEAT_INTERVAL_MS

    def receive_heartbeat(self, message: Message) -> None:
        valid = message.term == self.current_term
        if not valid:
            self.log("STALE", f"reject heartbeat from node={message.source} term={message.term}")
        else:
            new_leader = self.leader_id != message.source
            self.change_state(State.FOLLOWER, f"valid heartbeat from leader={message.source}")
            self.votes.clear()
            # Preserve voted_for for this term: accepting a heartbeat must NOT
            # allow this node to cast another vote in the same term.
            self.leader_id = message.source
            self.last_heartbeat = self.cluster.now
            self.reset_election_timer()
            if new_leader or self.cluster.verbose_heartbeats:
                self.log(
                    "HEARTBEAT",
                    f"received from leader={message.source}; timer reset to {self.election_timeout}ms",
                )
        self.cluster.send(
            Message(Kind.HEARTBEAT_RESPONSE, self.current_term, self.node_id, message.source, valid)
        )


class Cluster:
    """Transport, clock, and observations; voting decisions remain in Node."""

    def __init__(self, seed: int = 7, output: TextIO | None = None,
                 partitioned: bool = False, verbose_heartbeats: bool = False):
        self.now = 0
        self.rng = random.Random(seed)
        self.output = output
        self.partitioned = partitioned
        self.verbose_heartbeats = verbose_heartbeats
        self.messages: list[tuple[int, int, Message]] = []
        self.sequence = 0
        self.elections: list[Election] = []
        # Observation only: (term, voter, candidate), useful for checking safety.
        self.vote_history: list[tuple[int, int, int]] = []
        self.nodes = {i: Node(i, self) for i in range(1, NODE_COUNT + 1)}

    def log(self, tag: str, message: str) -> None:
        if self.output is not None:
            print(f"[{self.now:05d}ms] {tag:<10} {message}", file=self.output)

    def connected(self, source: int, destination: int) -> bool:
        return not self.partitioned or (source <= 4) == (destination <= 4)

    def send(self, message: Message) -> None:
        if not self.nodes[message.source].alive:
            return
        if not self.connected(message.source, message.destination):
            self.log("DROP", f"{message.kind.value} {message.source}->{message.destination}; 4/4 partition")
            return
        self.sequence += 1
        heapq.heappush(self.messages, (self.now + MESSAGE_DELAY_MS, self.sequence, message))

    def step(self) -> None:
        """Advance 1 ms; arrivals precede timeouts at an identical timestamp."""
        self.now += 1
        while self.messages and self.messages[0][0] <= self.now:
            _, _, message = heapq.heappop(self.messages)
            if self.connected(message.source, message.destination):
                self.nodes[message.destination].receive(message)
        for node in self.nodes.values():
            node.tick()

    def crash(self, node_id: int) -> None:
        node = self.nodes[node_id]
        if node.alive:
            node.alive = False
            self.log(
                "CRASH",
                f"node={node_id} stopped; last role={node.state.value} term={node.current_term}; "
                f"no more heartbeats or votes; quorum stays {MAJORITY}/8",
            )

    def show_status(self) -> None:
        self.log("STATUS", "node | alive | state     | term | voted_for | votes | leader | last heartbeat")
        for node in self.nodes.values():
            last = "never" if node.last_heartbeat is None else f"{node.last_heartbeat}ms"
            self.log(
                "STATUS",
                f"{node.node_id:>4} | {'yes' if node.alive else 'no':<5} | {node.state.value:<9} | "
                f"{node.current_term:>4} | {str(node.voted_for or '-'):>9} | {node.vote_count:>5} | "
                f"{str(node.leader_id or '-'):>6} | {last}",
            )
