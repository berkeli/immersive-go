package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	CP "github.com/berkeli/raft-otel/service/consensus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	Leader    = "leader"
	Follower  = "follower"
	Candidate = "candidate"
)

const (
	ReqTimeout = 5 * time.Second
)

type ConsensusServer struct {
	sync.Mutex
	CP.UnimplementedConsensusServiceServer

	id int64

	state       string
	currentTerm int64
	votedFor    int64
	log         []*CP.Entry

	commitIndex int64 // index of highest log entry known to be committed (initialized to 0, increases monotonically)
	lastApplied int64 // index of highest log entry applied to state machine (initialized to 0, increases monotonically)

	nextIndex  map[int64]int64 // for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex map[int64]int64 // for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)

	peers map[int64]CP.ConsensusServiceClient

	lastHeartbeat time.Time
}

func NewConsensusServer() *ConsensusServer {

	idInt, err := strconv.Atoi(os.Getenv("ID"))

	if err != nil {
		log.Fatal(err)
	}

	id := int64(idInt)

	return &ConsensusServer{
		state: Follower,
		id:    id,
		peers: make(map[int64]CP.ConsensusServiceClient),
		log: []*CP.Entry{
			{Term: 0, Command: nil},
		}, // log is 1-indexed
		nextIndex:  make(map[int64]int64),
		matchIndex: make(map[int64]int64),
	}
}

// Receiver implementation:
// 1. Reply false if term < currentTerm (§5.1)
// 2. If votedFor is null or candidateId, and candidate’s log is at
// least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)

func (cs *ConsensusServer) RequestVote(ctx context.Context, req *CP.RequestVoteRequest) (*CP.RequestVoteResponse, error) {
	cs.Lock()
	defer cs.Unlock()

	if req.Term < cs.currentTerm {
		return &CP.RequestVoteResponse{
			Term:        cs.currentTerm,
			VoteGranted: false,
		}, status.Errorf(codes.FailedPrecondition, "Term is less than current term")
	}

	if (cs.votedFor == 0 || cs.votedFor == req.CandidateId) && req.LastLogIndex >= int64(len(cs.log)) {
		cs.votedFor = req.CandidateId
		return &CP.RequestVoteResponse{
			Term:        cs.currentTerm,
			VoteGranted: true,
		}, nil
	}

	return &CP.RequestVoteResponse{
		Term:        cs.currentTerm,
		VoteGranted: false,
	}, status.Errorf(codes.FailedPrecondition, "Already voted for %d", cs.votedFor)
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

	if cs.state != Follower {
		log.Println("Received AppendEntries from leader, changing state to follower")
		cs.state = Follower
	}

	if req.Term < cs.currentTerm {
		return &CP.AppendEntriesResponse{
			Term:    cs.currentTerm,
			Success: false,
		}, nil
	}

	cs.currentTerm = req.Term

	if len(cs.log) < int(req.PrevLogIndex) || cs.log[req.PrevLogIndex].Term != req.PrevLogTerm {
		return &CP.AppendEntriesResponse{
			Term:    cs.currentTerm,
			Success: false,
		}, nil
	}

	if len(cs.log) > int(req.PrevLogIndex) {
		cs.log = cs.log[:req.PrevLogIndex]
	}

	for _, entry := range req.Entries {
		cs.log = append(cs.log, entry)
	}

	if req.LeaderCommit > cs.commitIndex {
		cs.commitIndex = Min(req.LeaderCommit, int64(len(cs.log)))
	}

	return &CP.AppendEntriesResponse{
		Term:    cs.currentTerm,
		Success: true,
	}, nil
}

func (cs *ConsensusServer) autodiscovery() {

	ticker := time.NewTicker(3 * time.Second)

	go func() {
		for {
			<-ticker.C
			cs.Lock()

			i := int64(1)

			for i < 6 {
				myId := os.Getenv("ID")
				if myId != fmt.Sprintf("%d", i) && cs.peers[i] == nil {
					log.Println("Connecting to peer", i)
					cs.peers[i] = ConnectToPeer(i)
					cs.nextIndex[i] = cs.lastApplied + 1
					cs.matchIndex[i] = 0
				}
				i++
			}
			cs.Unlock()
		}
	}()
}

func (cs *ConsensusServer) BecomeLeader() error {
	cs.Lock()
	defer cs.Unlock()

	cs.state = Leader

	return nil
}

func (cs *ConsensusServer) BecomeFollower() error {
	cs.Lock()
	defer cs.Unlock()

	cs.state = Follower

	return nil
}

func (cs *ConsensusServer) BecomeCandidate() error {
	cs.Lock()
	defer cs.Unlock()

	cs.state = Candidate

	return nil
}

func (cs *ConsensusServer) Heartbeat() {
	cs.Lock()
	defer cs.Unlock()

	frequency := 50 * time.Millisecond

	ticker := time.NewTicker(frequency)

	go func() {
		for {
			<-ticker.C
			cs.Lock()

			if time.Since(cs.lastHeartbeat) < frequency {
				cs.Unlock()
				continue
			}

			if cs.state == Leader {
				for _, peer := range cs.peers {
					if peer == nil {
						continue
					}

				}
			}

			cs.Unlock()
		}
	}()
}

func (cs *ConsensusServer) appendEntriesRPC(id int64, n int) error {
	peer := cs.peers[id]

	if peer == nil {
		return nil
	}

	r, err := peer.AppendEntries(context.Background(), &CP.AppendEntriesRequest{
		Term:         cs.currentTerm,
		LeaderId:     cs.id,
		PrevLogIndex: int64(len(cs.log) - n - 1),
		PrevLogTerm:  cs.log[len(cs.log)-n-1].Term,
		Entries:      cs.log[len(cs.log)-n:],
		LeaderCommit: cs.commitIndex,
	})

	if err != nil {
		return err
	}

	if r.Term > cs.currentTerm {
		cs.currentTerm = r.Term
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
