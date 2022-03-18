package main

import (
	"flag"
	"os"

	"github.com/openinfradev/tks-common/pkg/log"
	"github.com/openinfradev/tks-common/pkg/argowf"
	"github.com/openinfradev/tks-common/pkg/grpc_client"
	"github.com/openinfradev/tks-common/pkg/grpc_server"
	pb "github.com/openinfradev/tks-proto/tks_pb"

)

type server struct {
	pb.UnimplementedClusterLcmServiceServer
}

var (
	argowfClient      argowf.Client
	contractClient    pb.ContractServiceClient
	cspInfoClient     pb.CspInfoServiceClient
	clusterInfoClient pb.ClusterInfoServiceClient
	appInfoClient     pb.AppInfoServiceClient
)


var (
	port            int
	tls                bool
	tlsClientCertPath  string
	tlsCertPath        string
	tlsKeyPath         string

	contractAddress string
	contractPort    int
	infoAddress     string
	infoPort        int
	argoAddress     string
	argoPort        int
	revision        string
	gitAccount      string
	gitToken        string
)

func init() {
	flag.IntVar(&port, "port", 9112, "service port")
	flag.BoolVar(&tls, "tls", false, "enabled tls")
	flag.StringVar(&tlsClientCertPath, "tls-client-cert-path", "../../cert/tks-ca.crt", "path of ca cert file for tls")
	flag.StringVar(&tlsCertPath, "tls-cert-path", "../../cert/tks-server.crt", "path of cert file for tls")
	flag.StringVar(&tlsKeyPath, "tls-key-path", "../../cert/tks-server.key", "path of key file for tls")
	flag.StringVar(&contractAddress, "contract-address", "localhost", "service address for tks-contract")
	flag.IntVar(&contractPort, "contract-port", 9110, "service port for tks-contract")
	flag.StringVar(&infoAddress, "info-address", "localhost", "service address for tks-info")
	flag.IntVar(&infoPort, "info-port", 9111, "service port for tks-info")
	flag.StringVar(&argoAddress, "argo-address", "192.168.70.10", "server address for argo-workflow-server")
	flag.IntVar(&argoPort, "argo-port", 2746, "server port for argo-workflow-server")
	flag.StringVar(&revision, "revision", "main", "revision for workflow parameter")
	flag.StringVar(&gitAccount, "repo-name", "tks-management", "git repository name for workflow parameter")

	gitToken = os.Getenv("TOKEN")
}

func main() {
	flag.Parse()

	log.Info("*** Arguments *** ")
	log.Info("tls : ", tls)
	log.Info("tlsClientCertPath : ", tlsClientCertPath)
	log.Info("tlsCertPath : ", tlsCertPath)
	log.Info("tlsKeyPath : ", tlsKeyPath)
	log.Info("contractAddress : ", contractAddress)
	log.Info("contractPort : ", contractPort)
	log.Info("infoAddress : ", infoAddress)
	log.Info("infoPort : ", infoPort)
	log.Info("argoAddress : ", argoAddress)
	log.Info("argoPort : ", argoPort)
	log.Info("revision : ", revision)
	log.Info("gitAccount : ", gitAccount)
	log.Info("****************** ")

	if gitToken = os.Getenv("TOKEN"); gitToken == "" {
		log.Fatal("Specify gitToken to environment variable (TOKEN).")
	}

	// initialize handlers
	var err error
	argowfClient, err = argowf.New(argoAddress, argoPort)
	if err != nil {
		log.Fatal("failed to create argowf client : ", err)
	}

	if _, contractClient, err = grpc_client.CreateContractClient(contractAddress, contractPort, tls, tlsClientCertPath); err != nil {
		log.Fatal("failed to create contract client : ", err)
	}

	if _, cspInfoClient, err = grpc_client.CreateCspInfoClient(infoAddress, infoPort, tls, tlsClientCertPath); err != nil {
		log.Fatal("failed to create cspinfo client : ", err)
	}

	if _, clusterInfoClient, err = grpc_client.CreateClusterInfoClient(infoAddress, infoPort, tls, tlsClientCertPath); err != nil {
		log.Fatal("failed to create cluster client : ", err)
	}

	if _, appInfoClient, err = grpc_client.CreateAppInfoClient(infoAddress, infoPort, tls, tlsClientCertPath); err != nil {
		log.Fatal("failed to create appinfo client : ", err)
	}


	// start server
	s, conn, err := grpc_server.CreateServer(port, tls, tlsCertPath, tlsKeyPath)
	if err != nil {
		log.Fatal("failed to crate grpc_server : ", err)
	}

	pb.RegisterClusterLcmServiceServer(s, &server{})
	if err := s.Serve(conn); err != nil {
		log.Fatal("failed to serve: ", err)
	}
}
