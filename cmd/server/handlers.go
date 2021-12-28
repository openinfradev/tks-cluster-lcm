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
	_client, err := argowf.New( argoAddress, argoPort );
	if err != nil {
		log.Fatal( "failed to create argowf client : ", err )
	}
	argowfClient = _client;

	_contractClient, err := tksContractClient.GetContractClient(contractAddress, contractPort, "tks-cluster-lcm");
	if err != nil {
		log.Fatal( "failed to create contract client : ", err )
	}
	contractClient = _contractClient

	_cspInfoClient, err := tksInfoClient.GetCspInfoClient(infoAddress, infoPort, "tks-cluster-lcm");
	if err != nil {
		log.Fatal( "failed to create csp client : ", err )
	}
	cspInfoClient = _cspInfoClient

	_clusterInfoClient, err := tksInfoClient.GetClusterInfoClient(infoAddress, infoPort, "tks-cluster-lcm");
	if err != nil {
		log.Fatal( "failed to create cluster client : ", err )
	}
	clusterInfoClient = _clusterInfoClient

	_appInfoClient, err := tksInfoClient.GetAppInfoClient(infoAddress, infoPort, "tks-cluster-lcm");
	if err != nil {
		log.Fatal( "failed to create appinfo client : ", err )
	}
	appInfoClient = _appInfoClient
}

func ValidateCreateClusterRequest(in *pb.CreateClusterRequest) (err error) {
	if _, err := uuid.Parse(in.GetContractId()); err != nil {
		return fmt.Errorf("invalid contract ID %s", in.GetContractId())
	}
	if _, err := uuid.Parse(in.GetCspId()); err != nil {
		return fmt.Errorf("invalid CSP ID %s", in.GetCspId())  
	}
	if in.GetName() == "" {
		return errors.New("Name must have value ")
	}
	return nil
}

func ValidateInstallAppGroupsRequest(in *pb.InstallAppGroupsRequest) (err error) {
	for _, appGroup := range in.GetAppGroups() {
		if _, err := uuid.Parse(appGroup.GetClusterId()); err != nil {
			log.Error( "Failed to validate clusterId : ", err );
			return errors.New("Invalid clusterId")
		}
		if appGroup.GetAppGroupName() == "" {
			return errors.New("Name must have value ")
		}
		if appGroup.GetExternalLabel() == "" {
			return errors.New("ExternalLabel must have value ")
		}
	}
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
	cspInfo, err := cspInfoClient.GetCSPInfo(ctx, &pb.IDRequest{ Id: in.GetCspId() })
	if err != nil {
		log.Error( "Failed to get csp info err : ", err )
		return &pb.IDResponse{
			Code: pb.Code_NOT_FOUND,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Invalid CSP Id %s", in.GetCspId()),
			},
		}, nil
	}

	if cspInfo.GetContractId() != in.GetContractId() {
		log.Error( "Invalid contractId by cspId : ", cspInfo.GetContractId() )
		return &pb.IDResponse{
			Code: pb.Code_NOT_FOUND,
			Error: &pb.Error{
				Msg: fmt.Sprintf("ContractId and CSP Id do not match. expected contractId : %s", cspInfo.GetContractId()),
			},
		}, nil
	}

	// check cluster
	// Exactly one of those must be provided
	/*
	res, err := clusterInfoClient.GetClusters(ctx, &pb.GetClustersRequest{
		ContractId : in.GetContractId(),
		CspId : "",
	})
	if err == nil {
		for _, cluster := range res.GetClusters() {
			if cluster.GetStatus() == pb.ClusterStatus_INSTALLING {
				log.Info( "Already existed installing workflow. cluster : ", cluster )
				return &pb.IDResponse{
					Code: pb.Code_ALREADY_EXISTS,
					Error: &pb.Error{
						Msg: fmt.Sprintf("Already existed installing workflow. : %s", cluster.GetName()),
					},
				}, nil
			}
		}
	}
	*/

	// create cluster info
	clusterId := ""
	resAddClusterInfo, err := clusterInfoClient.AddClusterInfo(ctx, &pb.AddClusterInfoRequest{
		ContractId : in.GetContractId(),
		CspId : in.GetCspId(),
		Name : in.GetName(),
		Conf : in.GetConf(),
	})
	if err != nil {
		log.Error( "Failed to add cluster info. err : ", err )
		return &pb.IDResponse{
			Code: pb.Code_INTERNAL,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Failed to add cluster info. err : %s", err),
			},
		}, nil
	}
	clusterId = resAddClusterInfo.Id

	log.Info( "Added cluster in tks-info. clusterId : ", clusterId )


	// create usercluster
	nameSpace := "argo"
	workflow := "create-tks-usercluster-refactor"
	templateName := "template-std"
	gitAccount := "tks-management"
	revision := "main"
	manifestRepoUrl := "https://github.com/tks-management/" + clusterId + "-manifests"

	parameters := []string{ 
		"contract_id=" + in.GetContractId(), 
		"cluster_id=" + clusterId,
		"site_name=" + clusterId,
		"template_name=" + templateName,
		"git_account=" + gitAccount,
		"manifest_repo_url=" + manifestRepoUrl,
		"revision=" + revision,
	};

	workflowName, err := argowfClient.SumbitWorkflowFromWftpl( ctx, workflow, nameSpace, parameters );
	if err != nil {
		log.Error( "failed to submit argo workflow template. err : ", err )
		return &pb.IDResponse{
			Code: pb.Code_INTERNAL,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Failed to call argo workflow : %s", err ),
			},
		}, nil
	}
	log.Debug("submited workflow name : ", workflowName )

	// update status : INSTALLING
	if err := s.updateClusterStatus( ctx, clusterId, pb.ClusterStatus_INSTALLING ); err != nil {
		log.Error("Failed to update cluster status : INSTALLING" )
	}
	

	/******************************************************/
	// FOR DEMO : DELETE BELOW'
	/*
	{
		if !argowfClient.WaitWorkflows(ctx, nameSpace, []string{workflowName}, false, false) {
			log.Error("Failed to wait workflow ", workflowName)

			if err := s.updateClusterStatus( ctx, clusterId, pb.ClusterStatus_ERROR ); err != nil {
				log.Error("Failed to update cluster status : INSTALLING" )
			}
					
			return &pb.IDResponse{
				Code: pb.Code_INTERNAL,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Failed to call argo workflow : %s", workflowName ),
				},
			}, nil
		}

		workflow := "setup-sealed-secrets-on-usercluster"
		git_account := "tks-management"
		revision := "main"
		tks_admin := "tks-admin"
		app_group := "sealed-secrets"

		parameters := []string{ 
			"contract_id=" + in.GetContractId(), 
			"cluster_id=" + clusterId,
			"git_account=" + git_account,
			"revision=" + revision,
			"tks_admin=" + tks_admin,
			"app_group=" + app_group,
		};

		_workflowName, err := argowfClient.SumbitWorkflowFromWftpl( ctx, workflow, nameSpace, parameters );
		if err != nil {
			log.Error( "failed to submit argo workflow template. err : ", err )
			return &pb.IDResponse{
				Code: pb.Code_INTERNAL,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Failed to call argo workflow : %s", err ),
				},
			}, nil
		}
		workflowName = _workflowName
		log.Info("submited workflow name : ", workflowName )

		if !argowfClient.WaitWorkflows(ctx, nameSpace, []string{workflowName}, false, false) {
			log.Error("Failed to wait workflow ", workflowName)

			if err := s.updateClusterStatus( ctx, clusterId, pb.ClusterStatus_ERROR ); err != nil {
				log.Error("Failed to update cluster status : INSTALLING" )
			}

			return &pb.IDResponse{
				Code: pb.Code_INTERNAL,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Failed to call argo workflow : %s", workflowName ),
				},
			}, nil
		}

		if err := s.updateClusterStatus( ctx, clusterId, pb.ClusterStatus_RUNNING ); err != nil {
			log.Error("Failed to update cluster status : INSTALLING" )
		}
	}
	*/
	/******************************************************/


	log.Info("cluster successfully created. clusterId : ", clusterId );
	return &pb.IDResponse{
		Code:  pb.Code_OK_UNSPECIFIED,
		Error: nil,
		Id: clusterId,
	}, nil
}

func (s *server) updateClusterStatus(ctx context.Context, clusterId string, status pb.ClusterStatus ) (error) {
	res, err := clusterInfoClient.UpdateClusterStatus(ctx, &pb.UpdateClusterStatusRequest{
		ClusterId: clusterId,
		Status : status,
	})
	if err != nil {
		log.Error( "Failed to update cluster status err : ", err )
		return err
	}
	log.Info("updated cluster status RUNNING ", res)

	return nil
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

	appGroupIds := []string{}
	for _, appGroup := range in.GetAppGroups() {
		log.Debug( "appGroup : ", appGroup )

		clusterId := appGroup.GetClusterId()
		contractId := ""
		
		// Check Cluster
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
		log.Debug( "contractId ", contractId )

		// Create AppGoup
		appGroupId := ""
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

		// Call argo workflow template
		log.Debug( "appGroup.GetType() : ", appGroup.GetType() )
		workflowTemplate := ""
		var parameters []string
		switch appGroup.GetType() {
			case pb.AppGroupType_LMA :
				workflowTemplate = "tks-lma-federation"
				gitToken := "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"	// [TODO] use secret
				siteRepoUrl := "https://" + gitToken + "@github.com/tks-management/" + clusterId
				manifestRepoUrl := "https://github.com/tks-management/" + clusterId + "-manifests"
				tksInfoHost := "tks-info.tks.svc"
				parameters = []string{ 
					"site_name=" + clusterId, 
					"logging_component=" + "efk", 
					"site_repo_url=" + siteRepoUrl,
					"manifest_repo_url=" + manifestRepoUrl,
					"revision=main",
					"cluster_id=" + clusterId,
					"app_group_id=" + appGroupId,
					"tks_info_host=" + tksInfoHost,
				};

			case pb.AppGroupType_SERVICE_MESH : 
				workflowTemplate = "tks-service-mesh"
				manifestRepoUrl := "https://github.com/tks-management/" + clusterId + "-manifests"
				revision := "main"
				parameters = []string{ 
					"site_name=" + clusterId, 
					"app_group=" + "service-mesh", 
					"manifest_repo_url=" + manifestRepoUrl,
					"revision=" + revision,
				};

			default :
				log.Error( "invalid appGroup type ", appGroup.GetType() )
				continue
		}
		log.Debug( "workflowTemplate : ", workflowTemplate )

		workflowName, err := argowfClient.SumbitWorkflowFromWftpl( ctx, workflowTemplate, "argo", parameters );
		if err != nil {
			log.Error( "failed to submit argo workflow template. err : ", err )
			return &pb.IDsResponse{
				Code: pb.Code_INTERNAL,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Failed to call argo workflow : %s", err ),
				},
			}, nil
		}
		log.Debug("submited workflow name :", workflowName)

/*
			if !argowfClient.WaitWorkflows(ctx, "argo", []string{workflowName}, false, false){
				log.Error( "Failed to execute workflow : ", workflowName)

				return &pb.IDsResponse{
					Code: pb.Code_INTERNAL,
					Error: &pb.Error{
						Msg: fmt.Sprintf("Failed to execute workflow : %s", workflowName ),
					},
				}, nil
			}
*/
		appGroupIds = append(appGroupIds, appGroupId)

		// 아래의 workflow 는 App 설치시 한꺼번에 병렬로 실행한다.
		if appGroup.GetType() == pb.AppGroupType_LMA {
			{
				workflowTemplate := "cp-aws-infrastructure"
				parameters := []string{ 
					"cluster_id=" + clusterId, 
				};
				workflowName, err := argowfClient.SumbitWorkflowFromWftpl( ctx, workflowTemplate, "argo", parameters );
				if err != nil {
					log.Error( "failed to submit argo workflow template. err : ", err )
					return &pb.IDsResponse{
						Code: pb.Code_INTERNAL,
						Error: &pb.Error{
							Msg: fmt.Sprintf("Failed to call argo workflow : %s", err ),
						},
					}, nil
				}
				log.Debug("submited workflow name :", workflowName)
			}
			{
				workflowTemplate := "setup-sealed-secrets-on-usercluster"
				parameters := []string{ 
					"contract_id=" + contractId, 
					"cluster_id=" + clusterId,
					"git_account=" + "tks-management",
					"revision=" + "main",
					"tks_admin=" + "tks_admin",
					"app_group=" + "sealed-secrets",
				};
				workflowName, err := argowfClient.SumbitWorkflowFromWftpl( ctx, workflowTemplate, "argo", parameters );
				if err != nil {
					log.Error( "failed to submit argo workflow template. err : ", err )
					return &pb.IDsResponse{
						Code: pb.Code_INTERNAL,
						Error: &pb.Error{
							Msg: fmt.Sprintf("Failed to call argo workflow : %s", err ),
						},
					}, nil
				}
				log.Debug("submited workflow name :", workflowName)
			}
			{
				workflowTemplate := "tks-install-ingress-controller"
				manifestRepoUrl := "https://github.com/tks-management/" + clusterId + "-manifests"
				parameters := []string{ 
					"site_name=" + clusterId,
					"manifest_repo_url=" + manifestRepoUrl,
					"revision=" + "main",
					//"app_prefix=" + "",
				};
				workflowName, err := argowfClient.SumbitWorkflowFromWftpl( ctx, workflowTemplate, "argo", parameters );
				if err != nil {
					log.Error( "failed to submit argo workflow template. err : ", err )
					return &pb.IDsResponse{
						Code: pb.Code_INTERNAL,
						Error: &pb.Error{
							Msg: fmt.Sprintf("Failed to call argo workflow : %s", err ),
						},
					}, nil
				}
				log.Debug("submited workflow name :", workflowName)
			}
		}
	}
 
	log.Info("completed installation. appGroupIds : ", appGroupIds)

	return &pb.IDsResponse{
		Code:  pb.Code_OK_UNSPECIFIED,
		Error: nil,
		Ids: appGroupIds,
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
