package main

import (
	"flag"
	"net"
	"os"
	"strconv"

	"github.com/openinfradev/tks-common/pkg/log"
	pb "github.com/openinfradev/tks-proto/tks_pb"
	"google.golang.org/grpc"
)

var (
	port            int
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

type server struct {
	pb.UnimplementedClusterLcmServiceServer
}

func init() {
	flag.IntVar(&port, "port", 9112, "service port")
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
	log.Info("tks-cluster-lcm server is starting...")
	flag.Parse()

	if gitToken == "" {
		log.Fatal("Specify gitToken to environment variable (TOKEN).")
	}

	log.Info("*** Connection Addresses *** ")
	log.Info("contractAddress : ", contractAddress)
	log.Info("contractPort : ", contractPort)
	log.Info("infoAddress : ", infoAddress)
	log.Info("infoPort : ", infoPort)
	log.Info("argoAddress : ", argoAddress)
	log.Info("argoPort : ", argoPort)
	log.Info("revision : ", revision)
	log.Info("gitAccount : ", gitAccount)

	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatal("an error failed to listen : ", err)
	}
	s := grpc.NewServer()

	log.Info("Started to listen port ", port)
	log.Info("****************************")

	InitHandlers(contractAddress, contractPort, infoAddress, infoPort, argoAddress, argoPort)

	pb.RegisterClusterLcmServiceServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatal("failed to serve: ", err)
	}
}
