package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/openinfradev/tks-common/pkg/argowf"
	"github.com/openinfradev/tks-common/pkg/grpc_client"
	"github.com/openinfradev/tks-common/pkg/log"
	pb "github.com/openinfradev/tks-proto/tks_pb"
)

var (
	argowfClient      argowf.Client
	contractClient    pb.ContractServiceClient
	cspInfoClient     pb.CspInfoServiceClient
	clusterInfoClient pb.ClusterInfoServiceClient
	appInfoClient     pb.AppInfoServiceClient
)

// 각 client lifecycle은 서버 종료시까지므로 close는 하지 않는다.
func InitHandlers(contractAddress string, contractPort int, infoAddress string, infoPort int, argoAddress string, argoPort int) {
	var err error

	argowfClient, err = argowf.New(argoAddress, argoPort)
	if err != nil {
		log.Fatal("failed to create argowf client : ", err)
	}

	_, contractClient, err = grpc_client.CreateContractClient(contractAddress, contractPort, "tks-cluster-lcm")
	if err != nil {
		log.Fatal("failed to create contract client : ", err)
	}

	_, cspInfoClient, err = grpc_client.CreateCspInfoClient(infoAddress, infoPort, "tks-cluster-lcm")
	if err != nil {
		log.Fatal("failed to create cspinfo client : ", err)
	}

	_, clusterInfoClient, err = grpc_client.CreateClusterInfoClient(infoAddress, infoPort, "tks-cluster-lcm")
	if err != nil {
		log.Fatal("failed to create cluster client : ", err)
	}

	_, appInfoClient, err = grpc_client.CreateAppInfoClient(infoAddress, infoPort, "tks-cluster-lcm")
	if err != nil {
		log.Fatal("failed to create appinfo client : ", err)
	}

	log.Info("All clients created successfully")
}

func validateCreateClusterRequest(in *pb.CreateClusterRequest) (err error) {
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

func validateDeleteClusterRequest(in *pb.IDRequest) (err error) {
	if _, err := uuid.Parse(in.GetId()); err != nil {
		return fmt.Errorf("invalid cluster ID %s", in.GetId())
	}
	return nil
}

func validateInstallAppGroupsRequest(in *pb.InstallAppGroupsRequest) (err error) {
	for _, appGroup := range in.GetAppGroups() {
		if _, err := uuid.Parse(appGroup.GetClusterId()); err != nil {
			log.Error("Failed to validate clusterId : ", err)
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
	if err := validateCreateClusterRequest(in); err != nil {
		return &pb.IDResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprint(err),
			},
		}, err
	}

	// check contract
	if _, err := contractClient.GetContract(ctx, &pb.GetContractRequest{ContractId: in.GetContractId()}); err != nil {
		log.Error("Failed to get contract info err : ", err)
		return &pb.IDResponse{
			Code: pb.Code_NOT_FOUND,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Invalid contract Id %s", in.GetContractId()),
			},
		}, err
	}

	// check csp
	cspInfo, err := cspInfoClient.GetCSPInfo(ctx, &pb.IDRequest{Id: in.GetCspId()})
	if err != nil {
		log.Error("Failed to get csp info err : ", err)
		return &pb.IDResponse{
			Code: pb.Code_NOT_FOUND,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Invalid CSP Id %s", in.GetCspId()),
			},
		}, err
	}

	if cspInfo.GetContractId() != in.GetContractId() {
		log.Error("Invalid contractId by cspId : ", cspInfo.GetContractId())
		return &pb.IDResponse{
			Code: pb.Code_NOT_FOUND,
			Error: &pb.Error{
				Msg: fmt.Sprintf("ContractId and CSP Id do not match. expected contractId : %s", cspInfo.GetContractId()),
			},
		}, err
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
		ContractId: in.GetContractId(),
		CspId:      in.GetCspId(),
		Name:       in.GetName(),
		Conf:       in.GetConf(),
	})
	if err != nil {
		log.Error("Failed to add cluster info. err : ", err)
		return &pb.IDResponse{
			Code: pb.Code_INTERNAL,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Failed to add cluster info. err : %s", err),
			},
		}, err
	}
	clusterId = resAddClusterInfo.Id

	log.Info("Added cluster in tks-info. clusterId : ", clusterId)

	// create usercluster
	nameSpace := "argo"
	workflow := "create-tks-usercluster"
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
	}

	workflowName, err := argowfClient.SumbitWorkflowFromWftpl(ctx, workflow, nameSpace, parameters)
	if err != nil {
		log.Error("failed to submit argo workflow template. err : ", err)
		return &pb.IDResponse{
			Code: pb.Code_INTERNAL,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Failed to call argo workflow : %s", err),
			},
		}, err
	}
	log.Debug("submited workflow name : ", workflowName)

	// update status : INSTALLING
	if err := s.updateClusterStatus(ctx, clusterId, pb.ClusterStatus_INSTALLING); err != nil {
		log.Error("Failed to update cluster status : INSTALLING")
	}

	log.Info("cluster successfully created. clusterId : ", clusterId)
	return &pb.IDResponse{
		Code:  pb.Code_OK_UNSPECIFIED,
		Error: nil,
		Id:    clusterId,
	}, nil
}

func (s *server) updateClusterStatus(ctx context.Context, clusterId string, status pb.ClusterStatus) error {
	res, err := clusterInfoClient.UpdateClusterStatus(ctx, &pb.UpdateClusterStatusRequest{
		ClusterId: clusterId,
		Status:    status,
	})
	if err != nil {
		log.Error("Failed to update cluster status err : ", err)
		return err
	}
	log.Info("updated cluster status : ", res)

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

func (s *server) DeleteCluster(ctx context.Context, in *pb.IDRequest) (*pb.SimpleResponse, error) {
	log.Info("Request 'DeleteCluster' for clusterId : ", in.GetId())

	if err := validateDeleteClusterRequest(in); err != nil {
		return &pb.SimpleResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprint(err),
			},
		}, err
	}
	clusterId := in.GetId()

	if _, err := clusterInfoClient.GetCluster(ctx, &pb.GetClusterRequest{ClusterId: clusterId}); err != nil {
		log.Error("Failed to get cluster info err : ", err)
		return &pb.SimpleResponse{
			Code: pb.Code_NOT_FOUND,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Invalid cluster Id %s", clusterId),
			},
		}, err
	}

	nameSpace := "argo"
	workflow := "tks-remove-usercluster"
	appGroup := "tks-cluster-aws"
	tksInfoHost := "tks-info.tks.svc"
	parameters := []string{
		"app_group=" + appGroup,
		"tks_info_host=" + tksInfoHost,
		"cluster_id=" + clusterId,
	}

	workflowName, err := argowfClient.SumbitWorkflowFromWftpl(ctx, workflow, nameSpace, parameters)
	if err != nil {
		log.Error("failed to submit argo workflow template. err : ", err)
		return &pb.SimpleResponse{
			Code: pb.Code_INTERNAL,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Failed to call argo workflow : %s", err),
			},
		}, err
	}
	log.Debug("submited workflow name : ", workflowName)

	// update status : DELETEING
	if err := s.updateClusterStatus(ctx, clusterId, pb.ClusterStatus_DELETING); err != nil {
		log.Error("Failed to update cluster status : DELETING")
	}

	log.Info("cluster successfully deleted. clusterId : ", clusterId)
	return &pb.SimpleResponse{
		Code:  pb.Code_OK_UNSPECIFIED,
		Error: nil,
	}, nil
}

// InstallAppGroups install apps, return a array of application id
func (s *server) InstallAppGroups(ctx context.Context, in *pb.InstallAppGroupsRequest) (*pb.IDsResponse, error) {
	log.Debug("Request 'InstallAppGroups' ")

	// [TODO] validation refactoring
	if err := validateInstallAppGroupsRequest(in); err != nil {
		return &pb.IDsResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprint(err),
			},
		}, err
	}

	appGroupIds := []string{}
	for _, appGroup := range in.GetAppGroups() {
		log.Debug("appGroup : ", appGroup)

		clusterId := appGroup.GetClusterId()
		contractId := ""

		// Check Cluster
		cluster, err := clusterInfoClient.GetCluster(ctx, &pb.GetClusterRequest{ClusterId: clusterId})
		if err != nil {
			log.Error("Failed to get cluster info err : ", err)
			continue
		}
		if cluster == nil {
			log.Error("Failed to get cluster info : ", appGroup.GetClusterId())
			continue
		}
		log.Debug("cluster : ", cluster)
		contractId = cluster.GetCluster().GetContractId()
		log.Debug("contractId ", contractId)

		appGroupId := ""
		res, err := appInfoClient.GetAppGroupsByClusterID(ctx, &pb.IDRequest{
			Id: clusterId,
		})
		if err == nil && res.Code == pb.Code_OK_UNSPECIFIED {
			for _, resAppGroup := range res.GetAppGroups() {
				if resAppGroup.GetAppGroupName() == appGroup.GetAppGroupName() &&
					resAppGroup.GetType() == appGroup.GetType() &&
					resAppGroup.GetExternalLabel() == appGroup.GetExternalLabel() {
					appGroupId = resAppGroup.GetAppGroupId()
					break
				}
			}
		}

		if appGroupId == "" {
			res, err := appInfoClient.CreateAppGroup(ctx, &pb.CreateAppGroupRequest{
				ClusterId: appGroup.GetClusterId(),
				AppGroup:  appGroup,
			})
			if err != nil {
				log.Error("Failed to create app group info err : ", err)
				continue
			}
			appGroupId = res.GetId()
		}
		log.Debug("appGroupId ", appGroupId)

		// Call argo workflow template
		log.Debug("appGroup.GetType() : ", appGroup.GetType())
		workflowTemplate := ""
		var parameters []string
		switch appGroup.GetType() {
		case pb.AppGroupType_LMA:
			workflowTemplate = "tks-lma-federation"
			gitToken := "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" // [TODO] use secret
			siteRepoUrl := "https://" + gitToken + "@github.com/tks-management/" + clusterId
			manifestRepoUrl := "https://github.com/tks-management/" + clusterId + "-manifests"
			tksInfoHost := "tks-info.tks.svc"
			parameters = []string{
				"site_name=" + clusterId,
				"logging_component=" + "loki",
				"site_repo_url=" + siteRepoUrl,
				"manifest_repo_url=" + manifestRepoUrl,
				"revision=main",
				"cluster_id=" + clusterId,
				"app_group_id=" + appGroupId,
				"tks_info_host=" + tksInfoHost,
			}

		case pb.AppGroupType_SERVICE_MESH:
			workflowTemplate = "tks-service-mesh"
			gitToken := "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" // [TODO] use secret
			siteRepoUrl := "https://" + gitToken + "@github.com/tks-management/" + clusterId
			manifestRepoUrl := "https://github.com/tks-management/" + clusterId + "-manifests"
			tksInfoHost := "tks-info.tks.svc"
			parameters = []string{
				"site_name=" + clusterId,
				"site_repo_url=" + siteRepoUrl,
				"manifest_repo_url=" + manifestRepoUrl,
				"revision=main",
				"cluster_id=" + clusterId,
				"app_group_id=" + appGroupId,
				"tks_info_host=" + tksInfoHost,
			}

		default:
			log.Error("invalid appGroup type ", appGroup.GetType())
			continue
		}
		log.Debug("workflowTemplate : ", workflowTemplate)

		workflowName, err := argowfClient.SumbitWorkflowFromWftpl(ctx, workflowTemplate, "argo", parameters)
		if err != nil {
			log.Error("failed to submit argo workflow template. err : ", err)
			return &pb.IDsResponse{
				Code: pb.Code_INTERNAL,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Failed to call argo workflow : %s", err),
				},
			}, nil
		}
		log.Debug("submited workflow name :", workflowName)

		appGroupIds = append(appGroupIds, appGroupId)
	}

	log.Info("completed installation. appGroupIds : ", appGroupIds)

	return &pb.IDsResponse{
		Code:  pb.Code_OK_UNSPECIFIED,
		Error: nil,
		Ids:   appGroupIds,
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
