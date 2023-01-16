package server

import CP "github.com/berkeli/raft-otel/service/consensus"

type State int

const (
	Follower State = iota
	Candidate
	Leader
)

func (s State) String() string {
	return [...]string{"Follower", "Candidate", "Leader"}[s]
}

type Status int

const (
	Online Status = iota
	Offline
)

func (s Status) String() string {
	return [...]string{"Online", "Offline"}[s]
}

type Peer struct {
	CP.ConsensusServiceClient
	Addr   string
	status Status
}
