package main

import (
	"flag"
	"net"
	"strconv"

	"github.com/openinfradev/tks-contract/pkg/log"
	pb "github.com/openinfradev/tks-proto/pbgo"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 9112, "service port")
	contractAddress = flag.String("contract-address", "localhost", "service address for tks-contract")
	contractPort = flag.Int("contract-port", 9110, "service port for tks-contract")
	infoAddress = flag.String("info-address", "localhost", "service address for tks-info")
	infoPort = flag.Int("info-port", 9111, "service port for tks-info")
	argoAddress = flag.String("argo-address", "192.168.70.10", "server address for argo-workflow-server")
	argoPort = flag.Int("argo-port", 2746, "server port for argo-workflow-server")
)

type server struct {
	pb.UnimplementedClusterLcmServiceServer
}

func main() {
	log.Info("tks-cluster-lcm server is starting...")
	flag.Parse()

	log.Info( "*** Connection Addresses *** " )
	log.Info( "contractAddress : ", *contractAddress )
	log.Info( "contractPort : ", *contractPort )
	log.Info( "infoAddress : ", *infoAddress )
	log.Info( "infoPort : ", *infoPort )
	log.Info( "argoAddress : ", *argoAddress )
	log.Info( "argoPort : ", *argoPort )

	lis, err := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if err != nil {
		log.Fatal("an error failed to listen : ", err)
	}
	s := grpc.NewServer()

	log.Info("Started to listen port ", *port)
	log.Info("****************************")

	InitHandlers( *contractAddress, *contractPort, *infoAddress, *infoPort, *argoAddress, *argoPort )

	pb.RegisterClusterLcmServiceServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatal("failed to serve: ", err)
	}
}
