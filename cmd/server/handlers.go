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

const MAX_SIZE_PER_AZ = 99

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

func validateUninstallAppGroupsRequest(in *pb.UninstallAppGroupsRequest) (err error) {
	for _, appGroupId := range in.GetAppGroupIds() {
		if _, err := uuid.Parse(appGroupId); err != nil {
			log.Error("Failed to validate appGroupId : ", err)
			return errors.New("Invalid appGroupId")
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

  /***************************
  * Pre-process cluster conf *
  ***************************/
  rawConf := in.GetConf()
  fmt.Printf("ClusterRawConf: %+v\n", rawConf)

  region := "ap-northeast-2"
  if rawConf != nil && rawConf.Region != "" {
    region = rawConf.Region
  }

  numOfAz := 3
  if rawConf != nil && rawConf.NumOfAz != 0 {
    numOfAz = int(rawConf.NumOfAz)
  }

  sshKeyName := "tks-seoul"
  if rawConf != nil && rawConf.SshKeyName != "" {
    sshKeyName = rawConf.SshKeyName
  }

  machineType := "t3.large"
  if rawConf != nil && rawConf.MachineType != "" {
    machineType = rawConf.MachineType
  }

  minSizePerAz := 1
  maxSizePerAz := 5

  // Check if numOfAz is correct based on pre-defined mapping table
  //validateNumOfAz(region, numOfAz)

  // TODO: Temporary table for test. Should be improved soon.
  azPerRegionTable := map[string]int{
	  "ap-northeast-1": 3,
	  "ap-northeast-2": 3,
  }

  maxAzForSelectedRegion := azPerRegionTable[region]
  if numOfAz > maxAzForSelectedRegion {
    log.Error("Invalid numOfAz: exceeded the number of Az in region ", region)
    temp_err := fmt.Errorf("Invalid numOfAz: exceeded the number of Az in region %s", region)
		return &pb.IDResponse{
			Code: pb.Code_INTERNAL,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Invalid numOfAz"),
			},
		}, temp_err
  }

  // Validate if machineReplicas is multiple of number of AZ
  replicas := int(rawConf.MachineReplicas)

  if replicas == 0 {
    log.Debug("No machineReplicas param. Using default values..")
  } else {
    if remainder := replicas % numOfAz; remainder != 0 {
      log.Error("Invalid machineReplicas: it should be multiple of numOfAz ", numOfAz)
      temp_err := fmt.Errorf("Invalid machineReplicas: it should be multiple of numOfAz %d", numOfAz)
      return &pb.IDResponse{
        Code: pb.Code_INTERNAL,
        Error: &pb.Error{
          Msg: fmt.Sprintf("Invalid machineReplicas!"),
        },
      }, temp_err
    } else {
      minSizePerAz = int(replicas/numOfAz)
      maxSizePerAz = minSizePerAz * 5

      // Validate if maxSizePerAx is within allowed range
      if maxSizePerAz > MAX_SIZE_PER_AZ {
        fmt.Printf("maxSizePerAz exceeded maximum value %d, so adjusted to %d", MAX_SIZE_PER_AZ, MAX_SIZE_PER_AZ)
        maxSizePerAz = MAX_SIZE_PER_AZ
      }
      log.Debug("Derived minSizePerAz: ", minSizePerAz)
      log.Debug("Derived maxSizePerAz: ", maxSizePerAz)
    }
  }

  // Construct cluster conf
  tempConf := pb.ClusterConf{
    SshKeyName: sshKeyName,
    Region: region,
    NumOfAz: int32(numOfAz),
    MachineType: machineType,
    MinSizePerAz: int32(minSizePerAz),
    MaxSizePerAz: int32(maxSizePerAz),
  }

  fmt.Printf("Newly constructed cluster conf: %+v\n", tempConf)

	// create cluster info
	clusterId := ""
	resAddClusterInfo, err := clusterInfoClient.AddClusterInfo(ctx, &pb.AddClusterInfoRequest{
		ContractId: in.GetContractId(),
		CspId:      in.GetCspId(),
		Name:       in.GetName(),
		Conf:       &tempConf,
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
	manifestRepoUrl := "https://github.com/" + gitAccount + "/" + clusterId + "-manifests"

	parameters := []string{
		"contract_id=" + in.GetContractId(),
		"cluster_id=" + clusterId,
		"site_name=" + clusterId,
		"template_name=template-std",
		"git_account=" + gitAccount,
		"manifest_repo_url=" + manifestRepoUrl,
		"revision=" + revision,
	}

	log.Info("Submitting workflow: ", workflow)

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
	log.Info("Successfully submited workflow: ", workflowName)

	// update status : INSTALLING
	if err := s.updateClusterStatus(ctx, clusterId, pb.ClusterStatus_INSTALLING); err != nil {
		log.Error("Failed to update cluster status to 'INSTALLING'")
	}

	log.Info("cluster successfully created. clusterId : ", clusterId)
	return &pb.IDResponse{
		Code:  pb.Code_OK_UNSPECIFIED,
		Error: nil,
		Id:    clusterId,
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

	res, err := clusterInfoClient.GetCluster(ctx, &pb.GetClusterRequest{ClusterId: clusterId})
	if err != nil {
		log.Error("Failed to get cluster info err : ", err)
		return &pb.SimpleResponse{
			Code: pb.Code_NOT_FOUND,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Could not find Cluster with ID %s", clusterId),
			},
		}, err
	}

	if res.GetCluster().GetStatus() == pb.ClusterStatus_DELETING || res.GetCluster().GetStatus() == pb.ClusterStatus_DELETED {
		log.Error("The cluster has been already deleted. status : ", res.GetCluster().GetStatus())
		return &pb.SimpleResponse{
			Code: pb.Code_NOT_FOUND,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Could not find cluster with ID. %s", clusterId),
			},
		}, fmt.Errorf("Could not find cluster with ID. %s", clusterId)
	}

	nameSpace := "argo"
	workflow := "tks-remove-usercluster"
	parameters := []string{
		"app_group=tks-cluster-aws",
		"tks_info_host=tks-info.tks.svc",
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

	if err := s.updateClusterStatus(ctx, clusterId, pb.ClusterStatus_DELETING); err != nil {
		log.Error("Failed to update cluster status to 'DELETING'")
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
		workflowTemplate := ""
		siteRepoUrl := "https://" + gitToken + "@github.com/" + gitAccount + "/" + clusterId
		manifestRepoUrl := "https://github.com/" + gitAccount + "/" + clusterId + "-manifests"
		parameters := []string{
			"site_name=" + clusterId,
			"cluster_id=" + clusterId,
			"site_repo_url=" + siteRepoUrl,
			"manifest_repo_url=" + manifestRepoUrl,
			"revision=" + revision,
			"app_group_id=" + appGroupId,
			"tks_info_host=tks-info.tks.svc",
		}

		switch appGroup.GetType() {
		case pb.AppGroupType_LMA:
			workflowTemplate = "tks-lma-federation"
			parameters = append(parameters, "logging_component=loki")

		case pb.AppGroupType_LMA_EFK:
			workflowTemplate = "tks-lma-federation"
			parameters = append(parameters, "logging_component=efk")


		case pb.AppGroupType_SERVICE_MESH:
			workflowTemplate = "tks-service-mesh"

		default:
			log.Error("invalid appGroup type ", appGroup.GetType())
			continue
		}
		log.Debug("workflowTemplate : ", workflowTemplate)

		workflowName, err := argowfClient.SumbitWorkflowFromWftpl(ctx, workflowTemplate, "argo", parameters)
		if err != nil {
			log.Error("failed to submit argo workflow template. err : ", err)
			continue
		}
		log.Debug("submited workflow name :", workflowName)

		if err := s.updateAppGroupStatus(ctx, appGroupId, pb.AppGroupStatus_APP_GROUP_INSTALLING); err != nil {
			log.Error("Failed to update appgroup status to 'APP_GROUP_INSTALLING'")
		}

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
func (s *server) UninstallAppGroups(ctx context.Context, in *pb.UninstallAppGroupsRequest) (*pb.IDsResponse, error) {
	log.Debug("Request 'UninstallAppGroups'")

	if err := validateUninstallAppGroupsRequest(in); err != nil {
		return &pb.IDsResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprint(err),
			},
		}, err
	}

	resAppGroupIds := []string{}
	for _, appGroupId := range in.GetAppGroupIds() {
		log.Debug("deleting appGroupId : ", appGroupId)

		res, err := appInfoClient.GetAppGroup(ctx, &pb.GetAppGroupRequest{
			AppGroupId: appGroupId,
		})
		if err != nil {
			log.Error("Failed to get app group info err : ", err)
			continue
		}

		appGroup := res.GetAppGroup()
		clusterId := appGroup.GetClusterId()

		// Call argo workflow template
		workflowTemplate := ""
		appGroupName := ""

		switch appGroup.GetType() {
		case pb.AppGroupType_LMA, pb.AppGroupType_LMA_EFK:
			workflowTemplate = "tks-remove-lma-federation"
			appGroupName = "lma"

		case pb.AppGroupType_SERVICE_MESH:
			workflowTemplate = "tks-remove-servicemesh"
			appGroupName = "service-mesh"

		default:
			log.Error("invalid appGroup type ", appGroup.GetType())
			continue
		}

		siteRepoUrl := "https://" + gitToken + "@github.com/" + gitAccount + "/" + clusterId
		parameters := []string{
			"app_group=" + appGroupName,
			"site_repo_url=" + siteRepoUrl,
			"tks_info_host=tks-info.tks.svc",
			"cluster_id=" + clusterId,
			"app_group_id=" + appGroupId,
		}

		workflowName, err := argowfClient.SumbitWorkflowFromWftpl(ctx, workflowTemplate, "argo", parameters)
		if err != nil {
			log.Error("failed to submit argo workflow template. err : ", err)
			continue
		}
		log.Debug("submited workflow name :", workflowName)

		resAppGroupIds = append(resAppGroupIds, appGroupId)
		if err := s.updateAppGroupStatus(ctx, appGroupId, pb.AppGroupStatus_APP_GROUP_DELETING); err != nil {
			log.Error("Failed to update appgroup status to 'APP_GROUP_DELETING'")
		}
	}

	return &pb.IDsResponse{
		Code:  pb.Code_OK_UNSPECIFIED,
		Error: nil,
		Ids:   resAppGroupIds,
	}, nil
}

func (s *server) updateClusterStatus(ctx context.Context, clusterId string, status pb.ClusterStatus) error {
	_, err := clusterInfoClient.UpdateClusterStatus(ctx, &pb.UpdateClusterStatusRequest{
		ClusterId: clusterId,
		Status:    status,
	})
	if err != nil {
		log.Error("Failed to update cluster status err : ", err)
		return err
	}

	return nil
}

func (s *server) updateAppGroupStatus(ctx context.Context, appGroupId string, status pb.AppGroupStatus) error {
	_, err := appInfoClient.UpdateAppGroupStatus(ctx, &pb.UpdateAppGroupStatusRequest{
		AppGroupId: appGroupId,
		Status:     status,
	})
	if err != nil {
		log.Error("Failed to update appgroup status err : ", err)
		return err
	}

	return nil
}
