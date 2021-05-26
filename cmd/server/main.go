package main

import (
	"flag"
	"net"
	"strconv"

	"github.com/sktelecom/tks-contract/pkg/log"
	pb "github.com/sktelecom/tks-proto/pbgo"
	"google.golang.org/grpc"
)

var (
	port               int    = 9112
	enableMockup       bool   = false
	infoServiceAddress string = ""
	infoServicePort    int    = 9111
)

type server struct {
	pb.UnimplementedClusterLcmServiceServer
}

func init() {
	setFlags()
}

func setFlags() {
	flag.IntVar(&port, "port", 9112, "service port")
	flag.StringVar(&infoServiceAddress, "info-address", "", "service address for tks-info")
	flag.IntVar(&infoServicePort, "info-port", 9111, "service port for tks-info")
}
func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	log.Info("Starting to listen port ", port)
	if err != nil {
		log.Fatal("an error failed to listen : ", err)
	}

	s := grpc.NewServer()
	pb.RegisterClusterLcmServiceServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatal("failed to serve: ", err)
	}
}
