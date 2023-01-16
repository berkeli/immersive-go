package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"sync"
	"time"

	CP "github.com/berkeli/raft-otel/service/consensus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ReqTimeout = 5 * time.Second
)

type ConsensusServer struct {
	sync.Mutex
	CP.UnimplementedConsensusServiceServer

	store *MapStorage

	id       int64
	leaderId int64

	state       State
	currentTerm int64
	votedFor    int64
	log         []*CP.Entry

	commitIndex int64 // index of highest log entry known to be committed (initialized to 0, increases monotonically)
	lastApplied int64 // index of highest log entry applied to state machine (initialized to 0, increases monotonically)

	nextIndex  map[int64]int64 // for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex map[int64]int64 // for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)

	peers map[int64]*Peer

	lastHeartbeat time.Time
}

func NewConsensusServer(s *MapStorage) *ConsensusServer {

	id, err := strconv.Atoi(os.Getenv("ID"))

	if err != nil {
		log.Fatal(err)
	}

	cs := &ConsensusServer{
		log: []*CP.Entry{
			{},
		},
		state:         Follower,
		id:            int64(id),
		peers:         make(map[int64]*Peer),
		nextIndex:     make(map[int64]int64),
		matchIndex:    make(map[int64]int64),
		lastApplied:   -1,
		commitIndex:   -1,
		votedFor:      -1,
		lastHeartbeat: time.Now(),
		store:         s,
	}

	go cs.electionTimer()
	go cs.stateReport()

	return cs
}

// Receiver implementation:
// 1. Reply false if term < currentTerm (§5.1)
// 2. If votedFor is null or candidateId, and candidate’s log is at
// least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)

func (cs *ConsensusServer) RequestVote(ctx context.Context, req *CP.RequestVoteRequest) (*CP.RequestVoteResponse, error) {
	cs.Lock()
	defer cs.Unlock()

	if req.Term > cs.currentTerm {
		cs.currentTerm = req.Term
		log.Println("Received RequestVote from candidate", req.CandidateId, "with term", req.Term, "changing state to follower")
		cs.BecomeFollower()
	}

	if (cs.votedFor == -1 || cs.votedFor == req.CandidateId) && req.Term == cs.currentTerm {
		cs.votedFor = req.CandidateId
		cs.lastHeartbeat = time.Now()

		return &CP.RequestVoteResponse{
			Term:        cs.currentTerm,
			VoteGranted: true,
		}, nil
	} else {
		return &CP.RequestVoteResponse{
			Term:        cs.currentTerm,
			VoteGranted: false,
		}, status.Errorf(codes.FailedPrecondition, "Already voted for %d", cs.votedFor)
	}
}

// Receiver implementation:
// 1. Reply false if term < currentTerm (§5.1)
// 2. Reply false if log doesn’t contain an entry at prevLogIndex
// whose term matches prevLogTerm (§5.3)
// 3. If an existing entry conflicts with a new one (same index
// but different terms), delete the existing entry and all that
// follow it (§5.3)
// 4. Append any new entries not already in the log
// 5. If leaderCommit > commitIndex, set commitIndex =
// min(leaderCommit, index of last new entry)

func (cs *ConsensusServer) AppendEntries(ctx context.Context, req *CP.AppendEntriesRequest) (*CP.AppendEntriesResponse, error) {
	cs.Lock()
	defer cs.Unlock()
	if req.Term < cs.currentTerm {
		return &CP.AppendEntriesResponse{
			Term:    cs.currentTerm,
			Success: false,
		}, status.Errorf(codes.FailedPrecondition, "Term %d is less than current term %d", req.Term, cs.currentTerm)
	}

	if req.Term > cs.currentTerm {
		log.Println("Received AppendEntries from leader, changing state to follower")
		cs.leaderId = req.LeaderId
		cs.currentTerm = req.Term
		cs.BecomeFollower()
	}

	cs.lastHeartbeat = time.Now()
	cs.leaderId = req.LeaderId

	if req.PrevLogIndex != -1 && req.PrevLogIndex < int64(len(cs.log)) && cs.log[req.PrevLogIndex].Term != req.PrevLogTerm {
		return &CP.AppendEntriesResponse{
			Term:    cs.currentTerm,
			Success: false,
		}, status.Errorf(codes.FailedPrecondition, "Log at index %d has term %d, expected %d", req.PrevLogIndex, cs.log[req.PrevLogIndex].Term, req.PrevLogTerm)
	}

	if len(cs.log) < int(req.PrevLogIndex) {
		return &CP.AppendEntriesResponse{
			Term:    cs.currentTerm,
			Success: false,
		}, status.Errorf(codes.FailedPrecondition, "Log is too short, expected at least %d entries", req.PrevLogIndex)
	}

	// Find an insertion point - where there's a term mismatch between
	// the existing log starting at PrevLogIndex+1 and the new entries sent
	// in the RPC.
	logInsertIndex := req.PrevLogIndex + 1
	newEntriesIndex := 0

	for {
		if int(logInsertIndex) >= len(cs.log) || int(newEntriesIndex) >= len(req.Entries) {
			break
		}
		if cs.log[logInsertIndex].Term != req.Entries[newEntriesIndex].Term {
			break
		}
		logInsertIndex++
		newEntriesIndex++
	}

	if newEntriesIndex < len(req.Entries) {
		new := req.Entries[newEntriesIndex:]
		cs.log = append(cs.log[:logInsertIndex], new...)
		cs.persist(new)
	}

	// Set commit index.
	if req.LeaderCommit > cs.commitIndex {
		cs.commitIndex = Min(req.LeaderCommit, int64(len(cs.log)-1))
	}

	return &CP.AppendEntriesResponse{
		Term:    cs.currentTerm,
		Success: true,
	}, nil
}

func (cs *ConsensusServer) BecomeLeader() error {
	log.Println("Becoming leader")
	cs.state = Leader
	cs.leaderId = cs.id

	for id := range cs.peers {
		cs.nextIndex[id] = int64(len(cs.log))
		cs.matchIndex[id] = -1
	}

	go cs.Heartbeat()

	return nil
}

func (cs *ConsensusServer) BecomeFollower() error {

	log.Println("Becoming follower")

	cs.state = Follower
	cs.votedFor = -1
	cs.lastHeartbeat = time.Now()

	return nil
}

func (cs *ConsensusServer) BecomeCandidate() error {

	if len(cs.peers) == 0 {
		return nil
	}

	cs.state = Candidate
	cs.currentTerm++
	savedCurrTerm := cs.currentTerm
	cs.votedFor = cs.id

	votes := 1
	chVotes := cs.requestVotesRPC(savedCurrTerm)
	voted := 0
	log.Println("Becoming candidate and requesting votes with term: ", cs.currentTerm)

	for {
		votes += <-chVotes

		if savedCurrTerm != cs.currentTerm {
			return nil
		}

		voted++

		log.Println("Votes: ", votes, "Voted: ", voted)

		if votes > len(cs.peers)/2 {
			cs.BecomeLeader()
			return nil
		}

		if voted == len(cs.peers) {
			fmt.Println("No majority, starting new election")
			cs.BecomeFollower()
			return nil
		}
	}
}

func (cs *ConsensusServer) stateReport() {
	ticker := time.NewTicker(5 * time.Second)

	defer ticker.Stop()

	for {
		<-ticker.C
		cs.Lock()
		log.Println(
			"State:", cs.state,
			"Term:", cs.currentTerm,
			"Leader:", cs.leaderId,
			"lastHeartbeat:", time.Since(cs.lastHeartbeat),
			"log len:", len(cs.log),
		)
		cs.Unlock()
	}
}

func (cs *ConsensusServer) Heartbeat() {
	frequency := 10 * time.Millisecond

	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		<-ticker.C
		cs.Lock()

		if cs.state != Leader {
			cs.Unlock()
			return
		}

		if cs.state == Leader {
			count := 0
			for id, peer := range cs.peers {
				if peer == nil {
					cs.Unlock()
					continue
				}
				err := cs.appendEntriesRPC(id, 0)

				if err != nil {
					log.Println("Error while sending heartbeat to peer: ", err)
				} else {
					count++
				}
			}
		}

		cs.Unlock()
	}

}

func (cs *ConsensusServer) electionTimer() {
	r, _ := rand.Int(rand.Reader, big.NewInt(150))
	timeout := time.Duration(time.Duration((150 + r.Int64())) * time.Millisecond)
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	cs.Lock()
	termStarted := cs.currentTerm
	cs.Unlock()
	for {
		<-ticker.C
		cs.Lock()

		if cs.state == Leader {
			log.Println("Already leader, skipping election")
			cs.Unlock()
			return
		}

		if time.Since(cs.lastHeartbeat) < timeout {
			// log.Println("Received heartbeat, skipping election")
			cs.Unlock()
			break
		}
		if termStarted != cs.currentTerm {
			log.Println("Term changed, skipping election")
			cs.Unlock()
			return
		}

		if len(cs.peers) < 3 {
			log.Println("Not enough peers, skipping election")
			cs.Unlock()
			break
		}

		if time.Since(cs.lastHeartbeat) > timeout && cs.state == Follower {
			log.Println("Starting election, timeout:", timeout, "last heartbeat:", time.Since(cs.lastHeartbeat))
			cs.BecomeCandidate()
			cs.Unlock()
			return
		}
	}

	go cs.electionTimer()
}

func (cs *ConsensusServer) requestVotesRPC(term int64) chan int {
	chVotes := make(chan int, len(cs.peers))
	log.Printf("Requesting votes from %d peers", len(cs.peers))
	for i, peer := range cs.peers {
		if peer == nil {
			continue
		}

		go func(id int64, peer CP.ConsensusServiceClient) {
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()
			r, err := peer.RequestVote(ctx, &CP.RequestVoteRequest{
				Term:        term,
				CandidateId: cs.id,
			})

			if err != nil {
				log.Println("Error while requesting vote from peer: ", err)
				chVotes <- 0
				return
			}

			if r.Term > cs.currentTerm {
				cs.currentTerm = r.Term
				cs.BecomeFollower()
				chVotes <- 0
				return
			}

			if r.VoteGranted {
				chVotes <- 1
				return
			}

			chVotes <- 0
		}(i, peer)
	}

	return chVotes
}

func (cs *ConsensusServer) appendEntriesRPC(id int64, n int) error {
	peer := cs.peers[id]

	ni := cs.nextIndex[id] - int64(n)

	prevLogTerm := int64(-1)

	if ni > 0 {
		prevLogTerm = cs.log[ni-1].Term
	}

	r, err := peer.AppendEntries(context.Background(), &CP.AppendEntriesRequest{
		Term:         cs.currentTerm,
		LeaderId:     cs.id,
		PrevLogIndex: ni - 1,
		PrevLogTerm:  prevLogTerm,
		Entries:      cs.log[ni:],
		LeaderCommit: cs.commitIndex,
	})

	if err != nil {
		return err
	}

	if r.Term > cs.currentTerm {
		cs.currentTerm = r.Term
		cs.leaderId = id
		cs.BecomeFollower()
	}

	return nil
}

func (cs *ConsensusServer) AppendOrDie(id int64, done chan struct{}) {
	n := 0

	for {
		err := cs.appendEntriesRPC(id, n)

		if err == nil {
			break
		}

		if n > len(cs.log) {
			log.Println("Failed to append to peer", id)
			break
		}

		n++
	}

	done <- struct{}{}
}

func (cs *ConsensusServer) Commit(key string, data []byte) (ok bool, err error) {
	cs.Lock()
	defer cs.Unlock()

	entry := &CP.Entry{
		Term: cs.currentTerm,
		Command: &CP.Command{
			Op:   "put",
			Key:  key,
			Data: data,
		},
	}

	cs.log = append(cs.log, entry)
	confirmations := 1 // 1 for self

	confChan := make(chan struct{}, len(cs.peers))

	start := time.Now()

	for id, peer := range cs.peers {
		if peer == nil {
			continue
		}

		go cs.AppendOrDie(id, confChan)
	}

	for {
		select {
		case <-confChan:
			confirmations++
		case <-time.After(ReqTimeout):
			return false, fmt.Errorf("timeout")
		}

		if confirmations > len(cs.peers)/2 {
			return true, nil
		}

		if time.Since(start) > ReqTimeout {
			return false, fmt.Errorf("timeout")
		}
	}
}

func (cs *ConsensusServer) Dummy(ctx context.Context, req *CP.DummyRequest) (*CP.DummyResponse, error) {
	cs.BecomeCandidate()
	return &CP.DummyResponse{}, nil
}

func (cs *ConsensusServer) persist(entries []*CP.Entry) {
	for _, entry := range entries {
		log.Println("Persisting", entry)
		cs.store.Set(entry.Command.Key, entry.Command.Data)
	}
}
