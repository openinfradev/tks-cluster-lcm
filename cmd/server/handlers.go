package main

import (
	"context"
	"fmt"
	"errors"

	"github.com/google/uuid"
	"github.com/openinfradev/tks-contract/pkg/log"
	"github.com/openinfradev/tks-cluster-lcm/pkg/argowf"

	pb "github.com/openinfradev/tks-proto/tks_pb"
	tksContractClient "github.com/openinfradev/tks-contract/pkg/client"
	tksInfoClient "github.com/openinfradev/tks-info/pkg/client"
)

var (
	argowfClient *argowf.Client

	contractClient pb.ContractServiceClient
	cspInfoClient pb.CspInfoServiceClient
	clusterInfoClient pb.ClusterInfoServiceClient
	appInfoClient pb.AppInfoServiceClient
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
	{
		_appInfoClient, err := tksInfoClient.GetAppInfoClient(infoAddress, infoPort, "tks-cluster-lcm");
		if err != nil {
			log.Fatal( "failed to create appinfo client : ", err )
		}
		appInfoClient = _appInfoClient
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

func ValidateInstallAppGroupsRequest(in *pb.InstallAppGroupsRequest) (err error) {
	return nil
}

func (s *server) CreateCluster(ctx context.Context, in *pb.CreateClusterRequest) (*pb.IDResponse, error) {
	log.Info("Request 'CreateCluster' for contractId : ", in.GetContractId())

	// [TODO] validation refactoring
	if err := ValidateCreateClusterRequest(in); err != nil {
		return &pb.IDResponse {
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprint(err),
			},
		}, nil
	}

	// check contract
	if _, err := contractClient.GetContract(ctx, &pb.GetContractRequest{ ContractId: in.GetContractId(), }); err != nil {
		log.Error( "Failed to get contract info err : ", err )
		return &pb.IDResponse {
			Code: pb.Code_NOT_FOUND,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Invalid contract Id %s", in.GetContractId()),
			},
		}, nil
	}

	// check csp
	if _, err := cspInfoClient.GetCSPInfo(ctx, &pb.IDRequest{ Id: in.GetCspId() }); err != nil {
		log.Error( "Failed to get csp info err : ", err )
		return &pb.IDResponse{
			Code: pb.Code_NOT_FOUND,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Invalid CSP Id %s", in.GetCspId()),
			},
		}, nil
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

	// [TODO] validation refactoring
	if err := ValidateInstallAppGroupsRequest(in); err != nil {
		return &pb.IDsResponse {
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprint(err),
			},
		}, nil
	}

	completed := false;
	for _, appGroup := range in.GetAppGroups() {
		log.Debug( "appGroup : ", appGroup )

		clusterId := appGroup.GetClusterId()
		contractId := ""
		
		// Check Cluster
		{
			cluster, err := clusterInfoClient.GetCluster(ctx, &pb.GetClusterRequest{ ClusterId: clusterId, })
			if err != nil {
				log.Error( "Failed to get cluster info err : ", err )
				continue;
			}
			if cluster == nil {
				log.Error( "Failed to get cluster info : ", appGroup.GetClusterId() )
				continue;
			}
			log.Debug( "cluster : ", cluster )
			contractId = cluster.GetCluster().GetContractId()
		}
		log.Debug( "contractId ", contractId )
		// clusterId : 771eede9-794e-427d-8183-999e96ea789f

		// Check AppGroup
		{
			_appGroup, err := appInfoClient.GetAppGroupsByClusterID(ctx, &pb.IDRequest{ Id: appGroup.GetClusterId(), })
			if err != nil {
				log.Error( "Failed to get appgroup info err : ", err )
				continue;
			}
			log.Debug("_appGroup ", _appGroup)
			if _appGroup != nil && len(_appGroup.GetAppGroups()) > 0 {
				log.Error( "appgroup already existed : ", appGroup.GetClusterId() )
				continue;
			}
		}

		// Create AppGoup
		appGroupId := ""
		{
			res, err := appInfoClient.CreateAppGroup(ctx, &pb.CreateAppGroupRequest{
				ClusterId: appGroup.GetClusterId(), 
				AppGroup: appGroup,
			})
			if err != nil {
				log.Error( "Failed to create app group info err : ", err )
				continue;
			}
			appGroupId = res.GetId()
			log.Debug( "appGroupId : ", appGroupId )
		}


		// Call argo workflow template
		{
			log.Debug( "appGroup.GetType() : ", appGroup.GetType() )
			workflow := ""
			opts := argowf.SubmitOptions{}
			switch appGroup.GetType() {
				case pb.AppGroupType_LMA :
					/*
					argo -n argo submit --from wftmpl/tks-lma-federation 
					-p site_name=6f1d121b-c979-4164-b8bf-cf83c367f423 
					-p site_repo_url=https://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@github.com/tks-management/8fcfe745-dee2-4d53-89a6-144cf17a68ab 
					-p manifest_repo_url=https://github.com/tks-management/8fcfe745-dee2-4d53-89a6-144cf17a68ab-manifests 
					-p cluster_id=6f1d121b-c979-4164-b8bf-cf83c367f423  
					-p app_group_id=2d390903-bc3b-40d1-8701-d63c6c2a862f 
					-p tks_info_host=a9024398d250c4b6c8630d1aa997917d-1399854961.ap-northeast-2.elb.amazonaws.com
					*/
					workflow = "tks-lma-federation"
					gitToken := "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"
					siteRepoUrl := "https://" + gitToken + "@github.com/tks-management/" + contractId
					manifestRepoUrl := "https://github.com/tks-management/" + contractId + "-manifests"
					tksInfoHost := "tks-info.tks.svc"
					opts.Parameters = []string{ 
						"site_name=" + clusterId, 
						"site_repo_url=" + siteRepoUrl,
						"manifest_repo_url=" + manifestRepoUrl,
						"cluster_id=" + clusterId,
						"app_group_id=" + appGroupId,
						"tks_info_host=" + tksInfoHost,
					};


				case pb.AppGroupType_SERVICE_MESH : 
					workflow = "tks-service-mesh"
					/*
					argo -n argo submit --from wftmpl/tks-service-mesh 
					-p site_name=6f1d121b-c979-4164-b8bf-cf83c367f423 
					-p app_name=service-mesh 
					-p manifest_repo_url=https://github.com/tks-management/8fcfe745-dee2-4d53-89a6-144cf17a68ab-manifests 
					-p revision=main
					*/
					workflow = "tks-service-mesh"
					manifestRepoUrl := "https://github.com/tks-management/" + contractId + "-manifests"
					revision := "main"
					opts.Parameters = []string{ 
						"site_name=" + clusterId, 
						"app_name=" + "service-mesh", 
						"manifest_repo_url=" + manifestRepoUrl,
						"revision=" + revision,
					};

				default :
					log.Error( "invalid appGroup type ", appGroup.GetType() )
					continue
			}
			log.Debug( "workflow : ", workflow )

			res, err := argowfClient.SumbitWorkflowFromWftpl( workflow, "argo", opts );
			if err != nil {
				log.Error( "failed to submit argo workflow %s template. err : %s", workflow, err )
				return &pb.IDsResponse{
					Code: pb.Code_INTERNAL,
					Error: &pb.Error{
						Msg: fmt.Sprintf("Failed to call argo workflow : %s", err ),
					},
				}, nil
			}
			log.Debug("submited workflow template :", res)
		}

	}

	log.Info("completed : ", completed)

/*
	// Check Cluster
	if cluster, err := clusterInfoClient.GetCluster(ctx, &pb.GetClusterRequest{ ClusterId: in.GetClusterId(), }); err != nil {
		log.Error( "Failed to get cluster info err : ", err )
		return &pb.IDResponse {
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Invalid cluster Id %s", in.GetClusterId()),
			},
		}, nil
	}

/*
	// rpc GetAppGroups(GetAppGroupsRequest) returns (GetAppGroupsResponse) {}
	if _, err := appInfoClient.GetAppGroups(ctx, &pb.GetAppGroupsResponse{ ClusterId: in.GetClusterId(), }); err != nil {
		log.Error( "Failed to get cluster info err : ", err )
		return &pb.IDResponse {
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Invalid cluster Id %s", in.GetClusterId()),
			},
		}, nil
	}
*/


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
