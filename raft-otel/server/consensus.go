package server

import (
	"context"

	pb "github.com/berkeli/raft-otel/service/consensus"
)

const (
	Leader    = "leader"
	Follower  = "follower"
	Candidate = "candidate"
)

type ConsensusServer struct {
	pb.UnimplementedConsensusServiceServer
	id    int
	peers map[int]string
	state string
}

func NewConsensusServer(id int) *ConsensusServer {
	peers := discoverPeers()
	return &ConsensusServer{
		state: Follower,
		id:    id,
		peers: peers,
	}
}

func (cs *ConsensusServer) RequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	return nil, nil
}

func (cs *ConsensusServer) AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	return nil, nil
}

func discoverPeers() map[int]string {
	return nil
}
