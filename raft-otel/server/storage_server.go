package server

import (
	"bytes"
	"context"

	pb "github.com/berkeli/raft-otel/service/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StorageServer struct {
	pb.UnimplementedStoreServer
	s *MapStorage
	c *ConsensusServer
}

func NewStorageServer(m *MapStorage) *StorageServer {
	server := &StorageServer{
		s: m,
	}

	return server
}

func (ss *StorageServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	val, found := ss.s.Get(ctx, req.Key)

	if !found {
		return nil, status.Error(codes.NotFound, "key not found")
	}

	return &pb.GetResponse{
		Value: val,
	}, nil
}

func (ss *StorageServer) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	ok, err := ss.c.Commit(ctx, req.Key, req.Value)

	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, status.Error(codes.Unavailable, "commit failed")
	}

	ss.s.Set(ctx, req.Key, req.Value)

	return &pb.PutResponse{
		Ok: true,
	}, nil
}

func (ss *StorageServer) CompareAndSet(ctx context.Context, req *pb.CompareAndSetRequest) (*pb.CompareAndSetResponse, error) {

	prev, found := ss.s.Get(ctx, req.Key)

	if !found || bytes.Equal(prev, req.PrevValue) {
		ss.s.Set(ctx, req.Key, req.Value)
		return &pb.CompareAndSetResponse{
			Ok: true,
		}, nil
	}

	return &pb.CompareAndSetResponse{
		Ok: false,
	}, status.Error(codes.FailedPrecondition, "prev value does not match")
}

func (ss *StorageServer) LeaderCheckInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	switch info.Server.(type) {
	case *StorageServer:
		ss.c.Lock()
		if ss.c.state != Leader {
			if ss.c.leaderId == -1 {
				ss.c.Unlock()
				return nil, status.Error(codes.Unavailable, "leader has not been elected yet")
			}

			status := status.New(codes.Unavailable, "not leader, redirecting")

			detSt, err := status.WithDetails(
				&pb.NotLeaderResponse{
					LeaderId:   ss.c.leaderId,
					LeaderAddr: ss.c.peers[ss.c.leaderId].Addr,
				},
			)

			if err == nil {
				ss.c.Unlock()
				return nil, detSt.Err()
			}

			ss.c.Unlock()
			return nil, status.Err()
		}
		ss.c.Unlock()
	}

	return handler(ctx, req)
}
