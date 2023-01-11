package server

import (
	"bytes"
	"context"

	pb "github.com/berkeli/raft-otel/service/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StorageServer struct {
	pb.UnimplementedStoreServer
	s *MapStorage
	c *ConsensusServer
}

func NewStorageServer() *StorageServer {
	return &StorageServer{
		s: NewMapStorage(),
	}
}

func (ss *StorageServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {

	if ss.c.state != Leader {
		return nil, status.Error(codes.Unavailable, "not leader")
	}

	val, found := ss.s.Get(req.Key)

	if !found {
		return nil, status.Error(codes.NotFound, "key not found")
	}

	return &pb.GetResponse{
		Value: val,
	}, nil
}

func (ss *StorageServer) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	if ss.c.state != Leader {
		return nil, status.Error(codes.Unavailable, "not leader")
	}

	ok, err := ss.c.Commit(req.Key, req.Value)

	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, status.Error(codes.Unavailable, "commit failed")
	}

	ss.s.Set(req.Key, req.Value)

	return &pb.PutResponse{
		Ok: true,
	}, nil
}

func (ss *StorageServer) CompareAndSet(ctx context.Context, req *pb.CompareAndSetRequest) (*pb.CompareAndSetResponse, error) {
	if ss.c.state != Leader {
		return nil, status.Error(codes.Unavailable, "not leader")
	}

	prev, found := ss.s.Get(req.Key)

	if !found || bytes.Equal(prev, req.PrevValue) {
		ss.s.Set(req.Key, req.Value)
		return &pb.CompareAndSetResponse{
			Ok: true,
		}, nil
	}

	return &pb.CompareAndSetResponse{
		Ok: false,
	}, status.Error(codes.FailedPrecondition, "prev value does not match")
}
