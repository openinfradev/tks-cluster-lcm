package main

import (
	"context"

	"github.com/openinfradev/tks-contract/pkg/log"
	pb "github.com/openinfradev/tks-proto/pbgo"
)

// CreateCluster creates a Kubernetes cluster and returns cluster id
func (s *server) CreateCluster(ctx context.Context, in *pb.CreateClusterRequest) (*pb.IDsResponse, error) {
	log.Debug("Request 'CreateContract' for contractID", in.GetContractId())
	log.Warn("Not Implemented gRPC API: 'CreateCluster'")
	return &pb.IDsResponse{
		Code:  pb.Code_UNIMPLEMENTED,
		Error: nil,
	}, nil
}

// ScaleCluster scales the Kubernetes cluster
func (s *server) ScaleCluster(ctx context.Context, in *pb.ScaleClusterRequest) (*pb.SimpleResponse, error) {
	log.Debug("Request 'ScaleCluster' for cluster ID:", in.GetClusterId())
	log.Warn("Not Implemented gRPC API: 'ScaleCluster'")
	return &pb.SimpleResponse{
		Code:  pb.Code_UNIMPLEMENTED,
		Error: nil,
	}, nil
}

// InstallAppGroups install apps, return a array of application id
func (s *server) InstallAppGroups(ctx context.Context, in *pb.InstallAppGroupsRequest) (*pb.IDsResponse, error) {
	log.Debug("Request 'InstallAppGroups' ")
	log.Warn("Not Implemented gRPC API: 'InstallAppGroups'")
	return &pb.IDsResponse{
		Code:  pb.Code_UNIMPLEMENTED,
		Error: nil,
	}, nil
}

// UninstallAppGroups uninstall apps
func (s *server) UninstallAppGroups(ctx context.Context, in *pb.UninstallAppGroupsRequest) (*pb.SimpleResponse, error) {
	log.Debug("Request 'UninstallAppGroups'")
	log.Warn("Not Implemented gRPC API: 'UninstallAppGroups'")
	return &pb.SimpleResponse{
		Code:  pb.Code_UNIMPLEMENTED,
		Error: nil,
	}, nil
}
