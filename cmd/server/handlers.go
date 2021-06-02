package main

import (
	"context"

	"github.com/sktelecom/tks-contract/pkg/log"
	pb "github.com/sktelecom/tks-proto/pbgo"
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

// InstallApps install apps, return a array of application id
func (s *server) InstallApps(ctx context.Context, in *pb.InstallAppsRequest) (*pb.IDsResponse, error) {
	log.Debug("Request 'InstallApps' for cluster ID:", in.GetClusterId())
	log.Warn("Not Implemented gRPC API: 'InstallApps'")
	return &pb.IDsResponse{
		Code:  pb.Code_UNIMPLEMENTED,
		Error: nil,
	}, nil
}

// UninstallApps uninstall apps
func (s *server) UninstallApps(ctx context.Context, in *pb.UninstallAppsRequest) (*pb.SimpleResponse, error) {
	log.Debug("Request 'UninstallApps' for cluster ID:", in.GetClusterId())
	log.Warn("Not Implemented gRPC API: 'UninstallApps'")
	return &pb.SimpleResponse{
		Code:  pb.Code_UNIMPLEMENTED,
		Error: nil,
	}, nil
}
