package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	mockargo "github.com/openinfradev/tks-common/pkg/argowf/mock"
	pb "github.com/openinfradev/tks-proto/tks_pb"
	mocktks "github.com/openinfradev/tks-proto/tks_pb/mock"

	"github.com/openinfradev/tks-common/pkg/log"
)

var (
	createClusterRequest    *pb.CreateClusterRequest
	installAppGroupsRequest *pb.InstallAppGroupsRequest

	createdClusterId  = uuid.New().String()
	createdAppGroupId = uuid.New().String()
)

func init() {
	log.Disable()

	// override for test
	filePathAzRegion = "../../files/az-per-region.txt"

	// for CreateCluster API
	installAppGroupsRequest = randomInstallAppGroupsRequest()
	createClusterRequest = randomCreateClusterRequest()
}

func TestCreateCluster(t *testing.T) {
	requestEmptyContractId := randomCreateClusterRequest()
	requestEmptyContractId.ContractId = ""
	requestEmptyContractId.CspId = ""

	testCases := []struct {
		name       string
		in         *pb.CreateClusterRequest
		buildStubs func(mockArgoClient *mockargo.MockClient,
			mockCspInfoClient *mocktks.MockCspInfoServiceClient,
			mockClusterInfoClient *mocktks.MockClusterInfoServiceClient,
			mockContractClient *mocktks.MockContractServiceClient)
		checkResponse func(req *pb.CreateClusterRequest, res *pb.IDResponse, err error)
	}{
		{
			name: "OK",
			in:   createClusterRequest,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockCspInfoClient *mocktks.MockCspInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient,
				mockContractClient *mocktks.MockContractServiceClient) {

				mockContractClient.EXPECT().GetContract(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetContractResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, nil)

				mockCspInfoClient.EXPECT().GetCSPInfo(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetCSPInfoResponse{
							Code:       pb.Code_OK_UNSPECIFIED,
							Error:      nil,
							ContractId: createClusterRequest.ContractId,
						}, nil)

				mockClusterInfoClient.EXPECT().AddClusterInfo(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.IDResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							Id:    createdClusterId,
						}, nil)

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), nil)

				mockClusterInfoClient.EXPECT().UpdateClusterStatus(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.SimpleResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, nil)
			},
			checkResponse: func(req *pb.CreateClusterRequest, res *pb.IDResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
			},
		},
		{
			name: "OK_EMPTY_CONTRACT_ID",
			in:   requestEmptyContractId,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockCspInfoClient *mocktks.MockCspInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient,
				mockContractClient *mocktks.MockContractServiceClient) {

				mockContractClient.EXPECT().GetDefaultContract(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetContractResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							Contract: &pb.Contract{
								ContractId: uuid.New().String(),
								CspId:      uuid.New().String(),
							},
						}, nil)

				mockCspInfoClient.EXPECT().GetCSPIDsByContractID(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.IDsResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							Ids:   []string{uuid.New().String()},
						}, nil)

				mockClusterInfoClient.EXPECT().AddClusterInfo(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.IDResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							Id:    createdClusterId,
						}, nil)

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), nil)

				mockClusterInfoClient.EXPECT().UpdateClusterStatus(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.SimpleResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, nil)
			},
			checkResponse: func(req *pb.CreateClusterRequest, res *pb.IDResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
			},
		},
		{
			name: "NO_DEFAULT_CONTRACT",
			in:   requestEmptyContractId,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockCspInfoClient *mocktks.MockCspInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient,
				mockContractClient *mocktks.MockContractServiceClient) {

				mockContractClient.EXPECT().GetDefaultContract(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetContractResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, errors.New("NO_DATA_DEFAULT_CONTRACT"))
			},
			checkResponse: func(req *pb.CreateClusterRequest, res *pb.IDResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_NOT_FOUND)
			},
		},
		{
			name: "INVALID_ARGUMENT_CONTRACTID",
			in: &pb.CreateClusterRequest{
				ContractId: "THIS_IS_NOT_UUID",
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockCspInfoClient *mocktks.MockCspInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient,
				mockContractClient *mocktks.MockContractServiceClient) {
			},
			checkResponse: func(req *pb.CreateClusterRequest, res *pb.IDResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INVALID_ARGUMENT)
			},
		},
		{
			name: "INVALID_ARGUMENT_CSPID",
			in: &pb.CreateClusterRequest{
				ContractId: uuid.New().String(),
				CspId:      "THIS_IS_NOT_UUID",
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockCspInfoClient *mocktks.MockCspInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient,
				mockContractClient *mocktks.MockContractServiceClient) {
			},
			checkResponse: func(req *pb.CreateClusterRequest, res *pb.IDResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INVALID_ARGUMENT)
			},
		},
		{
			name: "INVALID_ARGUMENT_NO_NAME",
			in: &pb.CreateClusterRequest{
				ContractId: uuid.New().String(),
				CspId:      uuid.New().String(),
				Name:       "",
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockCspInfoClient *mocktks.MockCspInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient,
				mockContractClient *mocktks.MockContractServiceClient) {
			},
			checkResponse: func(req *pb.CreateClusterRequest, res *pb.IDResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INVALID_ARGUMENT)
			},
		},
		{
			name: "NOT_FOUND_CONTRACT_ID",
			in:   createClusterRequest,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockCspInfoClient *mocktks.MockCspInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient,
				mockContractClient *mocktks.MockContractServiceClient) {

				mockContractClient.EXPECT().GetContract(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetContractResponse{
							Code: pb.Code_NOT_FOUND,
							Error: &pb.Error{
								Msg: "NOT FOUND CONTRACTID FROM TKS-CONTRACT",
							},
						}, errors.New("NOT FOUND CONTRACTID FROM TKS-CONTRACT"))
			},
			checkResponse: func(req *pb.CreateClusterRequest, res *pb.IDResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_NOT_FOUND)
			},
		},
		{
			name: "CSP_ID_IS_NOT_MATCHED_TO_CONTRACT_ID",
			in:   createClusterRequest,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockCspInfoClient *mocktks.MockCspInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient,
				mockContractClient *mocktks.MockContractServiceClient) {

				mockContractClient.EXPECT().GetContract(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetContractResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, nil)

				mockCspInfoClient.EXPECT().GetCSPInfo(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetCSPInfoResponse{
							Code:       pb.Code_OK_UNSPECIFIED,
							Error:      nil,
							ContractId: uuid.New().String(),
						}, nil)
			},
			checkResponse: func(req *pb.CreateClusterRequest, res *pb.IDResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_NOT_FOUND)
			},
		},
		{
			name: "FAILED_TO_ADD_CLUSTER",
			in:   createClusterRequest,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockCspInfoClient *mocktks.MockCspInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient,
				mockContractClient *mocktks.MockContractServiceClient) {

				mockContractClient.EXPECT().GetContract(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetContractResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, nil)

				mockCspInfoClient.EXPECT().GetCSPInfo(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetCSPInfoResponse{
							Code:       pb.Code_OK_UNSPECIFIED,
							Error:      nil,
							ContractId: createClusterRequest.ContractId,
						}, nil)

				mockClusterInfoClient.EXPECT().AddClusterInfo(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.IDResponse{}, errors.New("FAILED TO ADD CLUSTER"))
			},
			checkResponse: func(req *pb.CreateClusterRequest, res *pb.IDResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INTERNAL)

			},
		},
		{
			name: "FAILED_TO_CALL_WORKFLOW",
			in:   createClusterRequest,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockCspInfoClient *mocktks.MockCspInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient,
				mockContractClient *mocktks.MockContractServiceClient) {

				mockContractClient.EXPECT().GetContract(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetContractResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, nil)

				mockCspInfoClient.EXPECT().GetCSPInfo(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetCSPInfoResponse{
							Code:       pb.Code_OK_UNSPECIFIED,
							Error:      nil,
							ContractId: createClusterRequest.ContractId,
						}, nil)

				mockClusterInfoClient.EXPECT().AddClusterInfo(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.IDResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							Id:    createdClusterId,
						}, nil)

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
					Return("", errors.New("FAILED_TO_CALL_WORKFLOW"))
			},
			checkResponse: func(req *pb.CreateClusterRequest, res *pb.IDResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INTERNAL)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// mocking and injection
			mockArgoClient := mockargo.NewMockClient(ctrl)
			argowfClient = mockArgoClient

			mockCspInfoClient := mocktks.NewMockCspInfoServiceClient(ctrl)
			cspInfoClient = mockCspInfoClient
			mockClusterInfoClient := mocktks.NewMockClusterInfoServiceClient(ctrl)
			clusterInfoClient = mockClusterInfoClient
			mockContarctClient := mocktks.NewMockContractServiceClient(ctrl)
			contractClient = mockContarctClient

			tc.buildStubs(mockArgoClient, mockCspInfoClient, mockClusterInfoClient, mockContarctClient)

			s := server{}
			res, err := s.CreateCluster(ctx, tc.in)
			tc.checkResponse(tc.in, res, err)
		})
	}
}

func TestDeleteCluster(t *testing.T) {
	testCases := []struct {
		name          string
		in            *pb.IDRequest
		buildStubs    func(mockArgoClient *mockargo.MockClient, mockClusterInfoClient *mocktks.MockClusterInfoServiceClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient)
		checkResponse func(req *pb.IDRequest, res *pb.SimpleResponse, err error)
	}{
		{
			name: "OK",
			in: &pb.IDRequest{
				Id: createdClusterId,
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient,
				mockAppInfoClient *mocktks.MockAppInfoServiceClient) {

				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetClusterResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							Cluster: &pb.Cluster{
								Status: pb.ClusterStatus_RUNNING,
							},
						}, nil)

				mockAppInfoClient.EXPECT().GetAppGroupsByClusterID(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetAppGroupsResponse{
							Code:      pb.Code_OK_UNSPECIFIED,
							Error:     nil,
							AppGroups: []*pb.AppGroup{},
						}, nil)

				mockClusterInfoClient.EXPECT().UpdateClusterStatus(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.SimpleResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, nil)

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), nil)
			},
			checkResponse: func(req *pb.IDRequest, res *pb.SimpleResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
			},
		},
		{
			name: "INVALID_ARGUMENT_CLUSTER_ID",
			in: &pb.IDRequest{
				Id: "THIS_IS_NOT_UUID",
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient, mockClusterInfoClient *mocktks.MockClusterInfoServiceClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient) {
			},
			checkResponse: func(req *pb.IDRequest, res *pb.SimpleResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INVALID_ARGUMENT)
			},
		},
		{
			name: "NOT_EXIST_CLUSTER",
			in: &pb.IDRequest{
				Id: uuid.New().String(),
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient, mockClusterInfoClient *mocktks.MockClusterInfoServiceClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient) {
				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetClusterResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, errors.New("NOT_EXISTED_CLUSTER"))
			},
			checkResponse: func(req *pb.IDRequest, res *pb.SimpleResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_NOT_FOUND)
			},
		},
		{
			name: "THE_CLUSTER_ALREADY_DELETED",
			in: &pb.IDRequest{
				Id: uuid.New().String(),
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient, mockClusterInfoClient *mocktks.MockClusterInfoServiceClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient) {
				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetClusterResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							Cluster: &pb.Cluster{
								Status: pb.ClusterStatus_DELETED,
							},
						}, nil)
			},
			checkResponse: func(req *pb.IDRequest, res *pb.SimpleResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INVALID_ARGUMENT)
			},
		},
		{
			name: "FAILED_TO_CALL_WORKFLOW",
			in: &pb.IDRequest{
				Id: uuid.New().String(),
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient, mockClusterInfoClient *mocktks.MockClusterInfoServiceClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient) {
				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetClusterResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							Cluster: &pb.Cluster{
								Status: pb.ClusterStatus_RUNNING,
							},
						}, nil)
				mockAppInfoClient.EXPECT().GetAppGroupsByClusterID(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetAppGroupsResponse{
							Code:      pb.Code_OK_UNSPECIFIED,
							Error:     nil,
							AppGroups: []*pb.AppGroup{},
						}, nil)
				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), errors.New("FAILED_TO_CALL_WORKFLOW"))
			},
			checkResponse: func(req *pb.IDRequest, res *pb.SimpleResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INTERNAL)
			},
		},
		{
			name: "CLUSTER_STATUS_IS_NOT_RUNNING",
			in: &pb.IDRequest{
				Id: uuid.New().String(),
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient, mockClusterInfoClient *mocktks.MockClusterInfoServiceClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient) {
				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetClusterResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							Cluster: &pb.Cluster{
								Status: pb.ClusterStatus_INSTALLING,
							},
						}, nil)
			},
			checkResponse: func(req *pb.IDRequest, res *pb.SimpleResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INVALID_ARGUMENT)
			},
		},
		{
			name: "APPGROUP_STATUS_IS_NOT_DELETED",
			in: &pb.IDRequest{
				Id: uuid.New().String(),
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient, mockClusterInfoClient *mocktks.MockClusterInfoServiceClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient) {
				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetClusterResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							Cluster: &pb.Cluster{
								Status: pb.ClusterStatus_RUNNING,
							},
						}, nil)

				mockAppInfoClient.EXPECT().GetAppGroupsByClusterID(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetAppGroupsResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							AppGroups: []*pb.AppGroup{
								{
									Status: pb.AppGroupStatus_APP_GROUP_DELETING,
								},
							},
						}, nil)
			},
			checkResponse: func(req *pb.IDRequest, res *pb.SimpleResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INVALID_ARGUMENT)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// mocking and injection
			mockArgoClient := mockargo.NewMockClient(ctrl)
			argowfClient = mockArgoClient

			mockClusterInfoClient := mocktks.NewMockClusterInfoServiceClient(ctrl)
			clusterInfoClient = mockClusterInfoClient
			mockAppInfoClient := mocktks.NewMockAppInfoServiceClient(ctrl)
			appInfoClient = mockAppInfoClient

			tc.buildStubs(mockArgoClient, mockClusterInfoClient, mockAppInfoClient)

			s := server{}
			res, err := s.DeleteCluster(ctx, tc.in)
			tc.checkResponse(tc.in, res, err)
		})
	}
}

func TestInstallAppGroups(t *testing.T) {
	testCases := []struct {
		name       string
		in         *pb.InstallAppGroupsRequest
		buildStubs func(mockArgoClient *mockargo.MockClient,
			mockAppInfoClient *mocktks.MockAppInfoServiceClient,
			mockClusterInfoClient *mocktks.MockClusterInfoServiceClient)
		checkResponse func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error)
	}{
		{
			name: "OK",
			in:   installAppGroupsRequest,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockAppInfoClient *mocktks.MockAppInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient) {
				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetClusterResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, nil)

				mockAppInfoClient.EXPECT().GetAppGroupsByClusterID(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetAppGroupsResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, errors.New("NOT_EXISTED_APPGROUPS"))

				mockAppInfoClient.EXPECT().UpdateAppGroupStatus(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.SimpleResponse{Code: pb.Code_OK_UNSPECIFIED, Error: nil}, nil)

				mockAppInfoClient.EXPECT().CreateAppGroup(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.IDResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							Id:    createdAppGroupId,
						}, nil)

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), "tks-lma-federation", gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), nil)
			},
			checkResponse: func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.Equal(t, len(installAppGroupsRequest.AppGroups), len(res.Ids))
				require.Equal(t, createdAppGroupId, res.Ids[0])

			},
		},
		{
			name: "OK_NO_APPGROUPS",
			in: &pb.InstallAppGroupsRequest{
				AppGroups: []*pb.AppGroup{},
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockAppInfoClient *mocktks.MockAppInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient) {
			},
			checkResponse: func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.Equal(t, len(res.Ids), 0)
			},
		},
		{
			name: "OK_EXISTED_APPGROUP",
			in:   installAppGroupsRequest,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockAppInfoClient *mocktks.MockAppInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient) {
				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetClusterResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, nil)

				mockAppInfoClient.EXPECT().GetAppGroupsByClusterID(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetAppGroupsResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							AppGroups: []*pb.AppGroup{
								{
									AppGroupId:    createdAppGroupId,
									Type:          installAppGroupsRequest.GetAppGroups()[0].GetType(),
									AppGroupName:  installAppGroupsRequest.GetAppGroups()[0].GetAppGroupName(),
									ExternalLabel: installAppGroupsRequest.GetAppGroups()[0].GetExternalLabel(),
								},
							},
						}, nil)

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), "tks-lma-federation", gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), nil)

				mockAppInfoClient.EXPECT().UpdateAppGroupStatus(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.SimpleResponse{Code: pb.Code_OK_UNSPECIFIED, Error: nil}, nil)
			},
			checkResponse: func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.Equal(t, res.Ids[0], createdAppGroupId)
			},
		},
		{
			name: "INVALID_ARGUMENT_CLUSTER_ID",
			in: &pb.InstallAppGroupsRequest{
				AppGroups: []*pb.AppGroup{
					{
						ClusterId: "THIS_IS_NOT_UUID",
					},
				},
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockAppInfoClient *mocktks.MockAppInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient) {
			},
			checkResponse: func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INVALID_ARGUMENT)
			},
		},
		{
			name: "INVALID_ARGUMENT_NO_NAME",
			in: &pb.InstallAppGroupsRequest{
				AppGroups: []*pb.AppGroup{
					{
						ClusterId:    uuid.New().String(),
						AppGroupName: "",
					},
				},
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockAppInfoClient *mocktks.MockAppInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient) {
			},
			checkResponse: func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INVALID_ARGUMENT)
			},
		},
		{
			name: "NOT_EXIST_CLUSTER_ID",
			in:   installAppGroupsRequest,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockAppInfoClient *mocktks.MockAppInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient) {
				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.GetClusterResponse{}, errors.New("NOT_EXIST_CLUSTER_ID"))
			},
			checkResponse: func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.True(t, len(installAppGroupsRequest.AppGroups) > len(res.Ids))
			},
		},
		{
			name: "FAILED_TO_CREATE_APPGROUP",
			in:   installAppGroupsRequest,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockAppInfoClient *mocktks.MockAppInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient) {
				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetClusterResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, nil)

				mockAppInfoClient.EXPECT().GetAppGroupsByClusterID(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetAppGroupsResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, errors.New("NOT_EXISTED_APPGROUPS"))

				mockAppInfoClient.EXPECT().CreateAppGroup(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.IDResponse{}, errors.New("Failed to create appgroup"))
			},
			checkResponse: func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.True(t, len(installAppGroupsRequest.AppGroups) > len(res.Ids))
			},
		},
		{
			name: "SERVICEMESH_CALL_WORKFLOW",
			in: &pb.InstallAppGroupsRequest{
				AppGroups: []*pb.AppGroup{
					{
						AppGroupId:    uuid.New().String(),
						AppGroupName:  randomString("APPGROUP"),
						Type:          pb.AppGroupType_SERVICE_MESH,
						ClusterId:     uuid.New().String(),
						Status:        pb.AppGroupStatus_APP_GROUP_UNSPECIFIED,
						ExternalLabel: randomString("EXTERNAL_LABEL"),
					},
				},
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockAppInfoClient *mocktks.MockAppInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient) {
				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.GetClusterResponse{}, nil)

				mockAppInfoClient.EXPECT().CreateAppGroup(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.IDResponse{Id: createdAppGroupId}, nil)

				mockAppInfoClient.EXPECT().GetAppGroupsByClusterID(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.GetAppGroupsResponse{Code: pb.Code_OK_UNSPECIFIED}, errors.New("NOT_EXISTED_APPGROUPS"))

				mockAppInfoClient.EXPECT().UpdateAppGroupStatus(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.SimpleResponse{Code: pb.Code_OK_UNSPECIFIED, Error: nil}, nil)

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), "tks-service-mesh", gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), nil)
			},
			checkResponse: func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.True(t, len(installAppGroupsRequest.AppGroups) == len(res.Ids))
			},
		},
		{
			name: "FAILED_TO_CALL_WORKFLOW",
			in:   installAppGroupsRequest,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockAppInfoClient *mocktks.MockAppInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient) {
				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.GetClusterResponse{}, nil)

				mockAppInfoClient.EXPECT().CreateAppGroup(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.IDResponse{Id: createdAppGroupId}, nil)

				mockAppInfoClient.EXPECT().GetAppGroupsByClusterID(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.GetAppGroupsResponse{Code: pb.Code_OK_UNSPECIFIED}, errors.New("NOT_EXISTED_APPGROUPS"))

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
					Return("", errors.New("FAILED_TO_CALL_WORKFLOW"))
			},
			checkResponse: func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.Equal(t, len(res.Ids), 0)
			},
		},
	}
	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// mocking and injection
			mockArgoClient := mockargo.NewMockClient(ctrl)
			argowfClient = mockArgoClient

			mockAppInfoClient := mocktks.NewMockAppInfoServiceClient(ctrl)
			appInfoClient = mockAppInfoClient
			mockClusterInfoClient := mocktks.NewMockClusterInfoServiceClient(ctrl)
			clusterInfoClient = mockClusterInfoClient

			tc.buildStubs(mockArgoClient, mockAppInfoClient, mockClusterInfoClient)

			s := server{}
			res, err := s.InstallAppGroups(ctx, tc.in)
			tc.checkResponse(tc.in, res, err)
		})
	}
}

func TestUninstallAppGroups(t *testing.T) {
	testCases := []struct {
		name          string
		in            *pb.UninstallAppGroupsRequest
		buildStubs    func(mockArgoClient *mockargo.MockClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient)
		checkResponse func(req *pb.UninstallAppGroupsRequest, res *pb.IDsResponse, err error)
	}{
		{
			name: "OK",
			in: &pb.UninstallAppGroupsRequest{
				AppGroupIds: []string{createdAppGroupId},
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient) {
				mockAppInfoClient.EXPECT().GetAppGroup(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetAppGroupResponse{
							Code:     pb.Code_OK_UNSPECIFIED,
							Error:    nil,
							AppGroup: installAppGroupsRequest.GetAppGroups()[0],
						}, nil)

				mockAppInfoClient.EXPECT().UpdateAppGroupStatus(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.SimpleResponse{Code: pb.Code_OK_UNSPECIFIED, Error: nil}, nil)

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), nil)
			},
			checkResponse: func(req *pb.UninstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.Equal(t, len(res.Ids), 1)
				require.Equal(t, createdAppGroupId, res.Ids[0])
			},
		},
		{
			name: "INVALID_ARGUMENT_APPGROUP_ID",
			in: &pb.UninstallAppGroupsRequest{
				AppGroupIds: []string{"THIS_IS_NOT_UUID"},
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient) {
			},
			checkResponse: func(req *pb.UninstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.Error(t, err)
				require.Equal(t, res.Code, pb.Code_INVALID_ARGUMENT)
			},
		},
		{
			name: "NOT_EXISTED_APPGROUP",
			in: &pb.UninstallAppGroupsRequest{
				AppGroupIds: []string{uuid.New().String()},
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient) {
				mockAppInfoClient.EXPECT().GetAppGroup(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetAppGroupResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, errors.New("NOT_EXISTED_APPGROUP"))
			},
			checkResponse: func(req *pb.UninstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.Equal(t, len(res.Ids), 0)
			},
		},
		{
			name: "FAILED_TO_CALL_WORKFLOW",
			in: &pb.UninstallAppGroupsRequest{
				AppGroupIds: []string{createdAppGroupId},
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient) {
				mockAppInfoClient.EXPECT().GetAppGroup(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetAppGroupResponse{
							Code:     pb.Code_OK_UNSPECIFIED,
							Error:    nil,
							AppGroup: installAppGroupsRequest.GetAppGroups()[0],
						}, nil)

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
					Return("", errors.New("FAILED_TO_CALL_WORKFLOW"))
			},
			checkResponse: func(req *pb.UninstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.Equal(t, len(res.Ids), 0)
			},
		},
		{
			name: "PARTIALLY_SUCCESS",
			in: &pb.UninstallAppGroupsRequest{
				AppGroupIds: []string{createdAppGroupId, uuid.New().String()},
			},
			buildStubs: func(mockArgoClient *mockargo.MockClient, mockAppInfoClient *mocktks.MockAppInfoServiceClient) {
				mockAppInfoClient.EXPECT().GetAppGroup(gomock.Any(), &pb.GetAppGroupRequest{AppGroupId: createdAppGroupId}).Times(1).
					Return(
						&pb.GetAppGroupResponse{
							Code:     pb.Code_OK_UNSPECIFIED,
							Error:    nil,
							AppGroup: installAppGroupsRequest.GetAppGroups()[0],
						}, nil)

				mockAppInfoClient.EXPECT().GetAppGroup(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.GetAppGroupResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
						}, errors.New("NOT_EXISTED_APPGROUP"))

				mockAppInfoClient.EXPECT().UpdateAppGroupStatus(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.SimpleResponse{Code: pb.Code_OK_UNSPECIFIED, Error: nil}, nil)

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), nil)
			},
			checkResponse: func(req *pb.UninstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.Equal(t, len(res.Ids), 1)
				require.Equal(t, createdAppGroupId, res.Ids[0])
			},
		},
	}
	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// mocking and injection
			mockArgoClient := mockargo.NewMockClient(ctrl)
			argowfClient = mockArgoClient

			mockAppInfoClient := mocktks.NewMockAppInfoServiceClient(ctrl)
			appInfoClient = mockAppInfoClient

			tc.buildStubs(mockArgoClient, mockAppInfoClient)

			s := server{}
			res, err := s.UninstallAppGroups(ctx, tc.in)
			tc.checkResponse(tc.in, res, err)
		})
	}

}

// Helpers

func randomString(prefix string) string {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	return fmt.Sprintf("%s-%d", prefix, r.Int31n(1000000000))
}

func randomCreateClusterRequest() *pb.CreateClusterRequest {
	return &pb.CreateClusterRequest{
		ContractId: uuid.New().String(),
		CspId:      uuid.New().String(),
		Name:       randomString("NAME"),
		Conf: &pb.ClusterRawConf{
			SshKeyName:      randomString("SSHKEYNAME"),
			Region:          "ap-northeast-2",
			NumOfAz:         3,
			MachineType:     randomString("MACHINETYPE"),
			MachineReplicas: 3,
		},
	}
}

func randomInstallAppGroupsRequest() *pb.InstallAppGroupsRequest {
	return &pb.InstallAppGroupsRequest{
		AppGroups: []*pb.AppGroup{
			{
				AppGroupId:    uuid.New().String(),
				AppGroupName:  randomString("APPGROUP"),
				Type:          pb.AppGroupType_LMA,
				ClusterId:     uuid.New().String(),
				Status:        pb.AppGroupStatus_APP_GROUP_RUNNING,
				ExternalLabel: randomString("EXTERNAL_LABEL"),
			},
		},
	}
}
