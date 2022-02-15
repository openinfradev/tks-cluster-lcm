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

	// for CreateCluster API
	installAppGroupsRequest = randomInstallAppGroupsRequest()
	createClusterRequest = randomCreateClusterRequest()
}

func TestCreateCluster(t *testing.T) {
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
				//log.Info( res )
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
				//log.Info( res )
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

				mockAppInfoClient.EXPECT().CreateAppGroup(gomock.Any(), gomock.Any()).Times(1).
					Return(
						&pb.IDResponse{
							Code:  pb.Code_OK_UNSPECIFIED,
							Error: nil,
							Id:    createdAppGroupId,
						}, nil)

				maxCallCnt := 4
				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(maxCallCnt).
					Return(randomString("workflowName"), nil)
			},
			checkResponse: func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.Equal(t, len(installAppGroupsRequest.AppGroups), len(res.Ids))
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
			name: "LMA_CALL_WORKFLOW_BY_4_TIMES",
			in:   installAppGroupsRequest,
			buildStubs: func(mockArgoClient *mockargo.MockClient,
				mockAppInfoClient *mocktks.MockAppInfoServiceClient,
				mockClusterInfoClient *mocktks.MockClusterInfoServiceClient) {
				mockClusterInfoClient.EXPECT().GetCluster(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.GetClusterResponse{}, nil)

				mockAppInfoClient.EXPECT().CreateAppGroup(gomock.Any(), gomock.Any()).Times(1).
					Return(&pb.IDResponse{Id: createdAppGroupId}, nil)

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), "tks-lma-federation", gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), nil)
				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), "cp-aws-infrastructure", gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), nil)
				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), "setup-sealed-secrets-on-usercluster", gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), nil)
				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), "tks-install-ingress-controller", gomock.Any(), gomock.Any()).Times(1).
					Return(randomString("workflowName"), nil)
			},
			checkResponse: func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_OK_UNSPECIFIED)
				require.True(t, len(installAppGroupsRequest.AppGroups) == len(res.Ids))
			},
		},
		{
			name: "SERVICEMESH_CALL_WORKFLOW_BY_1_TIME",
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

				mockArgoClient.EXPECT().SumbitWorkflowFromWftpl(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).
					Return("", errors.New("FAILED_TO_CALL_WORKFLOW"))
			},
			checkResponse: func(req *pb.InstallAppGroupsRequest, res *pb.IDsResponse, err error) {
				require.NoError(t, err)
				require.Equal(t, res.Code, pb.Code_INTERNAL)
				require.True(t, res.Error != nil)
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
		Conf: &pb.ClusterConf{
			MasterFlavor:   randomString("MASTERFLAVOR"),
			MasterReplicas: 3,
			MasterRootSize: 30,
			WorkerFlavor:   randomString("WORKERFLAVOR"),
			WorkerReplicas: 3,
			WorkerRootSize: 30,
			K8SVersion:     "1.21",
			Region:         randomString("REGION"),
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
				Status:        pb.AppGroupStatus_APP_GROUP_UNSPECIFIED,
				ExternalLabel: randomString("EXTERNAL_LABEL"),
			},
		},
	}
}
