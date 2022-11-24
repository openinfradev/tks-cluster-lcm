module github.com/openinfradev/tks-cluster-lcm

go 1.16

require (
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/openinfradev/tks-common v0.0.0-20221124045547-fbf60e9529da
	github.com/openinfradev/tks-proto v0.0.6-0.20221117013032-f3e8aa863671
	github.com/stretchr/testify v1.7.0
	google.golang.org/genproto v0.0.0-20211013025323-ce878158c4d4 // indirect
)

replace github.com/openinfradev/tks-cluster-lcm => ./

//replace github.com/openinfradev/tks-contract => ../tks-contract
//replace github.com/openinfradev/tks-proto => ../tks-proto
//replace github.com/openinfradev/tks-info => ../tks-info
//replace github.com/openinfradev/tks-common => ../tks-common
