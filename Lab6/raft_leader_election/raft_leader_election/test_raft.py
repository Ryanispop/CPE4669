"""Behavioral checks for election safety, timers, and recovery; no dependencies."""

import io
import unittest

from main import Demo
from raft import Cluster, Kind, MAJORITY, Message, NODE_COUNT, State


def advance(cluster, milliseconds):
    for _ in range(milliseconds):
        cluster.step()


def run_demo(scenario, seed=7, milliseconds=3000):
    cluster = Cluster(seed)
    demo = Demo(cluster, scenario)
    for _ in range(milliseconds):
        demo.step()
    return cluster, demo


class ProtocolTests(unittest.TestCase):
    def test_eight_followers_start_with_random_timeouts(self):
        cluster = Cluster()
        self.assertEqual(len(cluster.nodes), 8)
        self.assertEqual(MAJORITY, 5)
        for node in cluster.nodes.values():
            self.assertEqual(node.state, State.FOLLOWER)
            self.assertEqual(node.current_term, 0)
            self.assertIsNone(node.voted_for)
            self.assertIsNone(node.last_heartbeat)
            self.assertEqual(node.vote_count, 0)
            self.assertTrue(150 <= node.election_timeout <= 300)

    def test_candidate_self_votes_and_requests_all_seven_peers(self):
        cluster = Cluster()
        node = cluster.nodes[1]
        node.start_election()
        self.assertEqual(node.state, State.CANDIDATE)
        self.assertEqual(node.current_term, 1)
        self.assertEqual(node.voted_for, 1)
        self.assertEqual(node.votes, {1})
        requests = [m for _, _, m in cluster.messages if m.kind == Kind.REQUEST_VOTE]
        self.assertEqual({m.destination for m in requests}, set(range(2, 9)))

    def test_only_one_candidate_gets_a_nodes_vote_per_term(self):
        cluster = Cluster()
        node = cluster.nodes[1]
        node.receive(Message(Kind.REQUEST_VOTE, 1, 2, 1))
        deadline = node.election_deadline
        node.receive(Message(Kind.REQUEST_VOTE, 1, 3, 1))
        self.assertEqual(node.voted_for, 2)
        self.assertEqual(node.election_deadline, deadline)
        responses = [m for _, _, m in cluster.messages]
        self.assertTrue(responses[0].granted)
        self.assertFalse(responses[1].granted)
        node.receive(Message(Kind.REQUEST_VOTE, 2, 3, 1))
        self.assertEqual(node.current_term, 2)
        self.assertEqual(node.voted_for, 3)

    def test_duplicate_vote_request_is_idempotent(self):
        cluster = Cluster()
        message = Message(Kind.REQUEST_VOTE, 1, 2, 1)
        cluster.nodes[1].receive(message)
        cluster.nodes[1].receive(message)
        self.assertEqual(cluster.vote_history, [(1, 1, 2)])
        self.assertTrue(all(m.granted for _, _, m in cluster.messages))

    def test_five_distinct_votes_required_and_duplicates_do_not_count(self):
        cluster = Cluster()
        node = cluster.nodes[1]
        node.start_election()
        for source in (2, 2, 3, 4):
            node.receive(Message(Kind.VOTE_RESPONSE, 1, source, 1, True))
        self.assertEqual(node.vote_count, 4)
        self.assertEqual(node.state, State.CANDIDATE)
        node.receive(Message(Kind.VOTE_RESPONSE, 1, 5, 1, True))
        self.assertEqual(node.state, State.LEADER)
        self.assertEqual(node.vote_count, 5)

    def test_stale_requests_and_heartbeats_do_not_reset_timer(self):
        cluster = Cluster()
        node = cluster.nodes[1]
        node.receive(Message(Kind.REQUEST_VOTE, 3, 2, 1))
        deadline = node.election_deadline
        node.receive(Message(Kind.HEARTBEAT, 2, 3, 1))
        node.receive(Message(Kind.REQUEST_VOTE, 2, 4, 1))
        self.assertEqual(node.election_deadline, deadline)
        self.assertIsNone(node.last_heartbeat)
        self.assertEqual(node.current_term, 3)
        self.assertEqual(node.voted_for, 2)

    def test_same_term_heartbeat_steps_candidate_down_without_erasing_vote(self):
        cluster = Cluster()
        node = cluster.nodes[1]
        node.start_election()
        cluster.now = 100
        node.receive(Message(Kind.HEARTBEAT, 1, 2, 1))
        self.assertEqual(node.state, State.FOLLOWER)
        self.assertEqual(node.voted_for, 1)
        self.assertEqual(node.last_heartbeat, 100)
        self.assertEqual(node.leader_id, 2)
        self.assertTrue(250 <= node.election_deadline <= 400)
        node.receive(Message(Kind.REQUEST_VOTE, 1, 3, 1))
        self.assertEqual(node.voted_for, 1)

    def test_higher_term_in_every_message_kind_steps_leader_down(self):
        for kind in Kind:
            with self.subTest(kind=kind):
                cluster = Cluster()
                node = cluster.nodes[1]
                node.state = State.LEADER
                node.current_term = 2
                node.voted_for = 1
                node.receive(Message(kind, 3, 2, 1))
                self.assertEqual(node.current_term, 3)
                self.assertEqual(node.state, State.FOLLOWER)
                self.assertEqual(node.voted_for, 2 if kind == Kind.REQUEST_VOTE else None)

    def test_stale_vote_response_cannot_elect_candidate(self):
        cluster = Cluster()
        node = cluster.nodes[1]
        node.start_election()
        node.start_election()
        for source in range(2, 9):
            node.receive(Message(Kind.VOTE_RESPONSE, 1, source, 1, True))
        self.assertEqual(node.current_term, 2)
        self.assertEqual(node.state, State.CANDIDATE)
        self.assertEqual(node.vote_count, 1)

    def test_candidate_retries_in_new_term_after_randomized_timeout(self):
        cluster = Cluster()
        for node_id in range(2, 9):
            cluster.crash(node_id)
        node = cluster.nodes[1]
        advance(cluster, node.election_deadline)
        self.assertEqual(node.state, State.CANDIDATE)
        self.assertEqual(node.current_term, 1)
        retry = node.election_deadline
        advance(cluster, retry - cluster.now - 1)
        self.assertEqual(node.current_term, 1)
        cluster.step()
        self.assertEqual(node.current_term, 2)
        self.assertEqual(node.votes, {1})
        self.assertTrue(150 <= node.election_timeout <= 300)


class SimulationTests(unittest.TestCase):
    def assert_election_safety(self, cluster):
        votes = {}
        for term, voter, candidate in cluster.vote_history:
            key = (term, voter)
            if key in votes:
                self.assertEqual(votes[key], candidate, "node voted for two candidates in one term")
            votes[key] = candidate
        leaders = {}
        for election in cluster.elections:
            if election.term in leaders:
                self.assertEqual(leaders[election.term], election.node_id, "two leaders in one term")
            leaders[election.term] = election.node_id
            self.assertGreaterEqual(len(set(election.voters)), 5)
            for voter in election.voters:
                self.assertEqual(votes[(election.term, voter)], election.node_id)

    def test_heartbeats_keep_healthy_cluster_stable(self):
        cluster = Cluster()
        advance(cluster, 3000)
        self.assertEqual(len(cluster.elections), 1)
        leader = cluster.elections[0].node_id
        for node in cluster.nodes.values():
            if node.node_id != leader:
                self.assertEqual(node.state, State.FOLLOWER)
                self.assertEqual(node.leader_id, leader)
                self.assertLessEqual(cluster.now - node.last_heartbeat, 55)
                self.assertGreater(node.election_deadline, cluster.now)
        self.assert_election_safety(cluster)

    def test_four_four_partition_cannot_elect_a_leader(self):
        cluster = Cluster(partitioned=True)
        advance(cluster, 2000)
        self.assertEqual(cluster.elections, [])
        self.assertTrue(any(n.current_term > 1 for n in cluster.nodes.values()))
        self.assertTrue(all(n.vote_count <= 4 for n in cluster.nodes.values()))
        self.assert_election_safety(cluster)

    def test_split_vote_demo_retries_and_recovers(self):
        output = io.StringIO()
        cluster = Cluster(output=output)
        demo = Demo(cluster, "split")
        for _ in range(3000):
            demo.step()
        self.assertTrue(demo.split_observed)
        self.assertTrue(demo.succeeded())
        self.assertIn("RETRY", output.getvalue())
        self.assertGreater(cluster.elections[0].time_ms, demo.healed_at)
        self.assertGreater(cluster.elections[0].term, demo.split_term)
        self.assert_election_safety(cluster)

    def test_failure_demo_elects_different_leader_and_dead_node_stays_stopped(self):
        cluster, demo = run_demo("failure")
        self.assertTrue(demo.succeeded())
        dead = cluster.nodes[demo.failed_id]
        frozen = (dead.current_term, dead.voted_for, dead.vote_count, dead.last_heartbeat)
        history = list(cluster.vote_history)
        advance(cluster, 500)
        self.assertFalse(dead.alive)
        self.assertEqual(frozen, (dead.current_term, dead.voted_for, dead.vote_count, dead.last_heartbeat))
        self.assertFalse(any(voter == dead.node_id for _, voter, _ in cluster.vote_history[len(history):]))
        leader = cluster.elections[-1].node_id
        for node in cluster.nodes.values():
            if node.alive and node.node_id != leader:
                self.assertEqual(node.leader_id, leader)
                self.assertLessEqual(cluster.now - node.last_heartbeat, 55)
        self.assert_election_safety(cluster)

    def test_four_survivors_still_need_five_votes(self):
        cluster = Cluster()
        for node_id in range(5, 9):
            cluster.crash(node_id)
        advance(cluster, 2000)
        self.assertEqual(cluster.elections, [])
        self.assert_election_safety(cluster)

    def test_five_survivors_can_elect_replacement(self):
        cluster = Cluster()
        advance(cluster, 500)
        old_leader = cluster.elections[-1]
        victims = [old_leader.node_id] + [i for i in cluster.nodes if i != old_leader.node_id][:2]
        for node_id in victims:
            cluster.crash(node_id)
        advance(cluster, 2000)
        self.assertEqual(sum(n.alive for n in cluster.nodes.values()), 5)
        self.assertGreater(cluster.elections[-1].term, old_leader.term)
        self.assertNotIn(cluster.elections[-1].node_id, victims)
        self.assert_election_safety(cluster)

    def test_seed_reproduces_message_order_and_results(self):
        traces = []
        for _ in range(2):
            output = io.StringIO()
            cluster = Cluster(7, output=output)
            demo = Demo(cluster, "failure")
            for _ in range(2000):
                demo.step()
            traces.append(output.getvalue())
        self.assertEqual(traces[0], traces[1])

    def test_recovery_and_election_safety_across_seeds(self):
        for scenario in ("failure", "split"):
            for seed in range(25):
                with self.subTest(scenario=scenario, seed=seed):
                    cluster, demo = run_demo(scenario, seed)
                    self.assertTrue(demo.succeeded())
                    self.assert_election_safety(cluster)
                    self.assertEqual(len(cluster.nodes), NODE_COUNT)


if __name__ == "__main__":
    unittest.main()
