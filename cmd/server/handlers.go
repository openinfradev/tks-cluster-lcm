package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"

	"github.com/openinfradev/tks-common/pkg/helper"
	"github.com/openinfradev/tks-common/pkg/log"
	pb "github.com/openinfradev/tks-proto/tks_pb"
)

var (
	filePathAzRegion = "./az-per-region.txt"
)

const MAX_SIZE_PER_AZ = 99

func validateCreateClusterRequest(in *pb.CreateClusterRequest) (err error) {
	if in.GetContractId() != "" {
		if !helper.ValidateContractId(in.GetContractId()) {
			return fmt.Errorf("invalid contract ID %s", in.GetContractId())
		}
		if _, err := uuid.Parse(in.GetCspId()); err != nil {
			return fmt.Errorf("invalid CSP ID %s", in.GetCspId())
		}
	}

	if in.GetName() == "" {
		return errors.New("Name must have value ")
	}
	return nil
}

func validateDeleteClusterRequest(in *pb.IDRequest) (err error) {
	if !helper.ValidateClusterId(in.GetId()) {
		return fmt.Errorf("invalid cluster ID %s", in.GetId())
	}
	return nil
}

func validateInstallAppGroupsRequest(in *pb.InstallAppGroupsRequest) (err error) {
	for _, appGroup := range in.GetAppGroups() {
		if !helper.ValidateClusterId(appGroup.GetClusterId()) {
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
		if !helper.ValidateApplicationGroupId(appGroupId) {
			return errors.New("Invalid appGroupId")
		}
	}
	return nil
}

func constructClusterConf(rawConf *pb.ClusterRawConf) (clusterConf *pb.ClusterConf, err error) {
	region := "ap-northeast-2"
	if rawConf != nil && rawConf.Region != "" {
		region = rawConf.Region
	}

	numOfAz := 3
	if rawConf != nil && rawConf.NumOfAz != 0 {
		numOfAz = int(rawConf.NumOfAz)

		if numOfAz > 3 {
			log.Error("Error: numOfAz cannot exceed 3.")
			temp_err := fmt.Errorf("Error: numOfAz cannot exceed 3.")
			return nil, temp_err
		}
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
	maxAzForSelectedRegion := 0

	file, err := os.Open(filePathAzRegion)
	if err != nil {
		log.Error(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var found bool = false
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), region) {
			log.Debug("Found region line: ", scanner.Text())
			azNum := strings.Split(scanner.Text(), ":")[1]
			maxAzForSelectedRegion, err = strconv.Atoi(strings.TrimSpace(azNum))
			if err != nil {
				log.Error("Error while converting azNum to int var: ", err)
			}
			log.Debug("Trimmed azNum var: ", maxAzForSelectedRegion)
			found = true
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error("Error while processing file: ", err)
	}
	if !found {
		log.Error("Couldn't find entry for region ", region)
	}

	if numOfAz > maxAzForSelectedRegion {
		log.Error("Invalid numOfAz: exceeded the number of Az in region ", region)
		temp_err := fmt.Errorf("Invalid numOfAz: exceeded the number of Az in region %s", region)
		return nil, temp_err
	}

	// Validate if machineReplicas is multiple of number of AZ
	replicas := int(rawConf.MachineReplicas)
	if replicas == 0 {
		log.Debug("No machineReplicas param. Using default values..")
	} else {
		if remainder := replicas % numOfAz; remainder != 0 {
			log.Error("Invalid machineReplicas: it should be multiple of numOfAz ", numOfAz)
			temp_err := fmt.Errorf("Invalid machineReplicas: it should be multiple of numOfAz %d", numOfAz)
			return nil, temp_err
		} else {
			log.Debug("Valid replicas and numOfAz. Caculating minSize & maxSize..")
			minSizePerAz = int(replicas / numOfAz)
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
		SshKeyName:   sshKeyName,
		Region:       region,
		NumOfAz:      int32(numOfAz),
		MachineType:  machineType,
		MinSizePerAz: int32(minSizePerAz),
		MaxSizePerAz: int32(maxSizePerAz),
	}

	fmt.Printf("Newly constructed cluster conf: %+v\n", &tempConf)
	return &tempConf, nil
}

func (s *server) CreateCluster(ctx context.Context, in *pb.CreateClusterRequest) (*pb.IDResponse, error) {
	log.Info("Request 'CreateCluster' for contractId : ", in.GetContractId())

	if err := validateCreateClusterRequest(in); err != nil {
		return &pb.IDResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprint(err),
			},
		}, err
	}

	contractId := in.GetContractId()
	cspId := in.GetCspId()

	// get default contract if contractId is empty
	if contractId == "" {
		contract, err := s.getDefaultContract(ctx)
		if err != nil {
			log.Error("Failed to get default contract. err : ", err)
			return &pb.IDResponse{
				Code: pb.Code_NOT_FOUND,
				Error: &pb.Error{
					Msg: "Failed to get default contract",
				},
			}, err
		}
		contractId = contract.GetContractId()

		res, err := cspInfoClient.GetCSPIDsByContractID(ctx, &pb.IDRequest{Id: contractId})
		if err != nil || len(res.Ids) == 0 {
			log.Error("Failed to get csp ids by contractId err : ", err)
			return &pb.IDResponse{
				Code: pb.Code_NOT_FOUND,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Invalid CSP Id %s", cspId),
				},
			}, err
		}

		// [TODO] Support AWS only!!!
		cspId = res.Ids[0]
	} else {
		// check contract
		if _, err := contractClient.GetContract(ctx, &pb.GetContractRequest{ContractId: contractId}); err != nil {
			log.Error("Failed to get contract info err : ", err)
			return &pb.IDResponse{
				Code: pb.Code_NOT_FOUND,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Invalid contract Id %s", contractId),
				},
			}, err
		}

		// check csp
		cspInfo, err := cspInfoClient.GetCSPInfo(ctx, &pb.IDRequest{Id: cspId})
		if err != nil {
			log.Error("Failed to get csp info err : ", err)
			return &pb.IDResponse{
				Code: pb.Code_NOT_FOUND,
				Error: &pb.Error{
					Msg: fmt.Sprintf("Invalid CSP Id %s", cspId),
				},
			}, err
		}
		if cspInfo.GetContractId() != contractId {
			log.Error("Invalid contractId by cspId : ", cspInfo.GetContractId())
			return &pb.IDResponse{
				Code: pb.Code_NOT_FOUND,
				Error: &pb.Error{
					Msg: fmt.Sprintf("ContractId and CSP Id do not match. expected contractId : %s", cspInfo.GetContractId()),
				},
			}, err
		}
	}

	/***************************
	 * Pre-process cluster conf *
	 ***************************/
	rawConf := in.GetConf()
	fmt.Printf("ClusterRawConf: %+v\n", rawConf)

	clConf, err := constructClusterConf(rawConf)
	if err != nil {
		return &pb.IDResponse{
			Code: pb.Code_INTERNAL,
			Error: &pb.Error{
				Msg: fmt.Sprint(err),
			},
		}, err
	}

	// create cluster info
	clusterId := ""
	resAddClusterInfo, err := clusterInfoClient.AddClusterInfo(ctx, &pb.AddClusterInfoRequest{
		ContractId: contractId,
		CspId:      cspId,
		Name:       in.GetName(),
		Conf:       clConf,
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
	manifestRepoUrl := "https://github.com/" + githubAccount + "/" + clusterId + "-manifests"

	parameters := []string{
		"contract_id=" + contractId,
		"cluster_id=" + clusterId,
		"site_name=" + clusterId,
		"template_name=template-std",
		"git_account=" + githubAccount,
		"manifest_repo_url=" + manifestRepoUrl,
		"revision=" + revision,
	}

	log.Info("Submitting workflow: ", workflow)

	workflowId, err := argowfClient.SumbitWorkflowFromWftpl(ctx, workflow, nameSpace, parameters)
	if err != nil {
		log.Error("failed to submit argo workflow template. err : ", err)
		return &pb.IDResponse{
			Code: pb.Code_INTERNAL,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Failed to call argo workflow : %s", err),
			},
		}, err
	}
	log.Info("Successfully submited workflow: ", workflowId)

	// update status : INSTALLING
	if err := s.updateClusterStatusWithWorkflowId(ctx, clusterId, pb.ClusterStatus_INSTALLING, workflowId); err != nil {
		log.Error("Failed to update cluster status to 'INSTALLING'")
	}

	log.Info("Successfully initiated user-cluster creation. clusterId: ", clusterId)
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

	// Validation : check request
	if err := validateDeleteClusterRequest(in); err != nil {
		return &pb.SimpleResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprint(err),
			},
		}, err
	}
	clusterId := in.GetId()

	// Validation : check cluster status
	// The cluster status must be (RUNNING|ERROR).
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
	if res.GetCluster().GetStatus() != pb.ClusterStatus_RUNNING &&
		res.GetCluster().GetStatus() != pb.ClusterStatus_ERROR {
		return &pb.SimpleResponse{
			Code: pb.Code_INVALID_ARGUMENT,
			Error: &pb.Error{
				Msg: fmt.Sprintf("The cluster can not be deleted. cluster status : %s", res.GetCluster().GetStatus()),
			},
		}, fmt.Errorf("The cluster can not be deleted. cluster status : %s", res.GetCluster().GetStatus())
	}

	// Validation : check appgroup status
	resAppGroups, err := appInfoClient.GetAppGroupsByClusterID(ctx, &pb.IDRequest{
		Id: clusterId,
	})
	if err == nil && resAppGroups.Code == pb.Code_OK_UNSPECIFIED {
		for _, resAppGroup := range resAppGroups.GetAppGroups() {
			if resAppGroup.GetStatus() != pb.AppGroupStatus_APP_GROUP_DELETED {
				return &pb.SimpleResponse{
					Code: pb.Code_INVALID_ARGUMENT,
					Error: &pb.Error{
						Msg: fmt.Sprintf("Undeleted services remain. %s", resAppGroup.GetAppGroupId()),
					},
				}, fmt.Errorf("Undeleted services remain. %s", resAppGroup.GetAppGroupId())
			}
		}
	}

	nameSpace := "argo"
	workflow := "tks-remove-usercluster"
	parameters := []string{
		"app_group=tks-cluster-aws",
		"tks_info_host=tks-info.tks.svc",
		"cluster_id=" + clusterId,
	}

	workflowId, err := argowfClient.SumbitWorkflowFromWftpl(ctx, workflow, nameSpace, parameters)
	if err != nil {
		log.Error("failed to submit argo workflow template. err : ", err)
		return &pb.SimpleResponse{
			Code: pb.Code_INTERNAL,
			Error: &pb.Error{
				Msg: fmt.Sprintf("Failed to call argo workflow : %s", err),
			},
		}, err
	}
	log.Debug("submited workflow name : ", workflowId)

	if err := s.updateClusterStatusWithWorkflowId(ctx, clusterId, pb.ClusterStatus_DELETING, workflowId); err != nil {
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
		manifestRepoUrl := "https://github.com/" + githubAccount + "/" + clusterId + "-manifests"
		parameters := []string{
			"site_name=" + clusterId,
			"cluster_id=" + clusterId,
			"github_account=" + githubAccount,
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

		workflowId, err := argowfClient.SumbitWorkflowFromWftpl(ctx, workflowTemplate, "argo", parameters)
		if err != nil {
			log.Error("failed to submit argo workflow template. err : ", err)
			continue
		}
		log.Debug("submited workflow name :", workflowId)

		if err := s.updateAppGroupStatusWithWorkflowId(ctx, appGroupId, pb.AppGroupStatus_APP_GROUP_INSTALLING, workflowId); err != nil {
			log.Error("Failed to update appgroup status to 'APP_GROUP_INSTALLING'")
		}

		appGroupIds = append(appGroupIds, appGroupId)
	}

	log.Info("Successfully submitted installation workflow. appGroupIds: ", appGroupIds)

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

		parameters := []string{
			"app_group=" + appGroupName,
			"github_account=" + githubAccount,
			"tks_info_host=tks-info.tks.svc",
			"cluster_id=" + clusterId,
			"app_group_id=" + appGroupId,
		}

		workflowId, err := argowfClient.SumbitWorkflowFromWftpl(ctx, workflowTemplate, "argo", parameters)
		if err != nil {
			log.Error("failed to submit argo workflow template. err : ", err)
			continue
		}
		log.Debug("submited workflow name :", workflowId)

		resAppGroupIds = append(resAppGroupIds, appGroupId)
		if err := s.updateAppGroupStatusWithWorkflowId(ctx, appGroupId, pb.AppGroupStatus_APP_GROUP_DELETING, workflowId); err != nil {
			log.Error("Failed to update appgroup status to 'APP_GROUP_DELETING'")
		}
	}

	return &pb.IDsResponse{
		Code:  pb.Code_OK_UNSPECIFIED,
		Error: nil,
		Ids:   resAppGroupIds,
	}, nil
}

func (s *server) updateClusterStatusWithWorkflowId(ctx context.Context, clusterId string, status pb.ClusterStatus, workflowId string) error {
	_, err := clusterInfoClient.UpdateClusterStatus(ctx, &pb.UpdateClusterStatusRequest{
		ClusterId:  clusterId,
		Status:     status,
		WorkflowId: workflowId,
	})
	if err != nil {
		log.Error("Failed to update cluster status err : ", err)
		return err
	}

	return nil
}

func (s *server) updateAppGroupStatusWithWorkflowId(ctx context.Context, appGroupId string, status pb.AppGroupStatus, workflowId string) error {
	_, err := appInfoClient.UpdateAppGroupStatus(ctx, &pb.UpdateAppGroupStatusRequest{
		AppGroupId: appGroupId,
		Status:     status,
		WorkflowId: workflowId,
	})
	if err != nil {
		log.Error("Failed to update appgroup status err : ", err)
		return err
	}

	return nil
}

func (s *server) getDefaultContract(ctx context.Context) (*pb.Contract, error) {
	resContract, err := contractClient.GetDefaultContract(ctx, &empty.Empty{})
	if err != nil {
		log.Error("Failed to get contract info err : ", err)
		return nil, err
	}

	return resContract.GetContract(), nil
}
