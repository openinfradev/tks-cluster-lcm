package main

import (
	"context"
	"fmt"
	"errors"

	"github.com/google/uuid"
	"github.com/openinfradev/tks-contract/pkg/log"
	"github.com/openinfradev/tks-cluster-lcm/pkg/argowf"

	pb "github.com/openinfradev/tks-proto/pbgo"
	tksContractClient "github.com/openinfradev/tks-contract/pkg/client"
	tksInfoClient "github.com/openinfradev/tks-info/pkg/client"
)

var (
	argowfClient *argowf.Client

	contractClient pb.ContractServiceClient
	cspInfoClient pb.CspInfoServiceClient
	clusterInfoClient pb.ClusterInfoServiceClient
)

func InitHandlers( contractAddress string, contractPort int, infoAddress string, infoPort int, argoAddress string, argoPort int ) {
	{
		_client, err := argowf.New( argoAddress, argoPort, false, "" );
		if err != nil {
			log.Fatal( "failed to create argowf client : ", err )
		}
		argowfClient = _client;
	}

	{
		_contractClient, err := tksContractClient.GetContractClient(contractAddress, contractPort, "tks-cluster-lcm");
		if err != nil {
			log.Fatal( "failed to create contract client : ", err )
		}
		contractClient = _contractClient
	}

	{
		_cspInfoClient, err := tksInfoClient.GetCspInfoClient(infoAddress, infoPort, "tks-cluster-lcm");
		if err != nil {
			log.Fatal( "failed to create csp client : ", err )
		}
		cspInfoClient = _cspInfoClient
	}

	{
		_clusterInfoClient, err := tksInfoClient.GetClusterInfoClient(infoAddress, infoPort, "tks-cluster-lcm");
		if err != nil {
			log.Fatal( "failed to create cluster client : ", err )
		}
		clusterInfoClient = _clusterInfoClient
	}
}

func ValidateCreateClusterRequest(in *pb.CreateClusterRequest) (err error) {
	if _, err := uuid.Parse(in.GetContractId()); err != nil {
		log.Error( "Failed to validate contractId : ", err );
		return errors.New("ContractId must have value ")
	}
	if _, err := uuid.Parse(in.GetCspId()); err != nil {
		log.Error( "Failed to validate cspId : ", err );
		return errors.New("CspId must have value ")
	}
	if in.GetName() == "" {
		return errors.New("Name must have value ")
	}
	return nil
}

func (s *server) CreateCluster(ctx context.Context, in *pb.CreateClusterRequest) (*pb.IDResponse, error) {
	log.Info("Request 'CreateCluster' for contractId : ", in.GetContractId())

	// [TODO] validation refactoring
	if err := ValidateCreateClusterRequest(in); err != nil {
		if err != nil {
			return &pb.IDResponse {
				Code: pb.Code_INVALID_ARGUMENT,
				Error: &pb.Error{
					Msg: fmt.Sprint(err),
				},
			}, nil
		}
	}

	// check contract
	if _, err := contractClient.GetContract(ctx, &pb.GetContractRequest{ ContractId: in.GetContractId(), }); err != nil {
		if err != nil {
			log.Error( "Failed to get contract info err : ", err )
			return &pb.IDResponse {
				Code: pb.Code_NOT_FOUND,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Invalid contract Id %s", in.GetContractId()),
				},
			}, nil
		}
	}

	// check csp
	if _, err := cspInfoClient.GetCSPInfo(ctx, &pb.IDRequest{ Id: in.GetCspId() }); err != nil {
		if err != nil {
			log.Error( "Failed to get csp info err : ", err )
			return &pb.IDResponse{
				Code: pb.Code_NOT_FOUND,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Invalid CSP Id %s", in.GetCspId()),
				},
			}, nil
		}
	}

	// check workflow
	{
		nameSpace := "argo"
		if err := argowfClient.IsRunningWorkflowByContractId(nameSpace, in.GetContractId()); err != nil {
			log.Error(fmt.Sprintf("Already running workflow. contractId : %s", in.GetContractId()))
			return &pb.IDResponse{
				Code: pb.Code_INTERNAL,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Already running workflow. contractId : %s", in.GetContractId() ),
				},
			}, nil
		}
	}

	// create cluster info
	clusterId := ""
	{
		res, err := clusterInfoClient.AddClusterInfo(ctx, &pb.AddClusterInfoRequest{
			ContractId : in.GetContractId(),
			CspId : in.GetCspId(),
			Name : in.GetName(),
			Conf : in.GetConf(),
		})
		if err != nil {
			log.Error( "Failed to get csp info err : ", err )
			return &pb.IDResponse{
				Code: pb.Code_INTERNAL,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Invalid contract ID %s", in.GetContractId()),
				},
			}, nil
		}
		clusterId = res.Id
	}

	log.Info( "Add cluster to tks-info. clusterId : ", clusterId )


	// actually, create usercluster
	{
		workflow := "create-tks-usercluster"
		nameSpace := "argo"
		git_account := "tks-management"
		revision := "main"
		tks_admin := "tks-admin"
		app_name := "tks-cluster"

		opts := argowf.SubmitOptions{}
		opts.Parameters = []string{ 
//			"contract_id=" + in.GetContractId(), 
//			"site_name=" + clusterId,
			"contract_id=" + in.GetContractId(), 
			"cluster_id=" + clusterId,
			"git_account=" + git_account,
			"revision=" + revision,
			"tks_admin=" + tks_admin,
			"app_name=" + app_name,
		};

		res, err := argowfClient.SumbitWorkflowFromWftpl( workflow, nameSpace, opts );
		if err != nil {
			log.Error( "failed to submit argo workflow %s template. err : %s", workflow, err )
			return &pb.IDResponse{
				Code: pb.Code_INTERNAL,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Failed to call argo workflow : %s", err ),
				},
			}, nil
		}
		log.Debug("submited workflow template :", res)
	}

	// update status : INSTALLING
	{
		res, err := clusterInfoClient.UpdateClusterStatus(ctx, &pb.UpdateClusterStatusRequest{
  		ClusterId: clusterId,
			Status : pb.ClusterStatus_INSTALLING,
		})
		if err != nil {
			log.Error( "Failed to update cluster status err : ", err )
			return &pb.IDResponse{
	      Code: pb.Code_INTERNAL,
	      Error: &pb.Error{
	        Msg: fmt.Sprintf("Failed to update cluster status %s", err),
	      },
			}, nil
		}
		log.Debug("updated cluster status INSTALLING ", res)
	}

	log.Info("cluster successfully created clusterId : ", clusterId );
	return &pb.IDResponse{
		Code:  pb.Code_OK_UNSPECIFIED,
		Error: nil,
		Id: clusterId,
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
