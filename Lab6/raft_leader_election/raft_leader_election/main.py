"""Command-line demonstrations. Run: python main.py --help."""

from __future__ import annotations

import argparse
import math
import sys
import time

from raft import Cluster, State


class Demo:
    """The harness introduces failures/partitions; it never assigns votes."""

    def __init__(self, cluster: Cluster, scenario: str, fail_after_ms: int = 1000):
        self.cluster = cluster
        self.scenario = scenario
        self.fail_after_ms = fail_after_ms
        self.failed_id: int | None = None
        self.failed_term: int | None = None
        self.failed_at: int | None = None
        self.split_observed = False
        self.split_term: int | None = None
        self.healed_at: int | None = None
        if scenario == "split":
            cluster.partitioned = True
            cluster.log("NETWORK", "temporary partition: nodes [1,2,3,4] | [5,6,7,8]")

    def step(self) -> None:
        c = self.cluster
        c.step()
        if self.scenario == "failure" and c.elections and self.failed_id is None:
            first = c.elections[0]
            if c.now >= first.time_ms + self.fail_after_ms:
                leaders = [n for n in c.nodes.values() if n.alive and n.state == State.LEADER]
                if leaders:
                    leader = leaders[0]
                    self.failed_id = leader.node_id
                    self.failed_term = leader.current_term
                    self.failed_at = c.now
                    c.crash(leader.node_id)
                    c.show_status()
        if self.scenario == "split" and not self.split_observed:
            candidates = [n for n in c.nodes.values()
                          if n.state == State.CANDIDATE and n.vote_count == 4]
            if (len(candidates) == 2
                    and (candidates[0].node_id <= 4) != (candidates[1].node_id <= 4)
                    and candidates[0].current_term == candidates[1].current_term):
                a, b = candidates
                self.split_observed = True
                self.split_term = a.current_term
                c.log("SPLIT", f"term={a.current_term}: node={a.node_id} has 4/8; "
                      f"node={b.node_id} has 4/8; NO MAJORITY, NO LEADER")
                c.show_status()
                c.partitioned = False
                self.healed_at = c.now
                c.log("NETWORK", "partition healed; candidates keep waiting for their randomized timeouts")

    def succeeded(self) -> bool:
        c = self.cluster
        if self.scenario == "failure":
            return self.failed_at is not None and any(
                e.time_ms > self.failed_at and e.node_id != self.failed_id and e.term > self.failed_term
                for e in c.elections
            )
        if self.scenario == "split":
            return self.split_observed and any(
                e.time_ms > self.healed_at and e.term > self.split_term for e in c.elections
            )
        return bool(c.elections)


def positive_seconds(value: str) -> float:
    try:
        number = float(value)
    except ValueError as exc:
        raise argparse.ArgumentTypeError("use a positive number of seconds") from exc
    if not math.isfinite(number) or number < 0.001:
        raise argparse.ArgumentTypeError("use a finite duration of at least 0.001 seconds")
    return number


def main() -> int:
    parser = argparse.ArgumentParser(description="Eight-node simplified Raft leader election")
    parser.add_argument("--scenario", choices=("failure", "split", "election"), default="failure",
                        help="failure (default), repeatable split vote, or healthy election")
    parser.add_argument("--seed", type=int, default=7, help="random seed; default 7")
    parser.add_argument("--duration", type=positive_seconds, default=3.0,
                        help="total simulated seconds; default 3")
    parser.add_argument("--fail-after", type=positive_seconds, default=1.0,
                        help="in failure mode, seconds after first election to crash leader; default 1")
    parser.add_argument("--fast", action="store_true", help="skip wall-clock waits; simulated timing is unchanged")
    parser.add_argument("--verbose-heartbeats", action="store_true", help="log every heartbeat timer reset")
    args = parser.parse_args()

    print(f"Raft leader election | 8 nodes | majority 5 | scenario={args.scenario} | seed={args.seed}")
    print("Election timeout: random 150-300ms | heartbeat: 50ms | one-way message delay: 5ms")
    print("Timestamps use simulated time. " + ("Fast mode: no wall-clock waits." if args.fast
                                             else "Pacing: about one real second per simulated second."))
    cluster = Cluster(args.seed, output=sys.stdout, verbose_heartbeats=args.verbose_heartbeats)
    demo = Demo(cluster, args.scenario, round(args.fail_after * 1000))
    cluster.show_status()
    wall_start = time.monotonic()
    try:
        for _ in range(round(args.duration * 1000)):
            # Sleep before the step so its messages appear near their timestamps.
            if not args.fast:
                delay = wall_start + (cluster.now + 1) / 1000 - time.monotonic()
                if delay > 0:
                    time.sleep(delay)
            demo.step()
            if cluster.now % 500 == 0:
                cluster.show_status()
    except KeyboardInterrupt:
        print("\nStopped by user.")
        return 130

    if cluster.now % 500:
        cluster.show_status()
    for election in cluster.elections:
        cluster.log("SUMMARY", f"leader={election.node_id} term={election.term} "
                    f"elected_at={election.time_ms}ms voters={list(election.voters)}")
    if demo.succeeded():
        messages = {
            "failure": "initial leader elected, leader crashed, and a different leader elected in a higher term",
            "split": "4/4 split observed, network healed, and a leader elected in a later term",
            "election": "leader elected and heartbeats demonstrated",
        }
        cluster.log("PASS", messages[args.scenario])
        return 0
    cluster.log("INCOMPLETE", "scenario did not finish; try --duration 6 (and --seed 7 for the split demo)")
    return 1


if __name__ == "__main__":
    sys.exit(main())
