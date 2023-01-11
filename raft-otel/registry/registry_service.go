package registry

import (
	"context"
	"time"

	RP "github.com/berkeli/raft-otel/service/registry"
)

const (
	Timeout = 10 * time.Second
)

type Node struct {
	Address     string
	LastCheckin time.Time
}

type RegistryService struct {
	RP.UnimplementedRegistryServer
	nodes map[int64]*Node
}

func NewRegistryService() *RegistryService {
	return &RegistryService{
		nodes: make(map[int64]*Node),
	}
}

func (r *RegistryService) List(ctx context.Context, req *RP.ListRequest) (*RP.ListResponse, error) {
	var nodes []*RP.Node

	for id, node := range r.nodes {

		if time.Since(node.LastCheckin) > Timeout {
			delete(r.nodes, id)
			continue
		}

		if req.Id != id {
			nodes = append(nodes, &RP.Node{
				Id:      id,
				Address: node.Address,
			})
		}
	}

	return &RP.ListResponse{
		Nodes: nodes,
	}, nil
}

func (r *RegistryService) Heartbeat(ctx context.Context, req *RP.HeartbeatRequest) (*RP.HeartbeatResponse, error) {
	node, ok := r.nodes[req.Id]

	if !ok {
		r.nodes[req.Id] = &Node{
			Address:     req.Address,
			LastCheckin: time.Now(),
		}
	} else {
		node.LastCheckin = time.Now()
	}

	return &RP.HeartbeatResponse{
		Ok: true,
	}, nil
}
